package pdf

import (
	"webtyp.com/color"
	. "webtyp.com/fmt"
)

type TableBuilder struct {
	doc          *Document
	colWidths    []string // "30%", "auto", "40mm"
	rows         [][]*CellElement
	bg           color.Color
	borderBottom struct {
		width float64
		color color.Color
	}
	keepTogether bool
}

func (d *Document) AddTable() *TableBuilder {
	return &TableBuilder{
		doc: d,
	}
}

func (t *TableBuilder) Cols(widths ...string) *TableBuilder {
	t.colWidths = widths
	return t
}

func (t *TableBuilder) Row(items ...any) *TableBuilder {
	row := make([]*CellElement, 0)
	for _, item := range items {
		switch v := item.(type) {
		case string:
			row = append(row, Cell(Text(v)))
		case *TextElement:
			row = append(row, Cell(v))
		case *ImageElement:
			row = append(row, Cell(v))
		case *LineElement:
			row = append(row, Cell(v))
		case *CellElement:
			row = append(row, v)
		case *TableBuilder:
			row = append(row, Cell(v))
		default:
			t.doc.addError(Errf("row arg: unsupported type %T", item))
		}
	}
	t.rows = append(t.rows, row)
	return t
}

func (t *TableBuilder) Background(c color.Color) *TableBuilder {
	t.bg = c
	return t
}

func (t *TableBuilder) BorderBottom(width float64, c color.Color) *TableBuilder {
	t.borderBottom.width = width
	t.borderBottom.color = c
	return t
}

func (t *TableBuilder) KeepTogether() *TableBuilder {
	t.keepTogether = true
	return t
}

func (t *TableBuilder) Draw() *Document {
	if t.doc.err != nil {
		return t.doc
	}

	// 1. Resolve column widths
	pageW, _ := t.doc.internal.GetPageSize()
	lMargin, _, rMargin, _ := t.doc.internal.GetMargins()
	availW := pageW - lMargin - rMargin

	resolvedWidths := t.resolveWidths(availW)

	// 2. Layout rows
	for _, row := range t.rows {
		// Measure row height
		rowHeight := 0.0
		cellHeights := make([]float64, len(row))
		for i, cell := range row {
			if i >= len(resolvedWidths) {
				break
			}
			_, h := cell.measure(t.doc, resolvedWidths[i])
			cellHeights[i] = h
			if h > rowHeight {
				rowHeight = h
			}
		}

		// Check for page break
		if t.doc.getCursorY()+rowHeight > t.doc.internal.GetPageBreakTrigger() {
			t.doc.AddPage()
		}

		// Draw row background
		if t.bg != "" {
			t.doc.drawFilledRect(lMargin, t.doc.getCursorY(), availW, rowHeight, t.bg)
		}

		// Draw cells
		currX := lMargin
		currY := t.doc.getCursorY()
		for i, cell := range row {
			if i >= len(resolvedWidths) {
				break
			}
			cell.drawWithHeight(t.doc, currX, currY, resolvedWidths[i], rowHeight)
			currX += resolvedWidths[i]
		}

		// Border bottom
		if t.borderBottom.width > 0 {
			t.doc.drawLineH(lMargin, currY+rowHeight, availW, t.borderBottom.color, t.borderBottom.width)
		}

		t.doc.setCursorY(currY + rowHeight)
	}

	// Leave the cursor at the left margin so subsequent flow elements
	// render from the page edge, not from the last cell's X position.
	t.doc.setCursorX(lMargin)
	return t.doc
}

func (t *TableBuilder) resolveWidths(availW float64) []float64 {
	n := len(t.colWidths)
	if n == 0 {
		// Default to equal widths if not specified
		if len(t.rows) > 0 {
			n = len(t.rows[0])
		}
	}
	if n == 0 {
		return nil
	}

	widths := make([]float64, n)
	remainingW := availW
	autoIndices := make([]int, 0)

	for i := 0; i < n; i++ {
		wStr := ""
		if i < len(t.colWidths) {
			wStr = t.colWidths[i]
		}

		if HasSuffix(wStr, "%") {
			pct, _ := Convert(wStr[:len(wStr)-1]).Float64()
			widths[i] = availW * pct / 100.0
			remainingW -= widths[i]
		} else if wStr == "auto" || wStr == "" {
			autoIndices = append(autoIndices, i)
		} else {
			// Assume mm
			val, _ := Convert(wStr).Float64()
			widths[i] = val
			remainingW -= widths[i]
		}
	}

	if len(autoIndices) > 0 {
		// Measure each auto column's intrinsic content width: the widest
		// cell in that column when given unbounded width.
		intrinsic := make([]float64, len(autoIndices))
		total := 0.0
		for k, idx := range autoIndices {
			maxW := 0.0
			for _, row := range t.rows {
				if idx >= len(row) {
					continue
				}
				cw, _ := row[idx].measure(t.doc, 0)
				if cw > maxW {
					maxW = cw
				}
			}
			intrinsic[k] = maxW
			total += maxW
		}

		switch {
		case total > remainingW && total > 0:
			// Overflow: scale autos down proportionally to fit.
			scale := remainingW / total
			for k, idx := range autoIndices {
				widths[idx] = intrinsic[k] * scale
			}
		case total < remainingW:
			// Surplus: absorb the leftover so the table always fills the
			// content area. Distribute proportionally to intrinsic width,
			// or equally when all intrinsic widths are zero.
			extra := remainingW - total
			for k, idx := range autoIndices {
				share := 0.0
				switch {
				case total > 0:
					share = extra * intrinsic[k] / total
				default:
					share = extra / float64(len(autoIndices))
				}
				widths[idx] = intrinsic[k] + share
			}
		default:
			for k, idx := range autoIndices {
				widths[idx] = intrinsic[k]
			}
		}
	}

	return widths
}

// Implement Element interface for nested tables
func (t *TableBuilder) draw(doc *Document, x, y, w float64) float64 {
	// Nested table drawing logic
	// For simplicity, we assume 'w' is availW for this sub-table
	resolvedWidths := t.resolveWidths(w)
	totalH := 0.0
	currY := y
	for _, row := range t.rows {
		rowHeight := 0.0
		for i, cell := range row {
			if i >= len(resolvedWidths) {
				break
			}
			_, h := cell.measure(doc, resolvedWidths[i])
			if h > rowHeight {
				rowHeight = h
			}
		}

		currX := x
		for i, cell := range row {
			if i >= len(resolvedWidths) {
				break
			}
			cell.drawWithHeight(doc, currX, currY, resolvedWidths[i], rowHeight)
			currX += resolvedWidths[i]
		}
		currY += rowHeight
		totalH += rowHeight
	}
	return totalH
}

func (t *TableBuilder) measure(doc *Document, w float64) (float64, float64) {
	resolvedWidths := t.resolveWidths(w)
	totalH := 0.0
	for _, row := range t.rows {
		rowHeight := 0.0
		for i, cell := range row {
			if i >= len(resolvedWidths) {
				break
			}
			_, h := cell.measure(doc, resolvedWidths[i])
			if h > rowHeight {
				rowHeight = h
			}
		}
		totalH += rowHeight
	}
	return w, totalH
}
