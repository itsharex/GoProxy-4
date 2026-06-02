package config

import (
	"errors"
	"fmt"
	"net"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// Validate checks the complete runtime configuration.
func Validate(cfg Config) error {
	if !cfg.Server.SOCKS5.Enabled && !cfg.Server.HTTP.Enabled {
		return errors.New("请至少开启一种入站协议：SOCKS5 或 HTTP CONNECT")
	}

	if cfg.Server.SOCKS5.Enabled {
		if err := validateProtocol("socks5", cfg.Server.SOCKS5); err != nil {
			return err
		}
	}
	if cfg.Server.HTTP.Enabled {
		if err := validateProtocol("http", cfg.Server.HTTP); err != nil {
			return err
		}
	}

	if cfg.Relay.DialTimeoutSec <= 0 {
		return errors.New("目标连接超时时间必须大于 0 秒")
	}
	if cfg.Relay.ReadTimeoutSec <= 0 {
		return errors.New("握手/读写超时时间必须大于 0 秒")
	}
	if cfg.Relay.MaxConnections <= 0 {
		return errors.New("最大并发连接数必须大于 0")
	}
	if cfg.Relay.KeepAliveSec <= 0 {
		return errors.New("Keep-Alive 间隔必须大于 0 秒")
	}
	if err := validateLog(cfg.Log); err != nil {
		return err
	}
	if err := validateUI(cfg.UI); err != nil {
		return err
	}
	if err := validateAuth(cfg.Auth); err != nil {
		return err
	}
	if err := ValidateRouteFileName(cfg.Route.ActiveFile); err != nil {
		return fmt.Errorf("当前规则文件无效: %w", err)
	}

	if cfg.Web.Enabled {
		if err := validateWeb(cfg.Web); err != nil {
			return err
		}
	}

	if cfg.Server.SOCKS5.Enabled && cfg.Server.HTTP.Enabled {
		if cfg.Server.SOCKS5.Host == cfg.Server.HTTP.Host && cfg.Server.SOCKS5.Port == cfg.Server.HTTP.Port {
			return errors.New("SOCKS5 和 HTTP CONNECT 不能使用完全相同的监听地址和端口")
		}
	}

	return nil
}

// ValidateRouteFileName checks a .rule file name without allowing path traversal.
func ValidateRouteFileName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("规则文件名称不能为空")
	}
	if !strings.HasSuffix(name, ".rule") {
		return errors.New("规则文件名称必须以 .rule 结尾")
	}
	if strings.Contains(name, "..") || strings.ContainsAny(name, `/\`) {
		return errors.New("规则文件名称不能包含路径分隔符")
	}
	if strings.EqualFold(name, "config.yaml") {
		return errors.New("规则文件名称不能为 config.yaml")
	}
	base := strings.TrimSuffix(name, ".rule")
	if base == "" {
		return errors.New("规则文件名称不能为空")
	}
	for _, r := range base {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' {
			continue
		}
		return errors.New("规则文件名称只能包含字母、数字、'-' 和 '_'")
	}
	return nil
}

func validateAuth(auth AuthConfig) error {
	if !auth.Enabled {
		return nil
	}
	if len(auth.Users) == 0 {
		return errors.New("开启认证前，请先新增至少一个认证用户")
	}

	seen := make(map[string]struct{}, len(auth.Users))
	for index, user := range auth.Users {
		username := strings.TrimSpace(user.Username)
		if username == "" {
			return fmt.Errorf("第 %d 个认证用户缺少用户名", index+1)
		}
		if _, exists := seen[username]; exists {
			return fmt.Errorf("认证用户名 %q 重复，请使用唯一用户名", username)
		}
		seen[username] = struct{}{}

		if strings.TrimSpace(user.Password) == "" {
			return fmt.Errorf("第 %d 个认证用户缺少密码", index+1)
		}
		if _, err := bcrypt.Cost([]byte(user.Password)); err != nil {
			return fmt.Errorf("第 %d 个认证用户的密码格式无效，请通过页面新增或重置密码", index+1)
		}
	}
	return nil
}

func validateProtocol(name string, protocol ProtocolConfig) error {
	label := protocolLabel(name)
	if protocol.Host == "" {
		return fmt.Errorf("%s 监听地址不能为空", label)
	}
	if ip := net.ParseIP(protocol.Host); ip == nil {
		return fmt.Errorf("%s 监听地址必须是 IP 地址，例如 0.0.0.0 或 127.0.0.1", label)
	}
	if protocol.Port < 1 || protocol.Port > 65535 {
		return fmt.Errorf("%s 端口必须在 1 到 65535 之间", label)
	}
	return nil
}

func protocolLabel(name string) string {
	switch name {
	case "socks5":
		return "SOCKS5"
	case "http":
		return "HTTP CONNECT"
	default:
		return name
	}
}

func validateLog(log LogConfig) error {
	switch log.Level {
	case "debug", "info", "warn", "error":
	default:
		return errors.New("日志级别只能选择 debug、info、warn 或 error")
	}
	if log.MaxSizeMB <= 0 {
		return errors.New("日志单文件大小必须大于 0 MB")
	}
	if log.MaxBackups < 0 {
		return errors.New("日志备份数量不能小于 0")
	}
	switch log.Output {
	case "file", "console", "both":
	default:
		return errors.New("日志输出方式只能选择文件、控制台或两者都输出")
	}
	return nil
}

func validateUI(ui UIConfig) error {
	switch ui.Theme {
	case "light", "dark", "auto":
	default:
		return errors.New("主题只能选择浅色、深色或跟随系统")
	}
	if ui.Language == "" {
		return errors.New("界面语言不能为空")
	}
	return nil
}

func validateWeb(web WebConfig) error {
	if web.Listen == "" {
		return errors.New("Web 管理面板监听地址不能为空")
	}
	host, port, err := net.SplitHostPort(web.Listen)
	if err != nil {
		return fmt.Errorf("Web 管理面板监听地址格式无效，应为 host:port: %w", err)
	}
	if host == "" {
		return errors.New("Web 管理面板监听主机不能为空")
	}
	if p := net.ParseIP(host); host != "0.0.0.0" && host != "[::]" && host != "::" && p == nil {
		return errors.New("Web 管理面板监听地址必须是有效的 IP 地址")
	}
	portNum := 0
	for _, c := range port {
		if c < '0' || c > '9' {
			return errors.New("Web 管理面板端口必须为数字")
		}
		portNum = portNum*10 + int(c-'0')
	}
	if portNum < 1 || portNum > 65535 {
		return errors.New("Web 管理面板端口必须在 1 到 65535 之间")
	}
	if strings.TrimSpace(web.Username) == "" {
		return errors.New("Web 管理面板用户名不能为空")
	}
	if web.JWTExpireHours <= 0 {
		return errors.New("Web 面板 Token 有效期必须大于 0 小时")
	}
	if web.TLSEnabled {
		if strings.TrimSpace(web.TLSCert) == "" {
			return errors.New("启用 TLS 时证书路径不能为空")
		}
		if strings.TrimSpace(web.TLSKey) == "" {
			return errors.New("启用 TLS 时私钥路径不能为空")
		}
	}
	return nil
}
