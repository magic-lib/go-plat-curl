package curl

import (
	"context"
	"github.com/magic-lib/go-plat-utils/cache"
	"github.com/magic-lib/go-plat-utils/conf"
	"github.com/magic-lib/go-plat-utils/logs"
	"net/http"
)

type InjectHandler interface {
	// BeforeHandler 发送前的方法
	BeforeHandler(ctx context.Context, rs *Request, httpReq *http.Request) error
	// AfterHandler 发送后的方法
	AfterHandler(ctx context.Context, rp *Response) error
}

// client 所有request包含公共的参数
type client struct {
	handler          InjectHandler
	httpClient       *httpClient
	httpCli          *http.Client
	clientHasChanged bool                    //client是否改变
	cacheIns         cache.CommCache[string] //缓存对象
	logger           logs.ILogger
}

// NewClient 客户端
func NewClient() *client {
	c := new(client)
	c.initHttpClientCfg()
	//初始化内存cache
	c.cacheIns = cache.NewMemGoCache[string](defaultMaxCacheTime, cacheCleanupInterval)
	return c
}

// WithHandler 设置执行前后方法
func (c *client) WithHandler(h InjectHandler) *client {
	c.handler = h
	return c
}

// WithCache 设置缓存实例
func (c *client) WithCache(cIns cache.CommCache[string]) *client {
	c.cacheIns = cIns
	return c
}

func (c *client) NewRequest(r *Request) *genRequest {
	gen := genRequestFromRequest(r)
	if c.clientHasChanged || c.httpCli == nil { // 如果改变了，则需要重新设置
		c.httpCli = c.httpClient.createClient()
		c.clientHasChanged = false
	}
	if c.logger == nil {
		c.logger = logs.DefaultLogger()
	}

	gen.defaultPrintLogInt = PrintError //默认只打印错误，后续可通过 SetPrintLog 方法覆盖默认值
	if conf.GetEnv() == conf.EnvLoc || conf.GetEnv() == conf.EnvDev {
		//测试环境默认全打印
		gen.defaultPrintLogInt = PrintAll
	}

	gen.setClient(c)
	return gen
}

// DefaultClient 默认客户端
func DefaultClient() *client {
	if defaultClient != nil {
		return defaultClient
	}
	defaultClient = NewClient()
	return defaultClient
}

// SetDefaultClient 默认客户端
func SetDefaultClient(cli *client) {
	defaultClientMutex.Lock()
	defer defaultClientMutex.Unlock()
	defaultClient = cli
}
