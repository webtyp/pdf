package fpdf_test

import (
	"bytes"
	"strings"
	"testing"

	"webtyp.com/pdf/fpdf"
)

// Without AutoFirstPage, drawing before the first AddPage is an error, never a corrupt file.
func TestContentBeforeAddPageIsAnError(t *testing.T) {
	f := fpdf.New(testDiskFiles{})
	f.Line(10, 10, 50, 50)
	var b bytes.Buffer
	err := f.Output(&b)
	if err == nil || !strings.Contains(err.Error(), "before the first AddPage") {
		t.Fatalf("Output error = %v, want the no-page error", err)
	}
}

// With AutoFirstPage, the same drawing opens page 1 and the file is valid.
func TestAutoFirstPageOpensPageOne(t *testing.T) {
	f := fpdf.New(testDiskFiles{}, fpdf.AutoFirstPage)
	f.Line(10, 10, 50, 50)
	var b bytes.Buffer
	if err := f.Output(&b); err != nil {
		t.Fatalf("Output: %v", err)
	}
	if !bytes.HasPrefix(b.Bytes(), []byte("%PDF-")) || f.PageNo() != 1 {
		t.Fatalf("want a valid PDF with page 1, got prefix %q page %d", b.Bytes()[:8], f.PageNo())
	}
}
