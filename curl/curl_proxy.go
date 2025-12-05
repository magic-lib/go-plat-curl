package curl

import (
	"context"
	"fmt"
	"github.com/magic-lib/go-plat-cache/cache"
	"github.com/magic-lib/go-plat-startupcfg/startupcfg"
	"github.com/magic-lib/go-plat-utils/conv"
	"net/http"
	"time"
)

// Proxy 利用config，封装了curl的提交方法
type Proxy interface {
	ServerApi() startupcfg.ServiceAPI                                                  //获取服务api配置
	Submit(ctx context.Context, proxyData *ProxyData, dstPoint any) (*Response, error) //提交数据
}

var (
	defaultCacheTime = time.Hour
)

type ProxyConfig struct {
	ServiceConfig *startupcfg.ConfigAPI
	ServiceName   string //config文件中服务名称，用于区分不同的服务
}
type ProxyCacheConfig struct {
	Namespace          string                    //缓存key的命名空间，避免相同的key冲突问题
	CacheKey           string                    //单纯的cacheKey
	CacheTime          time.Duration             //缓存时间，默认1小时
	CacheCheckFunc     func(resp *Response) bool //返回true表示满足缓存条件
	CacheDontUseExpire bool                      //如果过期，直接请求，但没有获取到，是否可以暂时使用过期数据, 默认false, 表示如果远程获取不到，一直使用db过期数据
	CachePool          cache.CommCache[string]   //缓存池对象，一般用mysql存储，如果不设置，默认为内存存储
}

type ProxyData struct {
	UrlCfgName      string                                 //url配置名称
	CurlReq         *Request                               //curl请求参数,GET\POST
	CacheConfig     *ProxyCacheConfig                      //缓存配置
	RetryConfig     *RetryPolicy                           //重试配置
	BuildReqHandler func(rb RequestBuilder) RequestBuilder //构建请求的函数，特殊处理，一般用不上
}

type curlProxy struct {
	serverApi startupcfg.ServiceAPI
}

func NewCurlProxy(cfg *ProxyConfig) (Proxy, error) {
	if cfg == nil {
		return nil, fmt.Errorf("cfg is nil")
	}
	if cfg.ServiceConfig == nil {
		return nil, fmt.Errorf("cfg.ServiceConfig is nil")
	}
	if cfg.ServiceName == "" {
		return nil, fmt.Errorf("cfg.ServiceName is empty")
	}

	dataApi := cfg.ServiceConfig.ServiceAPI(cfg.ServiceName)
	if dataApi == nil {
		return nil, fmt.Errorf("%s not set", cfg.ServiceName)
	}
	return &curlProxy{
		serverApi: dataApi,
	}, nil
}

func (l *curlProxy) ServerApi() startupcfg.ServiceAPI {
	return l.serverApi
}

func (l *curlProxy) Submit(ctx context.Context, proxyData *ProxyData, dstPoint any) (*Response, error) {
	if proxyData == nil {
		return nil, fmt.Errorf("proxyData is nil")
	}

	proxyData = buildCurlReq(l, proxyData)
	proxyData = buildCurlProxyRetry(proxyData)
	proxyData = buildCurlProxyCache(proxyData)

	var rb RequestBuilder = NewClient().NewRequest(proxyData.CurlReq)
	if proxyData.RetryConfig != nil {
		rb = rb.SetRetryPolicy(proxyData.RetryConfig)
	}
	if proxyData.BuildReqHandler != nil {
		rb = proxyData.BuildReqHandler(rb)
	}

	allRetStr := ""
	useCache := checkOpenCache(proxyData)
	if useCache {
		retStr, err := cache.NsGet(ctx, proxyData.CacheConfig.CachePool, proxyData.CacheConfig.Namespace, proxyData.CacheConfig.CacheKey)
		allRetStr = retStr              // 保存可能过期的数据
		if err == nil && retStr != "" { //表示为过期数据
			retResp := &Response{
				Request:    proxyData.CurlReq,
				StatusCode: http.StatusOK,
				Response:   retStr,
			}
			if dstPoint == nil {
				return retResp, nil
			} else {
				if err = conv.Unmarshal(retStr, dstPoint); err == nil {
					return retResp, nil
				}
			}
		}
	}

	resp := rb.Submit(ctx)

	useExpireDataFunc := func(resp *Response, outErr error)(*Response, error){
		if !proxyData.CacheConfig.CacheDontUseExpire && allRetStr != "" {
			resp.Response = allRetStr
			resp.Error = nil
			resp.StatusCode = http.StatusOK
			if dstPoint != nil {
				if err := conv.Unmarshal(resp.Response, dstPoint); err != nil {
					return resp, err
				}
			}
			return resp, nil
		}
		return resp, outErr
	}


	if resp.Error != nil {
		return useExpireDataFunc(resp, resp.Error)
	}
	if resp.StatusCode != http.StatusOK {
		return useExpireDataFunc(resp, fmt.Errorf("%d:%s", resp.StatusCode, http.StatusText(resp.StatusCode)))
	}

	var cacheFunction = func() {
		if useCache {
			// 如果满足缓存条件，则缓存数据
			if proxyData.CacheConfig.CacheCheckFunc(resp) {
				cache.NsSet(ctx,
					proxyData.CacheConfig.CachePool,
					proxyData.CacheConfig.Namespace,
					proxyData.CacheConfig.CacheKey,
					resp.Response, proxyData.CacheConfig.CacheTime)
			}
		}
	}

	if dstPoint != nil {
		if err := conv.Unmarshal(resp.Response, dstPoint); err != nil {
			return resp, err
		}
	}
	cacheFunction()

	return resp, nil
}

