package fpdf

import (
	"bytes"
	"io"
	"sort"

	. "webtyp.com/fmt"
)

// PageSize returns the width and height of the specified page in the units
// established in New(). These return values are followed by the unit of
// measure itself. If pageNum is zero or otherwise out of bounds, it returns
// the default page size, that is, the size of the page that would be added by
// AddPage().
func (f *Fpdf) PageSize(pageNum int) (wd, ht float64, unitStr string) {
	sz, ok := f.pageSizes[pageNum]
	if ok {
		// Convert from points back to user units
		return sz.Wd / f.k, sz.Ht / f.k, string(f.unitType)
	} else {
		// Return default page size, converting from points to user units
		return f.defPageSize.Wd / f.k, f.defPageSize.Ht / f.k, string(f.unitType)
	}
}

// AddPageFormat adds a new page with non-default orientation or size. See
// AddPage() for more details.
//
// See New() for a description of orientationStr.
//
// size specifies the size of the new page in the units established in New().
//
// The PageSize() example demonstrates this method.
func (f *Fpdf) AddPageFormat(orientationStr orientationType, size PageSize) {
	if f.err != nil {
		return
	}
	if f.page != len(f.pages)-1 {
		f.page = len(f.pages) - 1
	}
	if f.state == 0 {
		f.open()
	}
	familyStr := f.fontFamily
	style := f.fontStyle
	if f.underline {
		style += "U"
	}
	if f.strikeout {
		style += "S"
	}
	fontsize := f.fontSizePt
	lw := f.lineWidth
	dc := f.color.draw
	fc := f.color.fill
	tc := f.color.text
	cf := f.colorFlag

	if f.page > 0 {
		f.inFooter = true
		// Page footer avoid double call on footer.
		if f.footerFnc != nil {
			f.footerFnc()

		} else if f.footerFncLpi != nil {
			f.footerFncLpi(false) // not last page.
		}
		f.inFooter = false
		// Close page
		f.endpage()
	}
	// Start new page
	f.beginpage(orientationStr, size)
	// 	Set line cap style to current value
	// f.out("2 J")
	f.outf("%d J", f.capStyle)
	// 	Set line join style to current value
	f.outf("%d j", f.joinStyle)
	// Set line width
	f.lineWidth = lw
	f.outf("%.2f w", lw*f.k)
	// Set dash pattern
	if len(f.dashArray) > 0 {
		f.outputDashPattern()
	}
	// 	Set font
	if familyStr != "" {
		f.SetFont(familyStr, style, fontsize)
		if f.err != nil {
			return
		}
	}
	// 	Set colors
	f.color.draw = dc
	if dc.str != "0 G" {
		f.out(dc.str)
	}
	f.color.fill = fc
	if fc.str != "0 g" {
		f.out(fc.str)
	}
	f.color.text = tc
	f.colorFlag = cf
	// 	Page header
	if f.headerFnc != nil {
		f.inHeader = true
		f.headerFnc()
		f.inHeader = false
		if f.headerHomeMode {
			f.SetHomeXY()
		}
	}
	// 	Restore line width
	if f.lineWidth != lw {
		f.lineWidth = lw
		f.outf("%.2f w", lw*f.k)
	}
	// Restore font
	if familyStr != "" {
		f.SetFont(familyStr, style, fontsize)
		if f.err != nil {
			return
		}
	}
	// Restore colors
	if f.color.draw.str != dc.str {
		f.color.draw = dc
		f.out(dc.str)
	}
	if f.color.fill.str != fc.str {
		f.color.fill = fc
		f.out(fc.str)
	}
	f.color.text = tc
	f.colorFlag = cf
}

// AddPage adds a new page to the document. If a page is already present, the
// Footer() method is called first to output the footer. Then the page is
// added, the current position set to the top-left corner according to the left
// and top margins, and Header() is called to display the header.
//
// The font which was set before calling is automatically restored. There is no
// need to call SetFont() again if you want to continue with the same font. The
// same is true for colors and line width.
//
// The origin of the coordinate system is at the top-left corner and increasing
// ordinates go downwards.
//
// See AddPageFormat() for a version of this method that allows the page size
// and orientation to be different than the default.
func (f *Fpdf) AddPage() {
	if f.err != nil {
		return
	}
	// dbg("AddPage")
	f.AddPageFormat(f.defOrientation, f.defPageSize)
}

// PageNo returns the current page number.
//
// See the example for AddPage() for a demonstration of this method.
func (f *Fpdf) PageNo() int {
	return f.page
}

// GetPageSize returns the current page's width and height. This is the paper's
// size. To compute the size of the area being used, subtract the margins (see
// GetMargins()).
func (f *Fpdf) GetPageSize() (width, height float64) {
	width = f.w
	height = f.h
	return
}

