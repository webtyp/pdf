package pdf_test

import (
	"strings"
	"testing"

	"webtyp.com/files"
	"webtyp.com/files/mem"
	"webtyp.com/font"
	"webtyp.com/pdf"
)

// A Document built WithFiles writes its output through that files.ReadWriter — here an
// in-memory one — instead of the disk. This is how a browser Worker hands it webtyp/opfs.
func TestDocument_WithFilesWritesOutputThere(t *testing.T) {
	tf, err := pdf.LoadDeclared(font.Declare(font.Family("Roboto"), fontDir))
	if err != nil {
		t.Fatalf("loading typeface: %v", err)
	}
	store := mem.New()
	doc := pdf.NewDocument(tf, pdf.WithFiles(store))
	doc.AddPage()
	doc.AddHeader1("Cotización")
	if err := doc.WritePdf("out/quote.pdf"); err != nil {
		t.Fatalf("WritePdf: %v", err)
	}

	got, err := store.ReadFile("out/quote.pdf")
	if err != nil {
		t.Fatalf("the PDF was not written to the injected files: %v", err)
	}
	if !strings.HasPrefix(string(got), "%PDF-") {
		t.Fatalf("output does not look like a PDF: %q", got[:8])
	}
}

// Reading an image that does not exist reports files.ErrNotExist from the injected files.
func TestDocument_WithFilesMissingImage(t *testing.T) {
	tf, err := pdf.LoadDeclared(font.Declare(font.Family("Roboto"), fontDir))
	if err != nil {
		t.Fatalf("loading typeface: %v", err)
	}
	doc := pdf.NewDocument(tf, pdf.WithFiles(mem.New()))
	if _, err := doc.RegisterImage("missing.png"); err != files.ErrNotExist {
		t.Fatalf("RegisterImage(missing) error = %v, want files.ErrNotExist", err)
	}
}
