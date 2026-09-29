package pdf

import (
	"webtyp.com/color"
	. "webtyp.com/fmt"
)

// Element is the interface for all layout components.
type Element interface {
	draw(doc *Document, x, y, w float64) (height float64)
	measure(doc *Document, w float64) (width, height float64)
}

// Text element represents a paragraph of text.
type TextElement struct {
	doc     *Document // set by Document.AddText (flow mode); nil for package-level pdf.Text
	content string
	bold    bool
	italic  bool
	size    float64
	color   color.Color
	align   string // L, C, R, J
}

func Text(s string) *TextElement {
	return &TextElement{content: s, align: "L"}
}

// Draw renders this text in flow mode. Requires the element to have been
// created via Document.AddText. Returns the document for chaining.
func (t *TextElement) Draw() *Document {
	if t.doc == nil {
		return nil
	}
	d := t.doc
	style := t.getStyle()
	size := t.getSize(d)
	color := t.color
	if color == "" {
		color = d.theme.Body
	}
	d.setTextColor(color)
	d.internal.SetFont(d.getActiveFontName(), style, size)

	// Force X back to the left margin so flow text always uses the full
	// content width, even if a previous primitive (e.g. a table) left the
	// cursor at the right edge.
	lMargin, _, _, _ := d.internal.GetMargins()
	d.setCursorX(lMargin)

	align := t.align
	if align == "" {
		align = "L"
	}
	// Width 0 = wrap to right margin. Line height = size * 0.45 gives a
	// ~1.27x ratio (size 10 -> 4.5mm), comfortable for body copy and close
	// to the Word/Google Docs default of 1.15-1.2x.
	d.internal.MultiCell(0, size*0.45, t.content, "", align, false)
	d.setTextColor(d.theme.Body)
	return d
}

func (t *TextElement) Bold() *TextElement {
	t.bold = true
	return t
}

func (t *TextElement) Italic() *TextElement {
	t.italic = true
	return t
}

func (t *TextElement) Size(pt float64) *TextElement {
	t.size = pt
	return t
}

func (t *TextElement) Color(c color.Color) *TextElement {
	t.color = c
	return t
}

func (t *TextElement) AlignLeft() *TextElement {
	t.align = "L"
	return t
}

func (t *TextElement) AlignCenter() *TextElement {
	t.align = "C"
	return t
}

func (t *TextElement) AlignRight() *TextElement {
	t.align = "R"
	return t
}

func (t *TextElement) Justify() *TextElement {
	t.align = "J"
	return t
}

func (t *TextElement) getStyle() string {
	style := ""
	if t.bold {
		style += "B"
	}
	if t.italic {
		style += "I"
	}
	return style
}

func (t *TextElement) getSize(doc *Document) float64 {
	if t.size > 0 {
		return t.size
	}
	return doc.theme.Sizes.Body
}

func (t *TextElement) draw(doc *Document, x, y, w float64) float64 {
	style := t.getStyle()
	size := t.getSize(doc)
	color := t.color
	if color == "" {
		color = doc.theme.Body
	}

	doc.setTextColor(color)
	doc.internal.SetFont(doc.getActiveFontName(), style, size)

	// MultiCell wraps subsequent lines to the document's left margin, not
	// to x. Temporarily move the left margin to x so wrapping happens
	// inside the cell. Restore it after rendering.
	origLMargin, _, _, _ := doc.internal.GetMargins()
	doc.internal.SetLeftMargin(x)
	doc.setPosition(x, y)

	lineHeight := size * 0.5
	yBefore := doc.internal.GetY()
	doc.internal.MultiCell(w, lineHeight, t.content, "", t.align, false)
	yAfter := doc.internal.GetY()

	doc.internal.SetLeftMargin(origLMargin)
	return yAfter - yBefore
}

func (t *TextElement) measure(doc *Document, w float64) (float64, float64) {
	style := t.getStyle()
	size := t.getSize(doc)
	doc.internal.SetFont(doc.getActiveFontName(), style, size)

	if w <= 0 {
		// Infinite width measurement for auto-sizing
		return doc.internal.GetStringWidth(t.content), size * 0.5
	}

	lines := doc.internal.SplitText(t.content, w)
	lineHeight := size * 0.5
	return w, float64(len(lines)) * lineHeight
}

