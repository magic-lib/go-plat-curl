package curl

import (
	"crypto/tls"
	"net/http"
	"net/url"
	"time"
)

type httpClient struct {
	transport         *http.Transport
	disableKeepAlives *bool
	tlsClientConfig   *tls.Config
	proxy             func(*http.Request) (*url.URL, error)
	jar               http.CookieJar
	checkRedirect     func(req *http.Request, via []*http.Request) error
	timeout           time.Duration
}

// createTransport 根据参数创建新的transport
func (h *httpClient) createTransport() http.RoundTripper {
	if h.transport == nil {
		h.transport = http.DefaultTransport.(*http.Transport).Clone() //避免更改default值
	}
	if !(h.disableKeepAlives != nil || h.tlsClientConfig != nil || h.proxy != nil) {
		return http.RoundTripper(h.transport) //没有新的改变，直接返回
	}

	if h.disableKeepAlives != nil {
		h.transport.DisableKeepAlives = *h.disableKeepAlives
		h.disableKeepAlives = nil
	}
	if h.tlsClientConfig != nil {
		h.transport.TLSClientConfig = h.tlsClientConfig
		h.tlsClientConfig = nil
	}
	if h.proxy != nil {
		h.transport.Proxy = h.proxy
		h.proxy = nil
	}
	return http.RoundTripper(h.transport)
}

// createClient 创建客户端
func (h *httpClient) createClient() *http.Client {
	return &http.Client{
		Transport:     h.createTransport(),
		Jar:           h.jar,
		CheckRedirect: h.checkRedirect,
		Timeout:       h.timeout,
	}
}
