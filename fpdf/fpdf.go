package fpdf

import (
	"bytes"
	"image/color"
	"image/jpeg"
	"io"
	"math"

	"webtyp.com/files"
	. "webtyp.com/fmt"
)

var gl struct {
	catalogSort  bool
	noCompress   bool // Initial zero value indicates compression
	creationDate pdfTime
	modDate      pdfTime
}

type fmtBuffer struct {
	bytes.Buffer
}

func (b *fmtBuffer) printf(fmtStr string, args ...any) {
	b.Buffer.WriteString(Sprintf(fmtStr, args...))
}

func New(options ...any) (f *Fpdf) {
	f = new(Fpdf)

	var size = PageSize{0, 0, false}
	var initType *InitType

	// Set default values
	f.defOrientation = Portrait
	f.rootDirectory = "."
	f.fontsDirName = "fonts"
	f.unitType = MM
	// Closed by default: until a files.ReadWriter is passed to New, every file access fails loudly.
	f.files = unconfiguredFiles{}

	for _, opt := range options {
		switch v := opt.(type) {
		case unit:
			if v != "" {
				f.unitType = v
			}
		case orientationType:
			f.defOrientation = v
		case PageSize:
			size = v
		case *InitType:
			initType = v
		case RootDirectoryType:
			f.rootDirectory = v
		case FontsDirName:
			f.fontsDirName = v
		case files.ReadWriter:
			f.files = v
		case FirstPage:
			f.autoFirstPage = v == AutoFirstPage

		}
	}
	if initType != nil {
		f.defOrientation = initType.OrientationStr
		if f.defOrientation == "" {
			f.defOrientation = Portrait
		}
		if initType.UnitType != "" {
			f.unitType = initType.UnitType
		}

		// Note: page size conversion happens later after scale factor is set
		f.rootDirectory = initType.RootDirectory
		f.fontsPath = initType.RootDirectory.MakePath(initType.FontDirName)
	}

	f.page = 0
	f.n = 2
	f.pages = make([]*bytes.Buffer, 0, 8)
	f.pages = append(f.pages, bytes.NewBufferString("")) // pages[0] is unused (1-based)
	f.pageSizes = make(map[int]PageSize)
	f.pageBoxes = make(map[int]map[string]PageBox)
	f.defPageBoxes = make(map[string]PageBox)
	f.state = 0
	f.fonts = make(map[string]fontDefType)
	f.fontFiles = make(map[string]fontFileType)
	f.diffs = make([]string, 0, 8)
	f.images = make(map[string]*ImageInfoType)
	f.pageLinks = make([][]linkType, 0, 8)
	f.pageLinks = append(f.pageLinks, make([]linkType, 0)) // pageLinks[0] is unused (1-based)
	f.links = make([]intLinkType, 0, 8)
	f.links = append(f.links, intLinkType{}) // links[0] is unused (1-based)
	f.pageAttachments = make([][]annotationAttach, 0, 8)
	f.pageAttachments = append(f.pageAttachments, []annotationAttach{}) //
	f.aliasMap = make(map[string]string)
	f.inHeader = false
	f.inFooter = false
	f.lasth = 0
	f.fontFamily = ""
	f.fontStyle = ""
	f.SetFontSize(12)
	f.underline = false
	f.strikeout = false
	f.setDrawColor(0, 0, 0)
	f.setFillColor(0, 0, 0)
	f.setTextColor(0, 0, 0)
	f.colorFlag = false
	f.ws = 0
	// Set fontsPath instance
	f.fontsPath = f.rootDirectory.MakePath(string(f.fontsDirName))

	// Scale factor
	switch f.unitType {
	case POINT:
		f.k = 1.0
	case MM:
		f.k = 72.0 / 25.4
	case CM:
		f.k = 72.0 / 2.54
	case IN:
		f.k = 72.0
	default:
		f.err = Err("format", "invalid")
		return
	}
	f.stdPageSizes = make(map[string]PageSize)
	f.stdPageSizes["a3"] = A3
	f.stdPageSizes["a4"] = A4
	f.stdPageSizes["a5"] = A5
	f.stdPageSizes["a6"] = A6
	f.stdPageSizes["a7"] = A7
	f.stdPageSizes["a2"] = A2
	f.stdPageSizes["a1"] = A1
	f.stdPageSizes["letter"] = Letter
	f.stdPageSizes["legal"] = Legal
	f.stdPageSizes["tabloid"] = Tabloid

	// Set default page size
	if initType != nil && initType.Size.Wd > 0 && initType.Size.Ht > 0 {
		// Convert user-specified page size to points for internal use
		f.defPageSize = PageSize{
			Wd:     initType.Size.Wd * f.k,
			Ht:     initType.Size.Ht * f.k,
			AutoHt: initType.Size.AutoHt,
		}
	} else if size.Wd > 0 && size.Ht > 0 {
		// Convert deprecated size parameter to points for internal use
		f.defPageSize = PageSize{
			Wd:     size.Wd * f.k,
			Ht:     size.Ht * f.k,
			AutoHt: size.AutoHt,
		}
	} else {
		// Use default A4 size in points
		f.defPageSize = A4
	}
	f.curPageSize = f.defPageSize // Page orientation
	switch f.defOrientation {
	case Portrait:
		f.w = f.defPageSize.Wd / f.k
		f.h = f.defPageSize.Ht / f.k
		// dbg("Assign h: %8.2f", f.h)
	case Landscape:
		f.w = f.defPageSize.Ht / f.k
		f.h = f.defPageSize.Wd / f.k
	default:
		f.err = Err("format", "invalid")
		return
	}
	f.curOrientation = f.defOrientation
	f.wPt = f.w * f.k
	f.hPt = f.h * f.k
	// Page margins (1 cm)
	margin := 28.35 / f.k
	f.SetMargins(margin, margin, margin)
	// Interior cell margin (1 mm)
	f.cMargin = margin / 10
	// Line width (0.2 mm)
	f.lineWidth = 0.567 / f.k
	// 	Automatic page break
	f.SetAutoPageBreak(true, 2*margin)
	// Default display mode
	f.SetDisplayMode("default", "default")
	if f.err != nil {
		return
	}
	f.acceptPageBreak = func() bool {
		return f.autoPageBreak
	}
	// Enable compression
	f.SetCompression(!gl.noCompress)
	f.spotColorMap = make(map[string]spotColorType)
	f.blendList = make([]blendModeType, 0, 8)
	f.blendList = append(f.blendList, blendModeType{}) // blendList[0] is unused (1-based)
	f.blendMap = make(map[string]int)
	f.blendMode = "Normal"
	f.alpha = 1
	f.gradientList = make([]gradientType, 0, 8)
	f.gradientList = append(f.gradientList, gradientType{}) // gradientList[0] is unused
	// Set default PDF version number
	f.pdfVersion = pdfVers1_3
	f.SetProducer("FPDF "+cnFpdfVersion, true)
	f.layerInit()
	f.catalogSort = gl.catalogSort
	f.creationDate = gl.creationDate
	f.modDate = gl.modDate
	f.userUnderlineThickness = 1

	// create a large enough buffer for formatting float64s.
	// math.MaxInt64  needs 19.
	// math.MaxUint64 needs 20.
	f.fmt.buf = make([]byte, 24)
	return
}

// Ok returns true if no processing errors have occurred.
func (f *Fpdf) Ok() bool {
	return f.err == nil
}

// Err returns true if a processing error has occurred.
func (f *Fpdf) Err() bool {
	return f.err != nil
}

// ClearError unsets the internal Fpdf error. This method should be used with
// care, as an internal error condition usually indicates an unrecoverable
// problem with the generation of a document. It is intended to deal with cases
// in which an error is used to select an alternate form of the document.
func (f *Fpdf) ClearError() {
	f.err = nil
}

// SetErrorf sets the internal Fpdf error with formatted text to halt PDF
// generation; this may facilitate error handling by application. If an error
// condition is already set, this call is ignored.
//
// See the documentation for printing in the standard fmt package for details
// about fmtStr and args.
func (f *Fpdf) SetErrorf(fmtStr string, args ...any) {
	if f.err == nil {
		f.err = Errf(fmtStr, args...)
	}
}