// SetPageBoxRec sets the page box for the current page, and any following
// pages. Allowable types are trim, trimbox, crop, cropbox, bleed, bleedbox,
// art and artbox box types are case insensitive. See SetPageBox() for a method
// that specifies the coordinates and extent of the page box individually.
func (f *Fpdf) SetPageBoxRec(t string, pb PageBox) {
	switch Convert(t).ToLower().String() {
	case "trim":
		fallthrough
	case "trimbox":
		t = "TrimBox"
	case "crop":
		fallthrough
	case "cropbox":
		t = "CropBox"
	case "bleed":
		fallthrough
	case "bleedbox":
		t = "BleedBox"
	case "art":
		fallthrough
	case "artbox":
		t = "ArtBox"
	default:
		f.err = Errf("%s is not a valid page box type", t)
		return
	}

	pb.X = pb.X * f.k
	pb.Y = pb.Y * f.k
	pb.Wd = (pb.Wd * f.k) + pb.X
	pb.Ht = (pb.Ht * f.k) + pb.Y

	if f.page > 0 {
		f.pageBoxes[f.page][t] = pb
	}

	// always override. page defaults are supplied in addPage function
	f.defPageBoxes[t] = pb
}

// SetPageBox sets the page box for the current page, and any following pages.
// Allowable types are trim, trimbox, crop, cropbox, bleed, bleedbox, art and
// artbox box types are case insensitive.
func (f *Fpdf) SetPageBox(t string, x, y, wd, ht float64) {
	f.SetPageBoxRec(t, PageBox{SizeType{Wd: wd, Ht: ht}, PointType{X: x, Y: y}})
}

// SetPage sets the current page to that of a valid page in the PDF document.
// pageNum is one-based. The SetPage() example demonstrates this method.
func (f *Fpdf) SetPage(pageNum int) {
	if (pageNum > 0) && (pageNum < len(f.pages)) {
		f.page = pageNum
	}
}

// PageCount returns the number of pages currently in the document. Since page
// numbers in gofpdf are one-based, the page count is the same as the page
// number of the current last page.
func (f *Fpdf) PageCount() int {
	return len(f.pages) - 1
}

// GetMargins returns the left, top, right, and bottom margins. The first three
// are set with the SetMargins() method. The bottom margin is set with the
// SetAutoPageBreak() method.
func (f *Fpdf) GetMargins() (left, top, right, bottom float64) {
	left = f.lMargin
	top = f.tMargin
	right = f.rMargin
	bottom = f.bMargin
	return
}

// SetAcceptPageBreakFunc allows the application to control where page breaks
// occur.
//
// fnc is an application function (typically a closure) that is called by the
// library whenever a page break condition is met. The break is issued if true
// is returned. The default implementation returns a value according to the
// mode selected by SetAutoPageBreak. The function provided should not be
// called by the application.
//
// See the example for SetLeftMargin() to see how this function can be used to
// manage multiple columns.
func (f *Fpdf) SetAcceptPageBreakFunc(fnc func() bool) {
	f.acceptPageBreak = fnc
}

// SetHeaderFuncMode sets the function that lets the application render the
// page header. See SetHeaderFunc() for more details. The value for homeMode
// should be set to true to have the current position set to the left and top
// margin after the header function is called.
func (f *Fpdf) SetHeaderFuncMode(fnc func(), homeMode bool) {
	f.headerFnc = fnc
	f.headerHomeMode = homeMode
}

// SetHeaderFunc sets the function that lets the application render the page
// header. The specified function is automatically called by AddPage() and
// should not be called directly by the application. The implementation in Fpdf
// is empty, so you have to provide an appropriate function if you want page
// headers. fnc will typically be a closure that has access to the Fpdf
// instance and other document generation variables.
//
// A header is a convenient place to put background content that repeats on
// each page such as a watermark. When this is done, remember to reset the X
// and Y values so the normal content begins where expected. Including a
// watermark on each page is demonstrated in the example for TransformRotate.
//
// This method is demonstrated in the example for AddPage().
func (f *Fpdf) SetHeaderFunc(fnc func()) {
	f.headerFnc = fnc
}

// SetFooterFunc sets the function that lets the application render the page
// footer. The specified function is automatically called by AddPage() and
// Close() and should not be called directly by the application. The
// implementation in Fpdf is empty, so you have to provide an appropriate
// function if you want page footers. fnc will typically be a closure that has
// access to the Fpdf instance and other document generation variables. See
// SetFooterFuncLpi for a similar function that passes a last page indicator.
//
// This method is demonstrated in the example for AddPage().
func (f *Fpdf) SetFooterFunc(fnc func()) {
	f.footerFnc = fnc
	f.footerFncLpi = nil
}

// SetFooterFuncLpi sets the function that lets the application render the page
// footer. The specified function is automatically called by AddPage() and
// Close() and should not be called directly by the application. It is passed a
// boolean that is true if the last page of the document is being rendered. The
// implementation in Fpdf is empty, so you have to provide an appropriate
// function if you want page footers. fnc will typically be a closure that has
// access to the Fpdf instance and other document generation variables.
func (f *Fpdf) SetFooterFuncLpi(fnc func(lastPage bool)) {
	f.footerFncLpi = fnc
	f.footerFnc = nil
}

