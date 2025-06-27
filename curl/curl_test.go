package curl_test

import (
	"context"
	"fmt"
	"github.com/magic-lib/go-plat-curl/curl"
	"github.com/magic-lib/go-plat-utils/conf"
	"github.com/magic-lib/go-plat-utils/goroutines"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"testing"
	"time"
)

const localUrl = "https://static.json.cn/r/json/search_list.json"

var data = map[string]interface{}{
	"name":    "HttpRequest",
	"version": "v1.0",
}

var defaultClient = curl.NewClient()

func TestGetResponseWithCache(t *testing.T) {
	conf.SetEnv(conf.EnvLoc)
	_ = defaultClient.NewRequest(&curl.Request{
		Url:    localUrl,
		Data:   data,
		Method: http.MethodGet,
		Header: nil,
	}).SetCacheTime(5 * time.Second).SetCacheCheckFunc(func(resp *curl.Response) bool {
		t.Log("check func")
		return true
	}).Submit(nil)

	goroutines.GoAsync(func(params ...any) {
		time.Sleep(2 * time.Second)
		_ = defaultClient.NewRequest(&curl.Request{
			Url:    localUrl,
			Data:   data,
			Method: http.MethodGet,
			Header: nil,
		}).SetCache(5*time.Second, func(resp *curl.Response) bool {
			t.Log("check func")
			return true
		}).Submit(nil)
	})

	time.Sleep(7 * time.Second)

	_ = defaultClient.NewRequest(&curl.Request{
		Url:    localUrl,
		Data:   data,
		Method: http.MethodGet,
		Header: nil,
	}).SetCacheTime(5 * time.Second).Submit(nil)

	time.Sleep(3 * time.Second)

	//t.Log(conv.String(resp))
}

func TestGetResponseWithRetry(t *testing.T) {
	conf.SetEnv(conf.EnvLoc)

	_ = defaultClient.NewRequest(&curl.Request{
		Url:    localUrl,
		Data:   data,
		Method: http.MethodGet,
		Header: nil,
	}).SetCacheTime(10*time.Second).SetRetryPolicy(&curl.RetryPolicy{
		RetryCondFunc: func(resp *curl.Response) error {
			return fmt.Errorf("get error")
		},
		Attempts: 3,
	}).SetRetry(3, func(resp *curl.Response) error {
		return fmt.Errorf("get error")
	}).Submit(nil)

	//t.Log(conv.String(resp))
}

type oneInject struct {
	Begin string
	After string
}

func (o *oneInject) BeforeHandler(ctx context.Context, rs *curl.Request, httpReq *http.Request) error {
	fmt.Println("begin:", o.Begin)
	return nil
}

func (o *oneInject) AfterHandler(ctx context.Context, rp *curl.Response) error {
	fmt.Println("after:", o.After)
	return nil
}

func TestGetResponseWithAllHandler(t *testing.T) {
	conf.SetEnv(conf.EnvLoc)

	defaultClient.WithHandler(&oneInject{
		Begin: "aaaa",
		After: "bbbb",
	})

	_ = defaultClient.NewRequest(&curl.Request{
		Url:    localUrl,
		Data:   data,
		Method: http.MethodGet,
		Header: nil,
	}).Submit(nil)
}
func TestGetResponseWithAllHandler2(t *testing.T) {
	conf.SetEnv(conf.EnvLoc)

	curl.SetDefaultHandler(&oneInject{
		Begin: "eeee",
		After: "ffff",
	})

	_ = defaultClient.NewRequest(&curl.Request{
		Url:    localUrl,
		Data:   data,
		Method: http.MethodGet,
		Header: nil,
	}).Submit(nil)
}
func TestSetDefaultClient(t *testing.T) {
	conf.SetEnv(conf.EnvLoc)

	cli := curl.DefaultClient()
	cli = cli.DisableKeepAlives(true).WithHandler(&oneInject{
		Begin: "pppp",
		After: "tttt",
	})
	curl.SetDefaultClient(cli)

	curl.DefaultClient().NewRequest(&curl.Request{
		Url:    localUrl,
		Data:   data,
		Method: http.MethodGet,
		Header: nil,
	}).Submit(nil)
}

