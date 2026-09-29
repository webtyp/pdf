package fpdf

import (
	"webtyp.com/files"
	"webtyp.com/model"
)

import (
	"bytes"
	"encoding/binary"
	"io"
	"math"
	"path"

	"webtyp.com/fmt"
)

// Version of FPDF from which this package is derived
const (
	cnFpdfVersion = "1.7"
)

type blendModeType struct {
	strokeStr, fillStr, modeStr string
	objNum                      int
}

type gradientType struct {
	tp                int // 2: linear, 3: radial
	clr1Str, clr2Str  string
	x1, y1, x2, y2, r float64
	objNum            int
}

// FirstPage says what happens to content written before the first AddPage.
type FirstPage uint8

// AutoFirstPage, passed to New, opens page 1 automatically when the first content is written.
// Without it, such content is an error (errNoPage) instead of a corrupt file.
const AutoFirstPage FirstPage = 1

type RootDirectoryType string // RootDirectoryType is the root directory of the executable default is "." but test can set it to a different directory

// MakePath joins the root directory with one or more path elements.
// For example, with root "/home/user/docpdf":
//
//	r.MakePath("file.txt") returns "/home/user/docpdf/file.txt"
//	r.MakePath("dir", "file.txt") returns "/home/user/docpdf/dir/file.txt"
func (r RootDirectoryType) MakePath(pathElements ...string) string {
	elements := append([]string{string(r)}, pathElements...)
	return path.Join(elements...)
}

type FontsDirName string // FontsDirName is the name of the font directory default is "fonts"

type orientationType string

const (
	// Portrait represents the portrait orientation.
	Portrait orientationType = "p"

	// Landscape represents the landscape orientation.
	Landscape orientationType = "l"
)

type unit string

const (
	// POINT represents the size unit point
	POINT unit = "pt"
	// MM represents the size unit millimeter
	MM unit = "mm"
	// CM represents the size unit centimeter
	CM unit = "cm"
	// IN represents the size unit inch
	IN unit = "inch"
)

// Standard page sizes in points (1/72 inch)
var (
	// A3 represents DIN/ISO A3 page size
	A3 = PageSize{Wd: 841.89, Ht: 1190.55, AutoHt: false}
	// A4 represents DIN/ISO A4 page size
	A4 = PageSize{Wd: 595.28, Ht: 841.89, AutoHt: false}
	// A5 represents DIN/ISO A5 page size
	A5 = PageSize{Wd: 420.94, Ht: 595.28, AutoHt: false}
	// A6 represents DIN/ISO A6 page size
	A6 = PageSize{Wd: 297.64, Ht: 420.94, AutoHt: false}
	// A7 represents DIN/ISO A7 page size
	A7 = PageSize{Wd: 209.76, Ht: 297.64, AutoHt: false}
	// A2 represents DIN/ISO A2 page size
	A2 = PageSize{Wd: 1190.55, Ht: 1683.78, AutoHt: false}
	// A1 represents DIN/ISO A1 page size
	A1 = PageSize{Wd: 1683.78, Ht: 2383.94, AutoHt: false}
	// Letter represents US Letter page size
	Letter = PageSize{Wd: 612, Ht: 792, AutoHt: false}
	// Legal represents US Legal page size
	Legal = PageSize{Wd: 612, Ht: 1008, AutoHt: false}
	// Tabloid represents US Tabloid page size
	Tabloid = PageSize{Wd: 792, Ht: 1224, AutoHt: false}
)

const (
	// BorderNone set no border
	BorderNone = ""
	// BorderFull sets a full border
	BorderFull = "1"
	// BorderLeft sets the border on the left side
	BorderLeft = "L"
	// BorderTop sets the border at the top
	BorderTop = "T"
	// BorderRight sets the border on the right side
	BorderRight = "R"
	// BorderBottom sets the border on the bottom
	BorderBottom = "B"
)

const (
	// LineBreakNone disables linebreak
	LineBreakNone = 0
	// LineBreakNormal enables normal linebreak
	LineBreakNormal = 1
	// LineBreakBelow enables linebreak below
	LineBreakBelow = 2
)

