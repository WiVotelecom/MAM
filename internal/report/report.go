// Package report assembles a diagnostics report and renders it as JSON, HTML,
// or PDF, plus a QR code that encodes a link/payload for quick sharing.
package report

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"time"

	"github.com/go-pdf/fpdf"
	qrcode "github.com/skip2/go-qrcode"
)

// Section is a titled group of key/value rows in the report.
type Section struct {
	Title string     `json:"title"`
	Rows  [][2]string `json:"rows"`
}

// Report is the top-level diagnostics document.
type Report struct {
	Title       string    `json:"title"`
	GeneratedAt time.Time `json:"generated_at"`
	Hostname    string    `json:"hostname"`
	Sections    []Section `json:"sections"`
}

// New creates an empty report stamped with the current time.
func New(title, hostname string, now time.Time) *Report {
	return &Report{Title: title, Hostname: hostname, GeneratedAt: now}
}

// Add appends a section with the given title and rows.
func (r *Report) Add(title string, rows [][2]string) {
	r.Sections = append(r.Sections, Section{Title: title, Rows: rows})
}

// JSON renders the report as indented JSON.
func (r *Report) JSON() ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}

var htmlTmpl = template.Must(template.New("report").Parse(`<!DOCTYPE html>
<html lang="en"><head><meta charset="utf-8">
<title>{{.Title}}</title>
<style>
body{font-family:system-ui,Segoe UI,Arial,sans-serif;margin:2rem;color:#111;background:#f8fafc}
h1{font-size:1.5rem}
.meta{color:#555;margin-bottom:1.5rem}
section{background:#fff;border:1px solid #e2e8f0;border-radius:8px;padding:1rem 1.25rem;margin-bottom:1rem}
h2{font-size:1.05rem;margin:0 0 .5rem;border-bottom:1px solid #eee;padding-bottom:.35rem}
table{width:100%;border-collapse:collapse}
td{padding:.25rem .5rem;vertical-align:top}
td.k{color:#475569;width:220px;font-weight:600}
</style></head><body>
<h1>{{.Title}}</h1>
<div class="meta">Host: {{.Hostname}} &middot; Generated: {{.GeneratedAt.Format "2006-01-02 15:04:05 MST"}}</div>
{{range .Sections}}<section><h2>{{.Title}}</h2><table>
{{range .Rows}}<tr><td class="k">{{index . 0}}</td><td>{{index . 1}}</td></tr>{{end}}
</table></section>{{end}}
</body></html>`))

// HTML renders the report as a standalone HTML document.
func (r *Report) HTML() ([]byte, error) {
	var buf bytes.Buffer
	if err := htmlTmpl.Execute(&buf, r); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// PDF renders the report as a simple, printable PDF document.
func (r *Report) PDF() ([]byte, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(15, 15, 15)
	pdf.AddPage()

	pdf.SetFont("Helvetica", "B", 16)
	pdf.CellFormat(0, 10, r.Title, "", 1, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 9)
	pdf.SetTextColor(90, 90, 90)
	pdf.CellFormat(0, 6, fmt.Sprintf("Host: %s   Generated: %s", r.Hostname,
		r.GeneratedAt.Format("2006-01-02 15:04:05 MST")), "", 1, "L", false, 0, "")
	pdf.Ln(3)

	for _, sec := range r.Sections {
		pdf.SetTextColor(20, 20, 20)
		pdf.SetFont("Helvetica", "B", 12)
		pdf.CellFormat(0, 8, sec.Title, "", 1, "L", false, 0, "")
		pdf.SetFont("Helvetica", "", 10)
		for _, row := range sec.Rows {
			pdf.SetTextColor(70, 80, 100)
			pdf.CellFormat(55, 6, row[0], "", 0, "L", false, 0, "")
			pdf.SetTextColor(20, 20, 20)
			pdf.MultiCell(0, 6, row[1], "", "L", false)
		}
		pdf.Ln(2)
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// QRPNG encodes content into a PNG QR code of the given pixel size.
func QRPNG(content string, size int) ([]byte, error) {
	if size <= 0 {
		size = 256
	}
	return qrcode.Encode(content, qrcode.Medium, size)
}
