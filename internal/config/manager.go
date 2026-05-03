package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Manager loads and saves YAML configuration files.
type Manager struct {
	path string
}

// NewManager creates a configuration manager for a YAML file path.
func NewManager(path string) *Manager {
	return &Manager{path: path}
}

// Path returns the managed configuration file path.
func (m *Manager) Path() string {
	return m.path
}

// Load reads the YAML config file, applies defaults for missing fields, and validates it.
func (m *Manager) Load() (Config, error) {
	if m.path == "" {
		return Config{}, errors.New("配置文件路径未设置")
	}

	data, err := os.ReadFile(m.path)
	if err != nil {
		return Config{}, err
	}

	cfg := Default()
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("解析配置文件失败: %w", err)
	}

	if err := Validate(cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

// Save validates and writes the YAML config file.
func (m *Manager) Save(cfg Config) error {
	if m.path == "" {
		return errors.New("配置文件路径未设置")
	}
	if err := Validate(cfg); err != nil {
		return err
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("序列化配置文件失败: %w", err)
	}

	dir := filepath.Dir(m.path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("创建配置目录失败: %w", err)
		}
	}

	return m.writeWithBackup(data)
}

func (m *Manager) writeWithBackup(data []byte) error {
	dir := filepath.Dir(m.path)
	if dir == "" {
		dir = "."
	}

	tmp, err := os.CreateTemp(dir, filepath.Base(m.path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("创建临时配置文件失败: %w", err)
	}
	tmpPath := tmp.Name()
	cleanupTmp := true
	defer func() {
		if cleanupTmp {
			_ = os.Remove(tmpPath)
		}
	}()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("写入临时配置文件失败: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("关闭临时配置文件失败: %w", err)
	}
	if err := os.Chmod(tmpPath, 0o600); err != nil {
		return fmt.Errorf("设置临时配置文件权限失败: %w", err)
	}

	existing, err := os.ReadFile(m.path)
	hadExisting := err == nil
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("读取当前配置文件用于备份失败: %w", err)
	}
	if hadExisting {
		if err := os.WriteFile(m.backupPath(), existing, 0o600); err != nil {
			return fmt.Errorf("写入配置备份失败: %w", err)
		}
	}

	if err := os.Rename(tmpPath, m.path); err != nil {
		if hadExisting {
			if removeErr := os.Remove(m.path); removeErr == nil {
				err = os.Rename(tmpPath, m.path)
			}
		}
		if err != nil {
			if hadExisting {
				_ = os.WriteFile(m.path, existing, 0o600)
			}
			return fmt.Errorf("替换配置文件失败: %w", err)
		}
	}

	cleanupTmp = false
	return nil
}

func (m *Manager) backupPath() string {
	return m.path + ".bak"
}