// String satisfies the fmt.Stringer interface and summarizes the Fpdf
// instance.
func (f *Fpdf) String() string {
	return "Fpdf " + cnFpdfVersion
}

// SetError sets an error to halt PDF generation. This may facilitate error
// handling by application. See also Ok(), Err() and Error().
func (f *Fpdf) SetError(err error) {
	if f.err == nil && err != nil {
		f.err = err
	}
}

// Error returns the internal Fpdf error; this will be nil if no error has occurred.
func (f *Fpdf) Error() error {
	return f.err
}

// GetPageBreakTrigger returns the threshold used to trigger page breaks.
func (f *Fpdf) GetPageBreakTrigger() float64 {
	return f.pageBreakTrigger
}

// GetCellMargin returns the cell margin. This is the amount of space before
// and after the text within a cell that's left blank, and is in units passed
// to New(). It defaults to 1mm.
func (f *Fpdf) GetCellMargin() float64 {
	return f.cMargin
}

// SetCellMargin sets the cell margin. This is the amount of space before and
// after the text within a cell that's left blank, and is in units passed to
// New().
func (f *Fpdf) SetCellMargin(margin float64) {
	f.cMargin = margin
}

// SetPageSizeMM sets the default page size in millimetres. Must be called
// before AddPage; has no effect once pages have been added.
func (f *Fpdf) SetPageSizeMM(widthMM, heightMM float64) {
	if f.page > 0 {
		return
	}
	f.defPageSize = PageSize{Wd: widthMM * f.k, Ht: heightMM * f.k}
	f.curPageSize = f.defPageSize
	f.w = widthMM
	f.h = heightMM
	f.wPt = widthMM * f.k
	f.hPt = heightMM * f.k
}

// SetDefaultCompression controls the default setting of the internal
// compression flag. See SetCompression() for more details. Compression is on
// by default.
func SetDefaultCompression(compress bool) {
	gl.noCompress = !compress
}

// GetCompression returns whether page compression is enabled.
func (f *Fpdf) GetCompression() bool {
	return f.compress
}

// SetCompression activates or deactivates page compression with zlib. When
// activated, the internal representation of each page is compressed, which
// leads to a compression ratio of about 2 for the resulting document.
// Compression is on by default.
func (f *Fpdf) SetCompression(compress bool) {
	f.compress = compress
}

// GetProducer returns the producer of the document as ISO-8859-1 or UTF-16BE.
func (f *Fpdf) GetProducer() string {
	return f.producer
}

// SetProducer defines the producer of the document. isUTF8 indicates if the string
// is encoded in ISO-8859-1 (false) or UTF-8 (true).
func (f *Fpdf) SetProducer(producerStr string, isUTF8 bool) {
	if isUTF8 {
		producerStr = utf8toutf16(producerStr)
	}
	f.producer = producerStr
}

// GetTitle returns the title of the document as ISO-8859-1 or UTF-16BE.
func (f *Fpdf) GetTitle() string {
	return f.title
}

// SetTitle defines the title of the document. isUTF8 indicates if the string
// is encoded in ISO-8859-1 (false) or UTF-8 (true).
func (f *Fpdf) SetTitle(titleStr string, isUTF8 bool) {
	if isUTF8 {
		titleStr = utf8toutf16(titleStr)
	}
	f.title = titleStr
}

// GetSubject returns the subject of the document as ISO-8859-1 or UTF-16BE.
func (f *Fpdf) GetSubject() string {
	return f.subject
}

// SetSubject defines the subject of the document. isUTF8 indicates if the
// string is encoded in ISO-8859-1 (false) or UTF-8 (true).
func (f *Fpdf) SetSubject(subjectStr string, isUTF8 bool) {
	if isUTF8 {
		subjectStr = utf8toutf16(subjectStr)
	}
	f.subject = subjectStr
}

// GetAuthor returns the author of the document as ISO-8859-1 or UTF-16BE.
func (f *Fpdf) GetAuthor() string {
	return f.author
}

// SetAuthor defines the author of the document. isUTF8 indicates if the string
// is encoded in ISO-8859-1 (false) or UTF-8 (true).
func (f *Fpdf) SetAuthor(authorStr string, isUTF8 bool) {
	if isUTF8 {
		authorStr = utf8toutf16(authorStr)
	}
	f.author = authorStr
}

// GetLang returns the natural language of the document (e.g. "de-CH").
func (f *Fpdf) GetLang() string {
	return f.lang
}

// SetLang defines the natural language of the document (e.g. "de-CH").
func (f *Fpdf) SetLang(lang string) {
	f.lang = lang
}

// GetKeywords returns the keywords of the document as ISO-8859-1 or UTF-16BE.
func (f *Fpdf) GetKeywords() string {
	return f.keywords
}

// SetKeywords defines the keywords of the document. keywordStr is a
// space-delimited string, for example "invoice August". isUTF8 indicates if
// the string is encoded
func (f *Fpdf) SetKeywords(keywordsStr string, isUTF8 bool) {
	if isUTF8 {
		keywordsStr = utf8toutf16(keywordsStr)
	}
	f.keywords = keywordsStr
}

// GetCreator returns the creator of the document as ISO-8859-1 or UTF-16BE.
func (f *Fpdf) GetCreator() string {
	return f.creator
}

// SetCreator defines the creator of the document. isUTF8 indicates if the
// string is encoded in ISO-8859-1 (false) or UTF-8 (true).
func (f *Fpdf) SetCreator(creatorStr string, isUTF8 bool) {
	if isUTF8 {
		creatorStr = utf8toutf16(creatorStr)
	}
	f.creator = creatorStr
}

// GetXmpMetadata returns the XMP metadata that will be embedded with the document.
func (f *Fpdf) GetXmpMetadata() []byte {
	return []byte(string(f.xmp))
}

// SetXmpMetadata defines XMP metadata that will be embedded with the document.
func (f *Fpdf) SetXmpMetadata(xmpStream []byte) {
	f.xmp = xmpStream
}

// AddOutputIntent adds an output intent with ICC color profile
func (f *Fpdf) AddOutputIntent(outputIntent OutputIntentType) {
	f.outputIntents = append(f.outputIntents, outputIntent)
	if f.pdfVersion < pdfVers1_4 {
		f.pdfVersion = pdfVers1_4
	}
}

// AliasNbPages defines an alias for the total number of pages. It will be
// substituted as the document is closed. An empty string is replaced with the
// string "{nb}".
//
// See the example for AddPage() for a demonstration of this method.
func (f *Fpdf) AliasNbPages(aliasStr string) {
	if aliasStr == "" {
		aliasStr = "{nb}"
	}
	f.aliasNbPagesStr = aliasStr
}

// RTL enables right-to-left mode
func (f *Fpdf) RTL() {
	f.isRTL = true
}

// LTR disables right-to-left mode
func (f *Fpdf) LTR() {
	f.isRTL = false
}

// open begins a document
func (f *Fpdf) open() {
	f.state = 1
}

// Close terminates the PDF document. It is not necessary to call this method
// explicitly because Output(), OutputAndClose() and OutputFileAndClose() do it
// automatically. If the document contains no page, AddPage() is called to
// prevent the generation of an invalid document.
func (f *Fpdf) Close() {
	if f.err == nil {
		if f.clipNest > 0 {
			f.err = Errf("clip procedure must be explicitly ended")
		} else if f.transformNest > 0 {
			f.err = Errf("transformation procedure must be explicitly ended")
		}
	}
	if f.err != nil {
		return
	}
	if f.state == 3 {
		return
	}
	if f.page == 0 {
		f.AddPage()
		if f.err != nil {
			return
		}
	}
	// Page footer
	f.inFooter = true
	if f.footerFnc != nil {
		f.footerFnc()
	} else if f.footerFncLpi != nil {
		f.footerFncLpi(true)
	}
	f.inFooter = false

	// Close page
	f.endpage()
	// Close document
	f.enddoc()
}

