package ReadImage

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type Image interface {
	io.Reader
	io.Closer
}

func OpenImage(path string) (Image, error) {
	if isURL(path) {
		return fileFromURL(path)
	} else {
		path = filepath.Join(filepath.Dir(path), filepath.Base(path))
		return os.Open(path)
	}
}

func fileFromURL(imageURL string) (Image, error) {
	resp, err := http.Get(imageURL)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("receive non 200 response code")
	}
	return resp.Body, nil
}

func isURL(path string) bool {
	u, err := url.Parse(path)
	return err == nil && u.Scheme != "" && u.Host != ""
}

func GetImageName(path string) string {
	if isURL(path) {
		u, _ := url.Parse(path)
		sl := strings.Split(u.Path, "/")
		return sl[len(sl)-1]
	} else {
		return filepath.Base(path)
	}
}
