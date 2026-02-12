package locate

import (
	"context"
	"fmt"

	"github.com/npillmayer/fontfind"
	"github.com/npillmayer/fontfind/fontregistry"
)

// notFound returns an application error for a missing resource.
func notFound(res string) error {
	return fmt.Errorf("font not found: %v", res)
}

type fontPlusErr struct {
	font fontfind.ScalableFont
	err  error
}

// TypefacePromise runs font searching asynchronously in the background.
// A call to `Typeface()` blocks until font loading is completed, or -- in
// the case of a context cancellation -- returns an error.
type TypefacePromise interface {
	Typeface() (fontfind.ScalableFont, error)
	TypefaceContext(ctx context.Context) (fontfind.ScalableFont, error)
}

type fontLoader struct {
	await func(ctx context.Context) (fontfind.ScalableFont, error)
}

func (loader fontLoader) Typeface() (fontfind.ScalableFont, error) {
	return loader.TypefaceContext(context.Background())
}

func (loader fontLoader) TypefaceContext(ctx context.Context) (fontfind.ScalableFont, error) {
	return loader.await(ctx)
}

// ResolveTypeface resolves a typefacee with given properties.
// It searches for fonts in the following order:
//
// ▪︎ Fonts packaged with the application binary
//
// ▪︎ System-fonts
//
// ▪︎ Google Fonts service (https://fonts.google.com/)
//
// ResolveTypeface will try to match style and weight requirements closely, but
// will load a font variant anyway if it matches approximately. If, for example,
// a system contains a font with weight 300, which would be considered a "light"
// variant, but no variant with weight 400 (normal), it will load the 300-variant.
//
// When looking for sytem-fonts, ResolveTypeface will use an existing fontconfig
// (https://www.freedesktop.org/wiki/Software/fontconfig/)
// installation, if present. fontconfig has to be configured in the global
// application setup by pointing to the absolute path of the `fc-list` binary.
// If fontconfig isn't installed or configured, then this step will silently be
// skipped and a file system scan of the sytem's fonts-folders will be done.
// (See also function `FindLocalFont`).
//
// A prerequisite to looking for Google fonts is a valid API-key (refer to
// https://developers.google.com/fonts/docs/developer_api). It has to be configured
// either in the application setup or as an environment variable GOOGLE_FONTS_API_KEY.
// (See also function `FindGoogleFont`).
//
// If no suitable font can be found, an application-wide fallback font will be
// returned.
//
// Typefaces are not returned synchronously, but rather as a promise
// of kind TypefacePromise (async/await).
func ResolveTypeface(desc fontfind.Descriptor, resolvers ...FontLocator) TypefacePromise {
	ctxResolvers := make([]FontLocatorWithContext, 0, len(resolvers))
	for _, r := range resolvers {
		ctxResolvers = append(ctxResolvers, adaptLocator(r))
	}
	return ResolveTypefaceContext(context.Background(), desc, ctxResolvers...)
}

// ResolveTypefaceContext resolves a typeface with context-aware cancellation.
func ResolveTypefaceContext(ctx context.Context, desc fontfind.Descriptor, resolvers ...FontLocatorWithContext) TypefacePromise {
	if ctx == nil {
		ctx = context.Background()
	}
	ch := make(chan fontPlusErr)
	go func(ch chan<- fontPlusErr) {
		result := searchScalableFont(ctx, desc, resolvers)
		ch <- result
		close(ch)
	}(ch)
	loader := fontLoader{}
	// `waitCtx` will be set by the caller using ResolveTypefaceContext(myCtx)
	loader.await = func(waitCtx context.Context) (fontfind.ScalableFont, error) {
		select {
		case <-waitCtx.Done():
			return fontfind.NullFont, waitCtx.Err()
		case r := <-ch:
			return r.font, r.err
		}
	}
	return loader
}

func adaptLocator(r FontLocator) FontLocatorWithContext {
	return func(_ context.Context, d fontfind.Descriptor) (fontfind.ScalableFont, error) {
		return r(d)
	}
}

func searchScalableFont(ctx context.Context, desc fontfind.Descriptor, resolvers []FontLocatorWithContext) (result fontPlusErr) {
	if err := ctx.Err(); err != nil {
		result.err = err
		return
	}
	name := fontregistry.NormalizeFontname(desc.Pattern, desc.Style, desc.Weight)
	if t, err := fontregistry.GlobalRegistry().Typeface(name); err == nil {
		result.font = t
		return
	}
	for _, resolver := range resolvers {
		if err := ctx.Err(); err != nil {
			result.err = err
			return
		}
		if f, err := resolver(ctx, desc); err == nil {
			fontregistry.GlobalRegistry().StoreTypeface(name, f)
			result.font = f
			return
		} else if ctxErr := ctx.Err(); ctxErr != nil {
			result.err = ctxErr
			return
		}
	}
	result.err = notFound(name)
	if f, err := fontregistry.GlobalRegistry().FallbackTypeface(); err == nil {
		result.font = f
	}
	return result
}