const (
	// AlignLeft left aligns the cell
	AlignLeft = "L"
	// AlignRight right aligns the cell
	AlignRight = "R"
	// AlignCenter centers the cell
	AlignCenter = "C"
	// AlignTop aligns the cell to the top
	AlignTop = "T"
	// AlignBottom aligns the cell to the bottom
	AlignBottom = "B"
	// AlignMiddle aligns the cell to the middle
	AlignMiddle = "M"
	// AlignBaseline aligns the cell to the baseline
	AlignBaseline = "B"
)

type colorMode int

const (
	colorModeRGB colorMode = iota
	colorModeSpot
)

type colorType struct {
	r, g, b    float64
	ir, ig, ib int
	mode       colorMode
	spotStr    string // name of current spot color
	gray       bool
	str        string
}

// SpotColorType specifies a named spot color value
type spotColorType struct {
	id, objID int
	val       cmykColorType
}

// cmykColorType specifies an ink-based CMYK color value
type cmykColorType struct {
	c, m, y, k byte // 0% to 100%
}

// SizeType fields Wd and Ht specify the horizontal and vertical extents of a
// document element such as a page.
type SizeType struct {
	Wd, Ht float64
}

// PageSize specifies the dimensions and properties of a page.
// Wd and Ht specify the horizontal and vertical extents in points.
// AutoHt indicates if the page height should grow automatically based on content.
type PageSize struct {
	Wd, Ht float64
	AutoHt bool // For cases where page size needs to grow automatically (e.g., thermal printer paper)
}

// ToSizeType converts a PageSize to SizeType for compatibility
func (ps PageSize) ToSizeType() SizeType {
	return SizeType{Wd: ps.Wd, Ht: ps.Ht}
}

// PointType fields X and Y specify the horizontal and vertical coordinates of
// a point, typically used in drawing.
type PointType struct {
	X, Y float64
}

// XY returns the X and Y components of the receiver point.
func (p PointType) XY() (float64, float64) {
	return p.X, p.Y
}

// Extent returns the width and height of the page size
func (ps PageSize) Extent() (wd, ht float64) {
	return ps.Wd, ps.Ht
}

// Width returns the width of the page size
func (ps PageSize) Width() float64 {
	return ps.Wd
}

// Height returns the height of the page size
func (ps PageSize) Height() float64 {
	return ps.Ht
}

// ImageInfoType contains size, color and other information about an image.
// Changes to this structure should be reflected in its GobEncode and GobDecode
// methods.
type ImageInfoType struct {
	data  []byte  // Raw image data
	smask []byte  // Soft Mask, an 8bit per-pixel transparency mask
	n     int     // Image object number
	w     float64 // Width
	h     float64 // Height
	cs    string  // Color space
	pal   []byte  // Image color palette
	bpc   int     // Bits Per Component
	f     string  // Image filter
	dp    string  // DecodeParms
	trns  []int   // Transparency mask
	scale float64 // Document scale factor
	dpi   float64 // Dots-per-inch found from image file (png only)
	i     string  // SHA-1 checksum of the above values.
}

type idEncoder struct {
	w   io.Writer
	buf []byte
	err error
}

func newIDEncoder(w io.Writer) *idEncoder {
	return &idEncoder{
		w:   w,
		buf: make([]byte, 8),
	}
}

func (enc *idEncoder) i64(v int64) {
	if enc.err != nil {
		return
	}
	binary.LittleEndian.PutUint64(enc.buf, uint64(v))
	_, enc.err = enc.w.Write(enc.buf)
}

func (enc *idEncoder) f64(v float64) {
	if enc.err != nil {
		return
	}
	binary.LittleEndian.PutUint64(enc.buf, math.Float64bits(v))
	_, enc.err = enc.w.Write(enc.buf)
}

func (enc *idEncoder) str(v string) {
	if enc.err != nil {
		return
	}
	_, enc.err = enc.w.Write([]byte(v))
}

func (enc *idEncoder) bytes(v []byte) {
	if enc.err != nil {
		return
	}
	_, enc.err = enc.w.Write(v)
}

// PointConvert returns the value of pt, expressed in points (1/72 inch), as a
// value expressed in the unit of measure specified in New(). Since font
// management in Fpdf uses points, this method can help with line height
// calculations and other methods that require user units.
func (f *Fpdf) PointConvert(pt float64) (u float64) {
	return pt / f.k
}

