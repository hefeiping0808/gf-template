// Package user 提供用户表初始化和管理员账号创建逻辑。
package user

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/gogf/gf/v2/frame/g"
	"golang.org/x/crypto/bcrypt"
)

const createTableSQL = "CREATE TABLE IF NOT EXISTS users (\n" +
	"  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,\n" +
	"  username VARCHAR(64) NOT NULL,\n" +
	"  password_hash VARCHAR(255) NOT NULL,\n" +
	"  `role` VARCHAR(32) NOT NULL DEFAULT 'user',\n" +
	"  status TINYINT UNSIGNED NOT NULL DEFAULT 1,\n" +
	"  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,\n" +
	"  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,\n" +
	"  PRIMARY KEY (id),\n" +
	"  UNIQUE KEY uk_users_username (username),\n" +
	"  KEY idx_users_role_status (`role`, status)\n" +
	") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci"

var usernamePattern = regexp.MustCompile(`^[A-Za-z0-9_.-]{3,64}$`)

// CreateAdmin 确保用户表存在并创建管理员；唯一索引保证并发创建时也不会产生重复用户名。
func CreateAdmin(ctx context.Context, username string, password []byte) error {
	username = strings.TrimSpace(username)
	if !usernamePattern.MatchString(username) {
		return errors.New("用户名需为 3 到 64 位字母、数字、点、下划线或连字符")
	}
	if len(password) < 12 || len(password) > 72 {
		return errors.New("密码长度需为 12 到 72 字节（bcrypt 限制）")
	}
	passwordHash, err := bcrypt.GenerateFromPassword(password, bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("生成管理员密码哈希失败：%w", err)
	}
	if _, err = g.DB().Exec(ctx, createTableSQL); err != nil {
		return fmt.Errorf("确保 users 表存在失败：%w", err)
	}
	if _, err = g.DB().Exec(ctx,
		"INSERT INTO users (username, password_hash, `role`, status) VALUES (?, ?, 'admin', 1)",
		username, string(passwordHash),
	); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "duplicate") {
			return fmt.Errorf("用户名 %q 已存在", username)
		}
		return fmt.Errorf("创建管理员记录失败：%w", err)
	}
	return nil
}
