package fallbackfont

import (
	"embed"
	"errors"

	"github.com/npillmayer/fontfind"
	"github.com/npillmayer/fontfind/locate"
	"github.com/npillmayer/schuko/tracing"
	"golang.org/x/image/font"
)

// tracer writes to trace with key 'tyse.font'
func tracer() tracing.Trace {
	return tracing.Select("tyse.font")
}

//go:embed packaged/*
var packaged embed.FS

const defaultFallbackFilename = "Go-Regular.otf"

func Find() locate.FontLocator {
	return func(descr fontfind.Descriptor) (fontfind.ScalableFont, error) {
		pattern := descr.Pattern
		style := descr.Style
		weight := descr.Weight
		return FindFallbackFont(pattern, style, weight)
	}
}

// Default returns the default packaged fallback font.
func Default() (fontfind.ScalableFont, error) {
	// Ensure packaged default exists in embedded resources.
	if _, err := packaged.Open("packaged/" + defaultFallbackFilename); err != nil {
		return fontfind.NullFont, err
	}
	return fontfind.ScalableFont{
		Name:       defaultFallbackFilename,
		Path:       "packaged/" + defaultFallbackFilename,
		FileSystem: packaged,
		Style:      font.StyleNormal,
		Weight:     font.WeightNormal,
	}, nil
}

func FindFallbackFont(pattern string, style font.Style, weight font.Weight) (fontfind.ScalableFont, error) {
	fonts, _ := packaged.ReadDir("packaged")
	var fname string // path to embedded font, if any
	for _, f := range fonts {
		if f.IsDir() {
			continue
		}
		if fontfind.Matches(f.Name(), pattern, style, weight) {
			tracer().Debugf("found embedded font file %s", f.Name())
			fname = f.Name()
			break
		}
		fname = f.Name()
	}
	var sFont fontfind.ScalableFont
	if fname == "" {
		return fontfind.NullFont, errors.New("font not found")
	}
	// font is packaged embedded font
	sFont.Name = fname
	sFont.Path = "packaged/" + fname
	sFont.FileSystem = packaged
	sFont.Style = style
	sFont.Weight = weight
	return sFont, nil
}