// SetTopMargin defines the top margin. The method can be called before
// creating the first page.
func (f *Fpdf) SetTopMargin(margin float64) {
	f.tMargin = margin
}

// SetMargins defines the left, top and right margins. By default, they equal 1
// cm. Call this method to change them. If the value of the right margin is
// less than zero, it is set to the same as the left margin.
func (f *Fpdf) SetMargins(left, top, right float64) {
	f.lMargin = left
	f.tMargin = top
	if right < 0 {
		right = left
	}
	f.rMargin = right
}

// SetLeftMargin defines the left margin. The method can be called before
// creating the first page. If the current abscissa gets out of page, it is
// brought back to the margin.
func (f *Fpdf) SetLeftMargin(margin float64) {
	f.lMargin = margin
	if f.page > 0 && f.x < margin {
		f.x = margin
	}
}

// SetRightMargin defines the right margin. The method can be called before
// creating the first page.
func (f *Fpdf) SetRightMargin(margin float64) {
	f.rMargin = margin
}

// GetAutoPageBreak returns true if automatic pages breaks are enabled, false
// otherwise. This is followed by the triggering limit from the bottom of the
// page. This value applies only if automatic page breaks are enabled.
func (f *Fpdf) GetAutoPageBreak() (auto bool, margin float64) {
	auto = f.autoPageBreak
	margin = f.bMargin
	return
}

// SetAutoPageBreak enables or disables the automatic page breaking mode. When
// enabling, the second parameter is the distance from the bottom of the page
// that defines the triggering limit. By default, the mode is on and the margin
// is 2 cm.
func (f *Fpdf) SetAutoPageBreak(auto bool, margin float64) {
	f.autoPageBreak = auto
	f.bMargin = margin
	f.pageBreakTrigger = f.h - margin
}

// SetProtection applies certain constraints on the finished PDF document.
//
// actionFlag is a bitflag that controls various document operations.
// CnProtectPrint allows the document to be printed. CnProtectModify allows a
// document to be modified by a PDF editor. CnProtectCopy allows text and
// images to be copied into the system clipboard. CnProtectAnnotForms allows
// annotations and forms to be added by a PDF editor. These values can be
// combined by or-ing them together, for example,
// CnProtectCopy|CnProtectModify. This flag is advisory; not all PDF readers
// implement the constraints that this argument attempts to control.
//
// userPassStr specifies the password that will need to be provided to view the
// contents of the PDF. The permissions specified by actionFlag will apply.
//
// ownerPassStr specifies the password that will need to be provided to gain
// full access to the document regardless of the actionFlag value. An empty
// string for this argument will be replaced with a random value, effectively
// prohibiting full access to the document.
func (f *Fpdf) SetProtection(actionFlag byte, userPassStr, ownerPassStr string) {
	if f.err != nil {
		return
	}
	f.protect.setProtection(actionFlag, userPassStr, ownerPassStr)
}

// OutputAndClose sends the PDF document to the writer specified by w. This
// method will close both f and w, even if an error is detected and no document
// is produced.
func (f *Fpdf) OutputAndClose(w io.WriteCloser) error {
	_ = f.Output(w)
	err := w.Close()
	if err != nil {
		return Errf("could not close writer: %v", err)
	}
	return f.err
}

// OutputFileAndClose creates or truncates the file specified by fileStr and
// writes the PDF document to it. This method will close f and the newly
// written file, even if an error is detected and no document is produced.
//
// Most examples demonstrate the use of this method.
func (f *Fpdf) OutputFileAndClose(fileStr string) error {
	if f.err != nil {
		return f.err
	}

	// Generate PDF content to a buffer
	var buf bytes.Buffer
	_ = f.Output(&buf)

	if f.err != nil {
		return f.err
	}

	err := f.files.WriteFile(fileStr, buf.Bytes())
	if err != nil {
		f.err = err
		return f.err
	}

	return f.err
}

// Output sends the PDF document to the writer specified by w. No output will
// take place if an error has occurred in the document generation process. w
// remains open after this function returns. After returning, f is in a closed
// state and its methods should not be called.
func (f *Fpdf) Output(w io.Writer) error {
	if f.err != nil {
		return f.err
	}
	// dbg("Output")
	if f.state < 3 {
		f.Close()
	}
	_, err := f.buffer.WriteTo(w)
	if err != nil {
		f.err = err
	}
	return f.err
}

