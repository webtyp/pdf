package fpdf

import (
	"bytes"
	"io"
	"path"
	"path/filepath"
	"sort"

	. "webtyp.com/fmt"
)

// AddUTF8FontFromBytes  imports a TrueType font with utf-8 symbols from static
// bytes within the executable and makes it available for use in the generated
// document.
//
// family specifies the font family. The name can be chosen arbitrarily. If it
// is a standard family name, it will override the corresponding font. This
// string is used to subsequently set the font with the SetFont method.
//
// style specifies the font style. Acceptable values are (case insensitive) the
// empty string for regular style, "B" for bold, "I" for italic, or "BI" or
// "IB" for bold and italic combined.
//
// jsonFileBytes contain all bytes of JSON file.
//
// zFileBytes contain all bytes of Z file.
func (f *Fpdf) AddUTF8FontFromBytes(familyStr, styleStr string, utf8Bytes []byte) {
	f.addFontFromBytes(fontFamilyEscape(familyStr), styleStr, nil, nil, utf8Bytes)
}

func (f *Fpdf) addFontFromBytes(familyStr, styleStr string, jsonFileBytes, zFileBytes, utf8Bytes []byte) {
	if f.err != nil {
		return
	}

	// load font key
	var ok bool
	fontkey := getFontKey(familyStr, styleStr)
	_, ok = f.fonts[fontkey]

	if ok {
		return
	}

	if utf8Bytes != nil {

		// if styleStr == "IB" {
		// 	styleStr = "BI"
		// }

		Type := "UTF8"
		reader := fileReader{readerPosition: 0, array: utf8Bytes}

		utf8File := newUTF8Font(&reader)

		err := utf8File.parseFile()
		if err != nil {
			println(Sprintf("get metrics Error: %v", err))
			return
		}
		desc := FontDescType{
			Ascent:       int(utf8File.Ascent),
			Descent:      int(utf8File.Descent),
			CapHeight:    utf8File.CapHeight,
			Flags:        utf8File.Flags,
			FontBBox:     utf8File.Bbox,
			ItalicAngle:  utf8File.ItalicAngle,
			StemV:        utf8File.StemV,
			MissingWidth: round(utf8File.DefaultWidth),
		}

		var sbarr map[int]int
		if f.aliasNbPagesStr == "" {
			sbarr = makeSubsetRange(57)
		} else {
			sbarr = makeSubsetRange(32)
		}
		def := fontDefType{
			Tp:        Type,
			Name:      fontkey,
			Desc:      desc,
			Up:        int(round(utf8File.UnderlinePosition)),
			Ut:        round(utf8File.UnderlineThickness),
			Cw:        utf8File.CharWidths,
			utf8File:  utf8File,
			usedRunes: sbarr,
		}
		def.i, _ = generateFontID(def)
		f.fonts[fontkey] = def
	} else {
		// load font definitions
		var info fontDefType
		err := unmarshalFontDef(jsonFileBytes, &info)

		if err != nil {
			f.err = err
		}

		if f.err != nil {
			return
		}

		if info.i, err = generateFontID(info); err != nil {
			f.err = err
			return
		}

		// search existing encodings
		if len(info.Diff) > 0 {
			n := -1

			for j, str := range f.diffs {
				if str == info.Diff {
					n = j + 1
					break
				}
			}

			if n < 0 {
				f.diffs = append(f.diffs, info.Diff)
				n = len(f.diffs)
			}

			info.DiffN = n
		}

		// embed font
		if len(info.File) > 0 {
			if info.Tp == "TrueType" {
				f.fontFiles[info.File] = fontFileType{
					length1:  int64(info.OriginalSize),
					embedded: true,
					content:  zFileBytes,
				}
			} else {
				f.fontFiles[info.File] = fontFileType{
					length1:  int64(info.Size1),
					length2:  int64(info.Size2),
					embedded: true,
					content:  zFileBytes,
				}
			}
		}

		f.fonts[fontkey] = info
	}
}

