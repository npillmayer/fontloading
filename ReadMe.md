# fontfind

`fontfind` is a Go package for discovering and loading scalable fonts from multiple sources.

The package is built around one practical goal: given a font request (`family pattern`, `style`, `weight`), return a usable font resource that can be loaded as bytes. Resolution is provider-based and can combine:

- embedded fallback fonts
- local/system fonts
- Google Fonts (with local caching)

The resolver pipeline is cache-backed (`fontregistry`) and designed so callers can still receive a fallback font object even when the requested font cannot be found.

## Installation

```bash
go get github.com/npillmayer/fontfind
```

## API Overview

### Core types (`package fontfind`)

- `Descriptor`: describes a requested font (`Pattern`, `Style`, `Weight`)
- `ScalableFont`: describes a resolved font variant and where to load it from
- `NullFont`: zero-value marker used for unresolved results
- `FallbackFont()`: returns packaged default fallback (`Go-Regular.otf`)

`ScalableFont` methods:

- `SetFS(fs fs.FS, path string)`
- `Path() string`
- `ReadFontData() ([]byte, error)`

### Resolution API (`package locate`)

- `ResolveFontLoc(desc, resolvers...) FontPromise`
- `ResolveFontLocWithContext(ctx, desc, resolvers...) FontPromise`

`FontPromise`:

- `Font() (fontfind.ScalableFont, error)`
- `FontWithContext(ctx) (fontfind.ScalableFont, error)`

Resolution behavior:

1. Normalize descriptor to registry key.
2. Try registry cache (`fontregistry.GlobalRegistry().GetFont`).
3. Run resolvers in provided order on cache miss.
4. Cache successful hits.
5. Return registry fallback font with an error if all resolvers fail.

### Resolver providers

- `locate/fallbackfont`: embedded packaged fonts (`Find`, `Default`)
- `locate/systemfont`: local/system lookup (`Find`, `FindLocalFont`)
- `locate/googlefont`: Google Fonts lookup + cache (`Find`, `FindWithIO`, `FindGoogleFont`)

See the documentation in the sub-packages for more details.

## Example Applications

### 1. General app font resolution

This example will search for: system fonts -> embedded fallback fonts, i.e.
[Go fonts](https://go.dev/blog/go-fonts).

```go
desc := fontfind.Descriptor{
	Pattern: "Noto Sans",
	Style:   font.StyleNormal,
	Weight:  font.WeightNormal,
}

system := systemfont.Find("myapp", USE_SYSTEM_IO)
fallback := fallbackfont.Find()

promise := locate.ResolveFontLoc(desc, system, google, fallback)
sf, err := promise.Font()
if err != nil {
	// err may be non-nil while sf is still a usable fallback font
	log.Printf("font lookup degraded: %v", err)
}

data, err := sf.ReadFontData()
if err != nil {
	log.Fatal(err)
}
fmt.Printf("resolved %s (%s), %d bytes\n", sf.Name, sf.Path(), len(data))
```

### 2. Timeout-aware async resolution

Use a context-aware resolver and wait with cancellation/deadline control.

```go
desc := fontfind.Descriptor{
	Pattern: "Any",
	Style:   font.StyleNormal,
	Weight:  font.WeightNormal,
}

myLongRunningResolver := …   // client-provided resolver

ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
defer cancel()

promise := locate.ResolveFontLocWithContext(ctx, desc, myLongRunningResolver)
_, err := promise.FontWithContext(ctx)
fmt.Println("result:", err) // context deadline exceeded
```

### 3. Embedded-only/offline deployments

For fully offline systems, use only the fallback resolver:

```go
desc := fontfind.Descriptor{
	Pattern: "Go",
	Style: font.StyleNormal,
	Weight: font.WeightNormal
}
promise := locate.ResolveFontLoc(desc, fallbackfont.Find())
sf, err := promise.Font()
```

This will return a packaged
[Go font](https://go.dev/blog/go-fonts).

## Notes

- Google Fonts access requires a valid Google API key (`GOOGLE_FONTS_API_KEY`) for live directory fetches.
- TTC (`*.ttc`) handling is not yet implemented.

## License

BSD 3-Clause. See `LICENSE` file in the top-level directory.
