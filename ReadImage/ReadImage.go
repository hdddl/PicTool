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

func GetImageName(path string) string {
	if isURL(path) {
		u, _ := url.Parse(path)
		sl := strings.Split(u.Path, "/")
		return sl[len(sl)-1]
	} else {
		return filepath.Base(path)
	}
}

func OpenImage(path string) (*os.File, error) {
	if isURL(path) {
		return fileFromURL(path)
	} else {
		path = filepath.Join(filepath.Dir(path), filepath.Base(path))
		return os.Open(path)
	}
}

func isURL(path string) bool {
	u, err := url.Parse(path)
	return err == nil && u.Scheme != "" && u.Host != ""
}

func fileFromURL(imageURL string) (*os.File, error) {
	fileName := GetImageName(imageURL)
	filePath := filepath.Join(filepath.Dir(os.Args[0]), "img", fileName)
	resp, err := http.Get(imageURL)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("receive non 200 response code")
	}
	file, err := os.Create(filePath)
	if err != nil {
		return nil, err
	}
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return nil, err
	}
	err = file.Close()
	if err != nil {
		return nil, err
	}
	file, err = os.Open(filePath)
	if err != nil {
		return nil, err
	}
	return file, nil
}