func getMethod(allUrl string) {
	resp, err := http.Get(allUrl)
	if err != nil {
		fmt.Println("请求失败:", err)
		return
	}
	defer resp.Body.Close()

	// 检查响应状态码是否为 200 OK
	if resp.StatusCode != http.StatusOK {
		fmt.Println("请求失败，状态码：", resp.StatusCode)
		return
	}

	fileName, err := getFileNameFromURL(allUrl)
	if err != nil {
		return
	}

	oneFile, err := saveToTempFile(fileName, resp.Body)
	if err != nil {
		return
	}
	fmt.Println("文件保存成功：", oneFile.Name())
}
func newMethod(allUrl string) {
	// 打开 JSON 文件
	file, err := os.Open("/var/folders/xd/yd6ypt0s3p34gpnh696fv7f40000gn/T/upload-1194355570-900e54effa78f1b9.jpeg")
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close()

	fileData, _ := io.ReadAll(file)

	resp := curl.NewClient().NewRequest(&curl.Request{
		Url:    allUrl,
		Method: http.MethodPost,
		Data: map[string]any{
			"file": &curl.UploadFile{
				FileName: file.Name(),
				FileData: fileData,
			},
		},
	}).Submit(nil)

	body := []byte(resp.Response)

	// 检查响应状态码是否为 200 OK
	if resp.StatusCode != http.StatusOK {
		fmt.Println("请求失败，状态码：", resp.StatusCode)
		return
	}

	fileName, err := getFileNameFromURL(allUrl)
	if err != nil {
		return
	}

	saveToTempFile2(fileName, body)
}

func getFileNameFromURL(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse URL: %w", err)
	}

	// 获取路径的最后一段作为文件名
	filename := path.Base(u.Path)
	return filename, nil
}

func saveToTempFile(fileName string, rc io.ReadCloser) (*os.File, error) {
	tmpfile, err := os.CreateTemp("", "upload-*-"+fileName)
	if err != nil {
		return nil, fmt.Errorf("create temp file failed: %w", err)
	}

	_, err = io.Copy(tmpfile, rc)
	if err != nil {
		tmpfile.Close()
		os.Remove(tmpfile.Name())
		return nil, fmt.Errorf("copy content failed: %w", err)
	}
	rc.Close()

	// 将文件指针移回开头
	_, err = tmpfile.Seek(0, io.SeekStart)
	if err != nil {
		tmpfile.Close()
		os.Remove(tmpfile.Name())
		return nil, fmt.Errorf("seek to start failed: %w", err)
	}

	return tmpfile, nil
}
func saveToTempFile2(fileName string, body []byte) {
	// 创建临时文件
	tmpfile, err := os.CreateTemp("", "upload-*-"+fileName)
	if err != nil {
		fmt.Println("创建临时文件失败:", err)
		return
	}
	defer tmpfile.Close()

	// 写入内容到临时文件
	_, err = tmpfile.Write(body)
	if err != nil {
		fmt.Println("写入文件失败:", err)
		tmpfile.Close()
		os.Remove(tmpfile.Name()) // 删除临时文件
		return
	}

	// 可选：将文件指针移回文件开头
	_, err = tmpfile.Seek(0, io.SeekStart)
	if err != nil {
		fmt.Println("定位文件指针失败:", err)
		tmpfile.Close()
		os.Remove(tmpfile.Name()) // 删除临时文件
		return
	}

	fmt.Println("文件保存成功：", tmpfile.Name())
}

func TestGetFile(t *testing.T) {
	allUrl := "https://zamloan.com/credit_attach/2024-5-29/900e54effa78f1b9.jpeg"
	//getMethod(allUrl)
	newMethod(allUrl)
}
func TestPostFile(t *testing.T) {
	allUrl := "http://127.0.0.1:10601/audit-mgr/api/v1/audit/order/27/attach"
	//getMethod(allUrl)
	newMethod(allUrl)
}