// getFontKey is used by AddFontFromReader and GetFontDesc
func getFontKey(familyStr, styleStr string) string {
	familyStr = Convert(familyStr).ToLower().String()
	styleStr = Convert(styleStr).ToUpper().String()
	if styleStr == "IB" {
		styleStr = "BI"
	}
	return familyStr + styleStr
}

// GetFontDesc returns the font descriptor, which can be used for
// example to find the baseline of a font. If familyStr is empty
// current font descriptor will be returned.
// See FontDescType for documentation about the font descriptor.
// See AddFont for details about familyStr and styleStr.
func (f *Fpdf) GetFontDesc(familyStr, styleStr string) FontDescType {
	if familyStr == "" {
		return f.currentFont.Desc
	}
	return f.fonts[getFontKey(fontFamilyEscape(familyStr), styleStr)].Desc
}

// SetFont sets the font used to print character strings. It is mandatory to
// call this method at least once before printing text or the resulting
// document will not be valid.
//
// The font can be either a standard one or a font added via the AddFont()
// method or AddFontFromReader() method. Standard fonts use the Windows
// encoding cp1252 (Western Europe).
//
// The method can be called before the first page is created and the font is
// kept from page to page. If you just wish to change the current font size, it
// is simpler to call SetFontSize().
//
// Note: the font definition file must be accessible. An error is set if the
// file cannot be read.
//
// familyStr specifies the font family. It can be either a name defined by
// AddFont(), AddFontFromReader() or one of the standard families (case
// insensitive): "Courier" for fixed-width, "Helvetica" or "Arial" for sans
// serif, "Times" for serif, "Symbol" or "ZapfDingbats" for symbolic.
//
// styleStr can be "B" (bold), "I" (italic), "U" (underscore), "S" (strike-out)
// or any combination. The default value (specified with an empty string) is
// regular. Bold and italic styles do not apply to Symbol and ZapfDingbats.
//
// size is the font size measured in points. The default value is the current
// size. If no size has been specified since the beginning of the document, the
// value taken is 12.
func (f *Fpdf) SetFont(familyStr, styleStr string, size float64) {
	// dbg("SetFont x %.2f, lMargin %.2f", f.x, f.lMargin)

	if f.err != nil {
		return
	}
	// dbg("SetFont")
	familyStr = fontFamilyEscape(familyStr)
	var ok bool
	if familyStr == "" {
		familyStr = f.fontFamily
	} else {
		familyStr = Convert(familyStr).ToLower().String()
	}
	styleStr = Convert(styleStr).ToUpper().String()
	f.underline = Contains(styleStr, "U")
	if f.underline {
		styleStr = Convert(styleStr).Replace("U", "").String()
	}
	f.strikeout = Contains(styleStr, "S")
	if f.strikeout {
		styleStr = Convert(styleStr).Replace("S", "").String()
	}
	if styleStr == "IB" {
		styleStr = "BI"
	}
	if size == 0.0 {
		size = f.fontSizePt
	}

	// Test if font is already loaded
	fontKey := familyStr + styleStr
	_, ok = f.fonts[fontKey]
	if !ok {
		f.err = Errf("undefined font: %s %s", familyStr, styleStr)
		return
	}
	// Select it
	f.fontFamily = familyStr
	f.fontStyle = styleStr
	f.fontSizePt = size
	f.fontSize = size / f.k
	f.currentFont = f.fonts[fontKey]
	if f.currentFont.Tp == "UTF8" {
		f.isCurrentUTF8 = true
	} else {
		f.isCurrentUTF8 = false
	}
	if f.page > 0 {
		f.outf("BT /F%s %.2f Tf ET", f.currentFont.i, f.fontSizePt)
	}
}

// GetFontFamily returns the family of the current font. See SetFont() for details.
func (f *Fpdf) GetFontFamily() string {
	return f.fontFamily
}

