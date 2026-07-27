package redaction

import (
	"regexp"
	"strings"
)

var (
	emailPattern    = regexp.MustCompile(`(?i)[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}`)
	phonePattern    = regexp.MustCompile(`\b1[3-9][0-9]{9}\b`)
	identityPattern = regexp.MustCompile(`\b[0-9]{17}[0-9Xx]\b`)
	orderPattern    = regexp.MustCompile(`(?i)\bSF[0-9]{8,}\b`)
)

func Text(value string) string {
	value = strings.TrimSpace(value)
	value = emailPattern.ReplaceAllString(value, "[邮箱已脱敏]")
	value = phonePattern.ReplaceAllString(value, "[手机号已脱敏]")
	value = identityPattern.ReplaceAllString(value, "[身份信息已脱敏]")
	return orderPattern.ReplaceAllStringFunc(value, func(order string) string {
		if len(order) <= 4 {
			return "[订单号已脱敏]"
		}
		return "[订单号尾号" + order[len(order)-4:] + "]"
	})
}
