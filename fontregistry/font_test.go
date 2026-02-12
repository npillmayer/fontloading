package fontregistry

import (
	"testing"

	"github.com/npillmayer/fontfind"
	"github.com/npillmayer/schuko/tracing/gotestingadapter"
	"golang.org/x/image/font"
)

type sw struct {
	s font.Style
	w font.Weight
}

func TestGuess(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "resources")
	defer teardown()
	//
	for k, v := range map[string]sw{
		"fonts/Clarendon-bold.ttf":               {font.StyleNormal, font.WeightBold},
		"Microsoft/Gill Sans MT Bold Italic.ttf": {font.StyleItalic, font.WeightBold},
		"Cambria Math.ttf":                       {font.StyleNormal, font.WeightNormal},
	} {
		style, weight := fontfind.GuessStyleAndWeight(k)
		t.Logf("style = %d, weight = %d", style, weight)
		if style != v.s || weight != v.w {
			t.Errorf("expected different style or weight for %s", k)
		}
	}
}

func TestMatch(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "resources")
	defer teardown()
	//
	if !fontfind.Matches("fonts/Clarendon-bold.ttf",
		"clarendon", font.StyleNormal, font.WeightBold) {
		t.Errorf("expected match for Clarendon, haven't")
	}
	if !fontfind.Matches("Microsoft/Gill Sans MT Bold Italic.ttf",
		"gill sans", font.StyleItalic, font.WeightBold) {
		t.Errorf("expected match for Gill, haven't")
	}
	if !fontfind.Matches("Cambria Math.ttf",
		"cambria", font.StyleNormal, font.WeightNormal) {
		t.Errorf("expected match for Cambria Math, haven't")
	}
}

func TestNormalizeFont(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "resources")
	defer teardown()
	//
	n := NormalizeFontname("Clarendon", font.StyleItalic, font.WeightBold)
	if n != "clarendon-italic-bold" {
		t.Errorf("expected different normalized name for clarendon")
	}
}

func TestRegistryFallbackTypeface(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "resources")
	defer teardown()
	//
	fr := NewRegistry()
	f, err := fr.FallbackTypeface()
	if err != nil {
		t.Fatal(err)
	}
	if f.FileSystem == nil {
		t.Fatal("expected fallback font filesystem to be set")
	}
	if f.Path == "" {
		t.Fatal("expected fallback font path to be set")
	}
	if f.Name != "Go-Regular.otf" {
		t.Fatalf("expected fallback font Go-Regular.otf, got %s", f.Name)
	}
}

func TestRegistryTypefaceReturnsFallbackOnMiss(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "resources")
	defer teardown()
	//
	fr := NewRegistry()
	f, err := fr.Typeface("font-not-in-registry")
	if err == nil {
		t.Fatal("expected miss error from registry lookup")
	}
	if f.FileSystem == nil {
		t.Fatal("expected fallback font filesystem to be set")
	}
	if f.Name != "Go-Regular.otf" {
		t.Fatalf("expected fallback font Go-Regular.otf, got %s", f.Name)
	}
}
