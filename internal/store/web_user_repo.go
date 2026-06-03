package store

import (
	"database/sql"
	"fmt"
	"time"

	"gitee.com/jiuhuidalan1/goproxy/internal/config"
	"golang.org/x/crypto/bcrypt"
)

type WebUser struct {
	Username      string
	Password      string
	MustChangePwd bool
}

func (s *Store) GetWebUser(username string) (*WebUser, error) {
	var u WebUser
	var mustChange int
	err := s.db.QueryRow(
		"SELECT username, password, must_change_pwd FROM web_users WHERE username = ?",
		username,
	).Scan(&u.Username, &u.Password, &mustChange)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("查询面板用户失败: %w", err)
	}
	u.MustChangePwd = mustChange == 1
	return &u, nil
}

func (s *Store) ListWebUsers() ([]WebUser, error) {
	rows, err := s.db.Query("SELECT username, password, must_change_pwd FROM web_users ORDER BY id")
	if err != nil {
		return nil, fmt.Errorf("列出面板用户失败: %w", err)
	}
	defer rows.Close()
	var users []WebUser
	for rows.Next() {
		var u WebUser
		var mustChange int
		if err := rows.Scan(&u.Username, &u.Password, &mustChange); err != nil {
			return nil, err
		}
		u.MustChangePwd = mustChange == 1
		users = append(users, u)
	}
	return users, nil
}

func (s *Store) CreateWebUser(username, password string, mustChangePwd bool) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("生成密码哈希失败: %w", err)
	}
	must := 0
	if mustChangePwd {
		must = 1
	}
	_, err = s.db.Exec(
		"INSERT INTO web_users (username, password, must_change_pwd) VALUES (?, ?, ?)",
		username, string(hash), must,
	)
	if err != nil {
		return fmt.Errorf("创建面板用户失败: %w", err)
	}
	return nil
}

func (s *Store) UpdateWebUserPassword(username, password string, mustChangePwd bool) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("生成密码哈希失败: %w", err)
	}
	must := 0
	if mustChangePwd {
		must = 1
	}
	res, err := s.db.Exec(
		"UPDATE web_users SET password = ?, must_change_pwd = ?, updated_at = ? WHERE username = ?",
		string(hash), must, time.Now().Format(time.RFC3339), username,
	)
	if err != nil {
		return fmt.Errorf("更新面板用户密码失败: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("面板用户 %q 不存在", username)
	}
	return nil
}

func (s *Store) VerifyWebUser(username, password string) (*WebUser, error) {
	u, err := s.GetWebUser(username)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, fmt.Errorf("用户名或密码错误")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password)); err != nil {
		return nil, fmt.Errorf("用户名或密码错误")
	}
	return u, nil
}

func (s *Store) GetJWTSecret() (string, error) {
	var secret string
	err := s.db.QueryRow("SELECT jwt_secret FROM web_settings WHERE id = 1").Scan(&secret)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("读取 JWT 密钥失败: %w", err)
	}
	return secret, nil
}

func (s *Store) GetJWTExpireHours() (int, error) {
	var hours int
	err := s.db.QueryRow("SELECT jwt_expire_hours FROM web_settings WHERE id = 1").Scan(&hours)
	if err == sql.ErrNoRows {
		return 24, nil
	}
	if err != nil {
		return 24, fmt.Errorf("读取 JWT 过期时间失败: %w", err)
	}
	return hours, nil
}

func (s *Store) UpdateJWTSecret(secret string) error {
	_, err := s.db.Exec(
		"UPDATE web_settings SET jwt_secret = ?, updated_at = ? WHERE id = 1",
		secret, time.Now().Format(time.RFC3339),
	)
	return err
}

func (s *Store) InitDefaultWebSettings(jwtSecret string, expireHours int) error {
	_, err := s.db.Exec(
		"INSERT OR IGNORE INTO web_settings (id, jwt_secret, jwt_expire_hours) VALUES (1, ?, ?)",
		jwtSecret, expireHours,
	)
	return err
}

func (s *Store) GetWebCredentialsForConfig() (string, string, string, int, error) {
	var username, password, jwtSecret string
	var expireHours int
	u, err := s.ListWebUsers()
	if err != nil {
		return "", "", "", 24, err
	}
	if len(u) > 0 {
		username = u[0].Username
		password = u[0].Password
	}
	jwtSecret, err = s.GetJWTSecret()
	if err != nil {
		return "", "", "", 24, err
	}
	expireHours, err = s.GetJWTExpireHours()
	if err != nil {
		return "", "", "", 24, err
	}
	return username, password, jwtSecret, expireHours, nil
}

func (s *Store) FillWebConfig(cfg *config.Config) {
	username, password, jwtSecret, expireHours, err := s.GetWebCredentialsForConfig()
	if err != nil {
		return
	}
	cfg.Web.Username = username
	cfg.Web.Password = password
	cfg.Web.JWTSecret = jwtSecret
	cfg.Web.JWTExpireHours = expireHours
}

func (s *Store) FillAuthUsers(cfg *config.Config) {
	users, err := s.ListAuthUsers()
	if err != nil {
		return
	}
	cfg.Auth.Users = users
}

func (s *Store) FillActiveRoute(cfg *config.Config) {
	active, err := s.GetActiveRouteFileName()
	if err != nil {
		return
	}
	if active != "" {
		cfg.Route.ActiveFile = active
	}
}