func (f *Fpdf) getpagesizestr(sizeStr string) (size PageSize) {
	if f.err != nil {
		return
	}
	sizeStr = Convert(sizeStr).ToLower().String()
	// dbg("Size [%s]", sizeStr)
	var ok bool
	size, ok = f.stdPageSizes[sizeStr]
	if ok {
		// dbg("found %s", sizeStr)
		size.Wd /= f.k
		size.Ht /= f.k

	} else {
		f.err = Errf("unknown page size %s", sizeStr)
	}
	return
}

// GetPageSizeStr returns the PageSize for the given sizeStr (that is A4, A3, etc..)
func (f *Fpdf) GetPageSizeStr(sizeStr string) (size PageSize) {
	return f.getpagesizestr(sizeStr)
}

func (f *Fpdf) beginpage(newPageOrientation orientationType, size PageSize) {
	if f.err != nil {
		return
	}
	f.page++
	// add the default page boxes, if any exist, to the page
	f.pageBoxes[f.page] = make(map[string]PageBox)
	for box, pb := range f.defPageBoxes {
		f.pageBoxes[f.page][box] = pb
	}
	f.pages = append(f.pages, bytes.NewBufferString(""))
	f.pageLinks = append(f.pageLinks, make([]linkType, 0))
	f.pageAttachments = append(f.pageAttachments, []annotationAttach{})
	f.state = 2
	f.x = f.lMargin
	f.y = f.tMargin
	f.fontFamily = ""
	if newPageOrientation != f.curOrientation || size.Wd != f.curPageSize.Wd || size.Ht != f.curPageSize.Ht {
		// New size or orientation
		// size is in points, convert to user units for f.w and f.h
		if newPageOrientation == Portrait {
			f.w = size.Wd / f.k
			f.h = size.Ht / f.k
		} else {
			f.w = size.Ht / f.k
			f.h = size.Wd / f.k
		}
		f.wPt = f.w * f.k
		f.hPt = f.h * f.k
		f.pageBreakTrigger = f.h - f.bMargin
		f.curOrientation = newPageOrientation
		f.curPageSize = size
	}
	if newPageOrientation != f.defOrientation || size.Wd != f.defPageSize.Wd || size.Ht != f.defPageSize.Ht {
		// Store the actual page dimensions (after orientation is applied) in points
		// size is already in points, so no conversion needed
		if newPageOrientation == Portrait {
			f.pageSizes[f.page] = PageSize{Wd: size.Wd, Ht: size.Ht, AutoHt: size.AutoHt}
		} else {
			// For landscape, swap dimensions
			f.pageSizes[f.page] = PageSize{Wd: size.Ht, Ht: size.Wd, AutoHt: size.AutoHt}
		}
	}
}

func (f *Fpdf) endpage() {
	f.EndLayer()
	f.state = 1
}

func implode(sep string, arr []int) string {
	var s fmtBuffer
	for i := 0; i < len(arr)-1; i++ {
		s.printf("%v", arr[i])
		s.printf("%s", sep)
	}
	if len(arr) > 0 {
		s.printf("%v", arr[len(arr)-1])
	}
	return s.String()
}

// arrayCountValues counts the occurrences of each item in the $mp array.
func arrayCountValues(mp []int) map[int]int {
	answer := make(map[int]int)
	for _, v := range mp {
		answer[v] = answer[v] + 1
	}
	return answer
}

