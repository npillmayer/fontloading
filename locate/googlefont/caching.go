package googlefont

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"

	"github.com/npillmayer/schuko"
)

// downloadFile will download a url to a local file (usually located in the
// user's cache directory).
func downloadCachedFile(hostio IO, filepath string, url string) error {
	resp, err := hostio.HTTPGet(url)
	if err != nil {
		return err
	}
	if resp == nil {
		return errors.New("download request returned nil response")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download request failed: %s", resp.Status)
	}
	out, err := hostio.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, resp.Body)
	return err
}

// cacheFontDirPath checks and possibly creates a folder in the user's font cache
// directory.
//
// First choice:
// Directory path taken from configuration-key "fonts-cache-dir" + subfolder.
//
// Second choice:
// The base cache directory is taken from `os.UserCacheDir()`, plus
// an application specific key, taken as `app-key` from the global configuration,
// and appending "fonts" + subfolder.
//
// Clients may specify a folder name which will be appended to
// the base cache path. Non-existing sub-folders will be created as necessary
// (with permissions 750).
//
// Returns the path to the cache-(sub-)folder or an error.
func cacheFontDirPath(hostio IO, conf schuko.Configuration, subfolder string) (cacheDir string, err error) {
	tracer().Debugf("config[%s] = %s", "app-key", conf.GetString("app-key"))
	if cacheDir = conf.GetString("fonts-cache-dir"); cacheDir != "" {
		cacheDir = path.Join(cacheDir, subfolder)
	} else {
		var appkey string
		if appkey = conf.GetString("app-key"); appkey == "" {
			return "", errors.New("application key is not set")
		}
		if cacheDir, err = hostio.UserCacheDir(); err != nil {
			return "", err
		}
		cacheDir = path.Join(cacheDir, conf.GetString("app-key"), "fonts", subfolder)
	}
	tracer().Debugf("caching resource in %s", cacheDir)
	if _, err = hostio.Stat(cacheDir); err != nil {
		err = hostio.MkdirAll(cacheDir, 0750)
	}
	return
}
