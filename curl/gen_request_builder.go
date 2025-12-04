package curl

import (
	"context"
	"github.com/magic-lib/go-plat-utils/logs"
	"github.com/samber/lo"
	"net/http"
	"time"
)

var _ RequestBuilder = new(genRequest)

type RequestBuilder interface {
	SetUrl(s string) RequestBuilder
	SetData(d interface{}) RequestBuilder
	SetMethod(m string) RequestBuilder
	SetHeaders(headers map[string]string) RequestBuilder
	SetHeader(h http.Header) RequestBuilder
	SetCookies(cookies map[string]string) RequestBuilder
	SetBasicAuth(username, password string) RequestBuilder
	SetTimeout(d time.Duration) RequestBuilder
	SetPrintLog(b int) RequestBuilder
	SetLogger(l logs.ILogger) RequestBuilder
	SetRespDateType(l string) RequestBuilder
	SetCache(cacheTime time.Duration, checkFunc func(resp *Response) bool) RequestBuilder
	SetRetry(attempts uint, checkFunc func(resp *Response) error) RequestBuilder
	SetCacheTime(cacheTime time.Duration) RequestBuilder
	SetCacheCheckFunc(checkFunc func(resp *Response) bool) RequestBuilder
	SetRetryPolicy(p *RetryPolicy) RequestBuilder
	Submit(ctx context.Context) *Response
}

func (g *genRequest) SetUrl(s string) RequestBuilder {
	g.Url = s
	return g
}

func (g *genRequest) SetData(d interface{}) RequestBuilder {
	g.Data = d
	return g
}
func (g *genRequest) SetMethod(m string) RequestBuilder {
	g.Method = m
	return g
}

// SetHeaders headers
func (g *genRequest) SetHeaders(headers map[string]string) RequestBuilder {
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
func (g *genRequest) SetHeader(h http.Header) RequestBuilder {
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
func (g *genRequest) SetCookies(cookies map[string]string) RequestBuilder {
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
func (g *genRequest) SetBasicAuth(username, password string) RequestBuilder {
	g.username = username
	g.password = password
	return g
}

// SetTimeout d
func (g *genRequest) SetTimeout(d time.Duration) RequestBuilder {
	g.timeout = d
	return g
}

// SetPrintLog PrintError只会打印错误，PrintAll全打，PrintClose不打
func (g *genRequest) SetPrintLog(b int) RequestBuilder {
	if b == PrintError || b == PrintClose || b == PrintAll {
		g.defaultPrintLogInt = b
	}
	return g
}

func (g *genRequest) SetLogger(l logs.ILogger) RequestBuilder {
	g.cli.logger = l
	return g
}
func (g *genRequest) SetRespDateType(l string) RequestBuilder {
	if lo.IndexOf(respDataTypeList, l) >= 0 { //只能有特殊的返回值
		g.respDateType = l
	}
	return g
}

func (g *genRequest) SetCache(cacheTime time.Duration, checkFunc func(resp *Response) bool) RequestBuilder {
	g.SetCacheTime(cacheTime)
	g.SetCacheCheckFunc(checkFunc)
	return g
}
func (g *genRequest) SetRetry(attempts uint, checkFunc func(resp *Response) error) RequestBuilder {
	g.SetRetryPolicy(&RetryPolicy{
		RetryCondFunc: checkFunc,
		Attempts:      attempts,
	})
	return g
}

func (g *genRequest) SetCacheTime(cacheTime time.Duration) RequestBuilder {
	g.cacheTime = cacheTime
	return g
}

// SetCacheCheckFunc 设置缓存检查函数，有些业务错误不允许缓存
func (g *genRequest) SetCacheCheckFunc(checkFunc func(resp *Response) bool) RequestBuilder {
	g.checkCacheFunc = checkFunc
	return g
}

func (g *genRequest) SetRetryPolicy(p *RetryPolicy) RequestBuilder {
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