// GetFontStyle returns the style of the current font. See SetFont() for details.
func (f *Fpdf) GetFontStyle() string {
	styleStr := f.fontStyle

	if f.underline {
		styleStr += "U"
	}
	if f.strikeout {
		styleStr += "S"
	}

	return styleStr
}

// SetFontStyle sets the style of the current font. See also SetFont()
func (f *Fpdf) SetFontStyle(styleStr string) {
	f.SetFont(f.fontFamily, styleStr, f.fontSizePt)
}

// SetFontSize defines the size of the current font. Size is specified in
// points (1/ 72 inch). See also SetFontUnitSize().
func (f *Fpdf) SetFontSize(size float64) {
	f.fontSizePt = size
	f.fontSize = size / f.k
	if f.page > 0 {
		f.outf("BT /F%s %.2f Tf ET", f.currentFont.i, f.fontSizePt)
	}
}

// SetFontUnitSize defines the size of the current font. Size is specified in
// the unit of measure specified in New(). See also SetFontSize().
func (f *Fpdf) SetFontUnitSize(size float64) {
	f.fontSizePt = size * f.k
	f.fontSize = size
	if f.page > 0 {
		f.outf("BT /F%s %.2f Tf ET", f.currentFont.i, f.fontSizePt)
	}
}

// GetFontSize returns the size of the current font in points followed by the
// size in the unit of measure specified in New(). The second value can be used
// as a line height value in drawing operations.
func (f *Fpdf) GetFontSize() (ptSize, unitSize float64) {
	return f.fontSizePt, f.fontSize
}

// GetFontLoader returns the loader used to read font files (.json and .z) from
// an arbitrary source.
func (f *Fpdf) GetFontLoader() FontLoader {
	return f.fontLoader
}

// SetFontLoader sets a loader used to read font files (.json and .z) from an
// arbitrary source. If a font loader has been specified, it is used to load
// the named font resources when AddFont() is called. If this operation fails,
// an attempt is made to load the resources from the configured font directory
// (see SetFontLocation()).
func (f *Fpdf) SetFontLoader(loader FontLoader) {
	f.fontLoader = loader
}

// AddFont imports a TrueType, OpenType or Type1 font and makes it available.
// It is necessary to generate a font definition file first with the makefont
// utility. It is not necessary to call this function for the core PDF fonts
// (courier, helvetica, times, zapfdingbats).
//
// The JSON definition file (and the font file itself when embedding) must be
// present in the font directory. If it is not found, the error "Could not
// include font definition file" is set.
//
// family specifies the font family. The name can be chosen arbitrarily. If it
// is a standard family name, it will override the corresponding font. This
// string is used to subsequently set the font with the SetFont method.
//
// style specifies the font style. Acceptable values are (case insensitive) the
// empty string for regular style, "B" for bold, "I" for italic, or "BI" or
// "IB" for bold and italic combined.
//

// AddUTF8Font imports a TrueType font with utf-8 symbols and makes it available.
// It is necessary to generate a font definition file first with the makefont
// utility. It is not necessary to call this function for the core PDF fonts
// (courier, helvetica, times, zapfdingbats).
//
// The JSON definition file (and the font file itself when embedding) must be
// present in the font directory. If it is not found, the error "Could not
// include font definition file" is set.
//
// family specifies the font family. The name can be chosen arbitrarily. If it
// is a standard family name, it will override the corresponding font. This
// string is used to subsequently set the font with the SetFont method.
//
// style specifies the font style. Acceptable values are (case insensitive) the
// empty string for regular style, "B" for bold, "I" for italic, or "BI" or
// "IB" for bold and italic combined.
//
// fileStr specifies the base name with ".json" extension of the font
// definition file to be added. The file will be loaded from the font directory
// specified in the call to New() or SetFontLocation().
func (f *Fpdf) AddUTF8Font(familyStr, styleStr, fileStr string) {
	f.addFont(fontFamilyEscape(familyStr), styleStr, fileStr)
}

