package engine

import (
	"fmt"
	"strings"

	"github.com/ledongthuc/pdf"
)

// PDFPage holds the extracted plain text for a single PDF page.
type PDFPage struct {
	PageNum int
	Text    string
}

// ReadPDFPages extracts plain text page by page from a PDF file.
// Pages are 1-indexed. Empty or whitespace-only pages are skipped.
func ReadPDFPages(path string) ([]PDFPage, error) {
	f, r, err := pdf.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open pdf %s: %w", path, err)
	}
	defer f.Close()

	total := r.NumPage()
	pages := make([]PDFPage, 0, total)
	for i := 1; i <= total; i++ {
		p := r.Page(i)
		if p.V.IsNull() {
			continue
		}
		text, err := p.GetPlainText(nil)
		if err != nil {
			// skip unreadable pages rather than aborting the whole document
			continue
		}
		text = strings.TrimSpace(text)
		if text == "" {
			continue
		}
		pages = append(pages, PDFPage{PageNum: i, Text: text})
	}
	return pages, nil
}
