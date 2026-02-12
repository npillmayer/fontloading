package systemfont

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"strings"
	"sync"

	"github.com/npillmayer/fontfind"
	"golang.org/x/image/font"
)

// findFontListConfig will create a sub-filesystem for the user's configuration directory,
// suffixed with "<appkey>/fontconfig".
func findFontListConfigDir(appkey string, io IO) (fs.FS, error) {
	if appkey == "" {
		return nil, errors.New("missing app-key for font list config search")
	}
	uconfdir, err := io.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("cannot open user configuration directory: %w", err)
	}
	fontListConfigDir := path.Join(uconfdir, appkey)
	tracer().Debugf("fontListConfigDir base = %v", fontListConfigDir)
	return fs.Sub(io.DirFS(fontListConfigDir), "fontconfig")
}

func findFontList(appkey string, io IO) (list []byte, err error) {
	const listfile = "fontlist.txt"
	var configFS fs.FS
	configFS, err = findFontListConfigDir(appkey, io)
	if err != nil {
		return nil, err
	}
	return readFile(configFS, listfile, io)
}

func readFile(fsys fs.FS, name string, io IO) ([]byte, error) {
	if readFS, ok := fsys.(fs.ReadFileFS); ok {
		// Fast file reading within the sandboxed config area.
		return readFS.ReadFile(name)
	}
	// else do it the traditional way
	file, err := fsys.Open(name)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return io.ReadAll(file)
}

var noFonts = []fontfind.FontVariantsLocation{}

// loadFontConfigList searches the user's configuration directory for a font list file,
// then reads the file and parses it into a list of font variants.
// This list of font variants is then stored globally.
func loadFontConfigList(appkey string, io IO) ([]fontfind.FontVariantsLocation, bool) {
	fclist, err := findFontList(appkey, io)
	if err != nil {
		return noFonts, false
	}
	r := bytes.NewReader(fclist)
	scanner := bufio.NewScanner(r)
	ttc := 0
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		fields := strings.Split(line, ":")
		if len(fields) < 3 {
			continue
		}
		fontpath := strings.TrimSpace(fields[0])
		fontname := strings.TrimSpace(fields[1])
		fontname = strings.TrimPrefix(fontname, ".")
		fontvari := strings.ToLower(fields[2])
		if strings.HasSuffix(fontpath, ".ttc") {
			ttc++
			continue
		}
		desc := fontfind.FontVariantsLocation{
			Family: fontname,
			Path:   fontpath,
		}
		if strings.Contains(fontvari, "regular") {
			desc.Variants = []string{"regular"}
		} else if strings.Contains(fontvari, "text") {
			desc.Variants = []string{"regular"}
		} else if strings.Contains(fontvari, "light") {
			desc.Variants = []string{"light"}
		} else if strings.Contains(fontvari, "italic") {
			desc.Variants = []string{"italic"}
		} else if strings.Contains(fontvari, "bold") {
			desc.Variants = []string{"bold"}
		} else if strings.Contains(fontvari, "black") {
			desc.Variants = []string{"bold"}
		}
		fontConfigDescriptors = append(fontConfigDescriptors, desc)
	}
	if err = scanner.Err(); err != nil {
		err = fmt.Errorf("encountered a problem during reading of fontconfig font list: %s", fclist)
		return fontConfigDescriptors, false
	}
	if ttc > 0 {
		tracer().Infof("skipping %d platform fonts: TTC not yet supported", ttc)
	}
	return fontConfigDescriptors, true
}

var loadFontConfigListTask sync.Once
var loadedFontConfigListOK bool
var fontConfigDescriptors []fontfind.FontVariantsLocation

// findFontConfigFont searches for a locally installed font variant using the fontconfig
// system (https://www.freedesktop.org/wiki/Software/fontconfig/).
// However, we need some preparation from the user to de-couple from the
// fontconfig library.
func findFontConfigFont(appkey string, io IO, pattern string, style font.Style, weight font.Weight) (
	desc fontfind.FontVariantsLocation, variant string) {
	//
	loadFontConfigListTask.Do(func() {
		_, loadedFontConfigListOK = loadFontConfigList(appkey, io)
		tracer().Infof("loaded fontconfig list")
	})
	if !loadedFontConfigListOK {
		return
	}
	var confidence fontfind.MatchConfidence
	desc, variant, confidence = fontfind.ClosestMatch(fontConfigDescriptors, pattern, style, weight)
	tracer().Debugf("closest fontconfig match confidence for %s|%s= %d", desc.Family, variant, confidence)
	if confidence > fontfind.LowConfidence {
		return
	}
	return fontfind.FontVariantsLocation{}, ""
}
