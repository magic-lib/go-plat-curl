package curl

import (
	"context"
	"fmt"
	"github.com/ChengjinWu/gojson"
	jsoniter "github.com/json-iterator/go"
	"github.com/magic-lib/go-plat-utils/conv"
	"github.com/magic-lib/go-plat-utils/logs"
	"github.com/magic-lib/go-plat-utils/id-generator/id"
	"github.com/magic-lib/go-plat-utils/utils/httputil/param"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"net/http"
	"net/textproto"
	"net/url"
	"reflect"
	"sort"
	"strings"
)

// getMethod 请求方法判断
func getMethod(method string) string {
	method = strings.ToUpper(method)
	methodList := []string{
		http.MethodGet,
		http.MethodPost,
		http.MethodDelete,
		http.MethodHead, http.MethodPut, http.MethodPatch} //允许的方法列表
	for _, one := range methodList {
		if one == method {
			return method
		}
	}
	return defaultMethod
}

func getHeaders(headers http.Header, method string, data interface{}) http.Header {
	if headers == nil {
		headers = http.Header{}
	}

	ct := headers.Get(headerContentType)
	if ct == "" {
		isSetType := false
		dataString, err := getDataString(data)
		if err == nil {
			if method == http.MethodGet {
				isSetType = true
				headers.Set(headerContentType, headerContentTypeFormUrlencoded)
			}
		}
		if !isSetType {
			if dataString != "" {
				checkJsonError := gojson.CheckValid([]byte(dataString))
				if checkJsonError == nil {
					//表示数据是json格式
					headers.Set(headerContentType, headerContentTypeJsonUtf8)
				}
			}
		}
	} else {
		//如果data数据是json，并且不是get的话，则不能是 x-www-form-urlencoded
		if method != http.MethodGet {
			if strings.Contains(ct, headerContentTypeFormUrlencodedKey) {
				dataString, err := getDataString(data)
				if err == nil {
					checkJsonError := gojson.CheckValid([]byte(dataString))
					if checkJsonError == nil {
						//表示数据是json格式
						headers.Set(headerContentType, headerContentTypeJsonUtf8)
					}
				}
			}
		}
	}

	headers = beautifulHeader(headers)

	return headers
}

func getDataString(data interface{}) (string, error) {
	var paramDataStr string
	typeData := fmt.Sprintf("%T", data)
	if typeData != "string" {
		paramDataByte, err2 := jsonApi.Marshal(data)
		if err2 != nil {
			return "", fmt.Errorf("data 格式目前不支持:%w", err2)
		}
		paramDataStr = string(paramDataByte)
	} else {
		paramDataStr = data.(string)
	}
	return paramDataStr, nil
}

// 需要执行header格式的，如果没有，则直接使用，如果有且value是相同的话，则直接覆盖，避免提交两份数据
func beautifulHeader(headers http.Header) http.Header {
	if headers == nil {
		return nil
	}
	//0,复制一个headers
	oldHeaders := headers.Clone()
	newHeaders := http.Header{}

	// 不写一起的原因是可能后面有key更满足要求的情况。

	//1、首先将header标准key的值取出来
	hasStoreKeyList := make([]string, 0)
	for key, val := range oldHeaders {
		newKey := textproto.CanonicalMIMEHeaderKey(key)
		if key == newKey {
			newHeaders = setHeaderValues(newHeaders, newKey, val...)
			hasStoreKeyList = append(hasStoreKeyList, newKey)
		}
	}

	//2、将已经存储的key删除掉
	for _, key := range hasStoreKeyList {
		oldHeaders.Del(key)
	}

	//3、剩下的看value是否相同，不同的就存储下来
	for key, val := range oldHeaders {
		newKey := textproto.CanonicalMIMEHeaderKey(key)
		allNewValues := newHeaders.Values(newKey)
		if len(allNewValues) == 0 {
			//如果完全不存在，则用new进行存储
			newHeaders = setHeaderValues(newHeaders, newKey, val...)
			continue
		}
		newHeaders = setHeaderValues(newHeaders, key, val...)
	}
	return newHeaders
}

// setHeaderValues 为headers添加value
func setHeaderValues(h http.Header, key string, values ...string) http.Header {
	if key == "" {
		return h
	}
	for _, v := range values {
		if v == "" {
			continue
		}
		hasValues := h.Values(key)
		isFind := false
		for _, one := range hasValues {
			if one == v {
				isFind = true
			}
		}
		if !isFind {
			h.Add(key, v)
		}
	}
	return h
}

func getNewUrl(url, method string, dataString string) string {
	if (method != http.MethodGet && method != http.MethodDelete) || dataString == "" {
		return url
	}

	err := gojson.CheckValid([]byte(dataString))
	param := dataString
	if err == nil {
		param2 := make(map[string]interface{})
		err3 := jsoniter.Unmarshal([]byte(dataString), &param2)
		if err3 == nil {
			param = createParamStrOrder(param2)
		}
	}
	newUrl := ""
	if strings.Index(url, "?") > 0 {
		newUrl = url + "&" + param
	} else {
		newUrl = url + "?" + param
	}
	return newUrl
}

