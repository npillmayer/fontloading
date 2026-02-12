package googlefont

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"path"
	"testing"

	"github.com/npillmayer/schuko/schukonf/testconfig"
)

func TestCacheDownload(t *testing.T) {
	hostio := newFakeIO(t)
	const payload = "<svg xmlns='http://www.w3.org/2000/svg'></svg>\n"
	hostio.fontBytes = []byte(payload)
	const url = "https://example.test/UAX-Logo-shadow.svg"

	conf := testconfig.Conf{
		"app-key":         "tyse-test",
		"fonts-cache-dir": t.TempDir(),
	}

	cachedir, err := cacheFontDirPath(hostio, conf, "A")
	if err != nil {
		t.Fatal(err)
	}
	dst := path.Join(cachedir, "test.svg")
	err = downloadCachedFile(hostio, dst, url)
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, []byte(payload)) {
		t.Fatalf("cached file differs from response payload")
	}
}

type failingStatusIO struct {
	*fakeIO
	status int
}

func (f failingStatusIO) HTTPGet(u string) (*http.Response, error) {
	return &http.Response{
		StatusCode: f.status,
		Status:     "502 Bad Gateway",
		Body:       io.NopCloser(bytes.NewReader(nil)),
		Header:     make(http.Header),
	}, nil
}

func TestCacheDownloadHTTPStatusError(t *testing.T) {
	hostio := failingStatusIO{
		fakeIO: newFakeIO(t),
		status: http.StatusBadGateway,
	}
	dst := path.Join(t.TempDir(), "test.svg")
	err := downloadCachedFile(hostio, dst, "https://example.test/failure.svg")
	if err == nil {
		t.Fatal("expected download failure for non-200 status")
	}
	if _, statErr := os.Stat(dst); statErr == nil {
		t.Fatal("expected no file to be created for failed download")
	}
}
