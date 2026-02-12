package googlefont

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strings"
	"sync"

	"github.com/npillmayer/fontfind"
	"github.com/npillmayer/schuko"
	"github.com/npillmayer/schuko/tracing"
	font "golang.org/x/image/font"
)

// GoogleFontInfo describes a font entry in the Google Font Service.
type GoogleFontInfo struct {
	fontfind.FontVariantsLocation
	Version string            `json:"version"`
	Subsets []string          `json:"subsets"`
	Files   map[string]string `json:"files"`
}

type googleFontsList struct {
	Items []GoogleFontInfo `json:"items"`
}

const defaultGoogleFontsAPI = `https://www.googleapis.com/webfonts/v1/webfonts?`

type googleService struct {
	io IO

	api string

	loadGoogleFontsDir sync.Once
	googleFontsDir     googleFontsList
	googleFontsLoadErr error
}

func newGoogleService(hostio IO) *googleService {
	if hostio == nil {
		hostio = systemIO{}
	}
	return &googleService{
		io:  hostio,
		api: defaultGoogleFontsAPI,
	}
}

var defaultGoogleService = newGoogleService(nil)

func setupGoogleFontsDirectory(conf schuko.Configuration) error {
	return defaultGoogleService.setupGoogleFontsDirectory(conf)
}

func (svc *googleService) setupGoogleFontsDirectory(conf schuko.Configuration) (err error) {
	svc.loadGoogleFontsDir.Do(func() {
		tracer().Infof("setting up Google Fonts service directory")
		apikey := conf.GetString("google-fonts-api-key")
		if apikey == "" {
			if apikey = svc.io.Getenv("GOOGLE_FONTS_API_KEY"); apikey == "" {
				tracer().Errorf("Google fonts API key not set")
				svc.googleFontsLoadErr = fmt.Errorf(`Google Fonts API-key must be set in global configuration or as GOOGLE_FONTS_API_KEY in environment;
      please refer to https://developers.google.com/fonts/docs/developer_api`)
				return
			}
		}
		values := url.Values{
			"sort": []string{"alpha"},
			"key":  []string{apikey},
		}
		resp, getErr := svc.io.HTTPGet(svc.api + values.Encode())
		if getErr != nil || resp == nil {
			tracer().Errorf("Google Fonts API request not OK, error = %v", getErr)
			svc.googleFontsLoadErr = fmt.Errorf("could not get fonts-directory from Google font service")
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			tracer().Errorf("Google Fonts API request not OK, status = %d", resp.StatusCode)
			svc.googleFontsLoadErr = fmt.Errorf("could not get fonts-directory from Google font service")
			return
		}
		var list googleFontsList
		dec := json.NewDecoder(resp.Body)
		if decErr := dec.Decode(&list); decErr != nil {
			svc.googleFontsLoadErr = fmt.Errorf("could not decode fonts-list from Google font service")
			return
		}
		svc.googleFontsDir = list
		tracer().Infof("transfered list of %d fonts from Google Fonts service",
			len(svc.googleFontsDir.Items))
	})
	return svc.googleFontsLoadErr
}

func FindGoogleFont(conf schuko.Configuration, pattern string, style font.Style, weight font.Weight) (
	fontfind.ScalableFont, error) {
	return defaultGoogleService.findGoogleFont(conf, pattern, style, weight)
}

func (svc *googleService) findGoogleFont(conf schuko.Configuration, pattern string, style font.Style, weight font.Weight) (
	fontfind.ScalableFont, error) {
	//
	fiList, err := svc.matchGoogleFontInfo(conf, pattern, style, weight)
	if err != nil {
		return fontfind.NullFont, err
	}
	if len(fiList) == 0 {
		return fontfind.NullFont, fmt.Errorf("no matching Google font found")
	}
	fi := fiList[0]
	variant, confidence := selectVariant(fi.Variants, style, weight)
	if confidence < fontfind.LowConfidence {
		return fontfind.NullFont, fmt.Errorf("no suitable variant for %s (confidence=%d)", fi.Family, confidence)
	}
	cachedir, name, err := svc.cacheGoogleFont(conf, fi, variant)
	if err != nil {
		return fontfind.NullFont, err
	}
	fsys := svc.io.DirFS(cachedir)
	sfnt := fontfind.ScalableFont{
		Name:   name,
		Style:  style,
		Weight: weight,
	}
	sfnt.SetFS(fsys, name)
	return sfnt, nil
}

func selectVariant(variants []string, style font.Style, weight font.Weight) (variant string, confidence fontfind.MatchConfidence) {
	for _, v := range variants {
		s := fontfind.MatchStyle(v, style)
		w := fontfind.MatchWeight(v, weight)
		c := (s + w) / 2
		if c > confidence {
			confidence = c
			variant = v
		}
	}
	return
}

