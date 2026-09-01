package service

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/go-pdf/fpdf"
	"github.com/google/uuid"
)

// ReportService generates downloadable reports for scan jobs.
type ReportService interface {
	GenerateScanReport(ctx context.Context, scanJobID uuid.UUID) ([]byte, error)
}

// DefaultReportService is the default implementation of ReportService.
type DefaultReportService struct {
	scanService ScanService
}

// NewReportService creates a new report service.
func NewReportService(scanService ScanService) ReportService {
	return &DefaultReportService{scanService: scanService}
}

// cellPadding is the inner padding used inside table cells so wrapped text
// does not touch the cell borders.
const cellPadding = 1.0

// maxFloat returns the larger of a and b.
func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

// rowLineCount returns the number of lines a piece of text needs when wrapped
// to the given cell width using the currently selected font.
func rowLineCount(pdf *fpdf.Fpdf, text string, width float64) int {
	if text == "" {
		return 1
	}
	lines := pdf.SplitLines([]byte(text), width-2*cellPadding)
	if len(lines) == 0 {
		return 1
	}
	return len(lines)
}

// wrappedRowHeight calculates the height required to render a full table row
// with text wrapping, given each column width and the base line height.
func wrappedRowHeight(pdf *fpdf.Fpdf, cells []string, widths []float64, lineHeight float64) float64 {
	h := lineHeight
	for i, cell := range cells {
		lines := rowLineCount(pdf, cell, widths[i])
		h = maxFloat(h, float64(lines)*lineHeight)
	}
	return h
}

// printWrappedRow renders a table row. Each cell's text is wrapped to its
// column width and the whole row uses the tallest cell's height. A page break
// is inserted automatically if the row does not fit on the current page.
// When a page break occurs, onNewPage is called after AddPage so the caller can
// reprint table headers or restore styling.
// The caller is responsible for ensuring the font is already set.
func printWrappedRow(pdf *fpdf.Fpdf, cells []string, widths []float64, lineHeight float64, fill bool, pageLimit float64, onNewPage func()) {
	h := wrappedRowHeight(pdf, cells, widths, lineHeight)

	// If the row does not fit, start a new page and let the caller redraw headers.
	// Rows taller than a page cannot fit; in that edge case we render them anyway
	// to avoid an infinite AddPage loop.
	if h <= pageLimit-20 && pdf.GetY()+h > pageLimit {
		pdf.AddPage()
		if onNewPage != nil {
			onNewPage()
		}
	}

	xStart := pdf.GetX()
	y := pdf.GetY()
	x := xStart
	for i, cell := range cells {
		pdf.SetXY(x, y)
		pdf.CellFormat(widths[i], h, "", "1", 0, "L", fill, 0, "")
		pdf.SetXY(x+cellPadding, y+cellPadding)
		pdf.MultiCell(widths[i]-2*cellPadding, lineHeight, cell, "0", "L", false)
		x += widths[i]
	}
	pdf.SetXY(xStart, y+h)
}

// printWrappedHeader renders a table header row with centered text and a grey
// background. It handles page breaks just like printWrappedRow.
func printWrappedHeader(pdf *fpdf.Fpdf, headers []string, widths []float64, lineHeight float64, pageLimit float64) {
	pdf.SetFont("Helvetica", "B", 9)
	pdf.SetFillColor(230, 230, 230)
	printWrappedRow(pdf, headers, widths, lineHeight, true, pageLimit, nil)
}

