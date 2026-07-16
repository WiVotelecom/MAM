package certs

import (
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"net"
	"testing"
	"time"
)

func TestTLSVersionName(t *testing.T) {
	cases := map[uint16]string{
		tls.VersionTLS10: "TLS 1.0",
		tls.VersionTLS12: "TLS 1.2",
		tls.VersionTLS13: "TLS 1.3",
		0:                "",
	}
	for v, want := range cases {
		if got := TLSVersionName(v); got != want {
			t.Errorf("TLSVersionName(%d) = %q, want %q", v, got, want)
		}
	}
}

func TestDisplayName(t *testing.T) {
	if got := DisplayName("example.com", []string{"Example Inc"}); got != "CN=example.com, O=Example Inc" {
		t.Errorf("DisplayName = %q", got)
	}
	if got := DisplayName("", nil); got != "" {
		t.Errorf("empty DisplayName = %q, want empty", got)
	}
}

func TestSummarize(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	chain := []*x509.Certificate{
		{
			Subject:     pkix.Name{CommonName: "web.corp", Organization: []string{"Corp"}},
			Issuer:      pkix.Name{CommonName: "Corp Root CA"},
			DNSNames:    []string{"web.corp", "www.web.corp"},
			IPAddresses: []net.IP{net.ParseIP("10.0.0.5")},
			NotBefore:   now.AddDate(0, 0, -10),
			NotAfter:    now.AddDate(0, 0, 30),
			IsCA:        false,
		},
		{
			Subject:   pkix.Name{CommonName: "Corp Root CA"},
			Issuer:    pkix.Name{CommonName: "Corp Root CA"},
			NotBefore: now.AddDate(-1, 0, 0),
			NotAfter:  now.AddDate(9, 0, 0),
			IsCA:      true,
		},
	}
	out := Summarize(chain, now)
	if len(out) != 2 {
		t.Fatalf("got %d certs", len(out))
	}
	leaf := out[0]
	if leaf.Subject != "CN=web.corp, O=Corp" {
		t.Errorf("subject = %q", leaf.Subject)
	}
	if leaf.DaysLeft != 30 {
		t.Errorf("days left = %d, want 30", leaf.DaysLeft)
	}
	wantSAN := []string{"web.corp", "www.web.corp", "10.0.0.5"}
	if len(leaf.SAN) != len(wantSAN) {
		t.Fatalf("SAN = %v", leaf.SAN)
	}
	for i := range wantSAN {
		if leaf.SAN[i] != wantSAN[i] {
			t.Errorf("SAN[%d] = %q, want %q", i, leaf.SAN[i], wantSAN[i])
		}
	}
	if !out[1].IsCA {
		t.Error("second cert should be a CA")
	}
}
