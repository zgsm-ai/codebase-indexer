package database

import (
	"database/sql"
	"fmt"
)

// ExecuteInTransaction 在事务中执行函数，自动处理提交和回滚（遵循 DRY 原则）
//
// 此函数封装了常见的事务处理模式：
// - 开启事务
// - 执行业务逻辑
// - 发生错误时回滚
// - 成功时提交
// - 处理 panic 确保回滚
//
// 示例:
//
//	err := ExecuteInTransaction(db, func(tx *sql.Tx) error {
//	    _, err := tx.Exec("INSERT INTO ...")
//	    return err
//	})
func ExecuteInTransaction(db DatabaseManager, fn func(tx *sql.Tx) error) error {
	tx, err := db.BeginTransaction()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// 处理 panic 确保回滚
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p) // 重新抛出 panic
		}
	}()

	// 执行业务逻辑
	if err := fn(tx); err != nil {
		// 回滚失败也要记录
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("transaction failed: %w, rollback error: %v", err, rbErr)
		}
		return err
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