func colorComp(v int) (int, float64) {
	if v < 0 {
		v = 0
	} else if v > 255 {
		v = 255
	}
	return v, float64(v) / 255.0
}

func (f *Fpdf) rgbColorValue(r, g, b int, grayStr, fullStr string) (clr colorType) {
	clr.ir, clr.r = colorComp(r)
	clr.ig, clr.g = colorComp(g)
	clr.ib, clr.b = colorComp(b)
	clr.mode = colorModeRGB
	clr.gray = clr.ir == clr.ig && clr.r == clr.b
	const prec = 3
	if len(grayStr) > 0 {
		if clr.gray {
			// clr.str = sprintf("%.3f %s", clr.r, grayStr)
			f.fmt.col.Reset()
			f.fmt.col.WriteString(f.fmtF64(clr.r, prec))
			f.fmt.col.WriteString(" ")
			f.fmt.col.WriteString(grayStr)
			clr.str = f.fmt.col.String()
		} else {
			// clr.str = sprintf("%.3f %.3f %.3f %s", clr.r, clr.g, clr.b, fullStr)
			f.fmt.col.Reset()
			f.fmt.col.WriteString(f.fmtF64(clr.r, prec))
			f.fmt.col.WriteString(" ")
			f.fmt.col.WriteString(f.fmtF64(clr.g, prec))
			f.fmt.col.WriteString(" ")
			f.fmt.col.WriteString(f.fmtF64(clr.b, prec))
			f.fmt.col.WriteString(" ")
			f.fmt.col.WriteString(fullStr)
			clr.str = f.fmt.col.String()
		}
	} else {
		// clr.str = sprintf("%.3f %.3f %.3f", clr.r, clr.g, clr.b)
		f.fmt.col.Reset()
		f.fmt.col.WriteString(f.fmtF64(clr.r, prec))
		f.fmt.col.WriteString(" ")
		f.fmt.col.WriteString(f.fmtF64(clr.g, prec))
		f.fmt.col.WriteString(" ")
		f.fmt.col.WriteString(f.fmtF64(clr.b, prec))
		clr.str = f.fmt.col.String()
	}
	return
}

func makeSubsetRange(end int) map[int]int {
	answer := make(map[int]int)
	for i := 0; i < end; i++ {
		answer[i] = 0
	}
	return answer
}

// AddLink creates a new internal link and returns its identifier. An internal
// link is a clickable area which directs to another place within the document.
// The identifier can then be passed to Cell(), Write(), Image() or Link(). The
// destination is defined with SetLink().
func (f *Fpdf) AddLink() int {
	f.links = append(f.links, intLinkType{})
	return len(f.links) - 1
}

// SetLink defines the page and position a link points to. See AddLink().
func (f *Fpdf) SetLink(link int, y float64, page int) {
	if y == -1 {
		y = f.y
	}
	if page == -1 {
		page = f.page
	}
	f.links[link] = intLinkType{page, y}
}

// newLink adds a new clickable link on current page
func (f *Fpdf) newLink(x, y, w, h float64, link int, linkStr string) {
	// linkList, ok := f.pageLinks[f.page]
	// if !ok {
	// linkList = make([]linkType, 0, 8)
	// f.pageLinks[f.page] = linkList
	// }
	f.pageLinks[f.page] = append(f.pageLinks[f.page],
		linkType{x * f.k, f.hPt - y*f.k, w * f.k, h * f.k, link, linkStr})
}

// Link puts a link on a rectangular area of the page. Text or image links are
// generally put via Cell(), Write() or Image(), but this method can be useful
// for instance to define a clickable area inside an image. link is the value
// returned by AddLink().
func (f *Fpdf) Link(x, y, w, h float64, link int) {
	f.newLink(x, y, w, h, link, "")
}

// LinkString puts a link on a rectangular area of the page. Text or image
// links are generally put via Cell(), Write() or Image(), but this method can
// be useful for instance to define a clickable area inside an image. linkStr
// is the target URL.
func (f *Fpdf) LinkString(x, y, w, h float64, linkStr string) {
	f.newLink(x, y, w, h, 0, linkStr)
}

// Bookmark sets a bookmark that will be displayed in a sidebar outline. txtStr
// is the title of the bookmark. level specifies the level of the bookmark in
// the outline; 0 is the top level, 1 is just below, and so on. y specifies the
// vertical position of the bookmark destination in the current page; -1
// indicates the current position.
func (f *Fpdf) Bookmark(txtStr string, level int, y float64) {
	if y == -1 {
		y = f.y
	}
	if f.isCurrentUTF8 {
		txtStr = utf8toutf16(txtStr)
	}
	f.outlines = append(f.outlines, outlineType{text: txtStr, level: level, y: y, p: f.PageNo(), prev: -1, last: -1, next: -1, first: -1})
}

// GetWordSpacing returns the spacing between words of following text.
func (f *Fpdf) GetWordSpacing() float64 {
	return f.ws
}

// SetWordSpacing sets spacing between words of following text. See the
// WriteAligned() example for a demonstration of its use.
func (f *Fpdf) SetWordSpacing(space float64) {
	f.ws = space
	f.out(sprintf("%.5f Tw", space*f.k))
}

// SetTextRenderingMode sets the rendering mode of following text.
// The mode can be as follows:
// 0: Fill text
// 1: Stroke text
// 2: Fill, then stroke text
// 3: Neither fill nor stroke text (invisible)
// 4: Fill text and add to path for clipping
// 5: Stroke text and add to path for clipping
// 6: Fills then stroke text and add to path for clipping
// 7: Add text to path for clipping
// This method is demonstrated in the SetTextRenderingMode example.
func (f *Fpdf) SetTextRenderingMode(mode int) {
	if mode >= 0 && mode <= 7 {
		f.out(sprintf("%d Tr", mode))
	}
}

