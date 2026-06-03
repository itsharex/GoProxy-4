package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"gitee.com/jiuhuidalan1/goproxy/internal/config"
)

func (s *Store) EnsureDefaultRoute() error {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM route_rule_sets WHERE file_name = 'default.rule'").Scan(&count)
	if err != nil {
		return fmt.Errorf("检查默认路由规则失败: %w", err)
	}
	if count > 0 {
		return nil
	}
	return s.CreateRouteRuleSet("default.rule")
}

func (s *Store) ListRouteRuleSets() ([]config.RouteFileInfo, error) {
	if err := s.EnsureDefaultRoute(); err != nil {
		return nil, err
	}
	rows, err := s.db.Query(
		"SELECT file_name, is_active, updated_at FROM route_rule_sets ORDER BY file_name",
	)
	if err != nil {
		return nil, fmt.Errorf("列出路由规则集失败: %w", err)
	}
	defer rows.Close()
	var files []config.RouteFileInfo
	for rows.Next() {
		var f config.RouteFileInfo
		var isActive int
		if err := rows.Scan(&f.Name, &isActive, &f.UpdatedAt); err != nil {
			return nil, err
		}
		f.IsActive = isActive == 1
		files = append(files, f)
	}
	return files, nil
}

func (s *Store) LoadRouteRuleSet(fileName string) (config.RouteRuleSet, error) {
	var id int
	var set config.RouteRuleSet
	var version int
	err := s.db.QueryRow(
		"SELECT id, name, version, description, updated_at FROM route_rule_sets WHERE file_name = ?",
		fileName,
	).Scan(&id, &set.Name, &version, &set.Description, &set.UpdatedAt)
	if err == sql.ErrNoRows {
		return config.RouteRuleSet{}, fmt.Errorf("路由规则集 %q 不存在", fileName)
	}
	if err != nil {
		return config.RouteRuleSet{}, fmt.Errorf("加载路由规则集失败: %w", err)
	}
	set.Version = version

	rows, err := s.db.Query(
		`SELECT rule_id, name, enabled, priority, match_type, protocols, targets,
		        outbound_mode, outbound_local_ip, outbound_interface, remark, sort_order
		 FROM route_rules WHERE rule_set_id = ? ORDER BY sort_order, priority DESC`,
		id,
	)
	if err != nil {
		return config.RouteRuleSet{}, fmt.Errorf("加载路由规则失败: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var r config.RouteRule
		var enabled int
		var protocolsJSON, targetsJSON string
		var sortOrder int
		if err := rows.Scan(
			&r.ID, &r.Name, &enabled, &r.Priority, &r.MatchType,
			&protocolsJSON, &targetsJSON,
			&r.Outbound.Mode, &r.Outbound.LocalIP, &r.Outbound.Interface,
			&r.Remark, &sortOrder,
		); err != nil {
			return config.RouteRuleSet{}, err
		}
		r.Enabled = enabled == 1
		_ = json.Unmarshal([]byte(protocolsJSON), &r.Protocols)
		_ = json.Unmarshal([]byte(targetsJSON), &r.Targets)
		if r.Outbound.Mode == "" {
			r.Outbound.Mode = "default"
		}
		set.Rules = append(set.Rules, r)
	}
	return set, nil
}

func (s *Store) SaveRouteRuleSet(fileName string, set config.RouteRuleSet) error {
	if set.UpdatedAt == "" {
		set.UpdatedAt = time.Now().Format(time.RFC3339)
	}

	var setID int
	err := s.db.QueryRow("SELECT id FROM route_rule_sets WHERE file_name = ?", fileName).Scan(&setID)
	if err == sql.ErrNoRows {
		return fmt.Errorf("路由规则集 %q 不存在", fileName)
	}
	if err != nil {
		return fmt.Errorf("查询路由规则集失败: %w", err)
	}

	_, err = s.db.Exec(
		"UPDATE route_rule_sets SET name = ?, version = ?, description = ?, updated_at = ? WHERE id = ?",
		set.Name, set.Version, set.Description, set.UpdatedAt, setID,
	)
	if err != nil {
		return fmt.Errorf("更新路由规则集失败: %w", err)
	}

	_, err = s.db.Exec("DELETE FROM route_rules WHERE rule_set_id = ?", setID)
	if err != nil {
		return fmt.Errorf("清除旧路由规则失败: %w", err)
	}

	for i, r := range set.Rules {
		protocolsJSON, _ := json.Marshal(r.Protocols)
		targetsJSON, _ := json.Marshal(r.Targets)
		mode := r.Outbound.Mode
		if mode == "" {
			mode = "default"
		}
		_, err = s.db.Exec(
			`INSERT INTO route_rules
			 (rule_set_id, rule_id, name, enabled, priority, match_type, protocols, targets,
			  outbound_mode, outbound_local_ip, outbound_interface, remark, sort_order)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			setID, r.ID, r.Name, r.Enabled, r.Priority, r.MatchType,
			string(protocolsJSON), string(targetsJSON),
			mode, r.Outbound.LocalIP, r.Outbound.Interface, r.Remark, i,
		)
		if err != nil {
			return fmt.Errorf("插入路由规则失败: %w", err)
		}
	}
	return nil
}

func (s *Store) CreateRouteRuleSet(fileName string) error {
	if err := config.ValidateRouteFileName(fileName); err != nil {
		return err
	}

	name := strings.TrimSuffix(fileName, ".rule")
	now := time.Now().Format(time.RFC3339)
	defSet := config.DefaultRouteRuleSet()
	defSet.Name = name

	res, err := s.db.Exec(
		"INSERT INTO route_rule_sets (name, file_name, version, description, updated_at) VALUES (?, ?, ?, ?, ?)",
		name, fileName, defSet.Version, defSet.Description, now,
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return fmt.Errorf("路由规则集 %q 已存在", fileName)
		}
		return fmt.Errorf("创建路由规则集失败: %w", err)
	}
	setID, _ := res.LastInsertId()

	for i, r := range defSet.Rules {
		protocolsJSON, _ := json.Marshal(r.Protocols)
		targetsJSON, _ := json.Marshal(r.Targets)
		mode := r.Outbound.Mode
		if mode == "" {
			mode = "default"
		}
		_, err = s.db.Exec(
			`INSERT INTO route_rules
			 (rule_set_id, rule_id, name, enabled, priority, match_type, protocols, targets,
			  outbound_mode, outbound_local_ip, outbound_interface, remark, sort_order)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			setID, r.ID, r.Name, r.Enabled, r.Priority, r.MatchType,
			string(protocolsJSON), string(targetsJSON),
			mode, r.Outbound.LocalIP, r.Outbound.Interface, r.Remark, i,
		)
		if err != nil {
			return fmt.Errorf("插入默认路由规则失败: %w", err)
		}
	}
	return nil
}

func (s *Store) DeleteRouteRuleSet(fileName string) error {
	if fileName == config.DefaultRouteFileName {
		return fmt.Errorf("默认规则文件不能删除")
	}
	res, err := s.db.Exec("DELETE FROM route_rule_sets WHERE file_name = ?", fileName)
	if err != nil {
		return fmt.Errorf("删除路由规则集失败: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("路由规则集 %q 不存在", fileName)
	}
	return nil
}

func (s *Store) SetActiveRouteRuleSet(fileName string) error {
	if err := config.ValidateRouteFileName(fileName); err != nil {
		return err
	}
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM route_rule_sets WHERE file_name = ?", fileName).Scan(&count)
	if err != nil {
		return fmt.Errorf("检查路由规则集失败: %w", err)
	}
	if count == 0 {
		return fmt.Errorf("路由规则集 %q 不存在", fileName)
	}

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("开启事务失败: %w", err)
	}
	_, err = tx.Exec("UPDATE route_rule_sets SET is_active = 0")
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("重置活跃状态失败: %w", err)
	}
	_, err = tx.Exec("UPDATE route_rule_sets SET is_active = 1, updated_at = ? WHERE file_name = ?",
		time.Now().Format(time.RFC3339), fileName)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("设置活跃规则集失败: %w", err)
	}
	return tx.Commit()
}

func (s *Store) GetActiveRouteFileName() (string, error) {
	var name string
	err := s.db.QueryRow("SELECT file_name FROM route_rule_sets WHERE is_active = 1 LIMIT 1").Scan(&name)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("查询活跃规则集失败: %w", err)
	}
	return name, nil
}