func (f *Fpdf) addFont(familyStr, styleStr, fileStr string) {
	if fileStr == "" {
		fileStr = Convert(familyStr).Replace(" ", "").String() + Convert(styleStr).ToLower().String() + ".ttf"
	}
	fontKey := getFontKey(familyStr, styleStr)
	_, ok := f.fonts[fontKey]
	if ok {
		return
	}
	var originalSize int64
	var err error
	// If fileStr is already an absolute path, use it directly
	// Otherwise, join it with the fonts path
	if !filepath.IsAbs(fileStr) {
		fileStr = path.Join(f.fontsPath, fileStr)
	}
	Type := "UTF8"
	var utf8Bytes []byte
	utf8Bytes, err = f.files.ReadFile(fileStr)
	if err != nil {
		f.SetError(err)
		return
	}
	originalSize = int64(len(utf8Bytes))
	reader := fileReader{readerPosition: 0, array: utf8Bytes}
	utf8File := newUTF8Font(&reader)
	err = utf8File.parseFile()
	if err != nil {
		f.SetError(err)
		return
	}

	desc := FontDescType{
		Ascent:       int(utf8File.Ascent),
		Descent:      int(utf8File.Descent),
		CapHeight:    utf8File.CapHeight,
		Flags:        utf8File.Flags,
		FontBBox:     utf8File.Bbox,
		ItalicAngle:  utf8File.ItalicAngle,
		StemV:        utf8File.StemV,
		MissingWidth: round(utf8File.DefaultWidth),
	}

	var sbarr map[int]int
	if f.aliasNbPagesStr == "" {
		sbarr = makeSubsetRange(57)
	} else {
		sbarr = makeSubsetRange(32)
	}
	def := fontDefType{
		Tp:        Type,
		Name:      fontKey,
		Desc:      desc,
		Up:        int(round(utf8File.UnderlinePosition)),
		Ut:        round(utf8File.UnderlineThickness),
		Cw:        utf8File.CharWidths,
		usedRunes: sbarr,
		File:      fileStr,
		utf8File:  utf8File,
	}
	def.i, _ = generateFontID(def)
	f.fonts[fontKey] = def
	f.fontFiles[fontKey] = fontFileType{
		length1:  originalSize,
		fontType: "UTF8",
	}
	f.fontFiles[fileStr] = fontFileType{
		fontType: "UTF8",
	}
}

// GetFontLocation returns the location in the file system of the font and font
// definition files.
func (f *Fpdf) GetFontLocation() string {
	return f.fontsPath
}

func (f *Fpdf) loadFontFile(name string) ([]byte, error) {
	if f.fontLoader != nil {
		reader, err := f.fontLoader.Open(name)
		if err == nil {
			data, err := io.ReadAll(reader)
			if closer, ok := reader.(io.Closer); ok {
				closer.Close()
			}
			return data, err
		}
	}
	return f.files.ReadFile(path.Join(f.fontsPath, name))
}

func isAbsolutePath(p string) bool {
	return filepath.IsAbs(p)
}