// PointToUnitConvert is an alias for PointConvert.
func (f *Fpdf) PointToUnitConvert(pt float64) (u float64) {
	return pt / f.k
}

// UnitToPointConvert returns the value of u, expressed in the unit of measure
// specified in New(), as a value expressed in points (1/72 inch). Since font
// management in Fpdf uses points, this method can help with setting font sizes
// based on the sizes of other non-font page elements.
func (f *Fpdf) UnitToPointConvert(u float64) (pt float64) {
	return u * f.k
}

// Extent returns the width and height of the image in the units of the Fpdf
// object.
func (info *ImageInfoType) Extent() (wd, ht float64) {
	return info.Width(), info.Height()
}

// Width returns the width of the image in the units of the Fpdf object.
func (info *ImageInfoType) Width() float64 {
	return info.w / (info.scale * info.dpi / 72)
}

// Height returns the height of the image in the units of the Fpdf object.
func (info *ImageInfoType) Height() float64 {
	return info.h / (info.scale * info.dpi / 72)
}

// SetDpi sets the dots per inch for an image. PNG images MAY have their dpi
// set automatically, if the image specifies it. DPI information is not
// currently available automatically for JPG and GIF images, so if it's
// important to you, you can set it here. It defaults to 72 dpi.
func (info *ImageInfoType) SetDpi(dpi float64) {
	info.dpi = dpi
}

type fontFileType struct {
	length1, length2 int64
	n                int
	embedded         bool
	content          []byte
	fontType         string
}

type linkType struct {
	x, y, wd, ht float64
	link         int    // Auto-generated internal link ID or...
	linkStr      string // ...application-provided external link string
}

type intLinkType struct {
	page int
	y    float64
}

// outlineType is used for a sidebar outline of bookmarks
type outlineType struct {
	text                                   string
	level, parent, first, last, next, prev int
	y                                      float64
	p                                      int
}

// InitType is used with NewCustom() to customize an Fpdf instance.
// OrientationStr, UnitType correspond to the arguments accepted by New().
// If the Wd and Ht fields of Size are each greater than zero, Size will be used
// to set the default page size. Wd and Ht are specified in the units of measure
// indicated by UnitType.
type InitType struct {
	OrientationStr orientationType // Landscape or Portrait
	UnitType       unit
	Size           PageSize
	RootDirectory  RootDirectoryType // Root directory of the executable default is "." but test can set it to a different directory
	FontDirName    string            // name to the font directory default is "fonts"
}

// FontLoader is used to read fonts (JSON font specification and zlib compressed font binaries)
// from arbitrary locations (e.g. files, zip files, embedded font resources).
//
// Open provides an io.Reader for the specified font file (.json or .z). The file name
// never includes a path. Open returns an error if the specified file cannot be opened.
type FontLoader interface {
	Open(name string) (io.Reader, error)
}

// OutputIntentSubtype any of the pre defined types below or a value defined by ISO 32000 extension.
type OutputIntentSubtype string

const (
	OutputIntent_GTS_PDFX  OutputIntentSubtype = "GTS_PDFX"
	OutputIntent_GTS_PDFA1 OutputIntentSubtype = "GTS_PDFA1"
	OutputIntent_GTS_PDFE1 OutputIntentSubtype = "GTS_PDFE1"
)

// OutputIntentType defines an output intent with name and ICC color profile.
type OutputIntentType struct {
	SubtypeIdent              OutputIntentSubtype
	OutputConditionIdentifier string
	Info                      string
	ICCProfile                []byte
}

// PageBox defines the coordinates and extent of the various page box types
type PageBox struct {
	SizeType
	PointType
}