func (f *Fpdf) putpages() {
	var wPt, hPt float64
	var pageSize PageSize
	var ok bool
	nb := f.page
	if len(f.aliasNbPagesStr) > 0 {
		// Replace number of pages
		f.RegisterAlias(f.aliasNbPagesStr, sprintf("%d", nb))
	}
	f.replaceAliases()
	// f.defPageSize is already in points, no need to multiply by f.k
	if f.defOrientation == Portrait {
		wPt = f.defPageSize.Wd
		hPt = f.defPageSize.Ht
	} else {
		wPt = f.defPageSize.Ht
		hPt = f.defPageSize.Wd
	}
	pagesObjectNumbers := make([]int, nb+1) // 1-based
	for n := 1; n <= nb; n++ {
		// Page
		f.newobj()
		pagesObjectNumbers[n] = f.n // save for /Kids
		f.out("<</Type /Page")
		f.out("/Parent 1 0 R")
		pageSize, ok = f.pageSizes[n]
		if ok {
			f.outf("/MediaBox [0 0 %.2f %.2f]", pageSize.Wd, pageSize.Ht)
		}
		for t, pb := range f.pageBoxes[n] {
			f.outf("/%s [%.2f %.2f %.2f %.2f]", t, pb.X, pb.Y, pb.Wd, pb.Ht)
		}
		f.out("/Resources 2 0 R")
		// Links
		if len(f.pageLinks[n])+len(f.pageAttachments[n]) > 0 {
			var annots fmtBuffer
			annots.printf("/Annots [")
			for _, pl := range f.pageLinks[n] {
				annots.printf("<</Type /Annot /Subtype /Link /Rect [%.2f %.2f %.2f %.2f] /Border [0 0 0] ",
					pl.x, pl.y, pl.x+pl.wd, pl.y-pl.ht)
				if pl.link == 0 {
					annots.printf("/A <</S /URI /URI %s>>>>", f.textstring(pl.linkStr))
				} else {
					l := f.links[pl.link]
					var sz PageSize
					var h float64
					sz, ok = f.pageSizes[l.page]
					if ok {
						h = sz.Ht
					} else {
						h = hPt
					}
					// dbg("h [%.2f], l.y [%.2f] f.k [%.2f]\n", h, l.y, f.k)
					annots.printf("/Dest [%d 0 R /XYZ 0 %.2f null]>>", 1+2*l.page, h-l.y*f.k)
				}
			}
			f.putAttachmentAnnotationLinks(&annots, n)
			annots.printf("]")
			f.out(annots.String())
		}
		if f.pdfVersion > pdfVers1_3 {
			f.out("/Group <</Type /Group /S /Transparency /CS /DeviceRGB>>")
		}
		f.outf("/Contents %d 0 R>>", f.n+1)
		f.out("endobj")
		// Page content
		f.newobj()
		if f.compress {
			mem := xmem.compress(f.pages[n].Bytes())
			data := mem.bytes()
			f.outf("<</Filter /FlateDecode /Length %d>>", len(data))
			f.putstream(data)
			mem.release()
		} else {
			f.outf("<</Length %d>>", f.pages[n].Len())
			f.putstream(f.pages[n].Bytes())
		}
		f.out("endobj")
	}
	// Pages root
	f.offsets[1] = f.buffer.Len()
	f.out("1 0 obj")
	f.out("<</Type /Pages")
	var kids fmtBuffer
	kids.printf("/Kids [")
	for i := 1; i <= nb; i++ {
		kids.printf("%d 0 R ", pagesObjectNumbers[i])
	}
	kids.printf("]")
	f.out(kids.String())
	f.outf("/Count %d", nb)
	f.outf("/MediaBox [0 0 %.2f %.2f]", wPt, hPt)
	f.out(">>")
	f.out("endobj")
}

func (f *Fpdf) putimages() {
	var keyList []string
	var key string
	for key = range f.images {
		keyList = append(keyList, key)
	}

	// Sort the keyList []string by the corresponding image's width.
	if f.catalogSort {
		sort.SliceStable(keyList, func(i, j int) bool { return f.images[keyList[i]].w < f.images[keyList[j]].w })
	}

	// Maintain a list of inserted image SHA-1 hashes, with their
	// corresponding object ID number.
	insertedImages := map[string]int{}

	for _, key = range keyList {
		image := f.images[key]

		// Check if this image has already been inserted using it's SHA-1 hash.
		insertedImageObjN, isFound := insertedImages[image.i]

		// If found, skip inserting the image as a new object, and
		// use the object ID from the insertedImages map.
		// If not, insert the image into the PDF and store the object ID.
		if isFound {
			image.n = insertedImageObjN
		} else {
			f.putimage(image)
			insertedImages[image.i] = image.n
		}
	}
}

func (f *Fpdf) putimage(info *ImageInfoType) {
	f.newobj()
	info.n = f.n
	f.out("<</Type /XObject")
	f.out("/Subtype /Image")
	f.outf("/Width %d", int(info.w))
	f.outf("/Height %d", int(info.h))
	if info.cs == "Indexed" {
		f.outf("/ColorSpace [/Indexed /DeviceRGB %d %d 0 R]", len(info.pal)/3-1, f.n+1)
	} else {
		f.outf("/ColorSpace /%s", info.cs)
		if info.cs == "DeviceCMYK" {
			f.out("/Decode [1 0 1 0 1 0 1 0]")
		}
	}
	f.outf("/BitsPerComponent %d", info.bpc)
	if len(info.f) > 0 {
		f.outf("/Filter /%s", info.f)
	}
	if len(info.dp) > 0 {
		f.outf("/DecodeParms <<%s>>", info.dp)
	}
	if len(info.trns) > 0 {
		var trns fmtBuffer
		for _, v := range info.trns {
			trns.printf("%d %d ", v, v)
		}
		f.outf("/Mask [%s]", trns.String())
	}
	if info.smask != nil {
		f.outf("/SMask %d 0 R", f.n+1)
	}
	f.outf("/Length %d>>", len(info.data))
	f.putstream(info.data)
	f.out("endobj")
	// 	Soft mask
	if len(info.smask) > 0 {
		smask := &ImageInfoType{
			w:     info.w,
			h:     info.h,
			cs:    "DeviceGray",
			bpc:   8,
			f:     info.f,
			dp:    sprintf("/Predictor 15 /Colors 1 /BitsPerComponent 8 /Columns %d", int(info.w)),
			data:  info.smask,
			scale: f.k,
		}
		f.putimage(smask)
	}
	// 	Palette
	if info.cs == "Indexed" {
		f.newobj()
		if f.compress {
			mem := xmem.compress(info.pal)
			pal := mem.bytes()
			f.outf("<</Filter /FlateDecode /Length %d>>", len(pal))
			f.putstream(pal)
			mem.release()
		} else {
			f.outf("<</Length %d>>", len(info.pal))
			f.putstream(info.pal)
		}
		f.out("endobj")
	}
}

