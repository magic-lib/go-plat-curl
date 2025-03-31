package curl

import (
	"github.com/samber/lo"
	"github.com/magic-lib/go-plat-utils/logs"
	"net/http"
	"time"
)

func (g *genRequest) SetUrl(s string) *genRequest {
	g.Url = s
	return g
}

func (g *genRequest) SetData(d interface{}) *genRequest {
	g.Data = d
	return g
}
func (g *genRequest) SetMethod(m string) *genRequest {
	g.Method = m
	return g
}

// SetHeaders headers
func (g *genRequest) SetHeaders(headers map[string]string) *genRequest {
	if headers != nil || len(headers) > 0 {
		if g.Header == nil {
			g.Header = make(http.Header)
		}
		for k, v := range headers {
			g.Header.Set(k, v)
		}
	}
	return g
}
func (g *genRequest) SetHeader(h http.Header) *genRequest {
	if g.Header == nil {
		g.Header = h
	} else {
		for k, v := range h {
			g.Header = setHeaderValues(g.Header, k, v...)
		}
	}
	return g
}

// SetCookies cookies
func (g *genRequest) SetCookies(cookies map[string]string) *genRequest {
	if cookies != nil || len(cookies) > 0 {
		if g.cookies == nil {
			g.cookies = make([]*http.Cookie, 0)
		}
		for k, v := range cookies {
			g.cookies = append(g.cookies, &http.Cookie{
				Name:  k,
				Value: v,
			})
		}
	}
	return g
}

// SetBasicAuth username, password
func (g *genRequest) SetBasicAuth(username, password string) *genRequest {
	g.username = username
	g.password = password
	return g
}

// SetTimeout d
func (g *genRequest) SetTimeout(d time.Duration) *genRequest {
	g.Timeout = d
	return g
}

// SetPrintLog PrintError只会打印错误，PrintAll全打，PrintClose不打
func (g *genRequest) SetPrintLog(b int) *genRequest {
	if b == PrintError || b == PrintClose || b == PrintAll {
		g.defaultPrintLogInt = b
	}
	return g
}

func (g *genRequest) SetLogger(l logs.ILogger) *genRequest {
	g.cli.logger = l
	return g
}
func (g *genRequest) SetRespDateType(l string) *genRequest {
	if lo.IndexOf(respDataTypeList, l) >= 0 { //只能有特殊的返回值
		g.respDateType = l
	}
	return g
}

func (g *genRequest) SetCache(cacheTime time.Duration, checkFunc func(resp *Response) bool) *genRequest {
	g.SetCacheTime(cacheTime)
	g.SetCacheCheckFunc(checkFunc)
	return g
}
func (g *genRequest) SetRetry(attempts uint, checkFunc func(resp *Response) error) *genRequest {
	g.SetRetryPolicy(&RetryPolicy{
		RetryCondFunc: checkFunc,
		Attempts:      attempts,
	})
	return g
}

func (g *genRequest) SetCacheTime(cacheTime time.Duration) *genRequest {
	g.cacheTime = cacheTime
	return g
}

// SetCacheCheckFunc 设置缓存检查函数，有些业务错误不允许缓存
func (g *genRequest) SetCacheCheckFunc(checkFunc func(resp *Response) bool) *genRequest {
	g.checkCacheFunc = checkFunc
	return g
}

func (g *genRequest) SetRetryPolicy(p *RetryPolicy) *genRequest {
	if p == nil {
		g.retryPolicy = nil //去掉重试条件
		return g
	}

	if g.retryPolicy == nil {
		g.retryPolicy = p
	}
	if p.Attempts > 0 {
		g.retryPolicy.Attempts = p.Attempts
	}
	if p.RetryCondFunc != nil {
		g.retryPolicy.RetryCondFunc = p.RetryCondFunc
	}

	if p.Delay > 0 {
		g.retryPolicy.Delay = p.Delay
	}
	if p.MaxJitter > 0 {
		g.retryPolicy.MaxJitter = p.MaxJitter
	}
	if p.DelayType != nil {
		g.retryPolicy.DelayType = p.DelayType
	}
	return g
}
