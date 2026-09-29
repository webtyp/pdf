package pdf

import (
	"bytes"
	"io"

	"webtyp.com/color"
	"webtyp.com/files"
	. "webtyp.com/fmt"
	"webtyp.com/font"
	"webtyp.com/pdf/fpdf"
)

type Typeface struct {
	regularData    []byte
	boldData       []byte
	italicData     []byte
	boldItalicData []byte
}

type TypefaceID int

type ImageID int

type imageEntry struct {
	name string
	path string
}

// LoadDeclared loads the four faces named by the font.Declaration.
// The .ttf extension is appended by this package.
func LoadDeclared(d font.Declaration) (Typeface, error) {
	dir := d.Dir()
	if dir != "" && dir[len(dir)-1] != '/' {
		dir += "/"
	}
	f := d.Family()

	load := func(style font.Style) ([]byte, error) {
		path := dir + f.Face(style) + ".ttf"
		data, err := defaultFiles().ReadFile(path)
		if err != nil {
			return nil, Errf("face %s missing: %s: %v", f.Face(style), path, err)
		}
		return data, nil
	}

	reg, err := load(font.Regular)
	if err != nil {
		return Typeface{}, err
	}
	bld, err := load(font.Bold)
	if err != nil {
		return Typeface{}, err
	}
	it, err := load(font.Italic)
	if err != nil {
		return Typeface{}, err
	}
	bi, err := load(font.BoldItalic)
	if err != nil {
		return Typeface{}, err
	}

	return Typeface{
		regularData:    reg,
		boldData:       bld,
		italicData:     it,
		boldItalicData: bi,
	}, nil
}

// Document wraps the internal fpdf.Fpdf to provide a fluent API.
type Document struct {
	internal *fpdf.Fpdf
	files    files.ReadWriter
	logger   func(message ...any)

	// Resource registries
	typefaces  []Typeface
	activeFont TypefaceID
	images     []imageEntry

	theme Theme
	err   error
}

type Option func(*Document)

// WithFiles sets where the Document reads fonts and images and writes its output, e.g.
// webtyp.com/opfs in a browser Worker or webtyp.com/files/mem in tests. Without it, a server
// build uses the disk and a browser build uses fetch plus a download.
func WithFiles(rw files.ReadWriter) Option {
	return func(d *Document) {
		d.files = rw
	}
}

// documentFiles lets fpdf always use the Document's current files, including one set by
// WithFiles after construction.
type documentFiles struct{ d *Document }

func (f documentFiles) ReadFile(path string) ([]byte, error) { return f.d.files.ReadFile(path) }
func (f documentFiles) WriteFile(path string, data []byte) error {
	return f.d.files.WriteFile(path, data)
}

func WithLogger(fn func(...any)) Option {
	return func(d *Document) {
		d.logger = fn
	}
}

func WithPageSize(w, h float64, unit string) Option {
	return func(d *Document) {
		// unit is ignored for now as fpdf is initialized with mm by default
	}
}

func WithMargins(left, top, right, bottom float64) Option {
	return func(d *Document) {
		d.internal.SetMargins(left, top, right)
		d.internal.SetAutoPageBreak(true, bottom)
	}
}

// NewDocument creates a new Document instance with UTF-8 support.
func NewDocument(t Typeface, opts ...Option) *Document {
	d := &Document{
		typefaces:  []Typeface{t},
		activeFont: TypefaceID(0),
		images:     []imageEntry{},
		theme:      DefaultTheme,
	}
	d.initIO() // initializes logger + IO depending on build tag
	d.files = defaultFiles()
	d.internal = fpdf.New(documentFiles{d})
	d.internal.SetMargins(20, 20, 20)
	d.internal.SetAutoPageBreak(true, 20)

	// Register the primary typeface in fpdf as font_0 and set it as active
	d.registerTypefaceInFpdf(TypefaceID(0), t)

	for _, opt := range opts {
		opt(d)
	}

	return d
}

func (d *Document) AddTypeface(t Typeface) TypefaceID {
	id := TypefaceID(len(d.typefaces))
	d.typefaces = append(d.typefaces, t)
	d.registerTypefaceInFpdf(id, t)
	return id
}

