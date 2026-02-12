package googlefont

import (
	"encoding/json"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/npillmayer/schuko/schukonf/testconfig"
	"golang.org/x/image/font"
)

type fakeIO struct {
	cacheDir string
	env      map[string]string

	webfontsJSON []byte
	fontBytes    []byte
	requestedURL []string
}

func newFakeIO(t *testing.T) *fakeIO {
	t.Helper()
	j, err := os.ReadFile(filepath.Join("testdata", "webfonts.json"))
	if err != nil {
		t.Fatalf("cannot read test fixture JSON: %v", err)
	}
	return &fakeIO{
		cacheDir:     t.TempDir(),
		env:          map[string]string{"GOOGLE_FONTS_API_KEY": "test-key"},
		webfontsJSON: j,
		fontBytes:    []byte("dummy-font-bytes"),
	}
}

func (f *fakeIO) Getenv(k string) string {
	return f.env[k]
}

func (f *fakeIO) HTTPGet(u string) (*http.Response, error) {
	f.requestedURL = append(f.requestedURL, u)
	if strings.HasPrefix(u, defaultGoogleFontsAPI) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Body:       io.NopCloser(strings.NewReader(string(f.webfontsJSON))),
			Header:     make(http.Header),
		}, nil
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Status:     "200 OK",
		Body:       io.NopCloser(strings.NewReader(string(f.fontBytes))),
		Header:     make(http.Header),
	}, nil
}

func (f *fakeIO) UserCacheDir() (string, error) {
	return f.cacheDir, nil
}

func (f *fakeIO) DirFS(path string) fs.FS {
	return os.DirFS(path)
}

func (f *fakeIO) Stat(path string) (os.FileInfo, error) {
	return os.Stat(path)
}

func (f *fakeIO) MkdirAll(path string, perm fs.FileMode) error {
	return os.MkdirAll(path, perm)
}

func (f *fakeIO) Create(path string) (io.WriteCloser, error) {
	return os.Create(path)
}

func TestGoogleRespDecode(t *testing.T) {
	hostio := newFakeIO(t)
	dec := json.NewDecoder(strings.NewReader(string(hostio.webfontsJSON)))
	var list googleFontsList
	err := dec.Decode(&list)
	if err != nil {
		t.Fatal(err)
	}
	listGoogleFonts(list, ".*")
}

func TestGoogleAPI(t *testing.T) {
	hostio := newFakeIO(t)
	svc := newGoogleService(hostio)
	conf := testconfig.Conf{
		"app-key": "tyse-test",
	}
	err := svc.setupGoogleFontsDirectory(conf)
	if err != nil {
		t.Fatal(err)
	}
	if len(hostio.requestedURL) != 1 {
		t.Fatalf("expected 1 API request, got %d", len(hostio.requestedURL))
	}
	url := hostio.requestedURL[0]
	if !strings.Contains(url, "key=test-key") {
		t.Fatalf("expected API key in request URL, got %q", url)
	}
	if !strings.Contains(url, "sort=alpha") {
		t.Fatalf("expected sort=alpha in request URL, got %q", url)
	}
}

func TestMatchFontname(t *testing.T) {
	pattern := "Inconsolata"
	r, err := regexp.Compile(strings.ToLower(pattern))
	if err != nil {
		t.Fatal(err)
	}
	if !r.MatchString(strings.ToLower("Inconsolata")) {
		t.Errorf("expected to find match, didn't")
	}
}

func TestGoogleFindFont(t *testing.T) {
	hostio := newFakeIO(t)
	svc := newGoogleService(hostio)
	conf := testconfig.Conf{
		"app-key": "tyse-test",
	}
	f, err := svc.findGoogleFont(conf, "Inconsolata", font.StyleNormal, font.WeightNormal)
	if err != nil {
		t.Fatal(err)
	}
	if f.Path() != "Inconsolata-regular.ttf" {
		t.Fatalf("unexpected cached font name %q", f.Path())
	}
	_, err = svc.findGoogleFont(conf, "Inconsolata", font.StyleItalic, font.WeightNormal)
	if err == nil {
		t.Error("expected search for Inconsolata Italic to fail, did not")
	}

	f, err = svc.findGoogleFont(conf, "Anonymous Pro", font.StyleNormal, font.WeightNormal)
	if err != nil {
		t.Fatal(err)
	}
	if f.Path() != "Anonymous Pro-regular.ttf" {
		t.Fatalf("expected regular variant, got %q", f.Path())
	}

	f, err = svc.findGoogleFont(conf, "Anonymous Pro", font.StyleItalic, font.WeightNormal)
	if err != nil {
		t.Fatal(err)
	}
	if f.Path() != "Anonymous Pro-italic.ttf" {
		t.Fatalf("expected italic variant, got %q", f.Path())
	}
}

func TestGoogleCacheFont(t *testing.T) {
	hostio := newFakeIO(t)
	svc := newGoogleService(hostio)
	conf := testconfig.Conf{
		"app-key": "tyse-test",
	}
	fi, err := svc.matchGoogleFontInfo(conf, "Inconsolata", font.StyleNormal, font.WeightNormal)
	if err != nil {
		t.Fatal(err)
	}
	cachedir, file, err := svc.cacheGoogleFont(conf, fi[0], "regular")
	if err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(cachedir, file)
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != string(hostio.fontBytes) {
		t.Fatalf("cached bytes differ from downloaded bytes")
	}
}