func buildCurlReq(l *curlProxy, proxyData *ProxyData) *ProxyData {
	if proxyData == nil {
		return nil
	}
	if proxyData.CurlReq == nil {
		proxyData.CurlReq = new(Request)
	}
	if proxyData.CurlReq.Method == "" {
		proxyData.CurlReq.Method = http.MethodPost //默认获取数据用Post方式
	}
	if proxyData.CurlReq.Url == "" {
		proxyData.CurlReq.Url = fmt.Sprintf("%s%s", l.serverApi.DomainName(), l.serverApi.Url(proxyData.UrlCfgName))
	}
	return proxyData
}

func buildCurlProxyRetry(proxyData *ProxyData) *ProxyData {
	if proxyData == nil {
		return nil
	}
	if proxyData.RetryConfig != nil {
		if proxyData.RetryConfig.Attempts <= 0 {
			proxyData.RetryConfig.Attempts = 3
		}
		defaultRetryCheck := func(resp *Response) error {
			if resp.Error != nil {
				return resp.Error
			}
			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("status code: %d", resp.StatusCode)
			}
			return nil
		}
		if proxyData.RetryConfig.RetryCondFunc == nil {
			proxyData.RetryConfig.RetryCondFunc = defaultRetryCheck
		} else {
			customerCheck := proxyData.RetryConfig.RetryCondFunc
			proxyData.RetryConfig.RetryCondFunc = func(resp *Response) error {
				err := defaultRetryCheck(resp)
				if err != nil {
					return err
				}
				return customerCheck(resp)
			}
		}
	}
	return proxyData
}
func buildCurlProxyCache(proxyData *ProxyData) *ProxyData {
	if proxyData == nil {
		return nil
	}
	if proxyData.CacheConfig != nil {
		if proxyData.CacheConfig.CacheTime <= 0 {
			proxyData.CacheConfig.CacheTime = defaultCacheTime
		}
		if proxyData.CacheConfig.CachePool == nil {
			proxyData.CacheConfig.CachePool = cache.NewMemGoCache[string](defaultCacheTime, defaultCacheTime)
		}
		if proxyData.CacheConfig.Namespace == "" {
			// 默认使用url+method作为namespace，可能存在相同的情况，最好自定义
			proxyData.CacheConfig.Namespace = fmt.Sprintf("%s_%s", proxyData.UrlCfgName, proxyData.CurlReq.Method)
		}
		// 默认缓存条件
		defaultCacheCheck := func(resp *Response) bool {
			if resp.Error != nil ||
				resp.StatusCode != http.StatusOK ||
				resp.Response == "" {
				return false
			}
			return true
		}
		if proxyData.CacheConfig.CacheCheckFunc == nil {
			proxyData.CacheConfig.CacheCheckFunc = defaultCacheCheck
		} else {
			customerCheck := proxyData.CacheConfig.CacheCheckFunc
			proxyData.CacheConfig.CacheCheckFunc = func(resp *Response) bool {
				defaultCheck := defaultCacheCheck(resp)
				if !defaultCheck {
					return false
				}
				return customerCheck(resp)
			}
		}
	}
	return proxyData
}

func checkOpenCache(proxyData *ProxyData) bool {
	if proxyData == nil {
		return false
	}
	if proxyData.CacheConfig != nil &&
		proxyData.CacheConfig.Namespace != "" &&
		proxyData.CacheConfig.CacheKey != "" &&
		proxyData.CacheConfig.CacheTime > 0 &&
		proxyData.CacheConfig.CacheCheckFunc != nil &&
		proxyData.CacheConfig.CachePool != nil {
		return true
	}
	return false
}