// CellFormat prints a rectangular cell with optional borders, background color
// and character string. The upper-left corner of the cell corresponds to the
// current position. The text can be aligned or centered. After the call, the
// current position moves to the right or to the next line. It is possible to
// put a link on the text.
//
// An error will be returned if a call to SetFont() has not already taken
// place before this method is called.
//
// If automatic page breaking is enabled and the cell goes beyond the limit, a
// page break is done before outputting.
//
// w and h specify the width and height of the cell. If w is 0, the cell
// extends up to the right margin. Specifying 0 for h will result in no output,
// but the current position will be advanced by w.
//
// txtStr specifies the text to display.
//
// borderStr specifies how the cell border will be drawn. An empty string
// indicates no border, "1" indicates a full border, and one or more of "L",
// "T", "R" and "B" indicate the left, top, right and bottom sides of the
// border.
//
// ln indicates where the current position should go after the call. Possible
// values are 0 (to the right), 1 (to the beginning of the next line), and 2
// (below). Putting 1 is equivalent to putting 0 and calling Ln() just after.
//
// alignStr specifies how the text is to be positioned within the cell.
// Horizontal alignment is controlled by including "L", "C" or "R" (left,
// center, right) in alignStr. Vertical alignment is controlled by including
// "T", "M", "B" or "A" (top, middle, bottom, baseline) in alignStr. The default
// alignment is left middle.
//
// fill is true to paint the cell background or false to leave it transparent.
//
// link is the identifier returned by AddLink() or 0 for no internal link.
//
// linkStr is a target URL or empty for no external link. A non--zero value for
// link takes precedence over linkStr.
func (f *Fpdf) CellFormat(w, h float64, txtStr, borderStr string, ln int,
	alignStr string, fill bool, link int, linkStr string) {
	// dbg("CellFormat. h = %.2f, borderStr = %s", h, borderStr)
	if f.err != nil {
		return
	}

	if f.currentFont.Name == "" {
		f.err = Errf("font has not been set; unable to render text")
		return
	}

	borderStr = Convert(borderStr).ToLower().String()
	k := f.k
	if f.y+h > f.pageBreakTrigger && !f.inHeader && !f.inFooter && f.acceptPageBreak() {
		// Automatic page break
		x := f.x
		ws := f.ws
		// dbg("auto page break, x %.2f, ws %.2f", x, ws)
		if ws > 0 {
			f.ws = 0
			f.out("0 Tw")
		}
		f.AddPageFormat(f.curOrientation, f.curPageSize)
		if f.err != nil {
			return
		}
		f.x = x
		if ws > 0 {
			f.ws = ws
			// f.outf("%.3f Tw", ws*k)
			f.putF64(ws*k, 3)
			f.put(" Tw\n")
		}
	}
	if w == 0 {
		w = f.w - f.rMargin - f.x
	}
	var s fmtBuffer
	if h > 0 && (fill || borderStr == "1") {
		var op string
		if fill {
			if borderStr == "1" {
				op = "B"
				// dbg("border is '1', fill")
			} else {
				op = "f"
				// dbg("border is empty, fill")
			}
		} else {
			// dbg("border is '1', no fill")
			op = "S"
		}
		/// dbg("(CellFormat) f.x %.2f f.k %.2f", f.x, f.k)
		s.printf("%.2f %.2f %.2f %.2f re %s ", f.x*k, (f.h-f.y)*k, w*k, -h*k, op)
	}
	if len(borderStr) > 0 && borderStr != "1" {
		// fmt.Printf("border is '%s', no fill\n", borderStr)
		x := f.x
		y := f.y
		left := x * k
		top := (f.h - y) * k
		right := (x + w) * k
		bottom := (f.h - (y + h)) * k
		if Contains(borderStr, "L") {
			s.printf("%.2f %.2f m %.2f %.2f l S ", left, top, left, bottom)
		}
		if Contains(borderStr, "T") {
			s.printf("%.2f %.2f m %.2f %.2f l S ", left, top, right, top)
		}
		if Contains(borderStr, "R") {
			s.printf("%.2f %.2f m %.2f %.2f l S ", right, top, right, bottom)
		}
		if Contains(borderStr, "B") {
			s.printf("%.2f %.2f m %.2f %.2f l S ", left, bottom, right, bottom)
		}
	}
	if len(txtStr) > 0 {
		var dx, dy float64
		// Horizontal alignment
		switch {
		case Contains(alignStr, "R"):
			dx = w - f.cMargin - f.GetStringWidth(txtStr)
		case Contains(alignStr, "C"):
			dx = (w - f.GetStringWidth(txtStr)) / 2
		default:
			dx = f.cMargin
		}

		// Vertical alignment
		switch {
		case Contains(alignStr, "T"):
			dy = (f.fontSize - h) / 2.0
		case Contains(alignStr, "B"):
			dy = (h - f.fontSize) / 2.0
		case Contains(alignStr, "A"):
			var descent float64
			d := f.currentFont.Desc
			if d.Descent == 0 {
				// not defined (standard font?), use average of 19%
				descent = -0.19 * f.fontSize
			} else {
				descent = float64(d.Descent) * f.fontSize / float64(d.Ascent-d.Descent)
			}
			dy = (h-f.fontSize)/2.0 - descent
		default:
			dy = 0
		}
		if f.colorFlag {
			s.printf("q %s ", f.color.text.str)
		}
		//If multibyte, Tw has no effect - do word spacing using an adjustment before each space
		if (f.ws != 0 || alignStr == "J") && f.isCurrentUTF8 { // && f.ws != 0
			if f.isRTL {
				txtStr = reverseText(txtStr)
			}
			wmax := int(math.Ceil((w - 2*f.cMargin) * 1000 / f.fontSize))
			for _, uni := range txtStr {
				f.currentFont.usedRunes[int(uni)] = int(uni)
			}
			space := f.escape(utf8toutf16(" ", false))
			strSize := f.GetStringSymbolWidth(txtStr)
			s.printf("BT 0 Tw %.2f %.2f Td [", (f.x+dx)*k, (f.h-(f.y+.5*h+.3*f.fontSize))*k)
			t := Convert(txtStr).Split(" ")
			shift := float64((wmax - strSize)) / float64(len(t)-1)
			numt := len(t)
			for i := 0; i < numt; i++ {
				tx := t[i]
				tx = "(" + f.escape(utf8toutf16(tx, false)) + ")"
				s.printf("%s ", tx)
				if (i + 1) < numt {
					s.printf("%.3f(%s) ", -shift, space)
				}
			}
			s.printf("] TJ ET")
		} else {
			var txt2 string
			if f.isCurrentUTF8 {
				if f.isRTL {
					txtStr = reverseText(txtStr)
				}
				txt2 = f.escape(utf8toutf16(txtStr, false))
				for _, uni := range txtStr {
					f.currentFont.usedRunes[int(uni)] = int(uni)
				}
			} else {

				txt2 = Convert(txtStr).Replace("\\", "\\\\").Replace("(", "\\(").Replace(")", "\\)").String()
			}
			bt := (f.x + dx) * k
			td := (f.h - (f.y + dy + .5*h + .3*f.fontSize)) * k
			s.printf("BT %.2f %.2f Td (%s)Tj ET", bt, td, txt2)
			//BT %.2F %.2F Td (%s) Tj ET',(f.x+dx)*k,(f.h-(f.y+.5*h+.3*f.FontSize))*k,txt2);
		}

		if f.underline {
			s.printf(" %s", f.dounderline(f.x+dx, f.y+dy+.5*h+.3*f.fontSize, txtStr))
		}
		if f.strikeout {
			s.printf(" %s", f.dostrikeout(f.x+dx, f.y+dy+.5*h+.3*f.fontSize, txtStr))
		}
		if f.colorFlag {
			s.printf(" Q")
		}
		if link > 0 || len(linkStr) > 0 {
			f.newLink(f.x+dx, f.y+dy+.5*h-.5*f.fontSize, f.GetStringWidth(txtStr), f.fontSize, link, linkStr)
		}
	}
	str := s.String()
	if len(str) > 0 {
		f.out(str)
	}
	f.lasth = h
	if ln > 0 {
		// Go to next line
		f.y += h
		if ln == 1 {
			f.x = f.lMargin
		}
	} else {
		f.x += w
	}
}

// Revert string to use in RTL languages
func reverseText(text string) string {
	oldText := []rune(text)
	newText := make([]rune, len(oldText))
	length := len(oldText) - 1
	for i, r := range oldText {
		newText[length-i] = r
	}
	return string(newText)
}

// Cell is a simpler version of CellFormat with no fill, border, links or
// special alignment. The Cell_strikeout() example demonstrates this method.
func (f *Fpdf) Cell(w, h float64, txtStr string) {
	f.CellFormat(w, h, txtStr, "", 0, "L", false, 0, "")
}

// Cellf is a simpler printf-style version of CellFormat with no fill, border,
// links or special alignment. See documentation for the fmt package for
// details on fmtStr and args.
func (f *Fpdf) Cellf(w, h float64, fmtStr string, args ...any) {
	f.CellFormat(w, h, sprintf(fmtStr, args...), "", 0, "L", false, 0, "")
}

// SplitLines splits text into several lines using the current font. Each line
// has its length limited to a maximum width given by w. This function can be
// used to determine the total height of wrapped text for vertical placement
// purposes.
//
// This method is useful for codepage-based fonts only. For UTF-8 encoded text,
// use SplitText().
//
// You can use MultiCell if you want to print a text on several lines in a
// simple way.
func (f *Fpdf) SplitLines(txt []byte, w float64) [][]byte {
	// Function contributed by Bruno Michel
	lines := [][]byte{}
	cw := f.currentFont.Cw
	wmax := int(math.Ceil((w - 2*f.cMargin) * 1000 / f.fontSize))
	s := []byte(Convert(string(txt)).Replace("\r", "").String())
	nb := len(s)
	for nb > 0 && s[nb-1] == '\n' {
		nb--
	}
	s = s[0:nb]
	sep := -1
	i := 0
	j := 0
	l := 0
	for i < nb {
		c := s[i]
		l += cw[c]
		if c == ' ' || c == '\t' || c == '\n' {
			sep = i
		}
		if c == '\n' || l > wmax {
			if sep == -1 {
				if i == j {
					i++
				}
				sep = i
			} else {
				i = sep + 1
			}
			lines = append(lines, s[j:sep])
			sep = -1
			j = i
			l = 0
		} else {
			i++
		}
	}
	if i != j {
		lines = append(lines, s[j:i])
	}
	return lines
}