// Image element.
type ImageElement struct {
	doc    *Document // set by Document.AddImage (flow mode); nil for package-level pdf.Image
	id     ImageID
	width  float64
	height float64
	align  string // L, C, R
}

func Image(id ImageID) *ImageElement {
	return &ImageElement{id: id, align: "L"}
}

// Draw renders this image in flow mode. Requires the element to have been
// created via Document.AddImage. Returns the document for chaining.
func (i *ImageElement) Draw() *Document {
	if i.doc == nil {
		return nil
	}
	d := i.doc
	x := d.getCursorX()
	y := d.getCursorY()
	pageW, _ := d.internal.GetPageSize()
	lMargin, _, rMargin, _ := d.internal.GetMargins()
	w := pageW - lMargin - rMargin - (x - lMargin)
	h := i.draw(d, x, y, w)
	d.setCursorY(y + h)
	return d
}

func (i *ImageElement) Width(mm float64) *ImageElement {
	i.width = mm
	return i
}

func (i *ImageElement) Height(mm float64) *ImageElement {
	i.height = mm
	return i
}

func (i *ImageElement) AlignLeft() *ImageElement {
	i.align = "L"
	return i
}

func (i *ImageElement) AlignCenter() *ImageElement {
	i.align = "C"
	return i
}

func (i *ImageElement) AlignRight() *ImageElement {
	i.align = "R"
	return i
}

func (i *ImageElement) draw(doc *Document, x, y, w float64) float64 {
	imgW := i.width
	imgH := i.height

	name := Sprintf("img_%d", int(i.id))
	info := doc.internal.GetImageInfo(name)
	if info != nil {
		if imgW == 0 && imgH == 0 {
			imgW = w // Fit to width by default if not specified?
			// Or keep original size? PLAN says Width(mm) optional.
			// Let's use info if available.
			imgW = info.Width()
			if imgW > w {
				imgW = w
			}
		}
		if imgW == 0 {
			imgW = imgH * info.Width() / info.Height()
		}
		if imgH == 0 {
			imgH = imgW * info.Height() / info.Width()
		}
	} else {
		if imgW == 0 {
			imgW = w
		}
		if imgH == 0 {
			imgH = 10
		} // Placeholder
	}

	posX := x
	if i.align == "C" {
		posX = x + (w-imgW)/2
	} else if i.align == "R" {
		posX = x + w - imgW
	}

	doc.drawImageAt(name, posX, y, imgW)
	return imgH
}

func (i *ImageElement) measure(doc *Document, w float64) (float64, float64) {
	imgW := i.width
	imgH := i.height
	name := Sprintf("img_%d", int(i.id))
	info := doc.internal.GetImageInfo(name)
	if info != nil {
		if imgW == 0 && imgH == 0 {
			imgW = info.Width()
			imgH = info.Height()
		} else if imgW == 0 {
			imgW = imgH * info.Width() / info.Height()
		} else if imgH == 0 {
			imgH = imgW * info.Height() / info.Width()
		}
	} else {
		if imgW == 0 {
			imgW = 20
		}
		if imgH == 0 {
			imgH = 20
		}
	}
	return imgW, imgH
}

// Line element.
type LineElement struct {
	width     float64
	color     color.Color
	thickness float64
}

func Line() *LineElement {
	return &LineElement{thickness: 0.2}
}

func (l *LineElement) Width(mm float64) *LineElement {
	l.width = mm
	return l
}

func (l *LineElement) Color(c color.Color) *LineElement {
	l.color = c
	return l
}

func (l *LineElement) Thickness(mm float64) *LineElement {
	l.thickness = mm
	return l
}

func (l *LineElement) draw(doc *Document, x, y, w float64) float64 {
	width := l.width
	if width == 0 {
		width = w
	}
	color := l.color
	if color == "" {
		color = doc.theme.Body
	}

	doc.drawLineH(x, y+l.thickness, width, color, l.thickness)
	return l.thickness * 2 // Some padding
}

