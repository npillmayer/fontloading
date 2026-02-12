# fallbackfont

## Purpose

`fallbackfont` resolves fonts from embedded package assets. As we are in a Go eco-sytem,
these are the
[Go fonts](https://go.dev/blog/go-fonts),
packaged and embedded (in OTF format).

It is also the deterministic last-resort provider and contains the default packaged fallback
(`Go-Regular.otf`).

## API

- `Find() locate.FontLocator`
- `Default() (fontfind.ScalableFont, error)`
- `FindFallbackFont(pattern, style, weight) (fontfind.ScalableFont, error)`

## Example Applications

### 1. Use as final resolver in a chain

```go
fallbackSearcher := fallbackfont.Find()
sf, err := locate.ResolveFontLoc(desc, system, google, fallbackSearcher).Font()
```

### 2. Access default fallback directly

```go
sf, err := fallbackfont.Default()
data, err := sf.ReadFontData()
```
