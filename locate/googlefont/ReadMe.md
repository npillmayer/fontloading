# googlefont

## Purpose

`googlefont` resolves fonts via the Google Fonts directory API and caches downloaded files locally.

## API

- `type IO` (env/http/fs abstraction)
- `Find(conf, io) locate.FontLocator`
- `FindGoogleFont(conf, pattern, style, weight) (fontfind.ScalableFont, error)`
- `ListGoogleFonts(conf, pattern)`
- `SimpleConfig(appkey) schuko.Configuration`

Configuration note:

Live API usage requires a Google web-fonts API key, either
  - under key `google-fonts-api-key` in configuration `conf`, or
  - `GOOGLE_FONTS_API_KEY` set to a valid API key

## Example: Resolve and cache a Google font

Clients must provide an application shortname. This shortname is used to
find/create the cache directory for downloaded font files.
For `appkey` equal to *myapp*, cached fonts will be located in `os.UserCacheDir()`/*myapp*.
(See package os: 
[os.UserCacheDir](https://pkg.go.dev/os#UserCacheDir))

```go
conf := googlefont.SimpleConfig("myapp") // provide your application shortname
resolver := googlefont.Find(conf, nil)
sf, err := locate.ResolveFontLoc(desc, resolver).Font()
```
