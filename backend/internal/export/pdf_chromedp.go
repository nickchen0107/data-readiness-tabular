package export

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"

	"github.com/safe-ai/excel-brushing-tool/pkg/config"
)

// GeneratePDFChromedp generates a PDF report by rendering an HTML template with chromedp.
// It writes an HTML file to disk and uses headless Chrome file:// protocol to render it.
func GeneratePDFChromedp(data *PDFReportData, cfg *config.Config, outputDir string) (string, error) {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("create output dir: %w", err)
	}

	isEn := data.Locale == "en"

	// Generate HTML content
	htmlContent := buildReportHTML(data, isEn)

	// Write HTML to file
	htmlPath := filepath.Join(outputDir, fmt.Sprintf("report_%s_%s.html", data.Session.ID.String()[:8], data.Locale))
	if err := os.WriteFile(htmlPath, []byte(htmlContent), 0644); err != nil {
		return "", fmt.Errorf("write HTML: %w", err)
	}

	// Use chromedp to generate PDF from file:// URL
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("allow-file-access-from-files", true),
	)

	// Use CHROME_PATH env var if set (Docker)
	if chromePath := os.Getenv("CHROME_PATH"); chromePath != "" {
		opts = append(opts, chromedp.ExecPath(chromePath))
	}

	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer allocCancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	absPath, _ := filepath.Abs(htmlPath)
	fileURL := "file://" + absPath

	var pdfBuf []byte
	err := chromedp.Run(ctx,
		chromedp.Navigate(fileURL),
		chromedp.WaitReady("body"),
		chromedp.Sleep(500*time.Millisecond),
		chromedp.ActionFunc(func(ctx context.Context) error {
			buf, _, err := page.PrintToPDF().
				WithPrintBackground(true).
				WithMarginTop(0.4).
				WithMarginBottom(0.4).
				WithMarginLeft(0.4).
				WithMarginRight(0.4).
				WithPaperWidth(8.27).
				WithPaperHeight(11.69).
				Do(ctx)
			if err != nil {
				return err
			}
			pdfBuf = buf
			return nil
		}),
	)
	if err != nil {
		// Fallback to fpdf if chromedp fails (e.g., no Chrome installed)
		return GeneratePDF(data, cfg, outputDir)
	}

	// Write PDF
	filename := fmt.Sprintf("report_%s_%s.pdf", data.Session.ID.String()[:8], data.Locale)
	filePath := filepath.Join(outputDir, filename)
	if err := os.WriteFile(filePath, pdfBuf, 0644); err != nil {
		return "", fmt.Errorf("write PDF: %w", err)
	}

	// Clean up HTML file
	os.Remove(htmlPath)

	return filePath, nil
}
