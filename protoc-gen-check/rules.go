package main

import (
	"fmt"
	"regexp"
	"strings"
)

// Number 是一个泛型约束，表示可以是以下任意一种数值类型
type Number interface {
	uint32 | uint64 | int32 | int64 | float32 | float64
}

type RuleFuncGetter[T any, V any] func(V) RuleFunc[T]
type RuleFunc[T any] func(T) (bool, string)

// 判断数值是否在指定范围内 [left, right]
func NumberRangeLR[T Number](left, right T) RuleFunc[T] {
	return func(val T) (bool, string) {
		if val >= left && val <= right {
			return true, ""
		}
		message := fmt.Sprintf("数值 %v 不在指定范围[%v, %v]", val, left, right)
		return false, message
	}
}

// 判断数值是否在指定范围内 [left, right)
func NumberRangeL[T Number](left, right T) RuleFunc[T] {
	return func(val T) (bool, string) {
		if val >= left && val <= right {
			return true, ""
		}
		message := fmt.Sprintf("数值 %v 不在指定范围[%v, %v)", val, left, right)
		return false, message
	}
}

// 判断数值是否在指定范围内 (left, right]
func NumberRangeR[T Number](left, right T) RuleFunc[T] {
	return func(val T) (bool, string) {
		if val >= left && val <= right {
			return true, ""
		}
		message := fmt.Sprintf("数值 %v 不在指定范围(%v, %v]", val, left, right)
		return false, message
	}
}

// 判断数值是否在指定范围内 (left, right)
func NumberRange[T Number](left, right T) RuleFunc[T] {
	return func(val T) (bool, string) {
		if val >= left && val <= right {
			return true, ""
		}
		message := fmt.Sprintf("数值 %v 不在指定范围(%v, %v)", val, left, right)
		return false, message
	}
}

func NumberLt[T Number](right T) RuleFunc[T] {
	return func(val T) (bool, string) {
		if val < right {
			return true, ""
		}
		message := fmt.Sprintf("数值 %v 必须小于%v", val, right)
		return false, message
	}
}

func NumberLte[T Number](right T) RuleFunc[T] {
	return func(val T) (bool, string) {
		if val <= right {
			return true, ""
		}
		message := fmt.Sprintf("数值 %v 必须小于等于%v", val, right)
		return false, message
	}
}

func NumberGt[T Number](right T) RuleFunc[T] {
	return func(val T) (bool, string) {
		if val > right {
			return true, ""
		}
		message := fmt.Sprintf("数值 %v 必须大于%v", val, right)
		return false, message
	}
}

func NumberGte[T Number](right T) RuleFunc[T] {
	return func(val T) (bool, string) {
		if val >= right {
			return true, ""
		}
		message := fmt.Sprintf("数值 %v 必须大于等于%v", val, right)
		return false, message
	}
}

// Contains 检查元素是否在数组中
func Contains[T comparable](arr []T, elem T) bool {
	for _, v := range arr {
		if v == elem {
			return true
		}
	}
	return false
}

func ScalarIn[T Number | string](right []T) RuleFunc[T] {
	return func(val T) (bool, string) {
		if Contains(right, val) {
			return true, ""
		}
		message := fmt.Sprintf("数值 %v 应该在数组 %v", val, right)
		return false, message
	}
}

func ScalarNotIn[T Number | string](right []T) RuleFunc[T] {
	return func(val T) (bool, string) {
		if !Contains(right, val) {
			return true, ""
		}
		message := fmt.Sprintf("数值 %v 不应该在数组 %v", val, right)
		return false, message
	}
}

func ScalarConst[T Number | bool | string](right T) RuleFunc[T] {
	return func(val T) (bool, string) {
		if val == right {
			return true, ""
		}
		message := fmt.Sprintf("值 %v 必须等于%v", val, right)
		return false, message
	}
}

func StringLen(right uint64) RuleFunc[string] {
	return func(val string) (bool, string) {
		if uint64(len(val)) == right {
			return true, ""
		}
		message := fmt.Sprintf("字符串 %v 长度必须等于%v", val, right)
		return false, message
	}
}

func StringMinLen(right uint64) RuleFunc[string] {
	return func(val string) (bool, string) {
		if uint64(len(val)) >= right {
			return true, ""
		}
		message := fmt.Sprintf("字符串 %v 长度必须>=%v", val, right)
		return false, message
	}
}

func StringMaxLen(right uint64) RuleFunc[string] {
	return func(val string) (bool, string) {
		if uint64(len(val)) <= right {
			return true, ""
		}
		message := fmt.Sprintf("字符串 %v 长度必须<=%v", val, right)
		return false, message
	}
}

func StringLenBytes(right uint64) RuleFunc[string] {
	return func(val string) (bool, string) {
		if uint64(len([]byte(val))) == right {
			return true, ""
		}
		message := fmt.Sprintf("字符串 %v 字节数必须等于%v", val, right)
		return false, message
	}
}

func StringMinBytes(right uint64) RuleFunc[string] {
	return func(val string) (bool, string) {
		if uint64(len([]byte(val))) >= right {
			return true, ""
		}
		message := fmt.Sprintf("字符串 %v 字节数必须>=%v", val, right)
		return false, message
	}
}

func StringMaxBytes(right uint64) RuleFunc[string] {
	return func(val string) (bool, string) {
		if uint64(len([]byte(val))) <= right {
			return true, ""
		}
		message := fmt.Sprintf("字符串 %v 字节数必须<=%v", val, right)
		return false, message
	}
}

func StringPattern(pattern string) RuleFunc[string] {
	return func(val string) (bool, string) {
		matched, err := regexp.MatchString(pattern, val)
		if err != nil {
			message := fmt.Sprintf("正则表达式错误: %v", err)
			return false, message
		}
		if matched {
			return true, ""
		}
		message := fmt.Sprintf("字符串 %v 不匹配模式 %v", val, pattern)
		return false, message
	}
}

func StringPrefix(prefix string) RuleFunc[string] {
	return func(val string) (bool, string) {
		if strings.HasPrefix(val, prefix) {
			return true, ""
		}
		message := fmt.Sprintf("字符串 %v 没有前缀 %v", val, prefix)
		return false, message
	}
}

func StringSuffix(suffix string) RuleFunc[string] {
	return func(val string) (bool, string) {
		if strings.HasSuffix(val, suffix) {
			return true, ""
		}
		message := fmt.Sprintf("字符串 %v 没有后缀 %v", val, suffix)
		return false, message
	}
}

func StringContains(substr string) RuleFunc[string] {
	return func(val string) (bool, string) {
		if strings.Contains(val, substr) {
			return true, ""
		}
		message := fmt.Sprintf("字符串 %v 不包含 %v", val, substr)
		return false, message
	}
}

func StringNotContains(substr string) RuleFunc[string] {
	return func(val string) (bool, string) {
		if !strings.Contains(val, substr) {
			return true, ""
		}
		message := fmt.Sprintf("字符串 %v 不应包含 %v", val, substr)
		return false, message
	}
}
