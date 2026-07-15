// Package sysinfo gathers host and network information from the machine
// running NetInsight. All lookups are local (interfaces, /etc/resolv.conf,
// /proc/net/route) so they work in fully isolated, air-gapped networks.
package sysinfo

import (
	"bufio"
	"encoding/binary"
	"net"
	"os"
	"sort"
	"strconv"
	"strings"
)

// NIC describes a single network interface.
type NIC struct {
	Name         string   `json:"name"`
	MAC          string   `json:"mac"`
	MTU          int      `json:"mtu"`
	Up           bool     `json:"up"`
	Addresses    []string `json:"addresses"`
	IsLoopback   bool     `json:"is_loopback"`
}

// Network is a snapshot of the host's networking configuration.
type Network struct {
	Hostname       string   `json:"hostname"`
	PrivateIPv4    []string `json:"private_ipv4"`
	PrivateIPv6    []string `json:"private_ipv6"`
	DNSServers     []string `json:"dns_servers"`
	DNSSuffix      []string `json:"dns_suffix"`
	DefaultGateway string   `json:"default_gateway"`
	NICs           []NIC    `json:"nics"`
	DualStack      bool     `json:"dual_stack"`
}

// Hostname returns the machine hostname, or "unknown" on error.
func Hostname() string {
	h, err := os.Hostname()
	if err != nil || h == "" {
		return "unknown"
	}
	return h
}

// Collect returns a full network snapshot of the host.
func Collect() Network {
	n := Network{Hostname: Hostname()}
	n.NICs = interfaces()
	for _, nic := range n.NICs {
		if nic.IsLoopback {
			continue
		}
		for _, a := range nic.Addresses {
			ip, _, err := net.ParseCIDR(a)
			if err != nil {
				ip = net.ParseIP(a)
			}
			if ip == nil {
				continue
			}
			if ip.To4() != nil {
				n.PrivateIPv4 = append(n.PrivateIPv4, ip.String())
			} else {
				n.PrivateIPv6 = append(n.PrivateIPv6, ip.String())
			}
		}
	}
	servers, suffix := parseResolvConf(readFile("/etc/resolv.conf"))
	n.DNSServers = servers
	n.DNSSuffix = suffix
	n.DefaultGateway = parseDefaultGatewayV4(readFile("/proc/net/route"))
	n.DualStack = len(n.PrivateIPv4) > 0 && len(n.PrivateIPv6) > 0
	sort.Strings(n.PrivateIPv4)
	sort.Strings(n.PrivateIPv6)
	return n
}

func interfaces() []NIC {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil
	}
	var out []NIC
	for _, iface := range ifaces {
		nic := NIC{
			Name:       iface.Name,
			MAC:        iface.HardwareAddr.String(),
			MTU:        iface.MTU,
			Up:         iface.Flags&net.FlagUp != 0,
			IsLoopback: iface.Flags&net.FlagLoopback != 0,
		}
		addrs, _ := iface.Addrs()
		for _, a := range addrs {
			nic.Addresses = append(nic.Addresses, a.String())
		}
		out = append(out, nic)
	}
	return out
}

func readFile(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(b)
}

// parseResolvConf extracts nameservers and search/domain suffixes from the
// contents of a resolv.conf file. Exported indirectly for testing.
func parseResolvConf(content string) (servers []string, suffix []string) {
	sc := bufio.NewScanner(strings.NewReader(content))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		switch fields[0] {
		case "nameserver":
			servers = append(servers, fields[1])
		case "search", "domain":
			suffix = append(suffix, fields[1:]...)
		}
	}
	return servers, suffix
}

// parseDefaultGatewayV4 parses the contents of /proc/net/route (Linux) and
// returns the IPv4 default gateway address, or "" if none is present.
func parseDefaultGatewayV4(content string) string {
	sc := bufio.NewScanner(strings.NewReader(content))
	first := true
	for sc.Scan() {
		if first { // header row
			first = false
			continue
		}
		fields := strings.Fields(sc.Text())
		if len(fields) < 4 {
			continue
		}
		// A default route has destination 00000000.
		if fields[1] != "00000000" {
			continue
		}
		gwHex := fields[2]
		v, err := strconv.ParseUint(gwHex, 16, 32)
		if err != nil {
			continue
		}
		// The route file stores the address little-endian.
		ip := make(net.IP, 4)
		binary.LittleEndian.PutUint32(ip, uint32(v))
		return ip.String()
	}
	return ""
}
