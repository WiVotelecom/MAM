package server

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/WiVotelecom/MAM/internal/store"
)

func newTestServer(t *testing.T) *Server {
	t.Helper()
	st, err := store.Open(":memory:")
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	t.Cleanup(func() { st.Close() })

	assets := fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte("<html>NetInsight SPA</html>")},
		"assets/app.js": &fstest.MapFile{Data: []byte("console.log('hi')")},
	}
	cfg := DefaultConfig()
	cfg.PublicIPURL = "" // disable online lookup in tests
	s := New(st, assets, cfg)
	s.now = func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) }
	return s
}

func do(t *testing.T, h http.Handler, method, path string, body io.Reader) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, body)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0) Chrome/120 Safari/537.36")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestInfoEndpoint(t *testing.T) {
	h := newTestServer(t).Handler()
	rec := do(t, h, http.MethodGet, "/api/info", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var ci ClientInfo
	if err := json.Unmarshal(rec.Body.Bytes(), &ci); err != nil {
		t.Fatal(err)
	}
	if ci.Browser.Browser != "Chrome" {
		t.Errorf("browser = %q", ci.Browser.Browser)
	}
	if ci.Timezone != "UTC" {
		t.Errorf("timezone = %q", ci.Timezone)
	}
}

func TestBrowserAndNetwork(t *testing.T) {
	h := newTestServer(t).Handler()
	for _, path := range []string{"/api/browser", "/api/network"} {
		rec := do(t, h, http.MethodGet, path, nil)
		if rec.Code != http.StatusOK {
			t.Errorf("%s status = %d", path, rec.Code)
		}
		if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
			t.Errorf("%s content-type = %q", path, ct)
		}
	}
}

func TestTargetsCRUDAndHealth(t *testing.T) {
	h := newTestServer(t).Handler()

	body := strings.NewReader(`{"name":"DC01","host":"127.0.0.1","port":389,"type":"ldap"}`)
	rec := do(t, h, http.MethodPost, "/api/targets", body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status = %d body=%s", rec.Code, rec.Body.String())
	}
	var created store.Target
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.ID == 0 {
		t.Fatal("expected non-zero id")
	}

	rec = do(t, h, http.MethodGet, "/api/targets", nil)
	var list []store.Target
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("got %d targets", len(list))
	}

	// health should include one check
	rec = do(t, h, http.MethodGet, "/api/health", nil)
	var health map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &health); err != nil {
		t.Fatal(err)
	}
	if health["total"].(float64) != 1 {
		t.Errorf("health total = %v, want 1", health["total"])
	}

	// delete
	rec = do(t, h, http.MethodDelete, "/api/targets/"+itoa(created.ID), nil)
	if rec.Code != http.StatusNoContent {
		t.Errorf("delete status = %d", rec.Code)
	}
	rec = do(t, h, http.MethodDelete, "/api/targets/999", nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("delete missing status = %d, want 404", rec.Code)
	}
}

func TestAddTargetInvalid(t *testing.T) {
	h := newTestServer(t).Handler()
	rec := do(t, h, http.MethodPost, "/api/targets", strings.NewReader(`{"host":"x"}`))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestReportFormats(t *testing.T) {
	h := newTestServer(t).Handler()
	cases := []struct {
		format string
		ct     string
		prefix []byte
	}{
		{"json", "application/json", []byte("{")},
		{"html", "text/html; charset=utf-8", []byte("<!DOCTYPE html>")},
		{"pdf", "application/pdf", []byte("%PDF")},
	}
	for _, tc := range cases {
		rec := do(t, h, http.MethodGet, "/api/report?format="+tc.format, nil)
		if rec.Code != http.StatusOK {
			t.Errorf("%s status = %d", tc.format, rec.Code)
		}
		if ct := rec.Header().Get("Content-Type"); ct != tc.ct {
			t.Errorf("%s content-type = %q, want %q", tc.format, ct, tc.ct)
		}
		if !bytes.HasPrefix(rec.Body.Bytes(), tc.prefix) {
			t.Errorf("%s body prefix mismatch", tc.format)
		}
	}
	rec := do(t, h, http.MethodGet, "/api/report?format=xml", nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("bad format status = %d, want 400", rec.Code)
	}
}

func TestQR(t *testing.T) {
	h := newTestServer(t).Handler()
	rec := do(t, h, http.MethodGet, "/api/qr?content=hello", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "image/png" {
		t.Errorf("content-type = %q", ct)
	}
}

func TestSpeedDownloadAndUpload(t *testing.T) {
	h := newTestServer(t).Handler()
	rec := do(t, h, http.MethodGet, "/api/speed/download?size=2048", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("download status = %d", rec.Code)
	}
	if rec.Body.Len() != 2048 {
		t.Errorf("download size = %d, want 2048", rec.Body.Len())
	}

	rec = do(t, h, http.MethodPost, "/api/speed/upload", bytes.NewReader(make([]byte, 4096)))
	if rec.Code != http.StatusOK {
		t.Fatalf("upload status = %d", rec.Code)
	}
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out["bytes"].(float64) != 4096 {
		t.Errorf("upload bytes = %v, want 4096", out["bytes"])
	}
}

func TestSPAFallback(t *testing.T) {
	h := newTestServer(t).Handler()
	// existing asset
	rec := do(t, h, http.MethodGet, "/assets/app.js", nil)
	if rec.Code != http.StatusOK {
		t.Errorf("asset status = %d", rec.Code)
	}
	// unknown route falls back to index.html
	rec = do(t, h, http.MethodGet, "/some/spa/route", nil)
	if rec.Code != http.StatusOK {
		t.Errorf("spa fallback status = %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "NetInsight SPA") {
		t.Errorf("spa fallback body = %q", rec.Body.String())
	}
}

func itoa(v int64) string {
	return strconv.FormatInt(v, 10)
}
