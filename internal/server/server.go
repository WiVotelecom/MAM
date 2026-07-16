// Package server wires the NetInsight HTTP API and serves the embedded
// frontend. All endpoints are designed to work without internet access; the
// only optional online lookup is the public IP, which fails gracefully.
package server

import (
	"context"
	"encoding/json"
	"io"
	"io/fs"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/WiVotelecom/MAM/internal/certs"
	"github.com/WiVotelecom/MAM/internal/connectivity"
	"github.com/WiVotelecom/MAM/internal/report"
	"github.com/WiVotelecom/MAM/internal/speed"
	"github.com/WiVotelecom/MAM/internal/store"
	"github.com/WiVotelecom/MAM/internal/sysinfo"
	"github.com/WiVotelecom/MAM/internal/uaparse"
)

// Config controls server behaviour.
type Config struct {
	// PublicIPURL is queried (best-effort, short timeout) to discover the
	// public IP. Empty disables the lookup entirely (pure air-gap mode).
	PublicIPURL string
	// ProbeTimeout bounds each connectivity/certificate probe.
	ProbeTimeout time.Duration
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		PublicIPURL:  "https://api.ipify.org",
		ProbeTimeout: 3 * time.Second,
	}
}

// Server holds dependencies shared across handlers.
type Server struct {
	store  *store.Store
	prober *connectivity.Prober
	cfg    Config
	assets fs.FS
	now    func() time.Time
}

// New constructs a Server.
func New(st *store.Store, assets fs.FS, cfg Config) *Server {
	return &Server{
		store:  st,
		prober: connectivity.NewProber(cfg.ProbeTimeout),
		cfg:    cfg,
		assets: assets,
		now:    time.Now,
	}
}

// Handler returns the fully-wired http.Handler.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/info", s.handleInfo)
	mux.HandleFunc("GET /api/network", s.handleNetwork)
	mux.HandleFunc("GET /api/browser", s.handleBrowser)
	mux.HandleFunc("GET /api/health", s.handleHealth)
	mux.HandleFunc("GET /api/report", s.handleReport)
	mux.HandleFunc("GET /api/qr", s.handleQR)
	mux.HandleFunc("GET /api/connectivity", s.handleConnectivity)
	mux.HandleFunc("GET /api/certificate", s.handleCertificate)

	mux.HandleFunc("GET /api/targets", s.handleListTargets)
	mux.HandleFunc("POST /api/targets", s.handleAddTarget)
	mux.HandleFunc("DELETE /api/targets/{id}", s.handleDeleteTarget)

	mux.HandleFunc("GET /api/speed/download", s.handleSpeedDownload)
	mux.HandleFunc("POST /api/speed/upload", s.handleSpeedUpload)

	mux.Handle("/", s.spaHandler())

	return withCommonHeaders(mux)
}

// ---- Client / info handlers ----

// ClientInfo describes the requesting client, derived from the HTTP request.
type ClientInfo struct {
	PublicIP     string          `json:"public_ip"`
	RemoteIP     string          `json:"remote_ip"`
	Hostname     string          `json:"server_hostname"`
	UserAgent    string          `json:"user_agent"`
	Browser      uaparse.Result  `json:"browser"`
	Time         time.Time       `json:"time"`
	Timezone     string          `json:"timezone"`
	Language     string          `json:"language"`
	Protocol     string          `json:"protocol"`
	HTTPS        bool            `json:"https"`
	TLSVersion   string          `json:"tls_version"`
	Forwarded    string          `json:"forwarded_for,omitempty"`
}

func (s *Server) clientInfo(r *http.Request) ClientInfo {
	ci := ClientInfo{
		RemoteIP:  remoteIP(r),
		Hostname:  sysinfo.Hostname(),
		UserAgent: r.UserAgent(),
		Browser:   uaparse.Parse(r.UserAgent()),
		Time:      s.now().UTC(),
		Timezone:  "UTC",
		Language:  r.Header.Get("Accept-Language"),
		Protocol:  r.Proto,
		HTTPS:     r.TLS != nil,
		Forwarded: r.Header.Get("X-Forwarded-For"),
	}
	if r.TLS != nil {
		ci.TLSVersion = certs.TLSVersionName(r.TLS.Version)
	}
	ci.PublicIP = s.lookupPublicIP(r.Context())
	return ci
}

func (s *Server) handleInfo(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.clientInfo(r))
}

func (s *Server) handleBrowser(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"user_agent": r.UserAgent(),
		"parsed":     uaparse.Parse(r.UserAgent()),
		"language":   r.Header.Get("Accept-Language"),
		"protocol":   r.Proto,
		"https":      r.TLS != nil,
	})
}

func (s *Server) handleNetwork(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, sysinfo.Collect())
}

// ---- Connectivity / health ----

func (s *Server) storedTargets() ([]connectivity.Target, error) {
	rows, err := s.store.ListTargets()
	if err != nil {
		return nil, err
	}
	out := make([]connectivity.Target, 0, len(rows))
	for _, t := range rows {
		if !t.Enabled {
			continue
		}
		out = append(out, connectivity.Target{Name: t.Name, Host: t.Host, Port: t.Port, Type: t.Type})
	}
	return out, nil
}

func (s *Server) handleConnectivity(w http.ResponseWriter, r *http.Request) {
	targets, err := s.storedTargets()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	results := s.prober.ProbeAll(r.Context(), targets)
	writeJSON(w, http.StatusOK, map[string]any{"results": results})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	targets, err := s.storedTargets()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	results := s.prober.ProbeAll(r.Context(), targets)
	up := 0
	for _, res := range results {
		if res.OK {
			up++
		}
	}
	status := "ok"
	if len(results) > 0 && up < len(results) {
		status = "degraded"
	}
	if len(results) > 0 && up == 0 {
		status = "down"
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":  status,
		"total":   len(results),
		"up":      up,
		"down":    len(results) - up,
		"checks":  results,
		"version": Version,
	})
}

