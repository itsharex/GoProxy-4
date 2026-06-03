package store

import (
	"fmt"
	"time"

	"gitee.com/jiuhuidalan1/goproxy/internal/config"
	"golang.org/x/crypto/bcrypt"
)

func (s *Store) ListAuthUsers() ([]config.AuthUser, error) {
	rows, err := s.db.Query("SELECT username, password FROM auth_users ORDER BY id")
	if err != nil {
		return nil, fmt.Errorf("列出鉴权用户失败: %w", err)
	}
	defer rows.Close()
	var users []config.AuthUser
	for rows.Next() {
		var u config.AuthUser
		if err := rows.Scan(&u.Username, &u.Password); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

func (s *Store) AddAuthUser(username, passwordHash string) error {
	if _, err := bcrypt.Cost([]byte(passwordHash)); err != nil {
		return fmt.Errorf("密码格式无效，必须是 bcrypt 哈希")
	}
	_, err := s.db.Exec(
		"INSERT INTO auth_users (username, password) VALUES (?, ?)",
		username, passwordHash,
	)
	if err != nil {
		return fmt.Errorf("添加鉴权用户失败: %w", err)
	}
	return nil
}

func (s *Store) RemoveAuthUser(username string) error {
	res, err := s.db.Exec("DELETE FROM auth_users WHERE username = ?", username)
	if err != nil {
		return fmt.Errorf("删除鉴权用户失败: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("用户 %q 不存在", username)
	}
	return nil
}

func (s *Store) UpdateAuthUserPassword(username, passwordHash string) error {
	if _, err := bcrypt.Cost([]byte(passwordHash)); err != nil {
		return fmt.Errorf("密码格式无效，必须是 bcrypt 哈希")
	}
	res, err := s.db.Exec(
		"UPDATE auth_users SET password = ?, updated_at = ? WHERE username = ?",
		passwordHash, time.Now().Format(time.RFC3339), username,
	)
	if err != nil {
		return fmt.Errorf("更新鉴权用户密码失败: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("用户 %q 不存在", username)
	}
	return nil
}
