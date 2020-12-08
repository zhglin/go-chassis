package session

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chassis/go-chassis/v2/core/common"
	"github.com/go-chassis/go-chassis/v2/core/invocation"
	"github.com/go-chassis/go-chassis/v2/pkg/util/httputil"
	"github.com/go-chassis/openlog"
	"github.com/patrickmn/go-cache"
)

// ErrResponseNil used for to represent the error response, when it is nil
var ErrResponseNil = errors.New("can not set session, resp is nil")

// Cache session cache variable
var Cache *cache.Cache

// SessionStickinessCache key: go-chassisLB , value is cookie
// 记录不同请求的服务对应的sessionId, 不要求客户端携带cookie
var SessionStickinessCache *cache.Cache

// session管理器
func init() {
	Cache = initCache()                  // sessionId => address
	SessionStickinessCache = initCache() // service namespace => 的sessionId
	cookieMap = make(map[string]string)
}
func initCache() *cache.Cache {
	value := cache.New(3e+10, time.Second*30)
	return value
}

// context中的cookie 与 SessionStickinessCache功能一致
var cookieMap map[string]string

// getLBCookie gets cookie from local map
func getLBCookie(key string) string {
	return cookieMap[key]
}

// setLBCookie sets cookie to local map
func setLBCookie(key, value string) {
	cookieMap[key] = value
}

// GetContextMetadata gets data from context
// 从context中读取cookie
func GetContextMetadata(ctx context.Context, key string) string {
	md, ok := ctx.Value(common.ContextHeaderKey{}).(map[string]string)
	if ok {
		return md[key]
	}
	return ""
}

// SetContextMetadata sets data to context
// 设置cookie到context中
func SetContextMetadata(ctx context.Context, key string, value string) context.Context {
	md, ok := ctx.Value(common.ContextHeaderKey{}).(map[string]string)
	if !ok {
		md = make(map[string]string)
	}

	if md[key] == value {
		return ctx
	}

	md[key] = value
	return context.WithValue(ctx, common.ContextHeaderKey{}, md)
}

//GetSessionFromResp return session uuid in resp if there is
// http.response中获取指定key的cookie
func GetSessionFromResp(cookieKey string, resp *http.Response) string {
	bytes := httputil.GetRespCookie(resp, cookieKey)
	if bytes != nil {
		return string(bytes)
	}
	return ""
}

// SaveSessionIDFromContext check session id in response ctx and save it to session storage
// 从context中获取sessionId 这里不再区分invocation 所有有的invocation的sessionId是同一个
func SaveSessionIDFromContext(ctx context.Context, ep string, autoTimeout int) context.Context {

	timeValue := time.Duration(autoTimeout) * time.Second

	// context中的cookie设置
	sessionIDStr := GetContextMetadata(ctx, common.LBSessionID)
	if sessionIDStr != "" {
		cookieKey := strings.Split(sessionIDStr, "=")
		if len(cookieKey) > 1 {
			sessionIDStr = cookieKey[1]
		}
	}

	// cache过期清理
	ClearExpired()
	var sessBool bool
	if sessionIDStr != "" {
		_, sessBool = Cache.Get(sessionIDStr)
	}

	// cookie存在 cache存在
	if sessionIDStr != "" && sessBool {
		cookie := common.LBSessionID + "=" + sessionIDStr
		setLBCookie(common.LBSessionID, cookie) // 设置到本地
		Save(sessionIDStr, ep, timeValue)       // 重置cache
		return ctx
	}

	// cache过期 重新生成
	sessionIDValue, err := GenerateSessionID()
	if err != nil {
		openlog.Warn("session id generate fail, it is impossible", openlog.WithTags(
			openlog.Tags{
				"err": err.Error(),
			}))
	}
	cookie := common.LBSessionID + "=" + sessionIDValue
	setLBCookie(common.LBSessionID, cookie)
	Save(sessionIDValue, ep, timeValue)
	return SetContextMetadata(ctx, common.LBSessionID, cookie) // 设置到context中
}

//Temporary responsewriter for SetCookie
type cookieResponseWriter http.Header

// Header implements ResponseWriter Header interface
func (c cookieResponseWriter) Header() http.Header {
	return http.Header(c)
}

//Write is a dummy function
func (c cookieResponseWriter) Write([]byte) (int, error) {
	panic("ERROR")
}

//WriteHeader is a dummy function
func (c cookieResponseWriter) WriteHeader(int) {
	panic("ERROR")
}

//setCookie appends cookie with already present cookie with ';' in between
// http.response中设置cookie session
func setCookie(resp *http.Response, value string) {

	newCookie := common.LBSessionID + "=" + value
	oldCookie := string(httputil.GetRespCookie(resp, common.LBSessionID))

	// 同名已存在
	if oldCookie != "" {
		//If cookie is already set, append it with ';'
		newCookie = newCookie + ";" + oldCookie
	}

	c1 := http.Cookie{Name: common.LBSessionID, Value: newCookie}

	w := cookieResponseWriter(resp.Header)
	http.SetCookie(w, &c1)
}

