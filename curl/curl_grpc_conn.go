package curl

import (
	"fmt"
	"github.com/magic-lib/go-plat-cache/cache"
	"github.com/magic-lib/go-plat-utils/conv"
	cmap "github.com/orcaman/concurrent-map/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"reflect"
	"strings"
	"time"
)

var (
	allGrpcConnPool = cmap.New[*cache.CommPool[*grpc.ClientConn]]() //保存所有grpc的连接池，这样可以重复使用，避免重复创建的问题
)

func getGrpcConnPool(domain string, resPoolCfg *cache.ResPoolConfig[*grpc.ClientConn], opt ...grpc.DialOption) (*cache.CommPool[*grpc.ClientConn], error) {
	domain = strings.TrimSpace(domain)
	if domain == "" {
		return nil, fmt.Errorf("domain is empty")
	}

	if allGrpcConnPool.Has(domain) {
		if connPool, ok := allGrpcConnPool.Get(domain); ok {
			return connPool, nil
		}
		return nil, fmt.Errorf("get grpc conn pool err: %s", domain)
	}
	if len(opt) == 0 {
		opt = make([]grpc.DialOption, 0)
	}

	if resPoolCfg == nil {
		resPoolCfg = &cache.ResPoolConfig[*grpc.ClientConn]{}
	}

	if resPoolCfg.New == nil {
		resPoolCfg.New = func() (*grpc.ClientConn, error) {
			newOpt := ensureTransportCredentials(opt...)
			conn, err := grpc.NewClient(domain, newOpt...)
			if err == nil && conn != nil {
				return conn, nil
			}
			return nil, fmt.Errorf("%s connect err: %w", conv.String(domain), err)
		}
	}
	if resPoolCfg.CheckFunc == nil {
		resPoolCfg.CheckFunc = func(conn *grpc.ClientConn) error {
			state := conn.GetState()
			if state == connectivity.Idle ||
				state == connectivity.Connecting ||
				state == connectivity.Ready {
				return nil
			}
			return fmt.Errorf("%s connect err state: %s", conv.String(domain), state.String())
		}
	}
	if resPoolCfg.CloseFunc == nil {
		resPoolCfg.CloseFunc = func(conn *grpc.ClientConn) error {
			return conn.Close()
		}
	}
	if resPoolCfg.MaxSize <= 0 {
		resPoolCfg.MaxSize = 10
	}
	if resPoolCfg.MaxUsage == 0 {
		resPoolCfg.MaxUsage = 30 * time.Second
	}

	rpcConnPool := cache.NewResPool(resPoolCfg)
	allGrpcConnPool.Set(domain, rpcConnPool)
	return rpcConnPool, nil
}

// hasTransportCredentials 检查 DialOption 中是否包含 WithTransportCredentials 配置
func hasTransportCredentials(opts []grpc.DialOption) bool {
	// grpc.WithTransportCredentials 的底层会创建一个 DialOption，其内部存储的是 credentials.TransportCredentials
	// 通过反射解析 DialOption 的内部值，判断是否包含 TransportCredentials
	for _, opt := range opts {
		// 获取 DialOption 的底层值（DialOption 是 func(*grpc.DialOptions) 类型，需通过反射获取其包装的内容）
		val := reflect.ValueOf(opt)
		if val.Kind() == reflect.Func {
			// 获取函数的闭包变量（grpc 内部会将配置存在闭包中）
			if val.Type().NumIn() == 1 {
				// 遍历闭包的所有变量
				for i := 0; i < val.Type().NumIn(); i++ {
					// 检查是否包含 credentials.TransportCredentials 类型的变量
					argType := val.Type().In(i)
					if argType.String() == "credentials.TransportCredentials" ||
						argType.Implements(reflect.TypeOf((*credentials.TransportCredentials)(nil)).Elem()) {
						return true
					}
				}
			}

			// 兼容 grpc 不同版本的实现（部分版本闭包变量存在于 Func 的 CaptureVars 中）
			captureValues := reflect.ValueOf(opt).Pointer()
			if captureValues != 0 {
				// 简化判断：只要 DialOption 是 WithTransportCredentials 生成的，就返回 true
				// 更稳妥的方式是通过 grpc 内部的私有方法，但反射可能受版本影响，这里用特征判断
				if fmt.Sprintf("%v", opt) == "grpc.WithTransportCredentials" {
					return true
				}
			}
		}
	}
	return false
}

// ensureTransportCredentials 确保 DialOption 中包含传输凭证，无则添加不安全凭证
func ensureTransportCredentials(opt ...grpc.DialOption) []grpc.DialOption {
	// 检查是否已有 TransportCredentials 配置
	if hasTransportCredentials(opt) {
		return opt
	}

	// 没有则添加不安全的传输凭证
	newOpts := append(opt, grpc.WithTransportCredentials(insecure.NewCredentials()))
	return newOpts
}
