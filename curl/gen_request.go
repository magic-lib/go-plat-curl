package curl

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

type genRequest struct {
	Request

	Timeout time.Duration `json:"timeout,omitempty"`

	cookies  []*http.Cookie
	username string
	password string

	cli *client

	retryPolicy *RetryPolicy

	respDateType       string //返回的数据类型，方便检测
	defaultPrintLogInt int    //0 表示默认，只打印一条，1表示完全打开所有信息，-1 表示完全关闭

	checkCacheFunc func(resp *Response) bool //检查是否需要缓存
	cacheTime      time.Duration             //缓存过期时间
}

func (g *genRequest) getNewRequest() *Request {
	req := new(Request)
	req.Url = g.Url
	req.Data = g.Data
	req.Method = g.Method
	req.Header = g.Header
	return req
}

func (g *genRequest) Submit(ctx context.Context) *Response {
	g.buildGenRequest()

	resp := newResponse(g.getNewRequest())

	err := g.checkParam()
	if err != nil {
		resp.Error = err
		return resp
	}

	if ctx == nil {
		ctx = context.Background()
	}

	if g.cacheTime > 0 {
		now := time.Now()
		respTxt := g.getDataFromCache(ctx)
		if respTxt != "" {
			resp.Response = respTxt
			resp.fromCache = true
			resp.setCostTime(now)

			logStr := fmt.Sprintf("[comm-request cache return]id:%s", resp.Id)
			printLog(ctx, g.cli.logger, 0, g.defaultPrintLogInt, logStr)

			return resp
		}
	}

	dataString, _ := getDataString(g.Data)

	postUrl := getNewUrl(g.Url, g.Method, dataString)

	logStr := fmt.Sprintf("[comm-request request] url:%s", postUrl)
	printLog(ctx, g.cli.logger, 0, g.defaultPrintLogInt, logStr)

	allResp := g.httpRequest(ctx, dataString, resp)

	//返回结果的日志
	printLoggerResponse(ctx, g.cli.logger, g.defaultPrintLogInt, allResp)

	if g.cacheTime > 0 && allResp.Response != "" {
		if allResp.Error == nil && g.cacheTime > 0 {
			//有个方法对是否进行缓存进行验证，避免保存了业务错误信息
			if g.checkCacheFunc != nil {
				if g.checkCacheFunc(allResp) {
					g.setDataToCache(ctx, allResp)
				}
			} else {
				if allResp.StatusCode == http.StatusOK {
					g.setDataToCache(ctx, allResp)
				}
			}
		}
	}

	return allResp
}
