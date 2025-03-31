package curl

import (
	"crypto/tls"
	"net/http"
	"net/url"
)

func (c *client) initHttpClientCfg() {
	if c.httpClient == nil {
		c.httpClient = new(httpClient)
	}
}

func (c *client) DisableKeepAlives(v bool) *client {
	c.initHttpClientCfg()
	c.httpClient.disableKeepAlives = &v
	c.clientHasChanged = true
	return c
}

func (c *client) Jar(v http.CookieJar) *client {
	c.initHttpClientCfg()
	c.httpClient.jar = v
	c.clientHasChanged = true
	return c
}

func (c *client) CheckRedirect(v func(req *http.Request, via []*http.Request) error) *client {
	c.initHttpClientCfg()
	c.httpClient.checkRedirect = v
	c.clientHasChanged = true
	return c
}

func (c *client) TLSClient(v *tls.Config) *client {
	c.initHttpClientCfg()
	c.httpClient.tlsClientConfig = v
	c.clientHasChanged = true
	return c
}

func (c *client) Proxy(v func(*http.Request) (*url.URL, error)) *client {
	c.initHttpClientCfg()
	c.httpClient.proxy = v
	c.clientHasChanged = true
	return c
}

func (c *client) Transport(v *http.Transport) *client {
	c.initHttpClientCfg()
	c.httpClient.transport = v
	c.clientHasChanged = true
	return c
}