func (f *Fpdf) putxobjectdict() {
	{
		var image *ImageInfoType
		var key string
		var keyList []string
		for key = range f.images {
			keyList = append(keyList, key)
		}
		if f.catalogSort {
			sort.SliceStable(keyList, func(i, j int) bool { return f.images[keyList[i]].i < f.images[keyList[j]].i })
		}
		for _, key = range keyList {
			image = f.images[key]
			f.outf("/I%s %d 0 R", image.i, image.n)
		}
	}
}

func (f *Fpdf) putresourcedict() {
	f.out("/ProcSet [/PDF /Text /ImageB /ImageC /ImageI]")
	f.out("/Font <<")
	{
		var keyList []string
		var font fontDefType
		var key string
		for key = range f.fonts {
			keyList = append(keyList, key)
		}
		if f.catalogSort {
			sort.SliceStable(keyList, func(i, j int) bool { return f.fonts[keyList[i]].i < f.fonts[keyList[j]].i })
		}
		for _, key = range keyList {
			font = f.fonts[key]
			f.outf("/F%s %d 0 R", font.i, font.N)
		}
	}
	f.out(">>")
	f.out("/XObject <<")
	f.putxobjectdict()
	f.out(">>")
	count := len(f.blendList)
	if count > 1 {
		f.out("/ExtGState <<")
		for j := 1; j < count; j++ {
			f.outf("/GS%d %d 0 R", j, f.blendList[j].objNum)
		}
		f.out(">>")
	}
	count = len(f.gradientList)
	if count > 1 {
		f.out("/Shading <<")
		for j := 1; j < count; j++ {
			f.outf("/Sh%d %d 0 R", j, f.gradientList[j].objNum)
		}
		f.out(">>")
	}
	// Layers
	f.layerPutResourceDict()
	f.spotColorPutResourceDict()
}

func (f *Fpdf) putBlendModes() {
	count := len(f.blendList)
	for j := 1; j < count; j++ {
		bl := f.blendList[j]
		f.newobj()
		f.blendList[j].objNum = f.n
		f.outf("<</Type /ExtGState /ca %s /CA %s /BM /%s>>",
			bl.fillStr, bl.strokeStr, bl.modeStr)
		f.out("endobj")
	}
}

func (f *Fpdf) putGradients() {
	count := len(f.gradientList)
	for j := 1; j < count; j++ {
		var f1 int
		gr := f.gradientList[j]
		if gr.tp == 2 || gr.tp == 3 {
			f.newobj()
			f.outf("<</FunctionType 2 /Domain [0.0 1.0] /C0 [%s] /C1 [%s] /N 1>>", gr.clr1Str, gr.clr2Str)
			f.out("endobj")
			f1 = f.n
		}
		f.newobj()
		f.outf("<</ShadingType %d /ColorSpace /DeviceRGB", gr.tp)
		if gr.tp == 2 {
			f.outf("/Coords [%.5f %.5f %.5f %.5f] /Function %d 0 R /Extend [true true]>>",
				gr.x1, gr.y1, gr.x2, gr.y2, f1)
		} else if gr.tp == 3 {
			f.outf("/Coords [%.5f %.5f 0 %.5f %.5f %.5f] /Function %d 0 R /Extend [true true]>>",
				gr.x1, gr.y1, gr.x2, gr.y2, gr.r, f1)
		}
		f.out("endobj")
		f.gradientList[j].objNum = f.n
	}
}

func (f *Fpdf) putresources() {
	if f.err != nil {
		return
	}
	f.layerPutLayers()
	f.putBlendModes()
	f.putGradients()
	f.putSpotColors()
	f.putfonts()
	if f.err != nil {
		return
	}
	f.putimages()
	// 	Resource dictionary
	f.offsets[2] = f.buffer.Len()
	f.out("2 0 obj")
	f.out("<<")
	f.putresourcedict()
	f.out(">>")
	f.out("endobj")
	f.putjavascript()
	if f.protect.encrypted {
		f.newobj()
		f.protect.objNum = f.n
		f.out("<<")
		f.out("/Filter /Standard")
		f.out("/V 1")
		f.out("/R 2")
		f.outf("/O (%s)", f.escape(string(f.protect.oValue)))
		f.outf("/U (%s)", f.escape(string(f.protect.uValue)))
		f.outf("/P %d", f.protect.pValue)
		f.out(">>")
		f.out("endobj")
	}
}

