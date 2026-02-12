package locate_test

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"testing"
	"testing/fstest"
	"time"

	"github.com/npillmayer/fontfind"
	"github.com/npillmayer/fontfind/locate"
	"github.com/npillmayer/fontfind/locate/fallbackfont"
	"github.com/npillmayer/fontfind/locate/googlefont"
	"github.com/npillmayer/fontfind/locate/systemfont"
	"github.com/npillmayer/schuko/schukonf/testconfig"
	"github.com/npillmayer/schuko/tracing/gotestingadapter"
	"golang.org/x/image/font"
)

func TestLoadPackagedFont(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "fontfind")
	defer teardown()
	//
	desc := fontfind.Descriptor{
		Pattern: "Go",
		Style:   font.StyleNormal,
		Weight:  font.WeightNormal,
	}
	fallback := fallbackfont.Find()
	loader := locate.ResolveFontLoc(desc, fallback)
	//time.Sleep(500)
	_, err := loader.Font()
	if err != nil {
		t.Error(err)
	}
}

func TestResolveGoogleFont(t *testing.T) {
	if os.Getenv("GOOGLE_FONTS_API_KEY") == "" {
		t.Skip("requires GOOGLE_FONTS_API_KEY")
	}
	teardown := gotestingadapter.QuickConfig(t, "resources")
	defer teardown()
	//
	conf := testconfig.Conf{
		"app-key": "tyse-test",
	}
	desc := fontfind.Descriptor{
		Pattern: "Antic",
		Style:   font.StyleNormal,
		Weight:  font.WeightNormal,
	}
	google := googlefont.Find(conf)
	loader := locate.ResolveFontLoc(desc, google)
	_, err := loader.Font()
	if err != nil {
		t.Error(err)
	}
}

var fclist = `
/System/Library/Fonts/Supplemental/NotoSansGothic-Regular.ttf: Noto Sans Gothic:style=Regular
/System/Library/Fonts/NotoSerifMyanmar.ttc: Noto Serif Myanmar,Noto Serif Myanmar Light:style=Light,Regular
/System/Library/Fonts/Supplemental/NotoSansCarian-Regular.ttf: Noto Sans Carian:style=Regular
/System/Library/Fonts/NotoSansMyanmar.ttc: Noto Sans Zawgyi:style=Regular
/System/Library/Fonts/Supplemental/NotoSansSylotiNagri-Regular.ttf: Noto Sans Syloti Nagri:style=Regular
/System/Library/Fonts/NotoNastaliq.ttc: Noto Nastaliq Urdu:style=Bold
/System/Library/Fonts/Supplemental/NotoSansCham-Regular.ttf: Noto Sans Cham:style=Regular
/System/Library/Fonts/NotoSansArmenian.ttc: Noto Sans Armenian:style=Bold
`

func TestFCFind(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "tyse.font")
	defer teardown()
	desc := fontfind.Descriptor{
		Pattern: "Noto Sans Cham",
		Style:   font.StyleNormal,
		Weight:  font.WeightNormal,
	}
	system := systemfont.Find("tyse-test", newIO())
	loader := locate.ResolveFontLoc(desc, system)
	f, err := loader.Font()
	if err != nil {
		t.Fatalf("expected fixture-based systemfont hit, got error: %v", err)
	}
	if f.Path() != "NotoSansCham-Regular.ttf" {
		t.Fatalf("expected path NotoSansCham-Regular.ttf, got %q", f.Path())
	}
}

func TestResolveTypefaceUsesRegistryCache(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "resources")
	defer teardown()

	desc := fontfind.Descriptor{
		Pattern: "zz-resolve-cache-probe",
		Style:   font.StyleNormal,
		Weight:  font.WeightNormal,
	}
	callCount := 0
	testFS := fstest.MapFS{
		"probe.ttf": &fstest.MapFile{
			Data: []byte("dummy"),
		},
	}
	resolver := func(d fontfind.Descriptor) (fontfind.ScalableFont, error) {
		callCount++
		sfnt := fontfind.ScalableFont{
			Name:   "probe.ttf",
			Style:  d.Style,
			Weight: d.Weight,
		}
		sfnt.SetFS(testFS, "probe.ttf")
		return sfnt, nil
	}

	f, err := locate.ResolveFontLoc(desc, resolver).Font()
	if err != nil {
		t.Fatalf("expected resolver success, got error: %v", err)
	}
	if f.Path() != "probe.ttf" {
		t.Fatalf("unexpected resolved path: %q", f.Path())
	}
	f, err = locate.ResolveFontLoc(desc, resolver).Font()
	if err != nil {
		t.Fatalf("expected cached resolver success, got error: %v", err)
	}
	if f.Path() != "probe.ttf" {
		t.Fatalf("unexpected cached path: %q", f.Path())
	}
	if callCount != 1 {
		t.Fatalf("expected resolver to be called once due to registry cache, got %d", callCount)
	}
}

func TestResolveTypefaceReturnsFallbackOnMiss(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "resources")
	defer teardown()

	desc := fontfind.Descriptor{
		Pattern: "zz-no-such-font",
		Style:   font.StyleItalic,
		Weight:  font.WeightBold,
	}
	f, err := locate.ResolveFontLoc(desc).Font()
	if err == nil {
		t.Fatalf("expected lookup error for missing font")
	}
	if f.Name != "Go-Regular.otf" {
		t.Fatalf("expected fallback Go-Regular.otf, got %q", f.Name)
	}
}

func TestResolveTypefaceContextCanceledBeforeStart(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "resources")
	defer teardown()
	//
	desc := fontfind.Descriptor{
		Pattern: "zz-canceled-before-start",
		Style:   font.StyleNormal,
		Weight:  font.WeightNormal,
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	f, err := locate.ResolveFontLocWithContext(ctx, desc).Font()
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if f != fontfind.NullFont {
		t.Fatalf("expected null font on canceled request")
	}
}

func TestResolveTypefaceContextDeadlineExceeded(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "resources")
	defer teardown()

	desc := fontfind.Descriptor{
		Pattern: "zz-canceled-during-resolver",
		Style:   font.StyleNormal,
		Weight:  font.WeightNormal,
	}
	blocking := func(ctx context.Context, _ fontfind.Descriptor) (fontfind.ScalableFont, error) {
		select {
		case <-ctx.Done():
			return fontfind.NullFont, ctx.Err()
		case <-time.After(100 * time.Millisecond):
			return fontfind.NullFont, errors.New("unexpected resolver completion")
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	f, err := locate.ResolveFontLocWithContext(ctx, desc, blocking).FontWithContext(ctx)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context deadline exceeded, got %v", err)
	}
	if f != fontfind.NullFont {
		t.Fatalf("expected null font on deadline exceeded")
	}
}

// --- Test IO (+ file system) ------------------------------------------

type testIO struct {
	fsys fs.FS
}

func newIO() *testIO {
	testFS := fstest.MapFS{
		"tyse-test":               &fstest.MapFile{Mode: fs.ModeDir},
		"fontconfig":              &fstest.MapFile{Mode: fs.ModeDir},
		"fontconfig/fontlist.txt": &fstest.MapFile{Data: []byte(fclist)},
	}
	return &testIO{
		fsys: testFS,
	}
}

func (s *testIO) UserConfigDir() (string, error) {
	return "home", nil
}
func (s *testIO) DirFS(path string) fs.FS {
	return s.fsys
}

func (s *testIO) ReadAll(r io.Reader) ([]byte, error) {
	return []byte(fclist), nil
}