// Fpdf is the principal structure for creating a single PDF document
type Fpdf struct {
	isCurrentUTF8    bool                       // is current font used in utf-8 mode
	isRTL            bool                       // is is right to left mode enabled
	page             int                        // current page number
	n                int                        // current object number
	offsets          []int                      // array of object offsets
	buffer           fmtBuffer                  // buffer holding in-memory PDF
	pages            []*bytes.Buffer            // slice[page] of page content; 1-based
	state            int                        // current document state
	autoFirstPage    bool                       // open page 1 automatically on the first content (AutoFirstPage)
	compress         bool                       // compression flag
	k                float64                    // scale factor (number of points in user unit)
	defOrientation   orientationType            // default orientation
	curOrientation   orientationType            // current orientation
	stdPageSizes     map[string]PageSize        // standard page sizes
	defPageSize      PageSize                   // default page size
	defPageBoxes     map[string]PageBox         // default page size
	curPageSize      PageSize                   // current page size
	pageSizes        map[int]PageSize           // used for pages with non default sizes or orientations
	pageBoxes        map[int]map[string]PageBox // used to define the crop, trim, bleed and art boxes
	unitType         unit                       // unit of measure for all rendered objects except fonts
	wPt, hPt         float64                    // dimensions of current page in points
	w, h             float64                    // dimensions of current page in user unit
	lMargin          float64                    // left margin
	tMargin          float64                    // top margin
	rMargin          float64                    // right margin
	bMargin          float64                    // page break margin
	cMargin          float64                    // cell margin
	x, y             float64                    // current position in user unit
	lasth            float64                    // height of last printed cell
	lineWidth        float64                    // line width in user unit
	rootDirectory    RootDirectoryType          // root directory of the executable default is "." for test change
	fontsDirName     FontsDirName               // fonts directory name default is "fonts"
	fontsPath        string                     // full path containing fonts directory included rootDirectory eg. "/home/user/docpdf/fonts"
	fontLoader       FontLoader                 // used to load font files from arbitrary locations
	files            files.ReadWriter           // where fonts, images and the output live; pass one to New
	fonts            map[string]fontDefType     // array of used fonts
	fontFiles        map[string]fontFileType    // array of font files
	diffs            []string                   // array of encoding differences
	fontFamily       string                     // current font family
	fontStyle        string                     // current font style
	underline        bool                       // underlining flag
	strikeout        bool                       // strike out flag
	currentFont      fontDefType                // current font info
	fontSizePt       float64                    // current font size in points
	fontSize         float64                    // current font size in user unit
	ws               float64                    // word spacing
	images           map[string]*ImageInfoType  // array of used images
	aliasMap         map[string]string          // map of alias->replacement
	pageLinks        [][]linkType               // pageLinks[page][link], both 1-based
	links            []intLinkType              // array of internal links
	attachments      []Attachment               // slice of content to embed globally
	pageAttachments  [][]annotationAttach       // 1-based array of annotation for file attachments (per page)
	outlines         []outlineType              // array of outlines
	outlineRoot      int                        // root of outlines
	autoPageBreak    bool                       // automatic page breaking
	acceptPageBreak  func() bool                // returns true to accept page break
	pageBreakTrigger float64                    // threshold used to trigger page breaks
	inHeader         bool                       // flag set when processing header
	headerFnc        func()                     // function provided by app and called to write header
	headerHomeMode   bool                       // set position to home after headerFnc is called
	inFooter         bool                       // flag set when processing footer
	footerFnc        func()                     // function provided by app and called to write footer
	footerFncLpi     func(bool)                 // function provided by app and called to write footer with last page flag
	zoomMode         string                     // zoom display mode
	layoutMode       string                     // layout display mode
	nXMP             int                        // XMP object number
	xmp              []byte                     // XMP metadata
	producer         string                     // producer
	title            string                     // title
	subject          string                     // subject
	author           string                     // author
	lang             string                     // lang
	keywords         string                     // keywords
	creator          string                     // creator
	creationDate     pdfTime                    // override for document CreationDate value
	modDate          pdfTime                    // override for document ModDate value
	aliasNbPagesStr  string                     // alias for total number of pages
	pdfVersion       pdfVersion                 // PDF version number
	capStyle         int                        // line cap style: butt 0, round 1, square 2
	joinStyle        int                        // line segment join style: miter 0, round 1, bevel 2
	dashArray        []float64                  // dash array
	dashPhase        float64                    // dash phase
	blendList        []blendModeType            // slice[idx] of alpha transparency modes, 1-based
	blendMap         map[string]int             // map into blendList
	blendMode        string                     // current blend mode
	alpha            float64                    // current transpacency
	gradientList     []gradientType             // slice[idx] of gradient records
	clipNest         int                        // Number of active clipping contexts
	transformNest    int                        // Number of active transformation contexts
	err              error                      // Set if error occurs during life cycle of instance
	protect          protectType                // document protection structure
	layer            layerRecType               // manages optional layers in document
	catalogSort      bool                       // sort resource catalogs in document
	nJs              int                        // JavaScript object number
	javascript       *string                    // JavaScript code to include in the PDF
	colorFlag        bool                       // indicates whether fill and text colors are different
	color            struct {
		// Composite values of colors
		draw, fill, text colorType
	}
	spotColorMap           map[string]spotColorType // Map of named ink-based colors
	outputIntents          []OutputIntentType       // OutputIntents
	outputIntentStartN     int                      // Start object number for
	userUnderlineThickness float64                  // A custom user underline thickness multiplier.

	fmt struct {
		buf []byte       // buffer used to format numbers.
		col bytes.Buffer // buffer used to build color strings.
	}
}

