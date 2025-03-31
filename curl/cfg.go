package curl

import (
	jsoniter "github.com/json-iterator/go"
	"net/http"
	"sync"
	"time"
)

const (
	PrintError = iota //只打印关键错误日志
	PrintAll          //所有的都打印,哪怕你没有传ILogger
	PrintClose        //关闭日志打印
)

var (
	defaultClientMutex sync.Mutex
	defaultClient      *client
	defaultHandler     InjectHandler

	defaultPrintLogDataLength = 200 //默认打印日志的时候，数据最长，避免显示太多了

	defaultMethod        = http.MethodPost
	defaultTimeoutSecond = 30 * time.Second
	defaultMaxCacheTime  = 3600 * 24 * 2 * time.Second //最大用来存2天

	headerContentType                  = "Content-Type"
	headerContentTypeJsonUtf8          = "application/json; charset=utf-8"
	headerContentTypeFormUrlencoded    = "application/x-www-form-urlencoded"
	headerContentTypeFormUrlencodedKey = "x-www-form-urlencoded"

	respDataTypeJson = "json"
	respDataTypeList = []string{respDataTypeJson, "text"}

	cacheNamespace       = "comm-request"
	cacheCleanupInterval = time.Minute * 5

	jsonApi = jsoniter.Config{
		SortMapKeys: true,
	}.Froze()
)

// SetDefaultHandler 设置全局通用的trace方法
func SetDefaultHandler(j InjectHandler) {
	defaultHandler = j
}