// ---- Certificate ----

func (s *Server) handleCertificate(w http.ResponseWriter, r *http.Request) {
	host := r.URL.Query().Get("host")
	if host == "" {
		writeError(w, http.StatusBadRequest, "host query parameter is required")
		return
	}
	port, _ := strconv.Atoi(r.URL.Query().Get("port"))
	rep := certs.Inspect(host, port, s.cfg.ProbeTimeout, s.now)
	writeJSON(w, http.StatusOK, rep)
}

// ---- Targets CRUD ----

func (s *Server) handleListTargets(w http.ResponseWriter, r *http.Request) {
	targets, err := s.store.ListTargets()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if targets == nil {
		targets = []store.Target{}
	}
	writeJSON(w, http.StatusOK, targets)
}

func (s *Server) handleAddTarget(w http.ResponseWriter, r *http.Request) {
	var t store.Target
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&t); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	t.Enabled = true
	created, err := s.store.AddTarget(t)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (s *Server) handleDeleteTarget(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := s.store.DeleteTarget(id); err != nil {
		if err == store.ErrNotFound {
			writeError(w, http.StatusNotFound, "target not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---- Speed test ----

func (s *Server) handleSpeedDownload(w http.ResponseWriter, r *http.Request) {
	size, _ := strconv.ParseInt(r.URL.Query().Get("size"), 10, 64)
	size = speed.ClampSize(size)
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", strconv.FormatInt(size, 10))
	w.Header().Set("Cache-Control", "no-store")
	_, _ = io.Copy(w, speed.PayloadReader(size))
}

func (s *Server) handleSpeedUpload(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	n, err := io.Copy(io.Discard, io.LimitReader(r.Body, speed.MaxPayloadBytes))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	elapsed := time.Since(start)
	writeJSON(w, http.StatusOK, map[string]any{
		"bytes":      n,
		"elapsed_ms": float64(elapsed.Microseconds()) / 1000.0,
		"mbps":       speed.Mbps(n, elapsed),
	})
}

// ---- Report / QR ----

func (s *Server) buildReport(r *http.Request) *report.Report {
	ci := s.clientInfo(r)
	net := sysinfo.Collect()
	rep := report.New("NetInsight Diagnostics Report", net.Hostname, s.now().UTC())
	rep.Add("Client", [][2]string{
		{"Public IP", ci.PublicIP},
		{"Remote IP", ci.RemoteIP},
		{"User Agent", ci.UserAgent},
		{"Browser", ci.Browser.Browser},
		{"OS", ci.Browser.OS},
		{"Protocol", ci.Protocol},
		{"HTTPS", strconv.FormatBool(ci.HTTPS)},
		{"TLS Version", ci.TLSVersion},
		{"Language", ci.Language},
	})
	rep.Add("Network", [][2]string{
		{"Hostname", net.Hostname},
		{"Private IPv4", strings.Join(net.PrivateIPv4, ", ")},
		{"Private IPv6", strings.Join(net.PrivateIPv6, ", ")},
		{"DNS Servers", strings.Join(net.DNSServers, ", ")},
		{"DNS Suffix", strings.Join(net.DNSSuffix, ", ")},
		{"Default Gateway", net.DefaultGateway},
		{"Dual Stack", strconv.FormatBool(net.DualStack)},
	})
	if targets, err := s.storedTargets(); err == nil && len(targets) > 0 {
		results := s.prober.ProbeAll(r.Context(), targets)
		rows := make([][2]string, 0, len(results))
		for _, res := range results {
			state := "DOWN"
			if res.OK {
				state = "UP"
			}
			rows = append(rows, [2]string{res.Name, state})
		}
		rep.Add("Connectivity", rows)
	}
	return rep
}

func (s *Server) handleReport(w http.ResponseWriter, r *http.Request) {
	rep := s.buildReport(r)
	switch strings.ToLower(r.URL.Query().Get("format")) {
	case "", "json":
		b, err := rep.JSON()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(b)
	case "html":
		b, err := rep.HTML()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(b)
	case "pdf":
		b, err := rep.PDF()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", `attachment; filename="netinsight-report.pdf"`)
		_, _ = w.Write(b)
	default:
		writeError(w, http.StatusBadRequest, "unsupported format (use json, html, or pdf)")
	}
}

func (s *Server) handleQR(w http.ResponseWriter, r *http.Request) {
	content := r.URL.Query().Get("content")
	if content == "" {
		content = "http://" + r.Host + "/api/report?format=html"
	}
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))
	png, err := report.QRPNG(content, size)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Type", "image/png")
	_, _ = w.Write(png)
}

// ---- Public IP (optional online lookup) ----

func (s *Server) lookupPublicIP(ctx context.Context) string {
	if s.cfg.PublicIPURL == "" {
		return ""
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.cfg.PublicIPURL, nil)
	if err != nil {
		return ""
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return ""
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 128))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

// ---- Static SPA ----

func (s *Server) spaHandler() http.Handler {
	fileServer := http.FileServer(http.FS(s.assets))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Serve the file if it exists, otherwise fall back to index.html so
		// client-side routing works.
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}
		if _, err := fs.Stat(s.assets, path); err != nil {
			r2 := r.Clone(r.Context())
			r2.URL.Path = "/"
			http.ServeFileFS(w, r2, s.assets, "index.html")
			return
		}
		fileServer.ServeHTTP(w, r)
	})
}

// ---- helpers ----

func remoteIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		parts := strings.Split(fwd, ",")
		return strings.TrimSpace(parts[0])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func withCommonHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		next.ServeHTTP(w, r)
	})
}