// MultiCell supports printing text with line breaks. They can be automatic (as
// soon as the text reaches the right border of the cell) or explicit (via the
// \n character). As many cells as necessary are output, one below the other.
//
// Text can be aligned, centered or justified. The cell block can be framed and
// the background painted. See CellFormat() for more details.
//
// The current position after calling MultiCell() is the beginning of the next
// line, equivalent to calling CellFormat with ln equal to 1.
//
// w is the width of the cells. A value of zero indicates cells that reach to
// the right margin.
//
// h indicates the line height of each cell in the unit of measure specified in New().
//
// Note: this method has a known bug that treats UTF-8 fonts differently than
// non-UTF-8 fonts. With UTF-8 fonts, all trailing newlines in txtStr are
// removed. With a non-UTF-8 font, if txtStr has one or more trailing newlines,
// only the last is removed. In the next major module version, the UTF-8 logic
// will be changed to match the non-UTF-8 logic. To prepare for that change,
// applications that use UTF-8 fonts and depend on having all trailing newlines
// removed should call TrimRight(txtStr, "\r\n") before calling this
// method.
func (f *Fpdf) MultiCell(w, h float64, txtStr, borderStr, alignStr string, fill bool) {
	if f.err != nil {
		return
	}
	// dbg("MultiCell")
	if alignStr == "" {
		alignStr = "J"
	}
	cw := f.currentFont.Cw
	if w == 0 {
		w = f.w - f.rMargin - f.x
	}
	wmax := int(math.Ceil((w - 2*f.cMargin) * 1000 / f.fontSize))
	s := Convert(txtStr).Replace("\r", "").String()
	srune := []rune(s)

	// remove extra line breaks
	var nb int
	if f.isCurrentUTF8 {
		nb = len(srune)
		for nb > 0 && srune[nb-1] == '\n' {
			nb--
		}
		srune = srune[0:nb]
	} else {
		nb = len(s)
		bytes2 := []byte(s)

		// for nb > 0 && bytes2[nb-1] == '\n' {

		// Prior to August 2019, if s ended with a newline, this code stripped it.
		// After that date, to be compatible with the UTF-8 code above, *all*
		// trailing newlines were removed. Because this regression caused at least
		// one application to break (see issue #333), the original behavior has been
		// reinstated with a caveat included in the documentation.
		if nb > 0 && bytes2[nb-1] == '\n' {
			nb--
		}
		s = s[0:nb]
	}
	// dbg("[%s]\n", s)
	var b, b2 string
	b = "0"
	if len(borderStr) > 0 {
		if borderStr == "1" {
			borderStr = "LTRB"
			b = "LRT"
			b2 = "LR"
		} else {
			b2 = ""
			if Contains(borderStr, "L") {
				b2 += "L"
			}
			if Contains(borderStr, "R") {
				b2 += "R"
			}
			if Contains(borderStr, "T") {
				b = b2 + "T"
			} else {
				b = b2
			}
		}
	}
	sep := -1
	i := 0
	j := 0
	l := 0
	ls := 0
	ns := 0
	nl := 1
	for i < nb {
		// Get next character
		var c rune
		if f.isCurrentUTF8 {
			c = srune[i]
		} else {
			c = rune(s[i])
		}
		if c == '\n' {
			// Explicit line break
			if f.ws > 0 {
				f.ws = 0
				f.out("0 Tw")
			}

			if f.isCurrentUTF8 {
				newAlignStr := alignStr
				if newAlignStr == "J" {
					if f.isRTL {
						newAlignStr = "R"
					} else {
						newAlignStr = "L"
					}
				}
				f.CellFormat(w, h, string(srune[j:i]), b, 2, newAlignStr, fill, 0, "")
			} else {
				f.CellFormat(w, h, s[j:i], b, 2, alignStr, fill, 0, "")
			}
			i++
			sep = -1
			j = i
			l = 0
			ns = 0
			nl++
			if len(borderStr) > 0 && nl == 2 {
				b = b2
			}
			continue
		}
		if c == ' ' || isChinese(c) {
			sep = i
			ls = l
			ns++
		}
		if int(c) >= len(cw) {
			f.err = Errf("character outside the supported range: %s", string(c))
			return
		}
		if cw[int(c)] == 0 { //Marker width 0 used for missing symbols
			l += f.currentFont.Desc.MissingWidth
		} else if cw[int(c)] != 65535 { //Marker width 65535 used for zero width symbols
			l += cw[int(c)]
		}
		if l > wmax {
			// Automatic line break
			if sep == -1 {
				if i == j {
					i++
				}
				if f.ws > 0 {
					f.ws = 0
					f.out("0 Tw")
				}
				if f.isCurrentUTF8 {
					f.CellFormat(w, h, string(srune[j:i]), b, 2, alignStr, fill, 0, "")
				} else {
					f.CellFormat(w, h, s[j:i], b, 2, alignStr, fill, 0, "")
				}
			} else {
				if alignStr == "J" {
					if ns > 1 {
						f.ws = float64((wmax-ls)/1000) * f.fontSize / float64(ns-1)
					} else {
						f.ws = 0
					}
					// f.outf("%.3f Tw", f.ws*f.k)
					f.putF64(f.ws*f.k, 3)
					f.put(" Tw\n")
				}
				if f.isCurrentUTF8 {
					f.CellFormat(w, h, string(srune[j:sep]), b, 2, alignStr, fill, 0, "")
				} else {
					f.CellFormat(w, h, s[j:sep], b, 2, alignStr, fill, 0, "")
				}
				i = sep + 1
			}
			sep = -1
			j = i
			l = 0
			ns = 0
			nl++
			if len(borderStr) > 0 && nl == 2 {
				b = b2
			}
		} else {
			i++
		}
	}
	// Last chunk
	if f.ws > 0 {
		f.ws = 0
		f.out("0 Tw")
	}
	if len(borderStr) > 0 && Contains(borderStr, "B") {
		b += "B"
	}
	if f.isCurrentUTF8 {
		if alignStr == "J" {
			if f.isRTL {
				alignStr = "R"
			} else {
				alignStr = ""
			}
		}
		f.CellFormat(w, h, string(srune[j:i]), b, 2, alignStr, fill, 0, "")
	} else {
		f.CellFormat(w, h, s[j:i], b, 2, alignStr, fill, 0, "")
	}
	f.x = f.lMargin
}

// write outputs text in flowing mode
func (f *Fpdf) write(h float64, txtStr string, link int, linkStr string) {
	// dbg("Write")
	cw := f.currentFont.Cw
	w := f.w - f.rMargin - f.x
	wmax := (w - 2*f.cMargin) * 1000 / f.fontSize
	s := Convert(txtStr).Replace("\r", "").String()
	var nb int
	if f.isCurrentUTF8 {
		nb = len([]rune(s))
		if nb == 1 && s == " " {
			f.x += f.GetStringWidth(s)
			return
		}
	} else {
		nb = len(s)
	}
	sep := -1
	i := 0
	j := 0
	l := 0.0
	nl := 1
	for i < nb {
		// Get next character
		var c rune
		if f.isCurrentUTF8 {
			c = []rune(s)[i]
		} else {
			c = rune(byte(s[i]))
		}
		if c == '\n' {
			// Explicit line break
			if f.isCurrentUTF8 {
				f.CellFormat(w, h, string([]rune(s)[j:i]), "", 2, "", false, link, linkStr)
			} else {
				f.CellFormat(w, h, s[j:i], "", 2, "", false, link, linkStr)
			}
			i++
			sep = -1
			j = i
			l = 0.0
			if nl == 1 {
				f.x = f.lMargin
				w = f.w - f.rMargin - f.x
				wmax = (w - 2*f.cMargin) * 1000 / f.fontSize
			}
			nl++
			continue
		}
		if c == ' ' {
			sep = i
		}
		l += float64(cw[int(c)])
		if l > wmax {
			// Automatic line break
			if sep == -1 {
				if f.x > f.lMargin {
					// Move to next line
					f.x = f.lMargin
					f.y += h
					w = f.w - f.rMargin - f.x
					wmax = (w - 2*f.cMargin) * 1000 / f.fontSize
					i++
					nl++
					continue
				}
				if i == j {
					i++
				}
				if f.isCurrentUTF8 {
					f.CellFormat(w, h, string([]rune(s)[j:i]), "", 2, "", false, link, linkStr)
				} else {
					f.CellFormat(w, h, s[j:i], "", 2, "", false, link, linkStr)
				}
			} else {
				if f.isCurrentUTF8 {
					f.CellFormat(w, h, string([]rune(s)[j:sep]), "", 2, "", false, link, linkStr)
				} else {
					f.CellFormat(w, h, s[j:sep], "", 2, "", false, link, linkStr)
				}
				i = sep + 1
			}
			sep = -1
			j = i
			l = 0.0
			if nl == 1 {
				f.x = f.lMargin
				w = f.w - f.rMargin - f.x
				wmax = (w - 2*f.cMargin) * 1000 / f.fontSize
			}
			nl++
		} else {
			i++
		}
	}
	// Last chunk
	if i != j {
		if f.isCurrentUTF8 {
			f.CellFormat(l/1000*f.fontSize, h, string([]rune(s)[j:]), "", 0, "", false, link, linkStr)
		} else {
			f.CellFormat(l/1000*f.fontSize, h, s[j:], "", 0, "", false, link, linkStr)
		}
	}
}

