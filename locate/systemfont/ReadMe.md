# systemfont

## Purpose

`systemfont` resolves fonts from local machine sources.

It prefers a fontconfig list (`fontlist.txt` under the app config area) and falls back to platform directory scanning. `fontlist.txt` is the output of fontconfig command `fc-list`. Place it into
`os.UserConfigDir()`/*myapp*/*fontlist.txt*, with «*myapp*» being the shortname of your application.

See package os: 
[os.UserConfigDir](https://pkg.go.dev/os#UserConfigDir)

## API

- `type IO` (injectable host I/O for tests)
- `Find(appkey, io) locate.FontLocator`
- `FindLocalFont(appkey, io, pattern, style, weight) (fontfind.ScalableFont, error)`

`appkey` determines where fontconfig list data is looked up.

## Example

```go
desc := fontfind.Descriptor{
	Pattern: "Noto Sans",
	Style:   font.StyleNormal,
	Weight:  font.WeightNormal,
}
systemSearcher := systemfont.Find("myapp", nil)
font, err := locate.ResolveFontLoc(desc, systemSearcher).Font()
```
