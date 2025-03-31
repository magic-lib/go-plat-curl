package curl_test

import (
	"context"
	"fmt"
	"github.com/magic-lib/go-plat-curl/curl"
	"github.com/magic-lib/go-plat-utils/conf"
	"github.com/magic-lib/go-plat-utils/goroutines"
	"net/http"
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
