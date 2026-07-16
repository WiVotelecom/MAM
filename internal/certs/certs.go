// Package certs inspects TLS certificates presented by a remote endpoint so a
// help-desk operator can spot expiring or misconfigured certificates.
package certs

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

// CertInfo summarises a single certificate in the presented chain.
type CertInfo struct {
	Subject   string    `json:"subject"`
	Issuer    string    `json:"issuer"`
	SAN       []string  `json:"san"`
	NotBefore time.Time `json:"not_before"`
	NotAfter  time.Time `json:"not_after"`
	DaysLeft  int       `json:"days_left"`
	IsCA      bool      `json:"is_ca"`
}

// Report is the full result of inspecting an endpoint's TLS configuration.
type Report struct {
	Host       string     `json:"host"`
	Port       int        `json:"port"`
	TLSVersion string     `json:"tls_version"`
	Cipher     string     `json:"cipher"`
	Chain      []CertInfo `json:"chain"`
	OK         bool       `json:"ok"`
	Error      string     `json:"error,omitempty"`
}

// TLSVersionName converts a crypto/tls version constant into a display string.
func TLSVersionName(v uint16) string {
	switch v {
	case tls.VersionTLS10:
		return "TLS 1.0"
	case tls.VersionTLS11:
		return "TLS 1.1"
	case tls.VersionTLS12:
		return "TLS 1.2"
	case tls.VersionTLS13:
		return "TLS 1.3"
	case 0:
		return ""
	default:
		return fmt.Sprintf("0x%04x", v)
	}
}

// Inspect connects to host:port, performs a TLS handshake (without verifying
// the chain so self-signed/internal CAs still yield useful data) and returns a
// summary. The clock function is injectable for deterministic tests.
func Inspect(host string, port int, timeout time.Duration, now func() time.Time) Report {
	if now == nil {
		now = time.Now
	}
	if port <= 0 {
		port = 443
	}
	rep := Report{Host: host, Port: port}
	dialer := &net.Dialer{Timeout: timeout}
	conn, err := tls.DialWithDialer(dialer, "tcp", net.JoinHostPort(host, strconv.Itoa(port)), &tls.Config{
		InsecureSkipVerify: true, //nolint:gosec // we intentionally inspect untrusted/internal certs
		ServerName:         host,
	})
	if err != nil {
		rep.Error = err.Error()
		return rep
	}
	defer conn.Close()

	state := conn.ConnectionState()
	rep.TLSVersion = TLSVersionName(state.Version)
	rep.Cipher = tls.CipherSuiteName(state.CipherSuite)
	rep.Chain = Summarize(state.PeerCertificates, now())
	rep.OK = true
	return rep
}

// Summarize converts parsed x509 certificates into CertInfo records. It is kept
// separate from Inspect so it can be unit-tested with synthetic certificates.
func Summarize(chain []*x509.Certificate, now time.Time) []CertInfo {
	out := make([]CertInfo, 0, len(chain))
	for _, c := range chain {
		san := make([]string, 0, len(c.DNSNames)+len(c.IPAddresses))
		san = append(san, c.DNSNames...)
		for _, ip := range c.IPAddresses {
			san = append(san, ip.String())
		}
		out = append(out, CertInfo{
			Subject:   DisplayName(c.Subject.CommonName, c.Subject.Organization),
			Issuer:    DisplayName(c.Issuer.CommonName, c.Issuer.Organization),
			SAN:       san,
			NotBefore: c.NotBefore,
			NotAfter:  c.NotAfter,
			DaysLeft:  int(c.NotAfter.Sub(now).Hours() / 24),
			IsCA:      c.IsCA,
		})
	}
	return out
}

// DisplayName joins non-empty fields into a readable subject string.
func DisplayName(cn string, org []string) string {
	parts := []string{}
	if cn != "" {
		parts = append(parts, "CN="+cn)
	}
	if len(org) > 0 {
		parts = append(parts, "O="+strings.Join(org, ","))
	}
	return strings.Join(parts, ", ")
}
