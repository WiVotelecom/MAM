// Package connectivity runs reachability probes against admin-defined targets.
// Probes are deliberately agentless and server-side: the NetInsight container
// performs the checks so a help-desk / NOC operator gets a full picture without
// installing anything on the client.
package connectivity

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

// DefaultPorts maps a probe type to its well-known TCP port. Used when a target
// does not specify an explicit port.
var DefaultPorts = map[string]int{
	"tcp":      0,
	"dns":      53,
	"ldap":     389,
	"ldaps":    636,
	"https":    443,
	"http":     80,
	"ssh":      22,
	"rdp":      3389,
	"winrm":    5985,
	"smtp":     25,
	"smtps":    465,
	"imap":     143,
	"imaps":    993,
	"pop3":     110,
	"sql":      1433,
	"mssql":    1433,
	"oracle":   1521,
	"postgres": 5432,
	"smb":      445,
	"kerberos": 88,
	"ntp":      123,
}

// Result is the outcome of a single probe.
type Result struct {
	Name    string  `json:"name"`
	Host    string  `json:"host"`
	Port    int     `json:"port"`
	Type    string  `json:"type"`
	OK      bool    `json:"ok"`
	Latency float64 `json:"latency_ms"`
	Detail  string  `json:"detail"`
}

// Dialer abstracts net.Dialer so probes can be unit-tested without real
// network access.
type Dialer interface {
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
}

// Resolver abstracts DNS lookups for testing.
type Resolver interface {
	LookupHost(ctx context.Context, host string) ([]string, error)
}

// Prober executes connectivity probes using the supplied dialer/resolver.
type Prober struct {
	Dialer   Dialer
	Resolver Resolver
	Timeout  time.Duration
}

// NewProber returns a Prober backed by the real network stack.
func NewProber(timeout time.Duration) *Prober {
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	return &Prober{
		Dialer:   &net.Dialer{Timeout: timeout},
		Resolver: net.DefaultResolver,
		Timeout:  timeout,
	}
}

// ResolvePort returns the effective port for a target, applying the well-known
// default for the probe type when no explicit port is set.
func ResolvePort(typ string, port int) int {
	if port > 0 {
		return port
	}
	return DefaultPorts[strings.ToLower(typ)]
}

// Probe checks a single target and returns a Result. It never returns an error;
// failures are captured in Result.OK/Detail so callers can render a dashboard.
func (p *Prober) Probe(ctx context.Context, name, host, typ string, port int) Result {
	typ = strings.ToLower(strings.TrimSpace(typ))
	if typ == "" {
		typ = "tcp"
	}
	effPort := ResolvePort(typ, port)
	res := Result{Name: name, Host: host, Port: effPort, Type: typ}

	start := time.Now()
	if typ == "dns" {
		res.OK, res.Detail = p.probeDNS(ctx, host)
		res.Latency = msSince(start)
		return res
	}

	if effPort <= 0 {
		res.OK = false
		res.Detail = "no port specified and no default known for type " + typ
		return res
	}

	addr := net.JoinHostPort(host, strconv.Itoa(effPort))
	conn, err := p.Dialer.DialContext(ctx, "tcp", addr)
	res.Latency = msSince(start)
	if err != nil {
		res.OK = false
		res.Detail = err.Error()
		return res
	}
	_ = conn.Close()
	res.OK = true
	res.Detail = fmt.Sprintf("connected to %s in %.0fms", addr, res.Latency)
	return res
}

func (p *Prober) probeDNS(ctx context.Context, host string) (bool, string) {
	addrs, err := p.Resolver.LookupHost(ctx, host)
	if err != nil {
		return false, err.Error()
	}
	if len(addrs) == 0 {
		return false, "no records returned"
	}
	return true, "resolved: " + strings.Join(addrs, ", ")
}

// ProbeAll runs the given targets concurrently and returns their results in the
// same order as the input.
func (p *Prober) ProbeAll(ctx context.Context, targets []Target) []Result {
	results := make([]Result, len(targets))
	type job struct {
		idx int
		t   Target
	}
	ch := make(chan job)
	done := make(chan struct{})
	workers := 8
	if len(targets) < workers {
		workers = len(targets)
	}
	for i := 0; i < workers; i++ {
		go func() {
			for j := range ch {
				results[j.idx] = p.Probe(ctx, j.t.Name, j.t.Host, j.t.Type, j.t.Port)
			}
			done <- struct{}{}
		}()
	}
	for i, t := range targets {
		ch <- job{idx: i, t: t}
	}
	close(ch)
	for i := 0; i < workers; i++ {
		<-done
	}
	return results
}

// Target is the minimal shape needed to run a probe.
type Target struct {
	Name string
	Host string
	Port int
	Type string
}

func msSince(t time.Time) float64 {
	return float64(time.Since(t).Microseconds()) / 1000.0
}
