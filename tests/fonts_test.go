package pdf_test

import (
	"os"
	"strings"
	"testing"

	"webtyp.com/font"
	"webtyp.com/pdf"
)

// fontDir is relative to this test's working directory: the default files (disk) are plain
// os.ReadFile, so a registered path resolves against the package dir.
const fontDir = "../fpdf/fonts/"

type testFamily struct {
	name                            string
	regular, bold, italic, boldItal string
	note                            string
}

var families = []testFamily{
	{
		name: "Roboto", regular: "Roboto-Regular.ttf", bold: "Roboto-Bold.ttf",
		italic: "Roboto-Italic.ttf", boldItal: "Roboto-BoldItalic.ttf",
		note: "subset latino, cursivas reales",
	},
	{
		name: "Inter", regular: "Inter-Regular.ttf", bold: "Inter-Bold.ttf",
		italic: "Inter-Italic.ttf", boldItal: "Inter-BoldItalic.ttf",
		note: "subset latino, cursivas reales",
	},
}

// The sample exercises what a quote document actually prints: Spanish
// diacritics, the inverted marks, typographic quotes and dashes, and the euro
// sign that the DroidSans subset is missing.
const sample = `Señor Muñoz — ítem “café orgánico” ¿1.250 €? ¡Sí! ÁÉÍÓÚ üÜ ñÑ ª º`

// TestFonts_AllStyles renders every family in all four styles into a single
// PDF so the faces can be compared side by side. Open tests/test_fonts.pdf.
func TestFonts_AllStyles(t *testing.T) {
	// First load primary font
	d0 := font.Declare(font.Family(families[0].name), fontDir)
	primaryTf, err := pdf.LoadDeclared(d0)
	if err != nil {
		t.Fatalf("loading primary typeface: %v", err)
	}

	doc := pdf.NewDocument(primaryTf)

	// Map to keep track of typeface IDs
	type testFontReg struct {
		id   pdf.TypefaceID
		note string
		name string
	}

	regs := []testFontReg{
		{id: pdf.TypefaceID(0), name: families[0].name, note: families[0].note},
	}

	// Register other fonts
	for i := 1; i < len(families); i++ {
		f := families[i]
		d := font.Declare(font.Family(f.name), fontDir)
		tf, err := pdf.LoadDeclared(d)
		if err != nil {
			t.Fatalf("loading typeface %s: %v", f.name, err)
		}
		id := doc.AddTypeface(tf)
		regs = append(regs, testFontReg{id: id, name: f.name, note: f.note})
	}

	doc.AddPage()
	doc.AddHeader1("Comparativa de fuentes")

	for _, reg := range regs {
		doc.Use(reg.id)
		doc.SetSize(12)
		doc.AddHeader2(reg.name + " — " + reg.note)

		doc.AddText("REGULAR   " + sample).Draw()
		doc.AddText("BOLD      " + sample).Bold().Draw()
		doc.AddText("ITALIC    " + sample).Italic().Draw()
		doc.AddText("BOLD+ITAL " + sample).Bold().Italic().Draw()

		doc.SpaceBefore(6)
		doc.AddSeparator()
		doc.SpaceBefore(6)
	}

	const out = "test_fonts.pdf"
	// Ensure we remove it first, so if the test fails or runs from cache, we don't look at an old copy
	_ = os.Remove(out)

	if err := doc.WritePdf(out); err != nil {
		t.Fatalf("WritePdf: %v", err)
	}

	st, err := os.Stat(out)
	if err != nil {
		t.Fatalf("stat %s: %v", out, err)
	}
	if st.Size() == 0 {
		t.Fatalf("PDF %s is empty", out)
	}
	t.Logf("escrito %s (%d bytes) — abrelo para ver el resultado", out, st.Size())
}

// TestFonts_MissingFaceErrors is the regression test for the removed silent
// fallbacks: LoadDeclared must fail — naming the missing face and its path —
// instead of substituting another face and producing a document without
// italics.
func TestFonts_MissingFaceErrors(t *testing.T) {
	copy := func(dstDir, name string) {
		src, err := os.ReadFile(fontDir + name)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if err := os.WriteFile(dstDir+"/"+name, src, 0644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	cases := []struct {
		name    string
		provide []string
		missing string // full path that must appear in the error
	}{
		{
			name:    "missing italic",
			provide: []string{"Roboto-Regular.ttf", "Roboto-Bold.ttf"},
			missing: "Roboto-Italic.ttf",
		},
		{
			name:    "missing bold italic",
			provide: []string{"Roboto-Regular.ttf", "Roboto-Bold.ttf", "Roboto-Italic.ttf"},
			missing: "Roboto-BoldItalic.ttf",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir, err := os.MkdirTemp("", "pdf-fonts-*")
			if err != nil {
				t.Fatalf("MkdirTemp: %v", err)
			}
			defer os.RemoveAll(dir)

			for _, f := range tc.provide {
				copy(dir, f)
			}

			d := font.Declare("Roboto", dir)
			_, err = pdf.LoadDeclared(d)
			if err == nil {
				t.Fatalf("LoadDeclared succeeded, want error for missing %s", tc.missing)
			}
			if !strings.Contains(err.Error(), tc.missing) {
				t.Fatalf("error %q does not name the missing face %s", err, tc.missing)
			}
		})
	}
}

// TestFonts_SubsetsAreSmallerThanDroid is the size claim, checked rather than
// asserted in prose: the subsets must come in under the DroidSans face they
// replace, despite carrying a real italic and the euro sign it lacks.
func TestFonts_SubsetsAreSmallerThanDroid(t *testing.T) {
	size := func(name string) int64 {
		st, err := os.Stat(fontDir + name)
		if err != nil {
			t.Fatalf("stat %s: %v", name, err)
		}
		return st.Size()
	}

	droid := size("DroidSans.ttf")
	for _, name := range []string{"Roboto-Regular.ttf", "Inter-Regular.ttf"} {
		got := size(name)
		if got >= droid {
			t.Errorf("%s pesa %d B, no mejora a DroidSans.ttf (%d B)", name, got, droid)
		}
		t.Logf("%-20s %6d B  (DroidSans.ttf %d B)", name, got, droid)
	}
}
