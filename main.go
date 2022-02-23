package main

import (
	"PicTool/ReadImage"
	"bytes"
	"encoding/json"
	"errors"
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

type Config struct {
	Host     string `json:"host"`
	Album    string `json:"album"`
	Username string `json:"username"`
	Password string `json:"password"`
}

func main() {
	// 读取配置文件
	var config Config
	configurePath := filepath.Join(filepath.Dir(os.Args[0]), "config.json")
	tem, err := ioutil.ReadFile(configurePath)
	checkErr(err)
	err = json.Unmarshal(tem, &config)
	checkErr(err)

	uploadURL := config.Host + "/img/upload"
	loginURL := config.Host + "/admin/login/?next=/admin/"
	// 构造Session保证Cookie连续
	jar, _ := cookiejar.New(nil)
	client := &http.Client{
		Jar: jar,
	}
	// 模拟登入
	_ = config.login(loginURL, client)

	err, CSRFToken := getCSRFToken(uploadURL, client)
	checkErr(err)

	for _, p := range os.Args[1:] {
		// 获取文件名称
		imageName := ReadImage.GetImageName(p)
		// 构建数据
		formData := map[string]io.Reader{
			"name":                strings.NewReader(imageName),
			"album":               strings.NewReader(config.Album),
			"csrfmiddlewaretoken": strings.NewReader(CSRFToken),
		}
		// 读取数据
		image, err := ReadImage.OpenImage(p)
		if err != nil {
			panic(err)
		}
		b, contentType := structureMultiform(formData, map[string]ReadImage.Image{
			imageName: image,
		})
		// 关闭文件
		_ = image.Close()

		// 发送数据
		req, _ := http.NewRequest("POST", uploadURL, &b)
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

func (c Config) login(url string, client *http.Client) bool {
	err, CSRFToken := getCSRFToken(url, client)
	checkErr(err)
	loginData := map[string]io.Reader{
		"username":            strings.NewReader(c.Username),
		"password":            strings.NewReader(c.Password),
		"csrfmiddlewaretoken": strings.NewReader(CSRFToken),
	}
	b, contentType := structureMultiform(loginData, nil)
	// 发送数据
	req, _ := http.NewRequest("POST", url, &b)
	req.Header.Set("Content-Type", contentType)
	res, err := client.Do(req)
	checkErr(err)
	return res.StatusCode != http.StatusOK
}

// 获取CSRFToken
func getCSRFToken(url string, client *http.Client) (error, string) {
	CSRFToken := ""
	// 开始发送GET请求获取CSRT token
	req, _ := http.NewRequest("GET", url, nil)
	resp, err := client.Do(req)
	if err != nil {
		return err, CSRFToken
	}

	if resp.StatusCode != 200 {
		return errors.New("response status code is not 200"), CSRFToken
	} else {
		for _, cooke := range resp.Cookies() {
			if cooke.Name == "csrftoken" {
				CSRFToken = cooke.Value
			}
		}
	}
	_ = resp.Close
	return nil, CSRFToken
}

// 构造多表单
func structureMultiform(formData map[string]io.Reader, formFile map[string]ReadImage.Image) (bytes.Buffer, string) {
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
	// 写入文件数据
	for name, file := range formFile {
		fw, err := w.CreateFormFile("image", name)
		_, err = io.Copy(fw, file)
		checkErr(err)
	}

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
