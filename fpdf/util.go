package fpdf

import (
	"math"

	. "webtyp.com/fmt"
)

func must(n int, err error) {
	if err != nil {
		panic(err)
	}
}

func must64(n int64, err error) {
	if err != nil {
		panic(err)
	}
}

func round(f float64) int {
	if f < 0 {
		return -int(math.Floor(-f + 0.5))
	}
	return int(math.Floor(f + 0.5))
}

func sprintf(fmtStr string, args ...any) string {
	return Sprintf(fmtStr, args...)
}

// utf8toutf16 converts UTF-8 to UTF-16BE; from http://www.fpdf.org/
func utf8toutf16(s string, withBOM ...bool) string {
	bom := true
	if len(withBOM) > 0 {
		bom = withBOM[0]
	}
	res := make([]byte, 0, 8)
	if bom {
		res = append(res, 0xFE, 0xFF)
	}
	nb := len(s)
	i := 0
	for i < nb {
		c1 := byte(s[i])
		i++
		switch {
		case c1 >= 224:
			// 3-byte character
			c2 := byte(s[i])
			i++
			c3 := byte(s[i])
			i++
			res = append(res, ((c1&0x0F)<<4)+((c2&0x3C)>>2),
				((c2&0x03)<<6)+(c3&0x3F))
		case c1 >= 192:
			// 2-byte character
			c2 := byte(s[i])
			i++
			res = append(res, ((c1 & 0x1C) >> 2),
				((c1&0x03)<<6)+(c2&0x3F))
		default:
			// Single-byte character
			res = append(res, 0, c1)
		}
	}
	return string(res)
}

// intIf returns a if cnd is true, otherwise b
func intIf(cnd bool, a, b int) int {
	if cnd {
		return a
	}
	return b
}

// strIf returns aStr if cnd is true, otherwise bStr
func strIf(cnd bool, aStr, bStr string) string {
	if cnd {
		return aStr
	}
	return bStr
}

// Transform moves a point by given X, Y offset
func (p *PointType) Transform(x, y float64) PointType {
	return PointType{p.X + x, p.Y + y}
}

// Orientation returns the orientation of a given size:
// "P" for portrait, "L" for landscape
func (s *SizeType) Orientation() string {
	if s == nil || s.Ht == s.Wd {
		return ""
	}
	if s.Wd > s.Ht {
		return "L"
	}
	return "P"
}

// ScaleBy expands a size by a certain factor
func (s *SizeType) ScaleBy(factor float64) SizeType {
	return SizeType{s.Wd * factor, s.Ht * factor}
}

// ScaleToWidth adjusts the height of a size to match the given width
func (s *SizeType) ScaleToWidth(width float64) SizeType {
	height := s.Ht * width / s.Wd
	return SizeType{width, height}
}

// ScaleToHeight adjusts the width of a size to match the given height
func (s *SizeType) ScaleToHeight(height float64) SizeType {
	width := s.Wd * height / s.Ht
	return SizeType{width, height}
}

// The untypedKeyMap structure and its methods are copyrighted 2019 by Arteom Korotkiy (Gmail: arteomkorotkiy).
// Imitation of untyped Map Array
type untypedKeyMap struct {
	keySet   []any
	valueSet []int
}

// Get position of key=>value in PHP Array
func (pa *untypedKeyMap) getIndex(key any) int {
	if key != nil {
		for i, mKey := range pa.keySet {
			if mKey == key {
				return i
			}
		}
		return -1
	}
	return -1
}

// Put key=>value in PHP Array
func (pa *untypedKeyMap) put(key any, value int) {
	if key == nil {
		var i int
		for n := 0; ; n++ {
			i = pa.getIndex(n)
			if i < 0 {
				key = n
				break
			}
		}
		pa.keySet = append(pa.keySet, key)
		pa.valueSet = append(pa.valueSet, value)
	} else {
		i := pa.getIndex(key)
		if i < 0 {
			pa.keySet = append(pa.keySet, key)
			pa.valueSet = append(pa.valueSet, value)
		} else {
			pa.valueSet[i] = value
		}
	}
}

// Delete value in PHP Array
func (pa *untypedKeyMap) delete(key any) {
	if pa == nil || pa.keySet == nil || pa.valueSet == nil {
		return
	}
	i := pa.getIndex(key)
	if i >= 0 {
		if i == 0 {
			pa.keySet = pa.keySet[1:]
			pa.valueSet = pa.valueSet[1:]
		} else if i == len(pa.keySet)-1 {
			pa.keySet = pa.keySet[:len(pa.keySet)-1]
			pa.valueSet = pa.valueSet[:len(pa.valueSet)-1]
		} else {
			pa.keySet = append(pa.keySet[:i], pa.keySet[i+1:]...)
			pa.valueSet = append(pa.valueSet[:i], pa.valueSet[i+1:]...)
		}
	}
}

// Get value from PHP Array
func (pa *untypedKeyMap) get(key any) int {
	i := pa.getIndex(key)
	if i >= 0 {
		return pa.valueSet[i]
	}
	return 0
}

// Imitation of PHP function pop()
func (pa *untypedKeyMap) pop() {
	pa.keySet = pa.keySet[:len(pa.keySet)-1]
	pa.valueSet = pa.valueSet[:len(pa.valueSet)-1]
}

// Imitation of PHP function array_merge()
func arrayMerge(arr1, arr2 *untypedKeyMap) *untypedKeyMap {
	answer := untypedKeyMap{}
	if arr1 == nil && arr2 == nil {
		answer = untypedKeyMap{
			make([]any, 0),
			make([]int, 0),
		}
	} else if arr2 == nil {
		answer.keySet = arr1.keySet[:]
		answer.valueSet = arr1.valueSet[:]
	} else if arr1 == nil {
		answer.keySet = arr2.keySet[:]
		answer.valueSet = arr2.valueSet[:]
	} else {
		answer.keySet = arr1.keySet[:]
		answer.valueSet = arr1.valueSet[:]
		for i := 0; i < len(arr2.keySet); i++ {
			if arr2.keySet[i] == "interval" {
				if arr1.getIndex("interval") < 0 {
					answer.put("interval", arr2.valueSet[i])
				}
			} else {
				answer.put(nil, arr2.valueSet[i])
			}
		}
	}
	return &answer
}

func remove(arr []int, key int) []int {
	n := 0
	for i, mKey := range arr {
		if mKey == key {
			n = i
		}
	}
	if n == 0 {
		return arr[1:]
	} else if n == len(arr)-1 {
		return arr[:len(arr)-1]
	}
	return append(arr[:n], arr[n+1:]...)
}

func isChinese(rune2 rune) bool {
	// chinese unicode: 4e00-9fa5
	if rune2 >= rune(0x4e00) && rune2 <= rune(0x9fa5) {
		return true
	}
	return false
}

// Condition font family string to PDF name compliance. See section 5.3 (Names)
// in https://resources.infosecinstitute.com/pdf-file-format-basic-structure/
func fontFamilyEscape(familyStr string) (escStr string) {
	escStr = Convert(familyStr).Replace(" ", "#20", -1).String()
	// Additional replacements can take place here
	return
}
