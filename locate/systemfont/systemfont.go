package systemfont

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/flopp/go-findfont"
	"github.com/npillmayer/fontfind"
	"github.com/npillmayer/fontfind/locate"
	"github.com/npillmayer/schuko/tracing"
	"golang.org/x/image/font"
)

// tracer writes to trace with key 'tyse.font'
func tracer() tracing.Trace {
	return tracing.Select("tyse.font")
}

var USE_SYSTEM_IO IO = nil

// Find creates a FontLocator that resolves fonts from local system sources.
//
// appkey identifies the caller's config area used for fontconfig list lookup.
// io customizes host I/O and may be nil.
func Find(appkey string, io IO) locate.FontLocator {
	if io == nil {
		io = &systemIO{}
	}
	return func(descr fontfind.Descriptor) (fontfind.ScalableFont, error) {
		pattern := descr.Pattern
		style := descr.Style
		weight := descr.Weight
		return FindLocalFont(appkey, io, pattern, style, weight)
	}
}

// IO decouples font lookup from OS I/O for testability.
type IO interface {
	UserConfigDir() (string, error)
	DirFS(string) fs.FS
	ReadAll(io.Reader) ([]byte, error)
}

type systemIO struct{}

func (s *systemIO) UserConfigDir() (string, error) {
	return os.UserConfigDir()
}

func (s *systemIO) DirFS(path string) fs.FS {
	return os.DirFS(path)
}

func (s *systemIO) ReadAll(r io.Reader) ([]byte, error) {
	return io.ReadAll(r)
}

// FindLocalFont searches for a locally installed font variant.
//
// If present and configured, FindLocalFont uses the fontconfig
// system (https://www.freedesktop.org/wiki/Software/fontconfig/).
//
// If fontconfig is not configured, FindLocalFont will fall back to scanning
// system font folders (OS dependent).
func FindLocalFont(appkey string, io IO, pattern string, style font.Style, weight font.Weight) (
	fontfind.ScalableFont, error) {
	//
	if io == nil {
		io = &systemIO{}
	}
	variants, _ := findFontConfigFont(appkey, io, pattern, style, weight)
	if variants.Family != "" {
		if fsys, path, err := wrapDirFS(variants.Path); err == nil {
			sfnt := fontfind.ScalableFont{
				Name:   pattern,
				Weight: weight,
				Style:  style,
			}
			sfnt.SetFS(fsys, path)
			return sfnt, nil
		}
		return fontfind.NullFont, errors.New("path error with fontconfig file path")
	}
	if loadedFontConfigListOK { // fontconfig is active, but didn't find a font
		// therefore don't do a file system scan
		return fontfind.NullFont, errors.New("no such font")
	}
	// otherwise fontconfig is not active => scan file system
	fpath, err := findfont.Find(pattern) // go-findfont lib does not accept style & weight
	if err == nil && fpath != "" {
		tracer().Debugf("%s is a system font: %s", pattern, fpath)
		if fsys, path, err := wrapDirFS(fpath); err == nil {
			sfnt := fontfind.ScalableFont{
				Name:   pattern,
				Weight: weight,
				Style:  style,
			}
			sfnt.SetFS(fsys, path)
			return sfnt, nil
		}
		return fontfind.NullFont, errors.New("path error with system font file path")
	}
	return fontfind.NullFont, errors.New("no such font")
}

func wrapDirFS(fontpath string) (fs.FS, string, error) {
	d, f := filepath.Split(fontpath)
	return os.DirFS(d), f, nil
}
