package report

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func sampleReport() *Report {
	r := New("Test Report", "host1", time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC))
	r.Add("Client", [][2]string{{"Public IP", "1.2.3.4"}, {"Browser", "Chrome"}})
	r.Add("Network", [][2]string{{"Hostname", "host1"}})
	return r
}

func TestJSONRoundTrip(t *testing.T) {
	b, err := sampleReport().JSON()
	if err != nil {
		t.Fatal(err)
	}
	var back Report
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if back.Title != "Test Report" || len(back.Sections) != 2 {
		t.Errorf("unexpected report: %+v", back)
	}
}

func TestHTMLContainsData(t *testing.T) {
	b, err := sampleReport().HTML()
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	for _, want := range []string{"Test Report", "Public IP", "1.2.3.4", "Chrome", "host1"} {
		if !strings.Contains(s, want) {
			t.Errorf("HTML missing %q", want)
		}
	}
}

func TestPDFMagic(t *testing.T) {
	b, err := sampleReport().PDF()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(b, []byte("%PDF")) {
		t.Errorf("output does not look like a PDF: %q", b[:min(8, len(b))])
	}
}

func TestQRPNG(t *testing.T) {
	b, err := QRPNG("http://netinsight/api/report", 128)
	if err != nil {
		t.Fatal(err)
	}
	// PNG signature
	if !bytes.HasPrefix(b, []byte{0x89, 'P', 'N', 'G'}) {
		t.Error("output is not a PNG")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