// GenerateScanReport builds a PDF report for a scan job.
func (s *DefaultReportService) GenerateScanReport(ctx context.Context, scanJobID uuid.UUID) ([]byte, error) {
	detail, err := s.scanService.GetScanDetail(ctx, scanJobID, true)
	if err != nil {
		return nil, fmt.Errorf("load scan detail: %w", err)
	}

	pdf := fpdf.New("L", "mm", "A4", "")
	pdf.SetMargins(10, 10, 10)
	pdf.AddPage()

	const pageLimit = 185.0
	usableWidth := 297.0 - 20.0 // A4 landscape width minus left/right margins.

	// Title
	pdf.SetFont("Helvetica", "B", 18)
	pdf.Cell(0, 10, "Scan Report")
	pdf.Ln(12)

	// Metadata
	pdf.SetFont("Helvetica", "B", 12)
	pdf.Cell(0, 8, "Scan Job Details")
	pdf.Ln(10)

	labelWidth := 50.0
	valueWidth := usableWidth - labelWidth

	pdf.SetFont("Helvetica", "", 10)
	meta := [][]string{
		{"Job ID", detail.Job.ID.String()},
		{"AAP Job ID", detail.Job.AnsibleJobID},
		{"Template ID", fmt.Sprintf("%d", detail.Job.JobTemplateID)},
		{"Limit", detail.Job.Limit},
		{"Status", string(detail.Job.Status)},
		{"Created", detail.Job.CreatedAt.Format(time.RFC3339)},
		{"Successful Hosts", fmt.Sprintf("%d", detail.Job.SuccessfulHosts)},
		{"Failed Hosts", fmt.Sprintf("%d", detail.Job.FailedHosts)},
	}
	if detail.Job.CompletedAt != nil {
		meta = append(meta, []string{"Completed", detail.Job.CompletedAt.Format(time.RFC3339)})
	}
	if detail.Job.ErrorMessage != "" {
		meta = append(meta, []string{"Error", detail.Job.ErrorMessage})
	}

	for _, row := range meta {
		if pdf.GetY()+6 > pageLimit {
			pdf.AddPage()
		}
		x := pdf.GetX()
		y := pdf.GetY()
		pdf.SetFont("Helvetica", "B", 10)
		pdf.SetXY(x, y)
		pdf.Cell(labelWidth, 6, row[0]+":")
		pdf.SetFont("Helvetica", "", 10)
		pdf.SetXY(x+labelWidth, y)
		pdf.MultiCell(valueWidth, 6, row[1], "0", "L", false)
	}

	// Baselines
	if len(detail.Job.BaselineSnapshot) > 0 {
		pdf.Ln(4)
		pdf.SetFont("Helvetica", "B", 12)
		pdf.Cell(0, 8, "Active Baselines")
		pdf.Ln(10)

		headers := []string{"Scope", "OS Type", "File", "Version", "Entries", "Description"}
		widths := []float64{60, 35, 30, 25, 25, 82}
		lineHeight := 6.0

		printWrappedHeader(pdf, headers, widths, lineHeight, pageLimit)
		pdf.SetFont("Helvetica", "", 9)

		for _, b := range detail.Job.BaselineSnapshot {
			cells := []string{
				"Global",
				string(b.OSType),
				string(b.FileType),
				fmt.Sprintf("%d", b.Version),
				fmt.Sprintf("%d", b.EntryCount),
				b.Description,
			}
			printWrappedRow(pdf, cells, widths, lineHeight, false, pageLimit, func() {
				printWrappedHeader(pdf, headers, widths, lineHeight, pageLimit)
				pdf.SetFont("Helvetica", "", 9)
			})
		}
	}

	// Host results
	pdf.AddPage()
	pdf.SetFont("Helvetica", "B", 12)
	pdf.Cell(0, 8, "Host Results")
	pdf.Ln(10)

	hostHeaders := []string{"Hostname", "Status", "OS Type", "OS Version", "Baseline", "Environment", "Datacenter", "Deviations", "Allowed"}
	hostWidths := []float64{50, 25, 25, 25, 40, 25, 25, 20, 20}
	hostLineHeight := 6.0

	printHostHeader := func() {
		printWrappedHeader(pdf, hostHeaders, hostWidths, hostLineHeight, pageLimit)
		pdf.SetFont("Helvetica", "", 9)
	}
	printHostHeader()

	for _, r := range detail.Results {
		status := string(r.Status)
		deviations := fmt.Sprintf("%d", r.DeviationsFound)
		allowed := fmt.Sprintf("%d", len(r.AllowedDeviations))
		baseline := "-"
		if r.BaselineVersionAtScan != nil {
			baseline = fmt.Sprintf("v%d", *r.BaselineVersionAtScan)
		}
		if r.NoBaseline {
			baseline = "No baseline"
			status = "no_baseline"
			deviations = "-"
			allowed = "-"
		}
		if status == "failed_host" {
			deviations = "-"
			allowed = "-"
		}
		cells := []string{
			r.Hostname,
			status,
			r.OSType,
			r.OSVersion,
			baseline,
			r.Environment,
			r.Datacenter,
			deviations,
			allowed,
		}
		printWrappedRow(pdf, cells, hostWidths, hostLineHeight, false, pageLimit, func() {
			printWrappedHeader(pdf, hostHeaders, hostWidths, hostLineHeight, pageLimit)
			pdf.SetFont("Helvetica", "", 9)
		})
	}

	// Failed hosts that did not produce a scan result
	if len(detail.Job.FailedHostNames) > 0 {
		pdf.SetFillColor(255, 200, 200)
		for _, name := range detail.Job.FailedHostNames {
			cells := []string{name, "failed_host", "-", "-", "-", "-", "-", "-", "-"}
			printWrappedRow(pdf, cells, hostWidths, hostLineHeight, true, pageLimit, func() {
				printWrappedHeader(pdf, hostHeaders, hostWidths, hostLineHeight, pageLimit)
				pdf.SetFillColor(255, 200, 200)
			})
		}
	}

	// Incident details
	hostsWithIncidents := 0
	for _, r := range detail.Results {
		if len(r.Incidents) > 0 {
			hostsWithIncidents++
		}
	}
	if hostsWithIncidents > 0 {
		pdf.AddPage()
		pdf.SetFont("Helvetica", "B", 12)
		pdf.Cell(0, 8, "Incident Details")
		pdf.Ln(10)

		incHeaders := []string{"Number", "Entry", "Expected", "Actual", "Severity", "Status"}
		incWidths := []float64{40, 55, 55, 55, 30, 32}
		incLineHeight := 6.0

		printIncidentHeader := func() {
			printWrappedHeader(pdf, incHeaders, incWidths, incLineHeight, pageLimit)
			pdf.SetFont("Helvetica", "", 9)
		}

		for _, r := range detail.Results {
			if len(r.Incidents) == 0 {
				continue
			}
			if pdf.GetY()+18 > pageLimit {
				pdf.AddPage()
			}
			pdf.SetFont("Helvetica", "B", 10)
			pdf.Cell(0, 7, fmt.Sprintf("Host: %s", r.Hostname))
			pdf.Ln(8)
			printIncidentHeader()

			for _, inc := range r.Incidents {
				expected := "-"
				if inc.ExpectedValue != nil {
					expected = *inc.ExpectedValue
				}
				cells := []string{
					inc.IncidentNumber,
					inc.EntryKey,
					expected,
					inc.ActualValue,
					string(inc.Severity),
					string(inc.Status),
				}
				printWrappedRow(pdf, cells, incWidths, incLineHeight, false, pageLimit, func() {
					printIncidentHeader()
				})
			}
			pdf.Ln(4)
		}
	}

	// Allowed deviation details
	hostsWithAllowed := 0
	for _, r := range detail.Results {
		if len(r.AllowedDeviations) > 0 {
			hostsWithAllowed++
		}
	}
	if hostsWithAllowed > 0 {
		pdf.AddPage()
		pdf.SetFont("Helvetica", "B", 12)
		pdf.Cell(0, 8, "Allowed Deviations (Exceptions)")
		pdf.Ln(10)

		allowedHeaders := []string{"Hostname", "File", "Entry", "Expected", "Actual"}
		allowedWidths := []float64{55, 30, 55, 55, 55}
		allowedLineHeight := 6.0

		printAllowedHeader := func() {
			printWrappedHeader(pdf, allowedHeaders, allowedWidths, allowedLineHeight, pageLimit)
			pdf.SetFont("Helvetica", "", 9)
		}
		printAllowedHeader()

		for _, r := range detail.Results {
			for _, d := range r.AllowedDeviations {
				expected := d.ExpectedValue
				if expected == "" {
					expected = "-"
				}
				cells := []string{
					r.Hostname,
					string(d.FileType),
					d.EntryKey,
					expected,
					d.ActualValue,
				}
				printWrappedRow(pdf, cells, allowedWidths, allowedLineHeight, false, pageLimit, func() {
					printAllowedHeader()
				})
			}
		}
	}

	// Footer on each page
	pdf.SetFooterFunc(func() {
		pdf.SetY(-15)
		pdf.SetFont("Helvetica", "I", 8)
		pdf.SetTextColor(128, 128, 128)
		pdf.CellFormat(0, 10, fmt.Sprintf("Generated: %s | Page %d", time.Now().UTC().Format(time.RFC3339), pdf.PageNo()), "0", 0, "C", false, 0, "")
	})

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("render pdf: %w", err)
	}
	return buf.Bytes(), nil
}