func (l *LineElement) measure(doc *Document, w float64) (float64, float64) {
	width := l.width
	if width == 0 {
		width = w
	}
	return width, l.thickness * 2
}

// Cell container.
type CellElement struct {
	children []Element
	padding  [4]float64 // top, right, bottom, left
	bg       color.Color
	border   struct {
		width float64
		color color.Color
		sides string // "LTRB"
	}
	span struct {
		cols, rows int
	}
	hAlign string // L, C, R
	vAlign string // T, M, B
}

func Cell(children ...Element) *CellElement {
	return &CellElement{
		children: children,
		hAlign:   "L",
		vAlign:   "T",
	}
}

func (c *CellElement) Padding(top, right, bottom, left float64) *CellElement {
	c.padding = [4]float64{top, right, bottom, left}
	return c
}

func (c *CellElement) Background(col color.Color) *CellElement {
	c.bg = col
	return c
}

func (c *CellElement) Border(sides string, width float64, col color.Color) *CellElement {
	c.border.sides = sides
	c.border.width = width
	c.border.color = col
	return c
}

func (c *CellElement) Span(cols, rows int) *CellElement {
	c.span.cols = cols
	c.span.rows = rows
	return c
}

func (c *CellElement) AlignLeft() *CellElement {
	c.hAlign = "L"
	return c
}

func (c *CellElement) AlignCenter() *CellElement {
	c.hAlign = "C"
	return c
}

func (c *CellElement) AlignRight() *CellElement {
	c.hAlign = "R"
	return c
}

func (c *CellElement) AlignTop() *CellElement {
	c.vAlign = "T"
	return c
}

func (c *CellElement) AlignMiddle() *CellElement {
	c.vAlign = "M"
	return c
}

func (c *CellElement) AlignBottom() *CellElement {
	c.vAlign = "B"
	return c
}

func (c *CellElement) draw(doc *Document, x, y, w float64) float64 {
	_, h := c.measure(doc, w)
	return c.drawWithHeight(doc, x, y, w, h)
}

func (c *CellElement) drawWithHeight(doc *Document, x, y, w, h float64) float64 {
	// Background
	if c.bg != "" {
		doc.drawFilledRect(x, y, w, h, c.bg)
	}

	// Border - simple implementation
	if c.border.width > 0 {
		r, g, b, _ := c.border.color.RGB()
		doc.internal.SetDrawColor(r, g, b)
		doc.internal.SetLineWidth(c.border.width)
		if c.border.sides == "1" || c.border.sides == "" {
			doc.internal.Rect(x, y, w, h, "D")
		} else {
			// individual sides...
		}
	}

	innerX := x + c.padding[3]
	innerY := y + c.padding[0]
	innerW := w - c.padding[1] - c.padding[3]

	// Vertical alignment
	contentHeight := 0.0
	for _, child := range c.children {
		_, ch := child.measure(doc, innerW)
		contentHeight += ch
	}

	if c.vAlign == "M" {
		innerY += (h - c.padding[0] - c.padding[2] - contentHeight) / 2
	} else if c.vAlign == "B" {
		innerY = y + h - c.padding[2] - contentHeight
	}

	currY := innerY
	for _, child := range c.children {
		ch := child.draw(doc, innerX, currY, innerW)
		currY += ch
	}

	return h
}

func (c *CellElement) measure(doc *Document, w float64) (float64, float64) {
	innerW := w - c.padding[1] - c.padding[3]
	if w <= 0 {
		innerW = 0 // auto
	}

	contentH := 0.0
	maxChildW := 0.0
	for _, child := range c.children {
		cw, ch := child.measure(doc, innerW)
		contentH += ch
		if cw > maxChildW {
			maxChildW = cw
		}
	}

	finalW := w
	if w <= 0 {
		finalW = maxChildW + c.padding[1] + c.padding[3]
	}
	finalH := contentH + c.padding[0] + c.padding[2]

	return finalW, finalH
}