// Write prints text from the current position. When the right margin is
// reached (or the \n character is met) a line break occurs and text continues
// from the left margin. Upon method exit, the current position is left just at
// the end of the text.
//
// It is possible to put a link on the text.
//
// h indicates the line height in the unit of measure specified in New().
func (f *Fpdf) Write(h float64, txtStr string) {
	f.write(h, txtStr, 0, "")
}

// Writef is like Write but uses printf-style formatting. See the documentation
// for package fmt for more details on fmtStr and args.
func (f *Fpdf) Writef(h float64, fmtStr string, args ...any) {
	f.write(h, sprintf(fmtStr, args...), 0, "")
}

// WriteLinkString writes text that when clicked launches an external URL. See
// Write() for argument details.
func (f *Fpdf) WriteLinkString(h float64, displayStr, targetStr string) {
	f.write(h, displayStr, 0, targetStr)
}

// WriteLinkID writes text that when clicked jumps to another location in the
// PDF. linkID is an identifier returned by AddLink(). See Write() for argument
// details.
func (f *Fpdf) WriteLinkID(h float64, displayStr string, linkID int) {
	f.write(h, displayStr, linkID, "")
}

// WriteAligned is an implementation of Write that makes it possible to align
// text.
//
// width indicates the width of the box the text will be drawn in. This is in
// the unit of measure specified in New(). If it is set to 0, the bounding box
// of the page will be taken (pageWidth - leftMargin - rightMargin).
//
// lineHeight indicates the line height in the unit of measure specified in
// New().
//
// alignStr sees to horizontal alignment of the given textStr. The options are
// "L", "C" and "R" (Left, Center, Right). The default is "L".
func (f *Fpdf) WriteAligned(width, lineHeight float64, textStr, alignStr string) {
	lMargin, _, rMargin, _ := f.GetMargins()

	pageWidth, _ := f.GetPageSize()
	if width == 0 {
		width = pageWidth - (lMargin + rMargin)
	}

	var lines []string

	if f.isCurrentUTF8 {
		lines = f.SplitText(textStr, width)
	} else {
		for _, line := range f.SplitLines([]byte(textStr), width) {
			lines = append(lines, string(line))
		}
	}

	for _, lineBt := range lines {
		lineStr := string(lineBt)
		lineWidth := f.GetStringWidth(lineStr)

		switch alignStr {
		case "C":
			f.SetLeftMargin(lMargin + ((width - lineWidth) / 2))
			f.Write(lineHeight, lineStr)
			f.SetLeftMargin(lMargin)
		case "R":
			f.SetLeftMargin(lMargin + (width - lineWidth) - 2.01*f.cMargin)
			f.Write(lineHeight, lineStr)
			f.SetLeftMargin(lMargin)
		default:
			f.SetRightMargin(pageWidth - lMargin - width)
			f.Write(lineHeight, lineStr)
			f.SetRightMargin(rMargin)
		}
	}
}

// Ln performs a line break. The current abscissa goes back to the left margin
// and the ordinate increases by the amount passed in parameter. A negative
// value of h indicates the height of the last printed cell.
//
// This method is demonstrated in the example for MultiCell.
func (f *Fpdf) Ln(h float64) {
	f.x = f.lMargin
	if h < 0 {
		f.y += f.lasth
	} else {
		f.y += h
	}
}

// ImageTypeFromMime returns the image type used in various image-related
// functions (for example, Image()) that is associated with the specified MIME
// type. For example, "jpg" is returned if mimeStr is "image/jpeg". An error is
// set if the specified MIME type is not supported.
func (f *Fpdf) ImageTypeFromMime(mimeStr string) (tp string) {
	switch mimeStr {
	case "image/png":
		tp = "png"
	case "image/jpg":
		tp = "jpg"
	case "image/jpeg":
		tp = "jpg"
	case "image/gif":
		tp = "gif"
	default:
		f.SetErrorf("unsupported image type: %s", mimeStr)
	}
	return
}

func (f *Fpdf) imageOut(info *ImageInfoType, x, y, w, h float64, allowNegativeX, flow bool, link int, linkStr string) {
	// Automatic width and height calculation if needed
	if w == 0 && h == 0 {
		// Put image at 96 dpi
		w = -96
		h = -96
	}
	if w == -1 {
		// Set image width to whatever value for dpi we read
		// from the image or that was set manually
		w = -info.dpi
	}
	if h == -1 {
		// Set image height to whatever value for dpi we read
		// from the image or that was set manually
		h = -info.dpi
	}
	if w < 0 {
		w = -info.w * 72.0 / w / f.k
	}
	if h < 0 {
		h = -info.h * 72.0 / h / f.k
	}
	if w == 0 {
		w = h * info.w / info.h
	}
	if h == 0 {
		h = w * info.h / info.w
	}
	// Flowing mode
	if flow {
		if f.y+h > f.pageBreakTrigger && !f.inHeader && !f.inFooter && f.acceptPageBreak() {
			// Automatic page break
			x2 := f.x
			f.AddPageFormat(f.curOrientation, f.curPageSize)
			if f.err != nil {
				return
			}
			f.x = x2
		}
		y = f.y
		f.y += h
	}
	if !allowNegativeX {
		if x < 0 {
			x = f.x
		}
	}
	// dbg("h %.2f", h)
	// q 85.04 0 0 NaN 28.35 NaN cm /I2 Do Q
	// f.outf("q %.5f 0 0 %.5f %.5f %.5f cm /I%s Do Q", w*f.k, h*f.k, x*f.k, (f.h-(y+h))*f.k, info.i)
	const prec = 5
	f.put("q ")
	f.putF64(w*f.k, prec)
	f.put(" 0 0 ")
	f.putF64(h*f.k, prec)
	f.put(" ")
	f.putF64(x*f.k, prec)
	f.put(" ")
	f.putF64((f.h-(y+h))*f.k, prec)
	f.put(" cm /I" + info.i + " Do Q\n")
	if link > 0 || len(linkStr) > 0 {
		f.newLink(x, y, w, h, link, linkStr)
	}
}

// Image puts a JPEG, PNG or GIF image in the current page.
//
// Deprecated in favor of ImageOptions -- see that function for
// details on the behavior of arguments
func (f *Fpdf) Image(imageNameStr string, x, y, w, h float64, flow bool, tp string, link int, linkStr string) {
	options := ImageOptions{
		ReadDpi:   false,
		ImageType: tp,
	}
	f.ImageOptions(imageNameStr, x, y, w, h, flow, options, link, linkStr)
}

