package store

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"gitee.com/jiuhuidalan1/goproxy/internal/config"
	"gopkg.in/yaml.v3"
)

func (s *Store) SeedDefaults() error {
	if !s.IsFreshDB() {
		return nil
	}

	secret := generateRandomSecret()

	if err := s.InitDefaultWebSettings(secret, 24); err != nil {
		return fmt.Errorf("初始化 Web 设置失败: %w", err)
	}
	if err := s.CreateWebUser("admin", "admin123", true); err != nil {
		return fmt.Errorf("创建默认面板用户失败: %w", err)
	}
	if err := s.EnsureDefaultRoute(); err != nil {
		return fmt.Errorf("创建默认路由规则失败: %w", err)
	}
	if err := s.MarkInitialized(); err != nil {
		return fmt.Errorf("标记初始化完成失败: %w", err)
	}
	log.Println("已创建默认配置: 用户名 admin, 密码 admin123")
	return nil
}

func (s *Store) ImportFromYAML(configPath string) error {
	if !s.IsFreshDB() {
		return nil
	}

	_, err := os.Stat(configPath)
	if os.IsNotExist(err) {
		return s.SeedDefaults()
	}
	if err != nil {
		return s.SeedDefaults()
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return s.SeedDefaults()
	}

	var raw map[string]interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return s.SeedDefaults()
	}

	cfg := config.Default()
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return s.SeedDefaults()
	}

	secret := cfg.Web.JWTSecret
	if secret == "" {
		secret = generateRandomSecret()
	}
	if err := s.InitDefaultWebSettings(secret, cfg.Web.JWTExpireHours); err != nil {
		return fmt.Errorf("迁移 Web 设置失败: %w", err)
	}

	if cfg.Web.Username != "" && cfg.Web.Password != "" {
		if err := s.CreateWebUser(cfg.Web.Username, "admin_temp_placeholder", false); err != nil {
			return fmt.Errorf("迁移面板用户失败: %w", err)
		}
		if err := s.dbRawUpdateWebUserPassword(cfg.Web.Username, cfg.Web.Password); err != nil {
			return fmt.Errorf("迁移面板用户密码失败: %w", err)
		}
	} else {
		if err := s.CreateWebUser("admin", "admin123", false); err != nil {
			return fmt.Errorf("创建默认面板用户失败: %w", err)
		}
	}

	for _, u := range cfg.Auth.Users {
		if err := s.AddAuthUser(u.Username, u.Password); err != nil {
			log.Printf("迁移鉴权用户 %s 失败: %v", u.Username, err)
		}
	}

	configDir := filepath.Dir(configPath)
	if err := s.importRouteFiles(configDir, cfg.Route.ActiveFile); err != nil {
		log.Printf("迁移路由规则失败: %v", err)
	}

	if err := s.MarkInitialized(); err != nil {
		return fmt.Errorf("标记迁移完成失败: %w", err)
	}
	log.Println("已从 YAML 配置迁移到 SQLite 数据库")
	return nil
}

func (s *Store) importRouteFiles(dir string, activeFile string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return s.EnsureDefaultRoute()
	}

	hasActive := false
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".rule") {
			continue
		}
		if err := config.ValidateRouteFileName(entry.Name()); err != nil {
			continue
		}
		filePath := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			log.Printf("读取规则文件 %s 失败: %v", entry.Name(), err)
			continue
		}
		var set config.RouteRuleSet
		if err := yaml.Unmarshal(data, &set); err != nil {
			log.Printf("解析规则文件 %s 失败: %v", entry.Name(), err)
			continue
		}
		if set.UpdatedAt == "" {
			if info, err := entry.Info(); err == nil {
				set.UpdatedAt = info.ModTime().Format("2006-01-02T15:04:05Z07:00")
			}
		}

		name := strings.TrimSuffix(entry.Name(), ".rule")
		isActive := 0
		if entry.Name() == activeFile {
			isActive = 1
			hasActive = true
		}
		now := "2006-01-02T15:04:05Z07:00"
		_, err = s.db.Exec(
			"INSERT INTO route_rule_sets (name, file_name, version, description, is_active, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
			name, entry.Name(), set.Version, set.Description, isActive, set.UpdatedAt,
		)
		if err != nil {
			log.Printf("插入规则集 %s 失败: %v", entry.Name(), err)
			continue
		}
		_ = now

		var setID int64
		err = s.db.QueryRow("SELECT id FROM route_rule_sets WHERE file_name = ?", entry.Name()).Scan(&setID)
		if err != nil {
			continue
		}
		for i, r := range set.Rules {
			importRouteRule(s, setID, r, i)
		}
	}

	if !hasActive {
		s.db.Exec("UPDATE route_rule_sets SET is_active = 1 WHERE file_name = 'default.rule'")
	}

	return s.EnsureDefaultRoute()
}

func importRouteRule(s *Store, setID int64, r config.RouteRule, sortOrder int) {
	protocolsJSON, _ := json.Marshal(r.Protocols)
	targetsJSON, _ := json.Marshal(r.Targets)
	mode := r.Outbound.Mode
	if mode == "" {
		mode = "default"
	}
	s.db.Exec(
		`INSERT INTO route_rules
		 (rule_set_id, rule_id, name, enabled, priority, match_type, protocols, targets,
		  outbound_mode, outbound_local_ip, outbound_interface, remark, sort_order)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		setID, r.ID, r.Name, r.Enabled, r.Priority, r.MatchType,
		string(protocolsJSON), string(targetsJSON),
		mode, r.Outbound.LocalIP, r.Outbound.Interface, r.Remark, sortOrder,
	)
}

func (s *Store) dbRawUpdateWebUserPassword(username, passwordHash string) error {
	_, err := s.db.Exec("UPDATE web_users SET password = ? WHERE username = ?", passwordHash, username)
	return err
}

func generateRandomSecret() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}
