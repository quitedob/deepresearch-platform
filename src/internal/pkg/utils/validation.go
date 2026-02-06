package utils

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

// SQL注入防护相关的正则表达式
var (
	// UUID格式验证
	uuidRegex = regexp.MustCompile(`^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{12}$`)
	
	// 简单ID格式（字母数字下划线）
	simpleIDRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	
	// 邮箱格式验证
	emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	
	// 用户名格式（字母数字下划线，3-50字符）
	usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_]{3,50}$`)
	
	// 危险SQL字符检测
	sqlInjectionPatterns = []string{
		"--", ";--", "/*", "*/", "@@",
		"char(", "nchar(", "varchar(", "nvarchar(",
		"alter ", "begin ", "cast(", "create ",
		"cursor ", "declare ", "delete ", "drop ",
		"end ", "exec ", "execute ", "fetch ",
		"insert ", "kill ", "select ", "sys.",
		"sysobjects", "syscolumns", "table ",
		"update ", "union ", "xp_",
	}
)

// IsValidUUID 验证UUID格式
func IsValidUUID(id string) bool {
	return uuidRegex.MatchString(id)
}

// IsValidSimpleID 验证简单ID格式（字母数字下划线横线）
func IsValidSimpleID(id string) bool {
	if len(id) == 0 || len(id) > 100 {
		return false
	}
	return simpleIDRegex.MatchString(id)
}

// IsValidEmail 验证邮箱格式
func IsValidEmail(email string) bool {
	if len(email) > 254 {
		return false
	}
	return emailRegex.MatchString(email)
}

// IsValidUsername 验证用户名格式
func IsValidUsername(username string) bool {
	return usernameRegex.MatchString(username)
}

// ContainsSQLInjection 检测是否包含SQL注入特征
func ContainsSQLInjection(input string) bool {
	lowerInput := strings.ToLower(input)
	for _, pattern := range sqlInjectionPatterns {
		if strings.Contains(lowerInput, pattern) {
			return true
		}
	}
	return false
}

// SanitizeString 清理字符串，移除潜在危险字符
func SanitizeString(input string) string {
	// 移除空字符
	input = strings.ReplaceAll(input, "\x00", "")
	// 移除控制字符
	var result strings.Builder
	for _, r := range input {
		if r >= 32 || r == '\n' || r == '\r' || r == '\t' {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// ValidateAndSanitizeID 验证并清理ID
// 返回清理后的ID和是否有效
func ValidateAndSanitizeID(id string) (string, bool) {
	id = strings.TrimSpace(id)
	if id == "" {
		return "", false
	}
	
	// 检查长度（最小1，最大100）
	if len(id) < 1 || len(id) > 100 {
		return "", false
	}
	
	// 检查是否包含空字符
	if strings.Contains(id, "\x00") {
		return "", false
	}
	
	// 检查是否包含SQL注入特征
	if ContainsSQLInjection(id) {
		return "", false
	}
	
	// 验证格式（UUID或简单ID）
	if IsValidUUID(id) || IsValidSimpleID(id) {
		return id, true
	}
	
	// 额外检查：纯数字ID也是有效的
	if IsValidNumericID(id) {
		return id, true
	}
	
	return "", false
}

// IsValidNumericID 验证纯数字ID
func IsValidNumericID(id string) bool {
	if len(id) == 0 || len(id) > 20 {
		return false
	}
	for _, c := range id {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// ValidateAndSanitizeIDs 批量验证ID
// 返回有效的ID列表和无效ID的数量
func ValidateAndSanitizeIDs(ids []string) ([]string, int) {
	validIDs := make([]string, 0, len(ids))
	invalidCount := 0
	
	for _, id := range ids {
		if sanitized, valid := ValidateAndSanitizeID(id); valid {
			validIDs = append(validIDs, sanitized)
		} else {
			invalidCount++
		}
	}
	
	return validIDs, invalidCount
}

// ValidateStringLength 验证字符串长度
func ValidateStringLength(s string, minLen, maxLen int) bool {
	length := utf8.RuneCountInString(s)
	return length >= minLen && length <= maxLen
}

// ValidateIntRange 验证整数范围
func ValidateIntRange(value, min, max int) bool {
	return value >= min && value <= max
}

// EscapeLikePattern 转义LIKE查询中的特殊字符
func EscapeLikePattern(pattern string) string {
	pattern = strings.ReplaceAll(pattern, "\\", "\\\\")
	pattern = strings.ReplaceAll(pattern, "%", "\\%")
	pattern = strings.ReplaceAll(pattern, "_", "\\_")
	return pattern
}
