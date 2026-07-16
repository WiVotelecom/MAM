package sysinfo

import (
	"reflect"
	"testing"
)

func TestParseResolvConf(t *testing.T) {
	content := `# comment line
; another comment
nameserver 10.0.0.1
nameserver 10.0.0.2
search corp.example.com example.com
domain fallback.local
`
	servers, suffix := parseResolvConf(content)
	wantServers := []string{"10.0.0.1", "10.0.0.2"}
	if !reflect.DeepEqual(servers, wantServers) {
		t.Errorf("servers = %v, want %v", servers, wantServers)
	}
	wantSuffix := []string{"corp.example.com", "example.com", "fallback.local"}
	if !reflect.DeepEqual(suffix, wantSuffix) {
		t.Errorf("suffix = %v, want %v", suffix, wantSuffix)
	}
}

func TestParseResolvConfEmpty(t *testing.T) {
	servers, suffix := parseResolvConf("")
	if len(servers) != 0 || len(suffix) != 0 {
		t.Errorf("expected empty results, got servers=%v suffix=%v", servers, suffix)
	}
}

func TestParseDefaultGatewayV4(t *testing.T) {
	// Iface Destination Gateway ... : default route (dest 00000000),
	// gateway 0100000A little-endian => 10.0.0.1
	content := `Iface	Destination	Gateway	Flags	RefCnt	Use	Metric	Mask	MTU	Window	IRTT
eth0	00000000	0100000A	0003	0	0	0	00000000	0	0	0
eth0	0000000A	00000000	0001	0	0	0	00FFFFFF	0	0	0
`
	got := parseDefaultGatewayV4(content)
	if got != "10.0.0.1" {
		t.Errorf("gateway = %q, want 10.0.0.1", got)
	}
}

func TestParseDefaultGatewayV4None(t *testing.T) {
	content := `Iface	Destination	Gateway	Flags	RefCnt	Use	Metric	Mask	MTU	Window	IRTT
eth0	0000000A	00000000	0001	0	0	0	00FFFFFF	0	0	0
`
	if got := parseDefaultGatewayV4(content); got != "" {
		t.Errorf("gateway = %q, want empty", got)
	}
}

func TestCollectDoesNotPanic(t *testing.T) {
	n := Collect()
	if n.Hostname == "" {
		t.Error("hostname should not be empty")
	}
}
