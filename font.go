/*
Package fontlonad is for typeface and font finding and loading.

There is a certain confusion with the nomenclature of typesetting. We will
stick to the following definitions:

▪︎ A "typeface" is a family of fonts. An example is "Helvetica".
This corresponds to a TrueType "collection" (*.ttc).

▪︎ A "scalable font" is a font, i.e. a variant of a typeface with a
certain weight, slant, etc.  An example is "Helvetica regular".

▪︎ A "typecase" is a scaled font, i.e. a font in a certain size for
a certain script and language. The name is reminiscend on the wooden
boxes of typesetters in the era of metal type.
An example is "Helvetica regular 11pt, Latin, en_US".

Please note that Go (Golang) does use the terms "font" and "face"
differently–actually more or less in an opposite manner.

# Status

Does not yet contain methods for font collections (*.ttc), e.g.,
/System/Library/Fonts/Helvetica.ttc on Mac OS.

# Links

OpenType explained:
https://docs.microsoft.com/en-us/typography/opentype/

______________________________________________________________________

# License

Governed by a 3-Clause BSD license. License file may be found in the root
folder of this module.

Copyright © Norbert Pillmayer <norbert@pillmayer.com>
*/
package fontfind

import (
	"embed"
	"errors"
	"io/fs"

	"github.com/npillmayer/schuko/tracing"
	"golang.org/x/image/font"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

// tracer writes to trace with key 'tyse.font'
func tracer() tracing.Trace {
	return tracing.Select("tyse.font")
}

const (
	StyleNormal = font.StyleNormal
	StyleItalic = font.StyleItalic
)

const (
	WeightLight    = font.WeightLight
	WeightNormal   = font.WeightNormal
	WeightSemiBold = font.WeightSemiBold
	WeightBold     = font.WeightBold
)

type Descriptor struct {
	Pattern string
	Style   font.Style
	Weight  font.Weight
}

type ScalableFont struct {
	Name       string
	Style      font.Style
	Weight     font.Weight
	fileSystem fs.FS
	path       string
}

func (f *ScalableFont) SetFS(fs fs.FS, path string) {
	f.fileSystem = fs
	f.path = path
}

func (f *ScalableFont) Path() string {
	return f.path
}

func (f *ScalableFont) ReadFontData() ([]byte, error) {
	if f.fileSystem == nil {
		return nil, errors.New("no file system to read from")
	}
	if f.path == "" {
		return nil, errors.New("path not set")
	}
	return fs.ReadFile(f.fileSystem, f.path)
}

var NullFont = ScalableFont{}

//go:embed locate/fallbackfont/packaged/Go-Regular.otf
var fallbackFS embed.FS

// FallbackFont returns the default packaged fallback font.
func FallbackFont() ScalableFont {
	return ScalableFont{
		Name:       "Go-Regular.otf",
		Style:      font.StyleNormal,
		Weight:     font.WeightNormal,
		path:       "locate/fallbackfont/packaged/Go-Regular.otf",
		fileSystem: fallbackFS,
	}
}

// ---------------------------------------------------------------------------

/*
u/em   = 2000
_em    = 12 pt  = 0,1666 in
_dpi   = 120
=>
_d/_em = 120 * 0,1666 = 19,992 pixels per em
=>
u1     = 150

2000 = 19,992
u1   = ?
=>  ? = _d/em * u1 / u/em

_u1    = 150 / _d/_em  = 7,503  pixels

Beispiel:
PT  = 12
DPI = 72
_d/_em = gtx.Px(DPI) * (PT / 72.27)
=> gtx.Px(12)  vereinfacht bei dpi = 72
*/

// PtIn is 72.27, i.e. printer's points per inch.
var PtIn fixed.Int26_6 = fixed.I(72) + fixed.I(27)/100

// PpEm calculates a ppem value for a given font point-size and an output resolution (dpi).
func PpEm(ptSize fixed.Int26_6, dpi float32) fixed.Int26_6 {
	_dpi := fixed.Int26_6(dpi * 64)
	return _dpi * (ptSize / PtIn)
}

// RasterCoords transforms `u`, a value in font-units, into pixel coordinates.
// Calculation is done for a font `sfont` at a given point-size `ptSize`.
func RasterCoords(u sfnt.Units, sfont *sfnt.Font, ptSize fixed.Int26_6, dpi float32) fixed.Int26_6 {
	_ppem := PpEm(ptSize, dpi)
	uem := sfont.UnitsPerEm()
	_uem := fixed.I(int(uem))
	_u := fixed.I(int(u)) * _ppem / _uem
	return _u
}