const (
	pdfVers1_3 = pdfVersion(uint16(1)<<8 | uint16(3))
	pdfVers1_4 = pdfVersion(uint16(1)<<8 | uint16(4))
	pdfVers1_5 = pdfVersion(uint16(1)<<8 | uint16(5))
)

type pdfVersion uint16

func pdfVersionFrom(maj, min uint) pdfVersion {
	if min > 255 {
		panic(fmt.Err("format", "invalid", maj, min))
	}
	return pdfVersion(uint16(maj)<<8 | uint16(min))
}

func (v pdfVersion) String() string {
	maj := int64(byte(v >> 8))
	min := int64(byte(v))
	return fmt.Sprintf("%d.%d", maj, min)
}

type encType struct {
	uv   int
	name string
}

type encListType [256]encType

var (
	fontBoxDef = model.Definition{
		Name: "font_box",
		Fields: model.Fields{
			{Name: "Xmin", Type: model.Int()},
			{Name: "Ymin", Type: model.Int()},
			{Name: "Xmax", Type: model.Int()},
			{Name: "Ymax", Type: model.Int()},
		},
	}

	fontDescDef = model.Definition{
		Name: "font_desc",
		Fields: model.Fields{
			{Name: "Ascent", Type: model.Int()},
			{Name: "Descent", Type: model.Int()},
			{Name: "CapHeight", Type: model.Int()},
			{Name: "Flags", Type: model.Int()},
			{Name: "FontBBox", Type: model.Struct(&fontBoxDef)},
			{Name: "ItalicAngle", Type: model.Float()},
			{Name: "StemV", Type: model.Int()},
			{Name: "MissingWidth", Type: model.Int()},
		},
	}
)

type fontBoxType struct {
	Xmin, Ymin, Xmax, Ymax int
}

func (f *fontBoxType) Schema() []model.Field {
	return []model.Field{
		{Name: "Xmin", Type: model.Int()},
		{Name: "Ymin", Type: model.Int()},
		{Name: "Xmax", Type: model.Int()},
		{Name: "Ymax", Type: model.Int()},
	}
}

func (f *fontBoxType) Pointers() []any {
	return []any{&f.Xmin, &f.Ymin, &f.Xmax, &f.Ymax}
}

func (f *fontBoxType) IsNil() bool { return f == nil }

func (f *fontBoxType) DecodeFields(r model.FieldReader) {
	if v, ok := r.Int("Xmin"); ok {
		f.Xmin = int(v)
	}
	if v, ok := r.Int("Ymin"); ok {
		f.Ymin = int(v)
	}
	if v, ok := r.Int("Xmax"); ok {
		f.Xmax = int(v)
	}
	if v, ok := r.Int("Ymax"); ok {
		f.Ymax = int(v)
	}
}

