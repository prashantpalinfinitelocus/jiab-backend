package utils

import (
	"fmt"
	"strconv"
	"strings"
)

type SuccessResponse struct {
	Success bool        `json:"success"`
	Status  int         `json:"status"`
	Data    interface{} `json:"data"`
}

func TitleWithAmount(title string, amount int) string {
	newStr := strings.Replace(title, "%", fmt.Sprintf("%d", amount), 1)
	return newStr
}

func NumberWithCountryCode(phoneNumber string) string {
	if len(phoneNumber) == 10 {
		return "91" + phoneNumber
	}
	return phoneNumber
}

// EscapeNewlinesInJSON escapes newlines in JSON string literals
func EscapeNewlinesInJSON(jsonStr string) string {
	var result strings.Builder
	inString := false
	escaped := false
	
	for i := 0; i < len(jsonStr); i++ {
		char := jsonStr[i]
		
		if escaped {
			result.WriteByte(char)
			escaped = false
			continue
		}
		
		if char == '\\' {
			escaped = true
			result.WriteByte(char)
			continue
		}
		
		if char == '"' {
			inString = !inString
			result.WriteByte(char)
			continue
		}
		
		if inString && char == '\n' {
			result.WriteString("\\n")
		} else {
			result.WriteByte(char)
		}
	}
	
	return result.String()
}

// SanitizeForLog strips CR/LF from untrusted input before it's written to logs,
// so a value like a phone number, URL path or request field can't be used to
// forge fake extra log lines or corrupt line-oriented log parsing (CWE-117 log
// injection). It does not escape any other character - this is for log
// hygiene, not for HTML/SQL/shell contexts.
func SanitizeForLog(s string) string {
	s = strings.ReplaceAll(s, "\n", "")
	s = strings.ReplaceAll(s, "\r", "")
	return s
}

// ParseIntDefault parses a string to int with a default value if parsing fails
func ParseIntDefault(str string, defaultValue int) int {
	if str == "" {
		return defaultValue
	}

	value, err := strconv.Atoi(str)
	if err != nil {
		return defaultValue
	}

	return value
}
