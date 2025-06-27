package curl

import (
	"fmt"
	"net/url"
	"strings"
)

// genRequestFromRequest 初始化参数
func genRequestFromRequest(r *Request) *genRequest {
	g := new(genRequest)
	g.Url = r.Url
	g.Data = r.Data
	g.Method = r.Method
	g.Header = r.Header
	return g
}

// buildGenRequest 优化一下参数
func (g *genRequest) buildGenRequest() {
	if g.Data == nil {
		g.Data = ""
	}
	g.Method = getMethod(g.Method)

	if g.Timeout <= 0 {
		g.Timeout = defaultTimeoutSecond
	}

	if g.cacheTime > 0 {
		if g.cacheTime > defaultMaxCacheTime {
			g.cacheTime = defaultMaxCacheTime
		}
	}

	g.Url = strings.TrimSpace(g.Url)
	g.Header = getHeaders(g.Header, g.Method, g.hasFile, g.Data)

	if g.cli.handler == nil {
		g.cli.handler = defaultHandler
	}
}

// checkParam 优化一下参数
func (g *genRequest) checkParam() error {

	if g.Url == "" {
		return fmt.Errorf("url请求地址为空")
	}

	_, err := url.Parse(g.Url)
	if err != nil {
		return fmt.Errorf("url格式错误：%s, %v", g.Url, err)
	}

	_, err = getDataString(g.hasFile, g.Data)
	if err != nil {
		return err
	}
	return nil
}

// setClient headers
func (g *genRequest) setClient(cli *client) *genRequest {
	if cli != nil {
		g.cli = cli
	}
	return g
}