func (d *Document) Use(id TypefaceID) *Document {
	if int(id) < 0 || int(id) >= len(d.typefaces) {
		return d
	}
	d.activeFont = id
	return d
}

func (d *Document) registerTypefaceInFpdf(id TypefaceID, t Typeface) {
	name := Sprintf("font_%d", int(id))
	d.internal.AddUTF8FontFromBytes(name, "", t.regularData)
	d.internal.AddUTF8FontFromBytes(name, "B", t.boldData)
	d.internal.AddUTF8FontFromBytes(name, "I", t.italicData)
	d.internal.AddUTF8FontFromBytes(name, "BI", t.boldItalicData)
}

func (d *Document) getActiveFontName() string {
	return Sprintf("font_%d", int(d.activeFont))
}

func (d *Document) SetSize(pt float64) *Document {
	d.internal.SetFontSize(pt)
	return d
}

// SetTheme sets the document theme. Missing numeric fields (Sizes, Spacing)
// inherit from DefaultTheme so callers can specify only the colors they care
// about without producing zero-height text.
func (d *Document) SetTheme(theme Theme) *Document {
	if theme.Sizes.H1 == 0 {
		theme.Sizes.H1 = DefaultTheme.Sizes.H1
	}
	if theme.Sizes.H2 == 0 {
		theme.Sizes.H2 = DefaultTheme.Sizes.H2
	}
	if theme.Sizes.H3 == 0 {
		theme.Sizes.H3 = DefaultTheme.Sizes.H3
	}
	if theme.Sizes.Body == 0 {
		theme.Sizes.Body = DefaultTheme.Sizes.Body
	}
	if theme.Sizes.Small == 0 {
		theme.Sizes.Small = DefaultTheme.Sizes.Small
	}
	if theme.Spacing.Paragraph == 0 {
		theme.Spacing.Paragraph = DefaultTheme.Spacing.Paragraph
	}
	if theme.Spacing.Section == 0 {
		theme.Spacing.Section = DefaultTheme.Spacing.Section
	}
	if theme.Spacing.Page == 0 {
		theme.Spacing.Page = DefaultTheme.Spacing.Page
	}
	d.theme = theme
	if theme.Page.Width > 0 && theme.Page.Height > 0 {
		d.internal.SetPageSizeMM(theme.Page.Width, theme.Page.Height)
	}
	if theme.Margin.Top > 0 || theme.Margin.Right > 0 || theme.Margin.Bottom > 0 || theme.Margin.Left > 0 {
		m := theme.Margin
		d.internal.SetMargins(m.Left, m.Top, m.Right)
		d.internal.SetAutoPageBreak(true, m.Bottom)
	}
	return d
}

// Err returns the first accumulated error.
func (d *Document) Err() error {
	return d.err
}

func (d *Document) addError(err error) {
	if d.err == nil && err != nil {
		d.err = err
	}
}

// SetLog sets the logger function.
func (d *Document) SetLog(fn func(...any)) *Document {
	d.logger = fn
	return d
}

// Log writes a message to the logger.
func (d *Document) Log(message ...any) {
	if d.logger != nil {
		d.logger(message...)
	}
}

// RegisterImage registers an image to be loaded immediately.
func (d *Document) RegisterImage(path string) (ImageID, error) {
	data, err := d.files.ReadFile(path)
	if err != nil {
		return 0, err
	}

	ext := ""
	if idx := LastIndex(path, "."); idx != -1 {
		ext = path[idx+1:]
	}

	id := len(d.images)
	name := Sprintf("img_%d", id)

	opt := fpdf.ImageOptions{ImageType: ext, ReadDpi: true}
	d.internal.RegisterImageOptionsReader(name, opt, bytes.NewReader(data))

	d.images = append(d.images, imageEntry{name: name, path: path})
	return ImageID(id), nil
}

// Draw is a placeholder for consistency, though currently operations draw immediately.
func (d *Document) Draw() *Document {
	return d
}

// WritePdf generates the PDF and writes it to the specified path.
func (d *Document) WritePdf(path string) error {
	if d.err != nil {
		return d.err
	}
	err := d.internal.OutputFileAndClose(path)
	if d.err == nil {
		return err
	}
	return d.err
}