// Font flags for FontDescType.Flags as defined in the pdf specification.
const (
	// FontFlagFixedPitch is set if all glyphs have the same width (as
	// opposed to proportional or variable-pitch fonts, which have
	// different widths).
	FontFlagFixedPitch = 1 << 0
	// FontFlagSerif is set if glyphs have serifs, which are short
	// strokes drawn at an angle on the top and bottom of glyph stems.
	// (Sans serif fonts do not have serifs.)
	FontFlagSerif = 1 << 1
	// FontFlagSymbolic is set if font contains glyphs outside the
	// Adobe standard Latin character set. This flag and the
	// Nonsymbolic flag shall not both be set or both be clear.
	FontFlagSymbolic = 1 << 2
	// FontFlagScript is set if glyphs resemble cursive handwriting.
	FontFlagScript = 1 << 3
	// FontFlagNonsymbolic is set if font uses the Adobe standard
	// Latin character set or a subset of it.
	FontFlagNonsymbolic = 1 << 5
	// FontFlagItalic is set if glyphs have dominant vertical strokes
	// that are slanted.
	FontFlagItalic = 1 << 6
	// FontFlagAllCap is set if font contains no lowercase letters;
	// typically used for display purposes, such as for titles or
	// headlines.
	FontFlagAllCap = 1 << 16
	// SmallCap is set if font contains both uppercase and lowercase
	// letters. The uppercase letters are similar to those in the
	// regular version of the same typeface family. The glyphs for the
	// lowercase letters have the same shapes as the corresponding
	// uppercase letters, but they are sized and their proportions
	// adjusted so that they have the same size and stroke weight as
	// lowercase glyphs in the same typeface family.
	SmallCap = 1 << 18
	// ForceBold determines whether bold glyphs shall be painted with
	// extra pixels even at very small text sizes by a conforming
	// reader. If the ForceBold flag is set, features of bold glyphs
	// may be thickened at small text sizes.
	ForceBold = 1 << 18
)

// FontDescType (font descriptor) specifies metrics and other
// attributes of a font, as distinct from the metrics of individual
// glyphs (as defined in the pdf specification).
type FontDescType struct {
	// The maximum height above the baseline reached by glyphs in this
	// font (for example for "S"). The height of glyphs for accented
	// characters shall be excluded.
	Ascent int
	// The maximum depth below the baseline reached by glyphs in this
	// font. The value shall be a negative number.
	Descent int
	// The vertical coordinate of the top of flat capital letters,
	// measured from the baseline (for example "H").
	CapHeight int
	// A collection of flags defining various characteristics of the
	// font. (See the FontFlag* constants.)
	Flags int
	// A rectangle, expressed in the glyph coordinate system, that
	// shall specify the font bounding box. This should be the smallest
	// rectangle enclosing the shape that would result if all of the
	// glyphs of the font were placed with their origins coincident
	// and then filled.
	FontBBox fontBoxType
	// The angle, expressed in degrees counterclockwise from the
	// vertical, of the dominant vertical strokes of the font. (The
	// 9-o’clock position is 90 degrees, and the 3-o’clock position
	// is –90 degrees.) The value shall be negative for fonts that
	// slope to the right, as almost all italic fonts do.
	ItalicAngle int
	// The thickness, measured horizontally, of the dominant vertical
	// stems of glyphs in the font.
	StemV int
	// The width to use for character codes whose widths are not
	// specified in a font dictionary’s Widths array. This shall have
	// a predictable effect only if all such codes map to glyphs whose
	// actual widths are the same as the value of the MissingWidth
	// entry. (Default value: 0.)
	MissingWidth int
}

func (f *FontDescType) Schema() []model.Field {
	return []model.Field{
		{Name: "Ascent", Type: model.Int()},
		{Name: "Descent", Type: model.Int()},
		{Name: "CapHeight", Type: model.Int()},
		{Name: "Flags", Type: model.Int()},
		{Name: "FontBBox", Type: model.Struct(&fontBoxDef)},
		{Name: "ItalicAngle", Type: model.Float()},
		{Name: "StemV", Type: model.Int()},
		{Name: "MissingWidth", Type: model.Int()},
	}
}

func (f *FontDescType) Pointers() []any {
	return []any{&f.Ascent, &f.Descent, &f.CapHeight, &f.Flags, &f.FontBBox, &f.ItalicAngle, &f.StemV, &f.MissingWidth}
}

func (f *FontDescType) IsNil() bool { return f == nil }

