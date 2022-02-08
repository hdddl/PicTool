package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"os"
	"path/filepath"
	"strings"
)

type RecvData struct {
	Status bool   `json:"status"`
	Msg    string `json:"msg"`
	Path   string `json:"path"`
}

func main() {
	url := "https://blog.dongliu.site/img/upload"
	var CSRFToken string
	// 构造Session保证Cookie连续
	jar, _ := cookiejar.New(nil)
	client := &http.Client{
		Jar: jar,
	}
	// 开始发送GET请求获取CSRT token
	req, _ := http.NewRequest("GET", url, nil)
	resp, err := client.Do(req)
	checkErr(err)

	if resp.StatusCode != 200 {
		fmt.Printf("%s\n", resp.Body)
	} else {
		for _, cooke := range resp.Cookies() {
			if cooke.Name == "csrftoken" {
				CSRFToken = cooke.Value
			}
		}
	}
	_ = resp.Close

	imageAlbum := "1"
	for _, p := range os.Args[1:] {
		imageName := filepath.Base(p)
		imageDir := filepath.Dir(p)
		imagePath := filepath.Join(imageDir, imageName)
		postData := map[string]io.Reader{
			"name":                strings.NewReader(imageName),
			"album":               strings.NewReader(imageAlbum),
			"csrfmiddlewaretoken": strings.NewReader(CSRFToken),
		}
		b, contentType := structureMultiform(postData, imagePath)

		// 发送数据
		req, _ = http.NewRequest("POST", url, &b)
		req.Header.Set("Content-Type", contentType)
		res, err := client.Do(req)
		checkErr(err)

		defer func(Body io.ReadCloser) {
			err := Body.Close()
			checkErr(err)
		}(res.Body)
		// 对接收到的JSON数据进行处理
		body, _ := ioutil.ReadAll(res.Body)
		var recvData RecvData
		err = json.Unmarshal(body, &recvData)
		checkErr(err)
		if recvData.Status {
			resURL := "https://blog.dongliu.site/media" + recvData.Path
			fmt.Println(resURL)
		} else {
			fmt.Println(recvData.Msg)
		}
	}
}

// 构造多表单
func structureMultiform(data map[string]io.Reader, imagePath string) (bytes.Buffer, string) {
	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	// 写入表单数据
	for key, r := range data {
		var fw io.Writer
		fw, err := w.CreateFormField(key)
		checkErr(err)
		_, err = io.Copy(fw, r)
		checkErr(err)
	}
	// 写入图片
	img, err := os.Open(imagePath)
	checkErr(err)
	fw, err := w.CreateFormFile("image", img.Name())
	_, err = io.Copy(fw, img)
	checkErr(err)
	contentType := w.FormDataContentType()

	// 关闭 multipart writer
	_ = w.Close()
	return b, contentType
}

// 错误检查
func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}