// OutputTo writes the generated PDF into the provided writer.
func (d *Document) OutputTo(w io.Writer) error {
	if d.err != nil {
		return d.err
	}
	err := d.internal.Output(w)
	if d.err == nil {
		return err
	}
	return d.err
}

// --- Base Components ---

// AddText adds a text paragraph in flow mode.
func (d *Document) AddText(text string) *TextElement {
	return &TextElement{doc: d, content: text, align: "L"}
}

// AddHeader1 adds a level 1 header.
func (d *Document) AddHeader1(text string) *Document {
	d.setTextColor(d.theme.Accent)
	d.internal.SetFont(d.getActiveFontName(), "B", d.theme.Sizes.H1)
	d.internal.CellFormat(0, d.theme.Sizes.H1/2, text, "", 1, "L", false, 0, "")
	d.setTextColor(d.theme.Body)
	d.internal.Ln(d.theme.Spacing.Section)
	return d
}

// AddHeader2 adds a level 2 header.
func (d *Document) AddHeader2(text string) *Document {
	d.setTextColor(d.theme.Accent)
	d.internal.SetFont(d.getActiveFontName(), "B", d.theme.Sizes.H2)
	d.internal.CellFormat(0, d.theme.Sizes.H2/2, text, "", 1, "L", false, 0, "")
	d.setTextColor(d.theme.Body)
	d.internal.Ln(d.theme.Spacing.Section / 2)
	return d
}

// AddHeader3 adds a level 3 header.
func (d *Document) AddHeader3(text string) *Document {
	d.setTextColor(d.theme.Accent)
	d.internal.SetFont(d.getActiveFontName(), "B", d.theme.Sizes.H3)
	d.internal.CellFormat(0, d.theme.Sizes.H3/2, text, "", 1, "L", false, 0, "")
	d.setTextColor(d.theme.Body)
	d.internal.Ln(d.theme.Spacing.Section / 4)
	return d
}

// AddSpace adds vertical space.
func (d *Document) AddSpace(units float64) *Document {
	d.internal.Ln(units)
	return d
}

// SpaceBefore is an alias for AddSpace.
func (d *Document) SpaceBefore(u float64) *Document {
	return d.AddSpace(u)
}

// AddPage adds a new page.
func (d *Document) AddPage() *Document {
	d.internal.AddPage()
	return d
}

// AddSeparator adds a horizontal line.
func (d *Document) AddSeparator() *Document {
	x := d.internal.GetX()
	y := d.internal.GetY()
	w, _ := d.internal.GetPageSize()
	lMargin, _, rMargin, _ := d.internal.GetMargins()
	width := w - lMargin - rMargin

	color := d.theme.Brand
	if color == "" {
		color = d.theme.Accent
	}
	d.drawLineH(x, y+2, width, color, 0.2)
	d.internal.Ln(5)
	return d
}

func (d *Document) drawLineH(x, y, width float64, c color.Color, thickness float64) {
	r, g, b, err := c.RGB()
	if err != nil {
		d.addError(err)
		return
	}
	d.internal.SetDrawColor(r, g, b)
	d.internal.SetLineWidth(thickness)
	d.internal.Line(x, y, x+width, y)
	d.internal.SetDrawColor(0, 0, 0)
	d.internal.SetLineWidth(0.2)
}

// AddImage adds an image by ID in flow mode.
func (d *Document) AddImage(id ImageID) *ImageElement {
	return &ImageElement{doc: d, id: id, align: "L"}
}

func (d *Document) drawImageAt(name string, x, y, width float64) {
	d.internal.Image(name, x, y, width, 0, false, "", 0, "")
}

func (d *Document) setPosition(x, y float64) {
	d.internal.SetXY(x, y)
}

func (d *Document) setCursorY(y float64) {
	d.internal.SetY(y)
}

func (d *Document) setCursorX(x float64) {
	d.internal.SetX(x)
}

func (d *Document) getCursorY() float64 {
	return d.internal.GetY()
}

