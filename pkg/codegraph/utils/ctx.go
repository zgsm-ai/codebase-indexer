package utils

import "context"

// CheckContext 检查上下文是否已取消
func CheckContext(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}
