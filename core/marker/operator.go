package marker

import (
	"regexp"
	"strconv"
	"strings"
)

// 精确比较
func exact(value, express string) bool {
	return value == express
}

// 子串
func contains(value, express string) bool {
	return strings.Contains(value, express)
}

// 正则
func regex(value, express string) bool {
	reg := regexp.MustCompilePOSIX(express)
	return reg.Match([]byte(value))
}

// 不等
func noEqu(value, express string) bool {
	return !(value == express)
}

// 大于等于
func noLess(value, express string) bool {
	return cmpInt(value, express, func(v, e int) bool {
		return v >= e
	})
}

// 小于
func less(value, express string) bool {
	return cmpInt(value, express, func(v, e int) bool {
		return v < e
	})
}

// 小于等于
func noGreater(value, express string) bool {
	return cmpInt(value, express, func(v, e int) bool {
		return v <= e
	})
}

// 大于
func greater(value, express string) bool {
	return cmpInt(value, express, func(v, e int) bool {
		return v > e
	})
}

// 转换成int的比较 转换失败就false
func cmpInt(value, express string, op func(v, e int) bool) bool {
	v, err := strconv.Atoi(value)
	if err != nil {
		return false
	}
	exp, err := strconv.Atoi(express)
	if err != nil {
		return false
	}
	return op(v, exp)
}
