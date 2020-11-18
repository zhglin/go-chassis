package jwt

import (
	"sync"
)

var signingMethods = map[string]func() SigningMethod{}
var signingMethodLock = new(sync.RWMutex)

// Implement SigningMethod to add new methods for signing or verifying tokens.
// 签名函数接口
type SigningMethod interface {
	// 签名生成
	Verify(signingString, signature string, key interface{}) error // Returns nil if signature is valid
	// 解析签名
	Sign(signingString string, key interface{}) (string, error) // Returns encoded signature or error
	// 签名标识
	Alg() string // returns the alg identifier for this method (example: 'HS256')
}

// Register the "alg" name and a factory function for signing method.
// This is typically done during init() in the method's implementation
// 注册签名函数
func RegisterSigningMethod(alg string, f func() SigningMethod) {
	signingMethodLock.Lock()
	defer signingMethodLock.Unlock()

	signingMethods[alg] = f
}

// Get a signing method from an "alg" string
// 签名函数
func GetSigningMethod(alg string) (method SigningMethod) {
	signingMethodLock.RLock()
	defer signingMethodLock.RUnlock()

	if methodF, ok := signingMethods[alg]; ok {
		method = methodF()
	}
	return
}