func (f *Fpdf) putfonts() {
	if f.err != nil {
		return
	}
	nf := f.n
	for _, diff := range f.diffs {
		// Encodings
		f.newobj()
		f.outf("<</Type /Encoding /BaseEncoding /WinAnsiEncoding /Differences [%s]>>", diff)
		f.out("endobj")
	}
	{
		var fileList []string
		var info fontFileType
		var file string
		for file = range f.fontFiles {
			fileList = append(fileList, file)
		}
		if f.catalogSort {
			sort.SliceStable(fileList, func(i, j int) bool { return fileList[i] < fileList[j] })
		}
		for _, file = range fileList {
			info = f.fontFiles[file]
			if info.fontType != "UTF8" {
				f.newobj()
				info.n = f.n
				f.fontFiles[file] = info

				var font []byte

				if info.embedded {
					font = info.content
				} else {
					var err error
					font, err = f.loadFontFile(file)
					if err != nil {
						f.err = err
						return
					}
				}
				compressed := file[len(file)-2:] == ".z"
				if !compressed && info.length2 > 0 {
					buf := font[6:info.length1]
					buf = append(buf, font[6+info.length1+6:info.length2]...)
					font = buf
				}
				f.outf("<</Length %d", len(font))
				if compressed {
					f.out("/Filter /FlateDecode")
				}
				f.outf("/Length1 %d", info.length1)
				if info.length2 > 0 {
					f.outf("/Length2 %d /Length3 0", info.length2)
				}
				f.out(">>")
				f.putstream(font)
				f.out("endobj")
			}
		}
	}
	{
		var keyList []string
		var font fontDefType
		var key string
		for key = range f.fonts {
			keyList = append(keyList, key)
		}
		if f.catalogSort {
			sort.SliceStable(keyList, func(i, j int) bool { return keyList[i] < keyList[j] })
		}
		for _, key = range keyList {
			font = f.fonts[key]
			// Font objects
			font.N = f.n + 1
			f.fonts[key] = font
			tp := font.Tp
			name := font.Name
			switch tp {
			case "Core":
				// Core font
				f.newobj()
				f.out("<</Type /Font")
				f.outf("/BaseFont /%s", name)
				f.out("/Subtype /Type1")
				if name != "Symbol" && name != "ZapfDingbats" {
					f.out("/Encoding /WinAnsiEncoding")
				}
				f.out(">>")
				f.out("endobj")
			case "Type1":
				fallthrough
			case "TrueType":
				// Additional Type1 or TrueType/OpenType font
				f.newobj()
				f.out("<</Type /Font")
				f.outf("/BaseFont /%s", name)
				f.outf("/Subtype /%s", tp)
				f.out("/FirstChar 32 /LastChar 255")
				f.outf("/Widths %d 0 R", f.n+1)
				f.outf("/FontDescriptor %d 0 R", f.n+2)
				if font.DiffN > 0 {
					f.outf("/Encoding %d 0 R", nf+font.DiffN)
				} else {
					f.out("/Encoding /WinAnsiEncoding")
				}
				f.out(">>")
				f.out("endobj")
				// Widths
				f.newobj()
				var s fmtBuffer
				s.WriteString("[")
				for j := 32; j < 256; j++ {
					s.printf("%d ", font.Cw[j])
				}
				s.WriteString("]")
				f.out(s.String())
				f.out("endobj")
				// Descriptor
				f.newobj()
				s.Truncate(0)
				s.printf("<</Type /FontDescriptor /FontName /%s ", name)
				s.printf("/Ascent %d ", font.Desc.Ascent)
				s.printf("/Descent %d ", font.Desc.Descent)
				s.printf("/CapHeight %d ", font.Desc.CapHeight)
				s.printf("/Flags %d ", font.Desc.Flags)
				s.printf("/FontBBox [%d %d %d %d] ", font.Desc.FontBBox.Xmin, font.Desc.FontBBox.Ymin,
					font.Desc.FontBBox.Xmax, font.Desc.FontBBox.Ymax)
				s.printf("/ItalicAngle %d ", font.Desc.ItalicAngle)
				s.printf("/StemV %d ", font.Desc.StemV)
				s.printf("/MissingWidth %d ", font.Desc.MissingWidth)
				var suffix string
				if tp != "Type1" {
					suffix = "2"
				}
				s.printf("/FontFile%s %d 0 R>>", suffix, f.fontFiles[font.File].n)
				f.out(s.String())
				f.out("endobj")
			case "UTF8":
				fontName := "utf8" + font.Name
				usedRunes := font.usedRunes
				delete(usedRunes, 0)
				utf8FontStream := font.utf8File.GenerateCutFont(usedRunes)
				utf8FontSize := len(utf8FontStream)
				CodeSignDictionary := font.utf8File.CodeSymbolDictionary
				delete(CodeSignDictionary, 0)

				f.newobj()
				f.out(Sprintf("<</Type /Font\n/Subtype /Type0\n/BaseFont /%s\n/Encoding /Identity-H\n/DescendantFonts [%d 0 R]\n/ToUnicode %d 0 R>>\nendobj", fontName, f.n+1, f.n+2))

				f.newobj()
				f.out("<</Type /Font\n/Subtype /CIDFontType2\n/BaseFont /" + fontName + "\n" +
					"/CIDSystemInfo " + Convert(f.n+2).String() + " 0 R\n/FontDescriptor " + Convert(f.n+3).String() + " 0 R")
				if font.Desc.MissingWidth != 0 {
					f.out("/DW " + Convert(font.Desc.MissingWidth).String())
				}
				f.generateCIDFontMap(&font, font.utf8File.LastRune)
				f.out("/CIDToGIDMap " + Convert(f.n+4).String() + " 0 R>>")
				f.out("endobj")

				f.newobj()
				f.out("<</Length " + Convert(len(toUnicode)).String() + ">>")
				f.putstream([]byte(toUnicode))
				f.out("endobj")

				// CIDInfo
				f.newobj()
				f.out("<</Registry (Adobe)\n/Ordering (UCS)\n/Supplement 0>>")
				f.out("endobj")

				// Font descriptor
				f.newobj()
				var s fmtBuffer
				s.printf("<</Type /FontDescriptor /FontName /%s\n /Ascent %d", fontName, font.Desc.Ascent)
				s.printf(" /Descent %d", font.Desc.Descent)
				s.printf(" /CapHeight %d", font.Desc.CapHeight)
				v := font.Desc.Flags
				v = v | 4
				v = v &^ 32
				s.printf(" /Flags %d", v)
				s.printf("/FontBBox [%d %d %d %d] ", font.Desc.FontBBox.Xmin, font.Desc.FontBBox.Ymin,
					font.Desc.FontBBox.Xmax, font.Desc.FontBBox.Ymax)
				s.printf(" /ItalicAngle %d", font.Desc.ItalicAngle)
				s.printf(" /StemV %d", font.Desc.StemV)
				s.printf(" /MissingWidth %d", font.Desc.MissingWidth)
				s.printf("/FontFile2 %d 0 R", f.n+2)
				s.printf(">>")
				f.out(s.String())
				f.out("endobj")

				// Embed CIDToGIDMap
				cidToGidMap := make([]byte, 256*256*2)

				for cc, glyph := range CodeSignDictionary {
					cidToGidMap[cc*2] = byte(glyph >> 8)
					cidToGidMap[cc*2+1] = byte(glyph & 0xFF)
				}

				mem := xmem.compress(cidToGidMap)
				cidToGidMap = mem.bytes()
				f.newobj()
				f.out("<</Length " + Convert(len(cidToGidMap)).String() + "/Filter /FlateDecode>>")
				f.putstream(cidToGidMap)
				f.out("endobj")
				mem.release()

				//Font file
				mem = xmem.compress(utf8FontStream)
				compressedFontStream := mem.bytes()
				f.newobj()
				f.out("<</Length " + Convert(len(compressedFontStream)).String())
				f.out("/Filter /FlateDecode")
				f.out("/Length1 " + Convert(utf8FontSize).String())
				f.out(">>")
				f.putstream(compressedFontStream)
				f.out("endobj")
				mem.release()
			default:
				f.err = Errf("unsupported font type: %s", tp)
				return
			}
		}
	}
}