// ImageOptions puts a JPEG, PNG or GIF image in the current page. The size it
// will take on the page can be specified in different ways. If both w and h
// are 0, the image is rendered at 96 dpi. If either w or h is zero, it will be
// calculated from the other dimension so that the aspect ratio is maintained.
// If w and/or h are -1, the dpi for that dimension will be read from the
// ImageInfoType object. PNG files can contain dpi information, and if present,
// this information will be populated in the ImageInfoType object and used in
// Width, Height, and Extent calculations. Otherwise, the SetDpi function can
// be used to change the dpi from the default of 72.
//
// If w and h are any other negative value, their absolute values
// indicate their dpi extents.
//
// Supported JPEG formats are 24 bit, 32 bit and gray scale. Supported PNG
// formats are 24 bit, indexed color, and 8 bit indexed gray scale. If a GIF
// image is animated, only the first frame is rendered. Transparency is
// supported. It is possible to put a link on the image.
//
// imageNameStr may be the name of an image as registered with a call to either
// RegisterImageReader() or RegisterImage(). In the first case, the image is
// loaded using an io.Reader. This is generally useful when the image is
// obtained from some other means than as a disk-based file. In the second
// case, the image is loaded as a file. Alternatively, imageNameStr may
// directly specify a sufficiently qualified filename.
//
// However the image is loaded, if it is used more than once only one copy is
// embedded in the file.
//
// If x is negative, the current abscissa is used.
//
// If flow is true, the current y value is advanced after placing the image and
// a page break may be made if necessary.
//
// If link refers to an internal page anchor (that is, it is non-zero; see
// AddLink()), the image will be a clickable internal link. Otherwise, if
// linkStr specifies a URL, the image will be a clickable external link.
func (f *Fpdf) ImageOptions(imageNameStr string, x, y, w, h float64, flow bool, options ImageOptions, link int, linkStr string) {
	if f.err != nil {
		return
	}
	info := f.RegisterImageOptions(imageNameStr, options)
	if f.err != nil {
		return
	}
	f.imageOut(info, x, y, w, h, options.AllowNegativePosition, flow, link, linkStr)
}

// RegisterImageReader registers an image, reading it from Reader r, adding it
// to the PDF file but not adding it to the page.
//
// This function is now deprecated in favor of RegisterImageOptionsReader
func (f *Fpdf) RegisterImageReader(imgName, tp string, r io.Reader) (info *ImageInfoType) {
	options := ImageOptions{
		ReadDpi:   false,
		ImageType: tp,
	}
	return f.RegisterImageOptionsReader(imgName, options, r)
}

// ImageOptions provides a place to hang any options we want to use while
// parsing an image.
//
// ImageType's possible values are (case insensitive):
// "JPG", "JPEG", "PNG" and "GIF". If empty, the type is inferred from
// the file extension.
//
// ReadDpi defines whether to attempt to automatically read the image
// dpi information from the image file. Normally, this should be set
// to true (understanding that not all images will have this info
// available). However, for backwards compatibility with previous
// versions of the API, it defaults to false.
//
// AllowNegativePosition can be set to true in order to prevent the default
// coercion of negative x values to the current x position.
type ImageOptions struct {
	ImageType             string
	ReadDpi               bool
	AllowNegativePosition bool
}

// RegisterImageOptionsReader registers an image, reading it from Reader r, adding it
// to the PDF file but not adding it to the page. Use Image() with the same
// name to add the image to the page. Note that tp should be specified in this
// case.
//
// See Image() for restrictions on the image and the options parameters.
func (f *Fpdf) RegisterImageOptionsReader(imgName string, options ImageOptions, r io.Reader) (info *ImageInfoType) {
	// Thanks, Ivan Daniluk, for generalizing this code to use the Reader interface.
	if f.err != nil {
		return
	}
	info, ok := f.images[imgName]
	if ok {
		return
	}

	// First use of this image, get info
	if options.ImageType == "" {
		f.err = Errf("image type should be specified if reading from custom reader")
		return
	}
	options.ImageType = Convert(options.ImageType).ToLower().String()
	if options.ImageType == "jpeg" {
		options.ImageType = "jpg"
	}
	switch options.ImageType {
	case "jpg":
		info = f.parsejpg(r)
	case "png":
		info = f.parsepng(r, options.ReadDpi)
	case "gif":
		info = f.parsegif(r)
	default:
		f.err = Errf("unsupported image type: %s", options.ImageType)
	}
	if f.err != nil {
		return
	}

	if info.i, f.err = generateImageID(info); f.err != nil {
		return
	}
	f.images[imgName] = info

	return
}

// RegisterImage registers an image, adding it to the PDF file but not adding
// it to the page. Use Image() with the same filename to add the image to the
// page. Note that Image() calls this function, so this function is only
// necessary if you need information about the image before placing it.
//
// This function is now deprecated in favor of RegisterImageOptions.
// See Image() for restrictions on the image and the "tp" parameters.
func (f *Fpdf) RegisterImage(fileStr, tp string) (info *ImageInfoType) {
	options := ImageOptions{
		ReadDpi:   false,
		ImageType: tp,
	}
	return f.RegisterImageOptions(fileStr, options)
}

// RegisterImageOptions registers an image, adding it to the PDF file but not
// adding it to the page. Use Image() with the same filename to add the image
// to the page. Note that Image() calls this function, so this function is only
// necessary if you need information about the image before placing it. See
// Image() for restrictions on the image and the "tp" parameters.
func (f *Fpdf) RegisterImageOptions(fileStr string, options ImageOptions) (info *ImageInfoType) {
	info, ok := f.images[fileStr]
	if ok {
		return
	}

	data, err := f.files.ReadFile(fileStr)
	if err != nil {
		f.err = err
		return
	}

	// First use of this image, get info
	if options.ImageType == "" {
		pos := LastIndex(fileStr, ".")
		if pos < 0 {
			f.err = Errf("image file has no extension and no type was specified: %s", fileStr)
			return
		}
		options.ImageType = fileStr[pos+1:]
	}

	return f.RegisterImageOptionsReader(fileStr, options, bytes.NewReader(data))
}

// GetImageInfo returns information about the registered image specified by
// imageStr. If the image has not been registered, nil is returned. The
// internal error is not modified by this method.
func (f *Fpdf) GetImageInfo(imageStr string) (info *ImageInfoType) {
	return f.images[imageStr]
}

// GetConversionRatio returns the conversion ratio based on the unit given when
// creating the PDF.
func (f *Fpdf) GetConversionRatio() float64 {
	return f.k
}

// SetHomeXY is a convenience method that sets the current position to the left
// and top margins.
func (f *Fpdf) SetHomeXY() {
	f.SetY(f.tMargin)
	f.SetX(f.lMargin)
}

// Escape special characters in strings
func (f *Fpdf) escape(s string) string {
	// Usar tinystring para reemplazos encadenados
	s = Convert(s).Replace("\\", "\\\\").Replace("(", "\\(").Replace(")", "\\)").Replace("\r", "\\r").String()
	return s
}

// textstring formats a text string
func (f *Fpdf) textstring(s string) string {
	if f.protect.encrypted {
		b := []byte(s)
		f.protect.rc4(uint32(f.n), &b)
		s = string(b)
	}
	return "(" + f.escape(s) + ")"
}

func blankCount(str string) (count int) {
	l := len(str)
	for j := 0; j < l; j++ {
		if byte(' ') == str[j] {
			count++
		}
	}
	return
}

// GetUnderlineThickness returns the current text underline thickness multiplier.
func (f *Fpdf) GetUnderlineThickness() float64 {
	return f.userUnderlineThickness
}

// SetUnderlineThickness accepts a multiplier for adjusting the text underline
// thickness, defaulting to 1. See SetUnderlineThickness example.
func (f *Fpdf) SetUnderlineThickness(thickness float64) {
	f.userUnderlineThickness = thickness
}

// Underline text
func (f *Fpdf) dounderline(x, y float64, txt string) string {
	up := float64(f.currentFont.Up)
	ut := float64(f.currentFont.Ut) * f.userUnderlineThickness
	w := f.GetStringWidth(txt) + f.ws*float64(blankCount(txt))
	return sprintf("%.2f %.2f %.2f %.2f re f", x*f.k,
		(f.h-(y-up/1000*f.fontSize))*f.k, w*f.k, -ut/1000*f.fontSizePt)
}

