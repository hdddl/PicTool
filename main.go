package main

import (
	"PicTool/ReadImage"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/url"
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

type Cookie struct {
	Csrftoken string `json:"csrftoken"`
	Sessionid string `json:"sessionid"`
}

var (
	currentPath   = filepath.Dir(os.Args[0])                  // 程序目录路径
	configurePath = filepath.Join(currentPath, "config.json") // 配置文件路径
	cookiePath    = filepath.Join(currentPath, "cookie.json") // cookie 路径
)

func main() {
	// 读取配置文件
	var config Config
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
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}, // 拒绝重定向
	}

	config.LoadCookie(client, loginURL) // 加载cookie

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
		if recvData.Status { // 打印出返回的URL
			resURL := config.Host + "/media" + recvData.Path
			fmt.Println(resURL)
		} else {
			fmt.Println(recvData.Msg)
		}
	}
}

// 登入获取cookie
func (c Config) login(loginURL string, client *http.Client) bool {
	err, CSRFToken := getCSRFToken(loginURL, client)
	checkErr(err)
	loginData := map[string]io.Reader{
		"username":            strings.NewReader(c.Username),
		"password":            strings.NewReader(c.Password),
		"csrfmiddlewaretoken": strings.NewReader(CSRFToken),
	}
	b, contentType := structureMultiform(loginData, nil)
	// 发送数据
	req, _ := http.NewRequest("POST", loginURL, &b)
	req.Header.Set("Content-Type", contentType)
	res, err := client.Do(req)
	checkErr(err)
	return res.StatusCode != http.StatusOK // 返回true则表示密码正确反之则是密码错误
}

// RefreshCookie 刷新cookie存储
func (c Config) RefreshCookie(client *http.Client, loginURL string) {
	if !c.login(loginURL, client) {
		log.Fatalln("wrong password")
	}
	u, _ := url.Parse(c.Host)
	cookies := client.Jar.Cookies(u)
	var cookie Cookie
	file, err := os.OpenFile(cookiePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	checkErr(err)
	for _, item := range cookies {
		if item.Name == "csrftoken" {
			cookie.Csrftoken = item.Value
		} else {
			cookie.Sessionid = item.Value
		}
	}
	b, err := json.Marshal(cookie)
	checkErr(err)
	_, _ = file.Write(b)
	_ = file.Close()
}

// LoadCookie 读取已经存储的cookie并验证是否可用
func (c Config) LoadCookie(client *http.Client, loginURL string) {
	_, err := os.Stat(cookiePath)
	if err != nil {
		if os.IsNotExist(err) { // 如果cookie不存在则刷新cookie
			c.RefreshCookie(client, loginURL)
		} else {
			log.Fatalln(err)
		}
	}
	// 从文件中读取cookie
	var cookie Cookie
	bCookie, _ := ioutil.ReadFile(cookiePath)
	_ = json.Unmarshal(bCookie, &cookie)
	u, _ := url.Parse(c.Host)

	csrftoken := http.Cookie{
		Name:  "csrftoken",
		Value: cookie.Csrftoken,
	}
	sessionid := http.Cookie{
		Name:  "sessionid",
		Value: cookie.Sessionid,
	}

	client.Jar.SetCookies(u, []*http.Cookie{
		&csrftoken,
		&sessionid,
	})
	// 开始验证cookie是否可用
	testURL := "https://blog.dongliu.site/admin/"
	res, _ := client.Get(testURL)
	if res.StatusCode != http.StatusOK { // 如果响应值不为200则说明cookie过期了
		c.RefreshCookie(client, loginURL)
	}
}

// 获取CSRFToken
func getCSRFToken(Url string, client *http.Client) (error, string) {
	CSRFToken := ""
	// 开始发送GET请求获取CSRT token
	req, _ := http.NewRequest("GET", Url, nil)
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
		log.Fatalln(err)
	}
}
