package curl

import (
	"context"
	"fmt"
	"github.com/magic-lib/go-plat-cache/cache"
	"github.com/magic-lib/go-plat-startupcfg/startupcfg"
	"google.golang.org/grpc"
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

type GrpcConfig struct {
	ServiceConfig *startupcfg.ConfigAPI
	ConnPoolCfg   *cache.ResPoolConfig[*grpc.ClientConn]
	ServiceName   string //config文件中服务名称，用于区分不同的服务
}

func buildCurlReq(serverApi startupcfg.ServiceAPI, proxyData *ProxyData) *ProxyData {
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
		proxyData.CurlReq.Url = fmt.Sprintf("%s%s", serverApi.DomainName(), serverApi.Url(proxyData.UrlCfgName))
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