// FindGoogleFont scans the Google Font Service for fonts matching `pattern` and
// having a given style and weight.
//
// Will include all fonts with a match-confidence greater than `font.LowConfidence`.
//
// A prerequisite to looking for Google fonts is a valid API-key (refer to
// https://developers.google.com/fonts/docs/developer_api). It has to be configured
// either in the application setup or as an environment variable GOOGLE_FONTS_API_KEY.
func matchGoogleFontInfo(conf schuko.Configuration, pattern string, style font.Style, weight font.Weight) (
	[]GoogleFontInfo, error) {
	return defaultGoogleService.matchGoogleFontInfo(conf, pattern, style, weight)
}

func (svc *googleService) matchGoogleFontInfo(conf schuko.Configuration, pattern string, style font.Style, weight font.Weight) (
	[]GoogleFontInfo, error) {
	//
	var fiList []GoogleFontInfo
	if err := svc.setupGoogleFontsDirectory(conf); err != nil {
		return fiList, err
	}
	r, err := regexp.Compile(strings.ToLower(pattern))
	if err != nil {
		return fiList, fmt.Errorf("cannot match Google font: invalid font name pattern: %v", err)
	}
	tracer().Debugf("trying to match (%s)", strings.ToLower(pattern))
	for _, finfo := range svc.googleFontsDir.Items {
		if r.MatchString(strings.ToLower(finfo.Family)) {
			tracer().Debugf("Google font name matches pattern: %s", finfo.Family)
			_, _, confidence := fontfind.ClosestMatch([]fontfind.FontVariantsLocation{finfo.FontVariantsLocation}, pattern,
				style, weight)
			if confidence > fontfind.LowConfidence {
				fiList = append(fiList, finfo)
				break
			}
		}
	}
	if len(fiList) == 0 {
		return fiList, errors.New("no Google font matches pattern")
	}
	tracer().Debugf("found Google font: %v", fiList[0])
	return fiList, nil
}

// ---------------------------------------------------------------------------

// cacheGoogleFont loads a font described by fi with a given variant.
// The loaded font is cached in the user's cache directory.
func (svc *googleService) cacheGoogleFont(conf schuko.Configuration, fi GoogleFontInfo, variant string) (
	cachedir, name string, err error) {
	//
	var fileurl string
	for _, v := range fi.Variants {
		if v == variant {
			fileurl = fi.Files[v]
		}
	}
	if fileurl == "" {
		return "", "", fmt.Errorf("no variant equals %s, cannot cache %s", variant, fi.Family)
	}
	letter := strings.ToUpper(fi.Family[:1])
	cachedir, err = cacheFontDirPath(svc.io, conf, letter)
	if err != nil {
		return "", "", err
	}
	ext := path.Ext(fileurl)
	name = fi.Family + "-" + variant + ext
	filepath := path.Join(cachedir, name)
	tracer().Infof("caching font %s as %s", fi.Family, filepath)
	if _, err := svc.io.Stat(filepath); err == nil {
		tracer().Infof("font already cached: %s", filepath)
	} else {
		err = downloadCachedFile(svc.io, filepath, fileurl)
	}
	return
}

// ---------------------------------------------------------------------------

// ListGoogleFonts produces a listing of available fonts from the Google webfont
// service, with font-family names matching a given pattern.
// Output goes into the trace file with log-level info.
//
// If not aleady done, the list of available fonts will be downloaded from Google.
func ListGoogleFonts(conf schuko.Configuration, pattern string) {
	defaultGoogleService.listGoogleFonts(conf, pattern)
}

func (svc *googleService) listGoogleFonts(conf schuko.Configuration, pattern string) {
	level := tracer().GetTraceLevel()
	tracer().SetTraceLevel(tracing.LevelInfo)
	if err := svc.setupGoogleFontsDirectory(conf); err != nil {
		tracer().Errorf("unable to list Google fonts: %v", err)
	} else {
		listGoogleFonts(svc.googleFontsDir, pattern)
	}
	tracer().SetTraceLevel(level)
}

func listGoogleFonts(list googleFontsList, pattern string) {
	r, err := regexp.Compile(pattern)
	if err != nil {
		tracer().Errorf("cannot list Google fonts: invalid pattern: %v", err)
	}
	tracer().Infof("%d fonts in Google font list", len(list.Items))
	tracer().Infof("======================================")
	for i, finfo := range list.Items {
		if r.MatchString(finfo.Family) {
			tracer().Infof("[%4d] %-20s: %s", i, finfo.Family, finfo.Version)
			tracer().Infof("       subsets: %v", finfo.Subsets)
			for k, v := range finfo.Files {
				tracer().Infof("       - %-18s: %s", k, v[len(v)-4:])
			}
		}
	}
}