func (f *FontDescType) DecodeFields(r model.FieldReader) {
	if v, ok := r.Int("Ascent"); ok {
		f.Ascent = int(v)
	}
	if v, ok := r.Int("Descent"); ok {
		f.Descent = int(v)
	}
	if v, ok := r.Int("CapHeight"); ok {
		f.CapHeight = int(v)
	}
	if v, ok := r.Int("Flags"); ok {
		f.Flags = int(v)
	}
	r.Object("FontBBox", &f.FontBBox)
	if v, ok := r.Int("ItalicAngle"); ok {
		f.ItalicAngle = int(v)
	}
	if v, ok := r.Int("StemV"); ok {
		f.StemV = int(v)
	}
	if v, ok := r.Int("MissingWidth"); ok {
		f.MissingWidth = int(v)
	}
}

type fontDefType struct {
	Tp           string        // "Core", "TrueType", ...
	Name         string        // "Courier-Bold", ...
	Desc         FontDescType  // Font descriptor
	Up           int           // Underline position
	Ut           int           // Underline thickness
	Cw           []int         // Character width by ordinal
	Enc          string        // "cp1252", ...
	Diff         string        // Differences from reference encoding
	File         string        // "Redressed.z"
	Size1, Size2 int           // Type1 values
	OriginalSize int           // Size of uncompressed font file
	N            int           // Set by font loader
	DiffN        int           // Position of diff in app array, set by font loader
	i            string        // 1-based position in font list, set by font loader, not this program
	utf8File     *utf8FontFile // UTF-8 font
	usedRunes    map[int]int   // Array of used runes
}

func (f *fontDefType) Schema() []model.Field {
	return []model.Field{
		{Name: "Tp", Type: model.Text()},
		{Name: "Name", Type: model.Text()},
		{Name: "Desc", Type: model.Struct(&fontDescDef)},
		{Name: "Up", Type: model.Int()},
		{Name: "Ut", Type: model.Int()},
		{Name: "Cw", Type: model.IntSlice()},
		{Name: "Enc", Type: model.Text()},
		{Name: "Diff", Type: model.Text()},
		{Name: "File", Type: model.Text()},
		{Name: "Size1", Type: model.Int()},
		{Name: "Size2", Type: model.Int()},
		{Name: "OriginalSize", Type: model.Int()},
		{Name: "N", Type: model.Int()},
		{Name: "DiffN", Type: model.Int()},
		{Name: "i", Type: model.Text()},
	}
}

func (f *fontDefType) Pointers() []any {
	return []any{&f.Tp, &f.Name, &f.Desc, &f.Up, &f.Ut, &f.Cw, &f.Enc, &f.Diff, &f.File, &f.Size1, &f.Size2, &f.OriginalSize, &f.N, &f.DiffN, &f.i}
}

func (f *fontDefType) IsNil() bool { return f == nil }

func (f *fontDefType) DecodeFields(r model.FieldReader) {
	if v, ok := r.String("Tp"); ok {
		f.Tp = v
	}
	if v, ok := r.String("Name"); ok {
		f.Name = v
	}
	r.Object("Desc", &f.Desc)
	if v, ok := r.Int("Up"); ok {
		f.Up = int(v)
	}
	if v, ok := r.Int("Ut"); ok {
		f.Ut = int(v)
	}
	if ar, ok := r.Array("Cw"); ok {
		n := ar.Len()
		f.Cw = make([]int, n)
		for i := 0; i < n; i++ {
			f.Cw[i] = int(ar.Int(i))
		}
	}
	if v, ok := r.String("Enc"); ok {
		f.Enc = v
	}
	if v, ok := r.String("Diff"); ok {
		f.Diff = v
	}
	if v, ok := r.String("File"); ok {
		f.File = v
	}
	if v, ok := r.Int("Size1"); ok {
		f.Size1 = int(v)
	}
	if v, ok := r.Int("Size2"); ok {
		f.Size2 = int(v)
	}
	if v, ok := r.Int("OriginalSize"); ok {
		f.OriginalSize = int(v)
	}
	if v, ok := r.Int("N"); ok {
		f.N = int(v)
	}
	if v, ok := r.Int("DiffN"); ok {
		f.DiffN = int(v)
	}
	if v, ok := r.String("i"); ok {
		f.i = v
	}
}

type fontInfoType struct {
	Data               []byte
	File               string
	OriginalSize       int
	FontName           string
	Bold               bool
	IsFixedPitch       bool
	UnderlineThickness int
	UnderlinePosition  int
	Widths             []int
	Size1, Size2       uint32
	Desc               FontDescType
}
