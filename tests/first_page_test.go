package pdf_test

import (
	"bytes"
	"testing"

	"webtyp.com/font"
	"webtyp.com/pdf"
)

func output(t *testing.T, doc *pdf.Document) []byte {
	t.Helper()
	var b bytes.Buffer
	if err := doc.OutputTo(&b); err != nil {
		t.Fatalf("OutputTo: %v", err)
	}
	return b.Bytes()
}

// Content added before any AddPage opens page 1 by itself: the file starts with the header.
func TestDocument_ContentWithoutAddPageOpensFirstPage(t *testing.T) {
	tf, err := pdf.LoadDeclared(font.Declare(font.Family("Roboto"), fontDir))
	if err != nil {
		t.Fatal(err)
	}
	doc := pdf.NewDocument(tf)
	doc.AddHeader1("Cotización")
	out := output(t, doc)
	if !bytes.HasPrefix(out, []byte("%PDF-")) {
		t.Fatalf("output must start with %%PDF-, got %q", out[:8])
	}
	if !bytes.Contains(out, []byte("/Count 1")) {
		t.Fatal("want exactly one page")
	}
}

// Calling AddPage first must not leave an extra blank page.
func TestDocument_AddPageFirstStillOnePage(t *testing.T) {
	tf, err := pdf.LoadDeclared(font.Declare(font.Family("Roboto"), fontDir))
	if err != nil {
		t.Fatal(err)
	}
	doc := pdf.NewDocument(tf)
	doc.AddPage()
	doc.AddHeader1("Cotización")
	if out := output(t, doc); !bytes.Contains(out, []byte("/Count 1")) {
		t.Fatal("AddPage before content must give exactly one page")
	}
}
