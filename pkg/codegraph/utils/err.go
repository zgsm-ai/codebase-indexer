package utils

const maxLen = 200

// TruncateError 截断错误信息，避免过长
func TruncateError(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	if len(msg) <= maxLen {
		return msg
	}
	return msg[:maxLen] + "..."
}
