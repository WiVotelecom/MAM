package connectivity

import (
	"context"
	"errors"
	"net"
	"sort"
	"testing"
	"time"
)

// fakeDialer records dialed addresses and returns a configurable error.
type fakeDialer struct {
	fail    map[string]error
	dialed  []string
}

func (f *fakeDialer) DialContext(_ context.Context, _, address string) (net.Conn, error) {
	f.dialed = append(f.dialed, address)
	if err, ok := f.fail[address]; ok {
		return nil, err
	}
	c1, c2 := net.Pipe()
	_ = c2.Close()
	return c1, nil
}

type fakeResolver struct {
	hosts map[string][]string
}

func (f *fakeResolver) LookupHost(_ context.Context, host string) ([]string, error) {
	if addrs, ok := f.hosts[host]; ok {
		return addrs, nil
	}
	return nil, errors.New("no such host")
}

func TestResolvePort(t *testing.T) {
	if got := ResolvePort("https", 0); got != 443 {
		t.Errorf("https default port = %d, want 443", got)
	}
	if got := ResolvePort("ldap", 0); got != 389 {
		t.Errorf("ldap default port = %d, want 389", got)
	}
	if got := ResolvePort("tcp", 8443); got != 8443 {
		t.Errorf("explicit port = %d, want 8443", got)
	}
	if got := ResolvePort("unknown", 0); got != 0 {
		t.Errorf("unknown type port = %d, want 0", got)
	}
}

func TestProbeTCPSuccess(t *testing.T) {
	p := &Prober{Dialer: &fakeDialer{}, Timeout: time.Second}
	res := p.Probe(context.Background(), "DC01", "dc01.corp", "ldap", 0)
	if !res.OK {
		t.Fatalf("expected OK, got detail=%q", res.Detail)
	}
	if res.Port != 389 {
		t.Errorf("port = %d, want 389", res.Port)
	}
}

func TestProbeTCPFailure(t *testing.T) {
	fd := &fakeDialer{fail: map[string]error{"dc01.corp:389": errors.New("connection refused")}}
	p := &Prober{Dialer: fd, Timeout: time.Second}
	res := p.Probe(context.Background(), "DC01", "dc01.corp", "ldap", 0)
	if res.OK {
		t.Fatal("expected failure")
	}
	if res.Detail != "connection refused" {
		t.Errorf("detail = %q", res.Detail)
	}
}

func TestProbeNoPort(t *testing.T) {
	p := &Prober{Dialer: &fakeDialer{}, Timeout: time.Second}
	res := p.Probe(context.Background(), "X", "host", "tcp", 0)
	if res.OK {
		t.Fatal("expected failure when no port and no default")
	}
}

func TestProbeDNS(t *testing.T) {
	p := &Prober{
		Resolver: &fakeResolver{hosts: map[string][]string{"dc01.corp": {"10.0.0.1"}}},
		Timeout:  time.Second,
	}
	res := p.Probe(context.Background(), "dns", "dc01.corp", "dns", 0)
	if !res.OK {
		t.Fatalf("expected DNS OK, got %q", res.Detail)
	}

	res = p.Probe(context.Background(), "dns", "missing.corp", "dns", 0)
	if res.OK {
		t.Fatal("expected DNS failure for unknown host")
	}
}

func TestProbeAllOrdering(t *testing.T) {
	fd := &fakeDialer{fail: map[string]error{"b:22": errors.New("refused")}}
	p := &Prober{Dialer: fd, Timeout: time.Second}
	targets := []Target{
		{Name: "a", Host: "a", Type: "ssh"},
		{Name: "b", Host: "b", Type: "ssh"},
		{Name: "c", Host: "c", Type: "ssh"},
	}
	results := p.ProbeAll(context.Background(), targets)
	if len(results) != 3 {
		t.Fatalf("got %d results", len(results))
	}
	if results[0].Name != "a" || results[1].Name != "b" || results[2].Name != "c" {
		t.Errorf("results out of order: %+v", results)
	}
	if results[1].OK {
		t.Error("target b should have failed")
	}
	// ensure all addresses were attempted
	sort.Strings(fd.dialed)
	want := []string{"a:22", "b:22", "c:22"}
	if len(fd.dialed) != 3 {
		t.Fatalf("dialed = %v", fd.dialed)
	}
	for i := range want {
		if fd.dialed[i] != want[i] {
			t.Errorf("dialed[%d] = %q, want %q", i, fd.dialed[i], want[i])
		}
	}
}