func (d *Document) getCursorX() float64 {
	return d.internal.GetX()
}

func (d *Document) drawFilledRect(x, y, w, h float64, c color.Color) {
	r, g, b, err := c.RGB()
	if err != nil {
		d.addError(err)
		return
	}
	d.internal.SetFillColor(r, g, b)
	d.internal.Rect(x, y, w, h, "F")
	d.internal.SetFillColor(255, 255, 255)
}

func (d *Document) setTextColor(c color.Color) {
	r, g, b, err := c.RGB()
	if err != nil {
		d.addError(err)
		return
	}
	d.internal.SetTextColor(r, g, b)
}

func (d *Document) drawTextAt(x, y float64, text, style string, size float64) {
	d.internal.SetXY(x, y)
	d.internal.SetFont(d.getActiveFontName(), style, size)
	d.internal.Cell(0, size/2.8, text)
}

func (d *Document) cellAt(x, y, w, h float64, text, style string, size float64, align string) {
	d.internal.SetXY(x, y)
	d.internal.SetFont(d.getActiveFontName(), style, size)
	d.internal.CellFormat(w, h, text, "", 0, align, false, 0, "")
}

func (d *Document) measureText(text, style string, size float64) (width, height float64) {
	d.internal.SetFont(d.getActiveFontName(), style, size)
	return d.internal.GetStringWidth(text), size / 2.8
}

// --- Page Header/Footer ---

type PageHeader struct {
	doc       *Document
	leftText  string
	rightText string
}

func (d *Document) SetPageHeader() *PageHeader {
	ph := &PageHeader{doc: d}
	d.internal.SetHeaderFunc(func() {
		d.internal.SetY(10) // Standard header position
		d.internal.SetFont(d.getActiveFontName(), "I", 8)
		if ph.leftText != "" {
			d.internal.Cell(0, 10, ph.leftText)
		}
		if ph.rightText != "" {
			d.internal.CellFormat(0, 10, ph.rightText, "", 0, "R", false, 0, "")
		}
		d.internal.Ln(20) // Space after header
	})
	return ph
}

func (ph *PageHeader) SetLeftText(t string) *PageHeader {
	ph.leftText = t
	return ph
}

func (ph *PageHeader) SetRightText(t string) *PageHeader {
	ph.rightText = t
	return ph
}

type PageFooter struct {
	doc        *Document
	centerText string
	leftText   string
	pageTotal  bool
}

func (d *Document) SetPageFooter() *PageFooter {
	pf := &PageFooter{doc: d}
	d.internal.SetFooterFunc(func() {
		d.internal.SetY(-15) // Standard footer position
		d.internal.SetFont(d.getActiveFontName(), "I", 8)

		if pf.leftText != "" {
			d.internal.SetTextColor(130, 130, 130)
			d.internal.CellFormat(0, 10, pf.leftText, "", 0, "L", false, 0, "")
			d.internal.SetTextColor(0, 0, 0)
		}

		if pf.centerText != "" {
			d.internal.CellFormat(0, 10, pf.centerText, "", 0, "C", false, 0, "")
		}

		if pf.pageTotal {
			pageNo := d.internal.PageNo()
			pageStr := Sprintf("Página %s/{nb}", Convert(pageNo).String())
			d.internal.CellFormat(0, 10, pageStr, "", 0, "R", false, 0, "")
		}
		d.internal.SetTextColor(0, 0, 0)
	})
	return pf
}

func (pf *PageFooter) SetCenterText(t string) *PageFooter {
	pf.centerText = t
	return pf
}

func (pf *PageFooter) WithLeftRight(leftText string) *PageFooter {
	pf.leftText = leftText
	pf.pageTotal = true
	pf.doc.internal.AliasNbPages("")
	return pf
}

func (pf *PageFooter) WithPageTotal(align string) *PageFooter {
	pf.pageTotal = true
	pf.doc.internal.AliasNbPages("")
	return pf
}

// --- Styles ---

type Style struct {
	FillColor color.Color
	TextColor color.Color
	Font      string // "B", "I", ""
	FontSize  float64
}

const (
	FontBold    = "B"
	FontItalic  = "I"
	FontRegular = ""
)
