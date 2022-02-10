package main

import (
	"PicTool/ReadImage"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"os"
	"strings"
)

type RecvData struct {
	Status bool   `json:"status"`
	Msg    string `json:"msg"`
	Path   string `json:"path"`
}

func main() {
	url := "https://blog.dongliu.site/img/upload"
	//url = "http://localhost:8000/img/upload"
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
		// 获取文件名称
		imageName := ReadImage.GetImageName(p)
		// 构建数据
		formData := map[string]io.Reader{
			"name":                strings.NewReader(imageName),
			"album":               strings.NewReader(imageAlbum),
			"csrfmiddlewaretoken": strings.NewReader(CSRFToken),
		}
		// 读取数据
		image, err := ReadImage.OpenImage(p)
		if err != nil {
			panic(err)
		}
		b, contentType := structureMultiform(formData, image, imageName)
		// 关闭文件
		_ = image.Close()

		// 发送数据
		req, _ = http.NewRequest("POST", url, &b)
		req.Header.Set("Content-Type", contentType)
		res, err := client.Do(req)
		checkErr(err)

		// 对接收到的JSON数据进行处理
		body, _ := ioutil.ReadAll(res.Body)
		// 关闭数据连接
		_ = res.Body.Close()
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
func structureMultiform(formData map[string]io.Reader, file ReadImage.Image, fileName string) (bytes.Buffer, string) {
	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	// 写入表单数据
	for key, r := range formData {
		var fw io.Writer
		fw, err := w.CreateFormField(key)
		checkErr(err)
		_, err = io.Copy(fw, r)
		checkErr(err)
	}

	fw, err := w.CreateFormFile("image", fileName)
	_, err = io.Copy(fw, file)
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