func (f *Fpdf) dostrikeout(x, y float64, txt string) string {
	up := float64(f.currentFont.Up)
	ut := float64(f.currentFont.Ut)
	w := f.GetStringWidth(txt) + f.ws*float64(blankCount(txt))
	return sprintf("%.2f %.2f %.2f %.2f re f", x*f.k,
		(f.h-(y+4*up/1000*f.fontSize))*f.k, w*f.k, -ut/1000*f.fontSizePt)
}

func (f *Fpdf) newImageInfo() *ImageInfoType {
	// default dpi to 72 unless told otherwise
	return &ImageInfoType{scale: f.k, dpi: 72}
}

// parsejpg extracts info from io.Reader with JPEG data
// Thank you, Bruno Michel, for providing this code.
func (f *Fpdf) parsejpg(r io.Reader) (info *ImageInfoType) {
	info = f.newImageInfo()
	var (
		data bytes.Buffer
		err  error
	)
	_, err = data.ReadFrom(r)
	if err != nil {
		f.err = err
		return
	}
	info.data = data.Bytes()

	config, err := jpeg.DecodeConfig(bytes.NewReader(info.data))
	if err != nil {
		f.err = err
		return
	}
	info.w = float64(config.Width)
	info.h = float64(config.Height)
	info.f = "DCTDecode"
	info.bpc = 8
	switch config.ColorModel {
	case color.GrayModel:
		info.cs = "DeviceGray"
	case color.YCbCrModel:
		info.cs = "DeviceRGB"
	case color.CMYKModel:
		info.cs = "DeviceCMYK"
	default:
		f.err = Errf("image JPEG buffer has unsupported color space (%v)", config.ColorModel)
		return
	}
	return
}

// parsepng extracts info from a PNG data
func (f *Fpdf) parsepng(r io.Reader, readdpi bool) (info *ImageInfoType) {
	buf, err := newRBuffer(r)
	if err != nil {
		f.err = err
		return
	}
	return f.parsepngstream(buf, readdpi)
}

// newobj begins a new object
func (f *Fpdf) newobj() {
	// dbg("newobj")
	f.n++
	for j := len(f.offsets); j <= f.n; j++ {
		f.offsets = append(f.offsets, 0)
	}
	f.offsets[f.n] = f.buffer.Len()
	f.outf("%d 0 obj", f.n)
}

func (f *Fpdf) putstream(b []byte) {
	// dbg("putstream")
	if f.protect.encrypted {
		f.protect.rc4(uint32(f.n), &b)
	}
	f.out("stream")
	f.out(string(b))
	f.out("endstream")
}

// errNoPage is set when content is written before the first page exists: without it the
// content would land before the %PDF header and corrupt the file silently.
const errNoPage = "fpdf: content written before the first AddPage; call AddPage first or pass fpdf.AutoFirstPage to New"

// beforeFirstPage handles content that arrives while no page exists (state 0): it opens the
// first page when AutoFirstPage was requested, otherwise it records errNoPage and reports
// that the content must be dropped.
func (f *Fpdf) beforeFirstPage() (drop bool) {
	if f.state != 0 {
		return false
	}
	if f.autoFirstPage {
		f.AddPage()
		return f.state != 2
	}
	if f.err == nil {
		f.err = Errf(errNoPage)
	}
	return true
}

// out; Add a line to the document
func (f *Fpdf) out(s string) {
	if f.beforeFirstPage() {
		return
	}
	if f.state == 2 {
		must(f.pages[f.page].WriteString(s))
		must(f.pages[f.page].WriteString("\n"))
	} else {
		must(f.buffer.WriteString(s))
		must(f.buffer.WriteString("\n"))
	}
}

func (f *Fpdf) put(s string) {
	if f.beforeFirstPage() {
		return
	}
	if f.state == 2 {
		f.pages[f.page].WriteString(s)
	} else {
		f.buffer.WriteString(s)
	}
}

// outbuf adds a buffered line to the document
func (f *Fpdf) outbuf(r io.Reader) {
	if f.state == 2 {
		must64(f.pages[f.page].ReadFrom(r))
		must(f.pages[f.page].WriteString("\n"))
	} else {
		must64(f.buffer.ReadFrom(r))
		must(f.buffer.WriteString("\n"))
	}
}

// RawWriteStr writes a string directly to the PDF generation buffer. This is a
// low-level function that is not required for normal PDF construction. An
// understanding of the PDF specification is needed to use this method
// correctly.
func (f *Fpdf) RawWriteStr(str string) {
	f.out(str)
}

// RawWriteBuf writes the contents of the specified buffer directly to the PDF
// generation buffer. This is a low-level function that is not required for
// normal PDF construction. An understanding of the PDF specification is needed
// to use this method correctly.
func (f *Fpdf) RawWriteBuf(r io.Reader) {
	f.outbuf(r)
}

// outf adds a formatted line to the document
func (f *Fpdf) outf(fmtStr string, args ...any) {
	f.out(sprintf(fmtStr, args...))
}

func (f *Fpdf) putF64(v float64, prec int) {
	f.put(f.fmtF64(v, prec))
}

// fmtF64 converts the floating-point number f to a string with precision prec.
func (f *Fpdf) fmtF64(v float64, prec int) string {
	// Usar tinystring para formatear float con precisión
	return Convert(v).Round(prec).String()
}

func (f *Fpdf) putInt(v int) {
	f.put(f.fmtInt(v))
}

func (f *Fpdf) fmtInt(v int) string {
	// Usar tinystring para convertir int a string
	return Convert(v).String()
}

// SetDefaultCatalogSort sets the default value of the catalog sort flag that
// will be used when initializing a new Fpdf instance. See SetCatalogSort() for
// more details.
func SetDefaultCatalogSort(flag bool) {
	gl.catalogSort = flag
}

// GetCatalogSort returns the document's internal catalog sort flag.
func (f *Fpdf) GetCatalogSort() bool {
	return f.catalogSort
}

// SetCatalogSort sets a flag that will be used, if true, to consistently order
// the document's internal resource catalogs. This method is typically only
// used for test purposes to facilitate PDF comparison.
func (f *Fpdf) SetCatalogSort(flag bool) {
	f.catalogSort = flag
}

// RegisterAlias adds an (alias, replacement) pair to the document so we can
// replace all occurrences of that alias after writing but before the document
// is closed. Functions ExampleFpdf_RegisterAlias() and
// ExampleFpdf_RegisterAlias_utf8() in fpdf_test.go demonstrate this method.
func (f *Fpdf) RegisterAlias(alias, replacement string) {
	// Note: map[string]string assignments embed literal escape ("\00") sequences
	// into utf16 key and value  Consequently, subsequent search/replace
	// operations will fail unexpectedly if utf8toutf16() conversions take place
	// here. Instead, conversions are deferred until the actual search/replace
	// operation takes place when the PDF output is generated.
	f.aliasMap[alias] = replacement
}

func (f *Fpdf) replaceAliases() {
	for mode := 0; mode < 2; mode++ {
		for alias, replacement := range f.aliasMap {
			if mode == 1 {
				alias = utf8toutf16(alias, false)
				replacement = utf8toutf16(replacement, false)
			}
			for n := 1; n <= f.page; n++ {
				s := f.pages[n].String()
				if Contains(s, alias) {
					s = Convert(s).Replace(alias, replacement).String()
					f.pages[n].Truncate(0)
					f.pages[n].WriteString(s)
				}
			}
		}
	}
}

// errFilesNotConfigured is returned by every file access of an Fpdf built without a
// files.ReadWriter option.
const errFilesNotConfigured = "fpdf: no files.ReadWriter configured; pass one to fpdf.New"

type unconfiguredFiles struct{}

func (unconfiguredFiles) ReadFile(string) ([]byte, error) { return nil, Errf(errFilesNotConfigured) }
func (unconfiguredFiles) WriteFile(string, []byte) error  { return Errf(errFilesNotConfigured) }
