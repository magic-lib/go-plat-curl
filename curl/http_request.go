package curl

import (
	"bytes"
	"context"
	"fmt"
	"github.com/magic-lib/go-plat-utils/logs"
	"net/http"
)

// initHeaders headers
func (g *genRequest) initHeaders(req *http.Request) {
	if g.Header == nil || len(g.Header) == 0 {
		return
	}
	for k, v := range g.Header {
		req.Header = setHeaderValues(req.Header, k, v...)
	}
}

// initCookies cookies
func (g *genRequest) initCookies(req *http.Request) {
	if g.cookies == nil {
		return
	}
	for _, v := range g.cookies {
		req.AddCookie(v)
	}
}

// initBasicAuth req
func (g *genRequest) initBasicAuth(req *http.Request) {
	if g.username != "" && g.password != "" {
		req.SetBasicAuth(g.username, g.password)
	}
}

// buildHttpRequest 汇总初始化http.Request
func (g *genRequest) buildHttpRequest(req *http.Request) *http.Request {
	if req == nil {
		req = &http.Request{}
	}
	g.initHeaders(req)
	g.initCookies(req)
	g.initBasicAuth(req)
	return req
}

func (g *genRequest) getHttpRequest(ctx context.Context, dataString string) (*http.Request, error) {
	newUrl := getNewUrl(g.Url, g.Method, dataString)

	httpReq, err := http.NewRequest(g.Method, newUrl, bytes.NewBufferString(dataString))
	if err != nil {
		logStr := fmt.Sprintf("[comm-request request] url:%s, error: %s", newUrl, err.Error())
		printLog(ctx, g.cli.logger, logs.ERROR, g.defaultPrintLogInt, logStr)
		return nil, err
	}

	httpReq = g.buildHttpRequest(httpReq)

	return httpReq, nil
}