// SaveSessionIDFromHTTP check session id
// 设置http sessionId  把sessionId设置到response中 上层解析response中的cookie设置SessionStickinessCache
func SaveSessionIDFromHTTP(ep string, autoTimeout int, resp *http.Response, req *http.Request) {
	if resp == nil {
		openlog.Warn(fmt.Sprintf("%s", ErrResponseNil))
		return
	}

	// sessionId过期时间
	timeValue := time.Duration(autoTimeout) * time.Second

	// request中的sessionId
	var sessionIDStr string
	if c, err := req.Cookie(common.LBSessionID); err != http.ErrNoCookie && c != nil {
		sessionIDStr = c.Value
	}

	// 清理过期的sessionId
	ClearExpired()
	var sessBool bool
	if sessionIDStr != "" {
		_, sessBool = Cache.Get(sessionIDStr)
	}

	// response中的sessionId
	valueChassisLb := GetSessionFromResp(common.LBSessionID, resp)
	//if session is in resp, then just save it
	// 首次 服务器端处理sessionId resp不变 直接缓存
	if valueChassisLb != "" {
		Save(valueChassisLb, ep, timeValue)
	} else if sessionIDStr != "" && sessBool { //后续 request携带 request中有 cache有
		setCookie(resp, sessionIDStr)     // 设置response中cookie 在最外层进行SessionStickiness的设置
		Save(sessionIDStr, ep, timeValue) // 重置session cache  sessionId不过期就一直有效
	} else {
		sessionIDValue, err := GenerateSessionID() // cache过期 或者 都不存在 重新生成sessionId
		if err != nil {
			openlog.Warn("session id generate fail, it is impossible", openlog.WithTags(
				openlog.Tags{
					"err": err.Error(),
				}))
		}
		setCookie(resp, sessionIDValue)
		Save(sessionIDValue, ep, timeValue)
	}

}

//GenerateSessionID generate a session id
// 生成sessionId
func GenerateSessionID() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// DeletingKeySuccessiveFailure deleting key successes and failures
// 响应失败的节点删除sessionId 对于基于会话的balance重新切换节点
func DeletingKeySuccessiveFailure(resp *http.Response) {
	// 清理过期
	Cache.DeleteExpired()
	// context方式
	if resp == nil {
		valueChassisLb := getLBCookie(common.LBSessionID)
		if valueChassisLb != "" {
			cookieKey := strings.Split(valueChassisLb, "=")
			if len(cookieKey) > 1 {
				Delete(cookieKey[1])
				setLBCookie(common.LBSessionID, "")
			}
		}
		return
	}

	// http方式
	valueChassisLb := GetSessionFromResp(common.LBSessionID, resp)
	if valueChassisLb != "" {
		cookieKey := strings.Split(valueChassisLb, "=")
		if len(cookieKey) > 1 {
			Delete(cookieKey[1])
		}
	}
}

// GetSessionCookie getting session cookie
func GetSessionCookie(ctx context.Context, resp *http.Response) string {
	if ctx != nil {
		return GetContextMetadata(ctx, common.LBSessionID)
	}

	if resp == nil {
		openlog.Warn(fmt.Sprintf("%s", ErrResponseNil))
		return ""
	}

	valueChassisLb := GetSessionFromResp(common.LBSessionID, resp)
	if valueChassisLb != "" {
		return valueChassisLb
	}

	return ""
}

// AddSessionStickinessToCache add new cookie or refresh old cookie
// 设置sessionId
func AddSessionStickinessToCache(cookie, namespace string) {
	key := getSessionStickinessCacheKey(namespace)
	value, ok := SessionStickinessCache.Get(key)
	// 不存在直接设置
	if !ok || value == nil {
		SessionStickinessCache.Set(key, cookie, 0)
		return
	}
	// 已存在 不是字符串类型 直接覆盖
	s, ok := value.(string)
	if !ok {
		SessionStickinessCache.Set(key, cookie, 0)
		return
	}
	// 已存在 已改变 直接覆盖
	if cookie != "" && s != cookie {
		SessionStickinessCache.Set(key, cookie, 0)
	}
}

// GetSessionID get sessionID from cache
// 获取sessionId
func GetSessionID(namespace string) string {
	value, ok := SessionStickinessCache.Get(getSessionStickinessCacheKey(namespace))
	if !ok || value == nil {
		openlog.Warn("not sessionID in cache")
		return ""
	}
	s, ok := value.(string)
	if !ok {
		openlog.Warn("get sessionID from cache failed")
		return ""
	}
	return s
}

// 根据namespace生成cache中的key
func getSessionStickinessCacheKey(namespace string) string {
	if namespace == "" {
		namespace = common.SessionNameSpaceDefaultValue
	}
	return strings.Join([]string{common.LBSessionID, namespace}, "|")
}

// GetSessionIDFromInv when use  SessionStickiness , get session id from inv
// 从请求响应中提取sessionId
func GetSessionIDFromInv(inv invocation.Invocation, key string) string {
	var metadata interface{}
	switch inv.Reply.(type) {
	case *http.Response:
		resp := inv.Reply.(*http.Response)
		value := httputil.GetRespCookie(resp, key)
		if string(value) != "" {
			metadata = string(value)
		}
	default:
		value := GetContextMetadata(inv.Ctx, key)
		if value != "" {
			metadata = value
		}
	}
	if metadata == nil {
		metadata = ""
	}
	return metadata.(string)
}
