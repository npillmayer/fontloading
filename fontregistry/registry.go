package fontregistry

import (
	"fmt"
	"strings"
	"sync"

	"github.com/npillmayer/fontfind"
	"github.com/npillmayer/schuko/tracing"
	xfont "golang.org/x/image/font"
)

// Registry is a type for holding information about loaded fonts.
type Registry struct {
	sync.Mutex
	fonts map[string]fontfind.ScalableFont
}

var globalFontRegistry *Registry

var globalRegistryCreation sync.Once

// GlobalRegistry is an application-wide singleton to hold information about
// loaded fonts and typecases.
func GlobalRegistry() *Registry {
	globalRegistryCreation.Do(func() {
		globalFontRegistry = NewRegistry()
	})
	return globalFontRegistry
}

func NewRegistry() *Registry {
	fr := &Registry{
		fonts: make(map[string]fontfind.ScalableFont),
	}
	return fr
}

const fallbackFontKey = "fallback"

// StoreFont pushes a font into the registry if it isn't contained yet.
//
// The font will be stored using the normalized font name as a key. If this
// key is already associated with a font, that font will not be overridden.
func (fr *Registry) StoreFont(normalizedName string, f fontfind.ScalableFont) {
	if f.Name == "" {
		tracer().Errorf("registry cannot store null font")
		return
	}
	fr.Lock()
	defer fr.Unlock()
	//style, weight := GuessStyleAndWeight(f.Fontname)
	//fname := NormalizeFontname(f.Fontname, style, weight)
	if _, ok := fr.fonts[normalizedName]; !ok {
		tracer().Debugf("registry stores font %s as %s", f.Name, normalizedName)
		fr.fonts[normalizedName] = f
	}
}

// GetFont returns a font with a given font, style and weight.
// If a suitable font has already been cached, GetFont will return the cached
// scalable font.
//
// If no font can be produced, GetFont will derive one from a system-wide
// fallback font and return it, together with an error message.
func (fr *Registry) GetFont(normalizedName string) (fontfind.ScalableFont, error) {
	//
	tracer().Debugf("registry searches for font %s", normalizedName)
	fr.Lock()
	if t, ok := fr.fonts[normalizedName]; ok {
		fr.Unlock()
		tracer().Infof("registry found font %s", normalizedName)
		return t, nil
	}
	fr.Unlock()
	tracer().Infof("registry does not contain font %s", normalizedName)
	missErr := fmt.Errorf("font %s not found in registry", normalizedName)
	f, fallbackErr := fr.FallbackFont()
	if fallbackErr != nil {
		return fontfind.NullFont, fmt.Errorf("%w; fallback failed: %v", missErr, fallbackErr)
	}
	return f, missErr
}

// FallbackFont returns the default fallback font from registry cache.
// If absent, it will load and cache the packaged fallback under key "fallback".
func (fr *Registry) FallbackFont() (fontfind.ScalableFont, error) {
	fr.Lock()
	if t, ok := fr.fonts[fallbackFontKey]; ok {
		fr.Unlock()
		return t, nil
	}
	fr.Unlock()

	f := fontfind.FallbackFont()
	fr.Lock()
	defer fr.Unlock()
	// Another goroutine may have inserted fallback while we were loading.
	if t, ok := fr.fonts[fallbackFontKey]; ok {
		return t, nil
	}
	tracer().Infof("font registry caches fallback font %s", fallbackFontKey)
	fr.fonts[fallbackFontKey] = f
	return f, nil
}

// LogFontList is a helper function to dump the list of the font known to a
// registry to the tracer (log-level Info).
func (fr *Registry) LogFontList(tracer tracing.Trace) {
	level := tracer.GetTraceLevel()
	tracer.SetTraceLevel(tracing.LevelInfo)
	tracer.Infof("--- registered fonts ---")
	for k, v := range fr.fonts {
		tracer.Infof("typeface [%s] = %s @ %v", k, v.Name, v.Path)
	}
	tracer.Infof("------------------------")
	tracer.SetTraceLevel(level)
}

func NormalizeFontname(fname string, style xfont.Style, weight xfont.Weight) string {
	fname = strings.TrimSpace(fname)
	fname = strings.ReplaceAll(fname, " ", "_")
	if dot := strings.LastIndex(fname, "."); dot > 0 {
		fname = fname[:dot]
	}
	fname = strings.ToLower(fname)
	switch style {
	case xfont.StyleItalic, xfont.StyleOblique:
		fname += "-italic"
	}
	switch weight {
	case xfont.WeightLight, xfont.WeightExtraLight:
		fname += "-light"
	case xfont.WeightBold, xfont.WeightExtraBold, xfont.WeightSemiBold:
		fname += "-bold"
	}
	return fname
}