func (f *Fpdf) generateCIDFontMap(font *fontDefType, LastRune int) {
	rangeID := 0
	cidArray := make(map[int]*untypedKeyMap)
	cidArrayKeys := make([]int, 0)
	prevCid := -2
	prevWidth := -1
	interval := false
	startCid := 1
	cwLen := LastRune + 1

	// for each character
	for cid := startCid; cid < cwLen; cid++ {
		if font.Cw[cid] == 0x00 {
			continue
		}
		width := font.Cw[cid]
		if width == 65535 {
			width = 0
		}
		if numb, OK := font.usedRunes[cid]; cid > 255 && (!OK || numb == 0) {
			continue
		}

		if cid == prevCid+1 {
			if width == prevWidth {

				if width == cidArray[rangeID].get(0) {
					cidArray[rangeID].put(nil, width)
				} else {
					cidArray[rangeID].pop()
					rangeID = prevCid
					r := untypedKeyMap{
						valueSet: make([]int, 0),
						keySet:   make([]any, 0),
					}
					cidArray[rangeID] = &r
					cidArrayKeys = append(cidArrayKeys, rangeID)
					cidArray[rangeID].put(nil, prevWidth)
					cidArray[rangeID].put(nil, width)
				}
				interval = true
				cidArray[rangeID].put("interval", 1)
			} else {
				if interval {
					// new range
					rangeID = cid
					r := untypedKeyMap{
						valueSet: make([]int, 0),
						keySet:   make([]any, 0),
					}
					cidArray[rangeID] = &r
					cidArrayKeys = append(cidArrayKeys, rangeID)
					cidArray[rangeID].put(nil, width)
				} else {
					cidArray[rangeID].put(nil, width)
				}
				interval = false
			}
		} else {
			rangeID = cid
			r := untypedKeyMap{
				valueSet: make([]int, 0),
				keySet:   make([]any, 0),
			}
			cidArray[rangeID] = &r
			cidArrayKeys = append(cidArrayKeys, rangeID)
			cidArray[rangeID].put(nil, width)
			interval = false
		}
		prevCid = cid
		prevWidth = width

	}
	previousKey := -1
	nextKey := -1
	isInterval := false
	for g := 0; g < len(cidArrayKeys); {
		key := cidArrayKeys[g]
		ws := *cidArray[key]
		cws := len(ws.keySet)
		if (key == nextKey) && (!isInterval) && (ws.getIndex("interval") < 0 || cws < 4) {
			if cidArray[key].getIndex("interval") >= 0 {
				cidArray[key].delete("interval")
			}
			cidArray[previousKey] = arrayMerge(cidArray[previousKey], cidArray[key])
			cidArrayKeys = remove(cidArrayKeys, key)
		} else {
			g++
			previousKey = key
		}
		nextKey = key + cws
		// ui := ws.getIndex("interval")
		// ui = ui + 1
		if ws.getIndex("interval") >= 0 {
			if cws > 3 {
				isInterval = true
			} else {
				isInterval = false
			}
			cidArray[key].delete("interval")
			nextKey--
		} else {
			isInterval = false
		}
	}
	var w fmtBuffer
	for _, k := range cidArrayKeys {
		ws := cidArray[k]
		if len(arrayCountValues(ws.valueSet)) == 1 {
			w.printf(" %d %d %d", k, k+len(ws.valueSet)-1, ws.get(0))
		} else {
			w.printf(" %d [ %s ]\n", k, implode(" ", ws.valueSet))
		}
	}
	f.out("/W [" + w.String() + " ]")
}

// Load a font definition file from the given Reader
func (f *Fpdf) loadfont(r io.Reader) (def fontDefType) {
	if f.err != nil {
		return
	}
	// dbg("Loading font [%s]", fontStr)
	var buf bytes.Buffer
	_, err := buf.ReadFrom(r)
	if err != nil {
		f.err = err
		return
	}
	err = unmarshalFontDef(buf.Bytes(), &def)
	if err != nil {
		f.err = err
		return
	}

	if def.i, err = generateFontID(def); err != nil {
		f.err = err
	}
	// dump(def)
	return
}