func (f *Fpdf) putinfo() {
	if len(f.producer) > 0 {
		f.outf("/Producer %s", f.textstring(f.producer))
	}
	if len(f.title) > 0 {
		f.outf("/Title %s", f.textstring(f.title))
	}
	if len(f.subject) > 0 {
		f.outf("/Subject %s", f.textstring(f.subject))
	}
	if len(f.author) > 0 {
		f.outf("/Author %s", f.textstring(f.author))
	}
	if len(f.keywords) > 0 {
		f.outf("/Keywords %s", f.textstring(f.keywords))
	}
	if len(f.creator) > 0 {
		f.outf("/Creator %s", f.textstring(f.creator))
	}
	f.outf("/CreationDate %s", f.textstring(formatPDFDate(f.creationDate)))
	f.outf("/ModDate %s", f.textstring(formatPDFDate(f.modDate)))
}

func (f *Fpdf) putcatalog() {
	f.out("/Type /Catalog")
	f.out("/Pages 1 0 R")
	f.putOutputIntents()
	if f.lang != "" {
		f.outf("/Lang (%s)", f.lang)
	}
	switch f.zoomMode {
	case "fullpage":
		f.out("/OpenAction [3 0 R /Fit]")
	case "fullwidth":
		f.out("/OpenAction [3 0 R /FitH null]")
	case "real":
		f.out("/OpenAction [3 0 R /XYZ null null 1]")
	}
	// } 	else if !is_string($this->zoomMode))
	// 		$this->out('/OpenAction [3 0 R /XYZ null null '.sprintf('%.2f',$this->zoomMode/100).']');
	switch f.layoutMode {
	case "single", "SinglePage":
		f.out("/PageLayout /SinglePage")
	case "continuous", "OneColumn":
		f.out("/PageLayout /OneColumn")
	case "two", "TwoColumnLeft":
		f.out("/PageLayout /TwoColumnLeft")
	case "TwoColumnRight":
		f.out("/PageLayout /TwoColumnRight")
	case "TwoPageLeft", "TwoPageRight":
		if f.pdfVersion < pdfVers1_5 {
			f.pdfVersion = pdfVers1_5
		}
		f.out("/PageLayout /" + f.layoutMode)
	}
	// Bookmarks
	if len(f.outlines) > 0 {
		f.outf("/Outlines %d 0 R", f.outlineRoot)
		f.out("/PageMode /UseOutlines")
	}
	// Layers
	f.layerPutCatalog()
	// XMP metadata
	if len(f.xmp) != 0 {
		f.outf("/Metadata %d 0 R", f.nXMP)
	}
	// Name dictionary :
	//	-> Javascript
	//	-> Embedded files
	f.out("/Names <<")
	// JavaScript
	if f.javascript != nil {
		f.outf("/JavaScript %d 0 R", f.nJs)
	}
	// Embedded files
	f.outf("/EmbeddedFiles %s", f.getEmbeddedFiles())
	f.out(">>")
}

func (f *Fpdf) putheader() {
	f.outf("%%PDF-%s", f.pdfVersion)
	f.out("%µ¶")
}

func (f *Fpdf) puttrailer() {
	f.outf("/Size %d", f.n+1)
	f.outf("/Root %d 0 R", f.n)
	f.outf("/Info %d 0 R", f.n-1)
	if f.protect.encrypted {
		f.outf("/Encrypt %d 0 R", f.protect.objNum)
		f.out("/ID [()()]")
	}
}

func (f *Fpdf) putxmp() {
	if len(f.xmp) == 0 {
		return
	}
	f.newobj()
	f.nXMP = f.n
	f.outf("<< /Type /Metadata /Subtype /XML /Length %d >>", len(f.xmp))
	f.putstream(f.xmp)
	f.out("endobj")
}

func (f *Fpdf) putbookmarks() {
	nb := len(f.outlines)
	if nb > 0 {
		lru := make(map[int]int)
		level := 0
		for i, o := range f.outlines {
			if o.level > 0 {
				parent := lru[o.level-1]
				f.outlines[i].parent = parent
				f.outlines[parent].last = i
				if o.level > level {
					f.outlines[parent].first = i
				}
			} else {
				f.outlines[i].parent = nb
			}
			if o.level <= level && i > 0 {
				prev := lru[o.level]
				f.outlines[prev].next = i
				f.outlines[i].prev = prev
			}
			lru[o.level] = i
			level = o.level
		}
		n := f.n + 1
		for _, o := range f.outlines {
			f.newobj()
			f.outf("<</Title %s", f.textstring(o.text))
			f.outf("/Parent %d 0 R", n+o.parent)
			if o.prev != -1 {
				f.outf("/Prev %d 0 R", n+o.prev)
			}
			if o.next != -1 {
				f.outf("/Next %d 0 R", n+o.next)
			}
			if o.first != -1 {
				f.outf("/First %d 0 R", n+o.first)
			}
			if o.last != -1 {
				f.outf("/Last %d 0 R", n+o.last)
			}
			f.outf("/Dest [%d 0 R /XYZ 0 %.2f null]", 1+2*o.p, (f.h-o.y)*f.k)
			f.out("/Count 0>>")
			f.out("endobj")
		}
		f.newobj()
		f.outlineRoot = f.n
		f.outf("<</Type /Outlines /First %d 0 R", n)
		f.outf("/Last %d 0 R>>", n+lru[0])
		f.out("endobj")
	}
}

