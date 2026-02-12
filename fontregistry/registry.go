package fontregistry

import (
	"fmt"
	"strings"
	"sync"

	"github.com/npillmayer/fontfind"
	"github.com/npillmayer/fontfind/locate/fallbackfont"
	"github.com/npillmayer/schuko/tracing"
	xfont "golang.org/x/image/font"
)

// Registry is a type for holding information about loaded fonts.
type Registry struct {
	sync.Mutex
	typefaces map[string]fontfind.ScalableFont
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
		typefaces: make(map[string]fontfind.ScalableFont),
	}
	return fr
}

const fallbackTypefaceKey = "fallback"

// StoreTypeface pushes a typeface into the registry if it isn't contained yet.
//
// The typeface will be stored using the normalized font name as a key. If this
// key is already associated with a font, that font will not be overridden.
func (fr *Registry) StoreTypeface(normalizedName string, f fontfind.ScalableFont) {
	if f.Name == "" || f.Path == "" {
		tracer().Errorf("registry cannot store null font")
		return
	}
	fr.Lock()
	defer fr.Unlock()
	//style, weight := GuessStyleAndWeight(f.Fontname)
	//fname := NormalizeFontname(f.Fontname, style, weight)
	if _, ok := fr.typefaces[normalizedName]; !ok {
		tracer().Debugf("registry stores font %s as %s", f.Name, normalizedName)
		fr.typefaces[normalizedName] = f
	}
}

// Typeface returns a typeface with a given font, style and weight.
// If a suitable typeface has already been cached, Typeface will return the cached
// typeface.
//
// If no typeface can be produced, Typeface will derive one from a system-wide
// fallback font and return it, together with an error message.
func (fr *Registry) Typeface(normalizedName string) (fontfind.ScalableFont, error) {
	//
	tracer().Debugf("registry searches for font %s", normalizedName)
	fr.Lock()
	if t, ok := fr.typefaces[normalizedName]; ok {
		fr.Unlock()
		tracer().Infof("registry found font %s", normalizedName)
		return t, nil
	}
	fr.Unlock()
	tracer().Infof("registry does not contain font %s", normalizedName)
	missErr := fmt.Errorf("font %s not found in registry", normalizedName)
	f, fallbackErr := fr.FallbackTypeface()
	if fallbackErr != nil {
		return fontfind.NullFont, fmt.Errorf("%w; fallback failed: %v", missErr, fallbackErr)
	}
	return f, missErr
}

// FallbackTypeface returns the default fallback typeface from registry cache.
// If absent, it will load and cache the packaged fallback under key "fallback".
func (fr *Registry) FallbackTypeface() (fontfind.ScalableFont, error) {
	fr.Lock()
	if t, ok := fr.typefaces[fallbackTypefaceKey]; ok {
		fr.Unlock()
		return t, nil
	}
	fr.Unlock()

	f, err := fallbackfont.Default()
	if err != nil {
		return fontfind.NullFont, err
	}
	fr.Lock()
	defer fr.Unlock()
	// Another goroutine may have inserted fallback while we were loading.
	if t, ok := fr.typefaces[fallbackTypefaceKey]; ok {
		return t, nil
	}
	tracer().Infof("font registry caches fallback font %s", fallbackTypefaceKey)
	fr.typefaces[fallbackTypefaceKey] = f
	return f, nil
}

// LogFontList is a helper function to dump the list of the typefaces konwn to a
// registry to the tracer (log-level Info).
func (fr *Registry) LogFontList(tracer tracing.Trace) {
	level := tracer.GetTraceLevel()
	tracer.SetTraceLevel(tracing.LevelInfo)
	tracer.Infof("--- registered fonts ---")
	for k, v := range fr.typefaces {
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

func appendSize(fname string, size float32) string {
	fname = fmt.Sprintf("%s-%.2f", fname, size)
	return fname
}