// createParamStrOrder 对参数进行排序，然后拼接成URL的字符串
func createParamStrOrder(params map[string]interface{}) string {
	aParams := make([]string, 0)
	for k, v := range params {
		val := fmt.Sprintf("%v", v)
		aParams = append(aParams, k+"="+url.QueryEscape(val))
	}
	sort.Strings(aParams)
	return strings.Join(aParams, "&")
}

func printLog(ctx context.Context, loggers logs.ILogger, logLevel logs.LogLevel, printLogInt int, logStr string) {
	if printLogInt == PrintClose {
		return
	}
	if printLogInt == PrintError {
		if logLevel < logs.WARNING {
			return
		}
	}

	//系统外的打印日志
	if !isNil(loggers) {
		if logLevel == 0 {
			logLevel = loggers.Level()
		}
		logs.Logger(loggers, logLevel, logStr)
		return
	}
	if logLevel == 0 {
		logLevel = logs.DEBUG //级别低，尽量不打印
	}
	// 默认的打印日志
	if printLogInt == PrintAll {
		var loggerTemp logs.ILogger
		if ctx != nil {
			loggerTemp = logs.CtxLogger(ctx)
		} else {
			loggerTemp = logs.DefaultLogger()
		}
		logs.Logger(loggerTemp, logLevel, logStr)
	}
}

func printLoggerResponse(ctx context.Context, cLogger logs.ILogger, defaultPrintLogInt int, resp *Response) {
	if cLogger == nil {
		return
	}
	logLevel := cLogger.Level()
	if resp.Error != nil {
		logLevel = logs.ERROR
	} else {
		if resp.StatusCode != http.StatusOK && resp.StatusCode != 0 {
			logLevel = logs.ERROR
		}
	}
	returnData := conv.String(resp)
	//这里默认打上日志，方便查问题，需要将数据量减少，避免默认内容太多了
	rData := []rune(gjson.Get(returnData, "request.data").String())
	rHeader := []rune(gjson.Get(returnData, "request.header").String())
	repData := []rune(gjson.Get(returnData, "response").String())
	maxLen := defaultPrintLogDataLength
	if len(rData) > maxLen {
		returnData, _ = sjson.Set(returnData, "request.data", string(rData[:maxLen]))
	}
	if len(rHeader) > maxLen {
		returnData, _ = sjson.Set(returnData, "request.headers", string(rHeader[:maxLen]))
	}
	if len(repData) > maxLen {
		returnData, _ = sjson.Set(returnData, "response", string(repData[:maxLen]))
	}

	logStrTemp := fmt.Sprintf("[comm-request print return]id:%s, data:%s, error: %v", resp.Id, conv.String(returnData), resp.Error)
	if logLevel >= logs.WARNING {
		printLog(ctx, cLogger, logLevel, defaultPrintLogInt, logStrTemp)
	}
}

func isNil(i interface{}) bool {
	if i == nil {
		return true
	}
	vi := reflect.ValueOf(i)
	kind := vi.Kind()
	if kind == reflect.Ptr ||
		kind == reflect.Chan ||
		kind == reflect.Func ||
		kind == reflect.UnsafePointer ||
		kind == reflect.Map ||
		kind == reflect.Interface ||
		kind == reflect.Slice {
		return vi.IsNil()
	}
	return false
}

func getRequestId(p *Request) string {
	paramDataOnlyStr := ""
	{
		paramDataStr, err := getDataString(p.Data)
		if err == nil {
			paramDataOnlyStr = getJsonOnlyKey(paramDataStr)
		}
	}

	headerDataOnlyStr := ""
	{
		if p.Header != nil {
			var keys []string
			for k := range p.Header {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			headerArray := make([]string, 0)
			for _, m := range keys {
				headerArray = append(headerArray, m+"="+p.Header.Get(m))
			}
			headerDataOnlyStr = getJsonOnlyKey(headerArray)
		}
	}

	return id.GetUUID(fmt.Sprintf("[%s][%s][%s][%s]", p.Url,
		paramDataOnlyStr, p.Method, headerDataOnlyStr))
}

// getJsonOnlyKey 传入map对象，取得唯一的返回key，用于cache中存储的时候
func getJsonOnlyKey(data interface{}) string {
	jsonData := conv.String(data)
	jsonMap := conv.KeyListFromMap(jsonData)
	if len(jsonMap) > 0 {
		return param.HttpBuildQuery(jsonMap)
	}
	strList := strings.Split(jsonData, "")
	sort.Strings(strList)
	return strings.Join(strList, "")
}