func (f *Fpdf) putOutputIntents() {
	if len(f.outputIntents) <= 0 {
		return
	}

	f.out("/OutputIntents [")
	for index, oi := range f.outputIntents {
		infoSegment := ""
		if oi.Info != "" {
			infoSegment = Sprintf("/Info (%s) ", oi.Info)
		}
		f.outf(
			`<< /Type /OutputIntent /S /%s /OutputConditionIdentifier (%s) %s/DestOutputProfile %d 0 R >>`,
			oi.SubtypeIdent, oi.OutputConditionIdentifier, infoSegment, f.outputIntentStartN+index,
		)
	}
	f.out("]")
}

func (f *Fpdf) putOutputIntentStreams() {
	if len(f.outputIntents) <= 0 {
		return
	}

	f.outputIntentStartN = f.n + 1
	for _, oi := range f.outputIntents {
		f.newobj()
		mem := xmem.compress(oi.ICCProfile)
		compressedICC := mem.bytes()
		f.outf("<< /N 3 /Alternate /DeviceRGB /Length %d /Filter /FlateDecode >>", len(compressedICC))
		f.putstream(compressedICC)
		f.out("endobj")

		mem.release()
	}
}

func (f *Fpdf) enddoc() {
	if f.err != nil {
		return
	}
	f.layerEndDoc()
	f.putheader()
	// Embedded files
	f.putAttachments()
	f.putAnnotationsAttachments()
	f.putpages()
	f.putresources()
	if f.err != nil {
		return
	}
	// Bookmarks
	f.putbookmarks()
	// Metadata
	f.putxmp()
	// 	Info
	f.newobj()
	f.out("<<")
	f.putinfo()
	f.out(">>")
	f.out("endobj")
	// Output intent color profile streams
	f.putOutputIntentStreams()
	// 	Catalog
	f.newobj()
	f.out("<<")
	f.putcatalog()
	f.out(">>")
	f.out("endobj")
	// Cross-ref
	o := f.buffer.Len()
	f.out("xref")
	f.outf("0 %d", f.n+1)
	f.out("0000000000 65535 f ")
	for j := 1; j <= f.n; j++ {
		f.outf("%010d 00000 n ", f.offsets[j])
	}
	// Trailer
	f.out("trailer")
	f.out("<<")
	f.puttrailer()
	f.out(">>")
	f.out("startxref")
	f.outf("%d", o)
	f.out("%%EOF")
	f.state = 3
}

// GetDisplayMode returns the current display mode. See SetDisplayMode() for details.
func (f *Fpdf) GetDisplayMode() (zoomStr, layoutStr string) {
	return f.zoomMode, f.layoutMode
}

// SetDisplayMode sets advisory display directives for the document viewer.
// Pages can be displayed entirely on screen, occupy the full width of the
// window, use real size, be scaled by a specific zooming factor or use viewer
// default (configured in the Preferences menu of Adobe Reader). The page
// layout can be specified so that pages are displayed individually or in
// pairs.
//
// zoomStr can be "fullpage" to display the entire page on screen, "fullwidth"
// to use maximum width of window, "real" to use real size (equivalent to 100%
// zoom) or "default" to use viewer default mode.
//
// layoutStr can be "single" (or "SinglePage") to display one page at once,
// "continuous" (or "OneColumn") to display pages continuously, "two" (or
// "TwoColumnLeft") to display two pages on two columns with odd-numbered pages
// on the left, or "TwoColumnRight" to display two pages on two columns with
// odd-numbered pages on the right, or "TwoPageLeft" to display pages two at a
// time with odd-numbered pages on the left, or "TwoPageRight" to display pages
// two at a time with odd-numbered pages on the right, or "default" to use
// viewer default mode.
func (f *Fpdf) SetDisplayMode(zoomStr, layoutStr string) {
	if f.err != nil {
		return
	}
	if layoutStr == "" {
		layoutStr = "default"
	}
	switch zoomStr {
	case "fullpage", "fullwidth", "real", "default":
		f.zoomMode = zoomStr
	default:
		f.err = Errf("incorrect zoom display mode: %s", zoomStr)
		return
	}
	switch layoutStr {
	case "single", "continuous", "two", "default", "SinglePage", "OneColumn",
		"TwoColumnLeft", "TwoColumnRight", "TwoPageLeft", "TwoPageRight":
		f.layoutMode = layoutStr
	default:
		f.err = Errf("incorrect layout display mode: %s", layoutStr)
		return
	}
}
