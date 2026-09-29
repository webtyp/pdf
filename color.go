package pdf

import (
	"webtyp.com/color"
)

type Theme struct {
	Accent color.Color // color para texto de headers (H1, H2, H3)
	Brand  color.Color // color de marca para elementos decorativos (líneas, bandas)
	Header color.Color
	Gray   color.Color
	Body   color.Color
	Sizes  struct {
		H1, H2, H3, Body, Small float64
	}
	Spacing struct {
		Paragraph, Section, Page float64
	}
	// Page size in mm. Zero values keep the fpdf default (A4).
	Page ThemePage
	// Margin in mm applied to all four sides. Zero values keep the default (20mm).
	Margin ThemeMargin
}

// ThemePage defines the page dimensions in millimetres.
type ThemePage struct {
	Width, Height float64
}

// ThemeMargin defines the four page margins in millimetres.
type ThemeMargin struct {
	Top, Right, Bottom, Left float64
}

var DefaultTheme = Theme{
	Accent: "#1E3C78",
	Header: "#F0F4FA",
	Gray:   "#646464",
	Body:   "#000000",
	Sizes: struct {
		H1, H2, H3, Body, Small float64
	}{
		H1:    16,
		H2:    12,
		H3:    10,
		Body:  10,
		Small: 8,
	},
	Spacing: struct {
		Paragraph, Section, Page float64
	}{
		Paragraph: 2,
		Section:   5,
		Page:      20,
	},
}
