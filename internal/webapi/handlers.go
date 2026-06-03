package webapi

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"gitee.com/jiuhuidalan1/goproxy/internal/config"
)

type handlers struct {
	app *WebApp
}

func newHandlers(app *WebApp) *handlers {
	return &handlers{app: app}
}

func (h *handlers) login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式无效")
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "用户名和密码不能为空")
		return
	}

	token, err := h.app.Auth().Authenticate(req.Username, req.Password)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	mustChangePwd := h.app.MustChangePwd(req.Username)
	expireHours := h.app.GetJWTExpireHours()
	expiresAt := time.Now().Add(time.Duration(expireHours) * time.Hour)
	writeJSON(w, http.StatusOK, loginResponse{
		Token:         token,
		ExpiresAt:     expiresAt.Format(time.RFC3339),
		Username:      req.Username,
		MustChangePwd: mustChangePwd,
	})
}

func (h *handlers) checkAuth(w http.ResponseWriter, r *http.Request) {
	username := r.Header.Get("X-User-Name")
	mustChangePwd := false
	if username != "" {
		mustChangePwd = h.app.MustChangePwd(username)
	}
	writeJSON(w, http.StatusOK, checkResponse{
		Valid:         true,
		Username:      username,
		MustChangePwd: mustChangePwd,
	})
}

func (h *handlers) changePassword(w http.ResponseWriter, r *http.Request) {
	username := r.Header.Get("X-User-Name")
	if username == "" {
		writeError(w, http.StatusUnauthorized, "未提供用户信息")
		return
	}

	var req changePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式无效")
		return
	}
	if req.OldPassword == "" || req.NewPassword == "" {
		writeError(w, http.StatusBadRequest, "旧密码和新密码不能为空")
		return
	}
	if len(req.NewPassword) < 6 {
		writeError(w, http.StatusBadRequest, "新密码长度不能少于 6 位")
		return
	}
	if req.OldPassword == req.NewPassword {
		writeError(w, http.StatusBadRequest, "新密码不能与旧密码相同")
		return
	}

	token, expiresAt, err := h.app.ChangePassword(username, req.OldPassword, req.NewPassword)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, changePasswordResponse{
		Token:     token,
		ExpiresAt: expiresAt.Format(time.RFC3339),
	})
}

func (h *handlers) getConfig(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.app.GetConfig())
}

func (h *handlers) saveConfig(w http.ResponseWriter, r *http.Request) {
	var cfg config.Config
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		writeError(w, http.StatusBadRequest, "配置格式无效")
		return
	}
	if err := h.app.SaveConfig(cfg); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, nil)
}

func (h *handlers) getServerStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.app.GetServerStatus())
}

func (h *handlers) startServer(w http.ResponseWriter, r *http.Request) {
	if err := h.app.StartServer(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, h.app.GetServerStatus())
}

func (h *handlers) stopServer(w http.ResponseWriter, r *http.Request) {
	if err := h.app.StopServer(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, h.app.GetServerStatus())
}

func (h *handlers) getStats(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.app.GetStats())
}

func (h *handlers) getActiveConnections(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.app.GetActiveConnections())
}

func (h *handlers) getRecentLogs(w http.ResponseWriter, r *http.Request) {
	n := 100
	if s := r.URL.Query().Get("n"); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v > 0 {
			n = v
		}
	}
	writeJSON(w, http.StatusOK, h.app.GetRecentLogs(n))
}

func (h *handlers) clearLogs(w http.ResponseWriter, r *http.Request) {
	if err := h.app.ClearLogs(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, nil)
}

func (h *handlers) setAuthEnabled(w http.ResponseWriter, r *http.Request) {
	var req authEnabledRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式无效")
		return
	}
	if err := h.app.SetAuthEnabled(req.Enabled); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, nil)
}

func (h *handlers) addUser(w http.ResponseWriter, r *http.Request) {
	var req createUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式无效")
		return
	}
	if err := h.app.AddUser(req.Username, req.Password); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, nil)
}

func (h *handlers) removeUser(w http.ResponseWriter, r *http.Request) {
	username := r.PathValue("username")
	if username == "" {
		writeError(w, http.StatusBadRequest, "用户名不能为空")
		return
	}
	if err := h.app.RemoveUser(username); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, nil)
}

func (h *handlers) resetUserPassword(w http.ResponseWriter, r *http.Request) {
	username := r.PathValue("username")
	if username == "" {
		writeError(w, http.StatusBadRequest, "用户名不能为空")
		return
	}
	var req resetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式无效")
		return
	}
	if err := h.app.ResetUserPassword(username, req.Password); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, nil)
}

func (h *handlers) listRouteFiles(w http.ResponseWriter, r *http.Request) {
	files, err := h.app.ListRouteFiles()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, files)
}

func (h *handlers) loadRouteFile(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	set, err := h.app.LoadRouteFile(name)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, set)
}

func (h *handlers) saveRouteFile(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	var set config.RouteRuleSet
	if err := json.NewDecoder(r.Body).Decode(&set); err != nil {
		writeError(w, http.StatusBadRequest, "规则格式无效")
		return
	}
	if err := h.app.SaveRouteFile(name, set); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, nil)
}

func (h *handlers) createRouteFile(w http.ResponseWriter, r *http.Request) {
	var req createRouteFileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式无效")
		return
	}
	if err := h.app.CreateRouteFile(req.Name); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, nil)
}

func (h *handlers) deleteRouteFile(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if err := h.app.DeleteRouteFile(name); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, nil)
}

func (h *handlers) setActiveRouteFile(w http.ResponseWriter, r *http.Request) {
	var req setActiveRouteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式无效")
		return
	}
	if err := h.app.SetActiveRouteFile(req.Name); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, nil)
}

func (h *handlers) getLocalIPs(w http.ResponseWriter, r *http.Request) {
	ips, err := h.app.GetLocalIPAddresses()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, ips)
}

func (h *handlers) getNetworkInterfaces(w http.ResponseWriter, r *http.Request) {
	ifaces, err := h.app.GetNetworkInterfaces()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, ifaces)
}

func (h *handlers) wsEndpoint(w http.ResponseWriter, r *http.Request) {
	serveWS(h.app.Hub(), w, r)
}

func (h *handlers) sseEndpoint(w http.ResponseWriter, r *http.Request) {
	h.app.serveSSE(w, r)
}

func RegisterRoutes(mux *http.ServeMux, app *WebApp) {
	h := newHandlers(app)
	auth := app.Auth()

	mux.HandleFunc("POST /api/v1/auth/login", h.login)
	mux.HandleFunc("GET /api/v1/auth/check", authMiddleware(auth, h.checkAuth))
	mux.HandleFunc("PUT /api/v1/auth/change-password", authMiddleware(auth, h.changePassword))

	protected := []struct {
		method  string
		pattern string
		handler http.HandlerFunc
	}{
		{"GET", "/api/v1/config", h.getConfig},
		{"PUT", "/api/v1/config", h.saveConfig},
		{"GET", "/api/v1/server/status", h.getServerStatus},
		{"POST", "/api/v1/server/start", h.startServer},
		{"POST", "/api/v1/server/stop", h.stopServer},
		{"GET", "/api/v1/server/stats", h.getStats},
		{"GET", "/api/v1/server/connections", h.getActiveConnections},
		{"GET", "/api/v1/logs", h.getRecentLogs},
		{"POST", "/api/v1/logs/clear", h.clearLogs},
		{"PUT", "/api/v1/auth/enabled", h.setAuthEnabled},
		{"POST", "/api/v1/auth/users", h.addUser},
		{"DELETE", "/api/v1/auth/users/{username}", h.removeUser},
		{"PUT", "/api/v1/auth/users/{username}/password", h.resetUserPassword},
		{"GET", "/api/v1/routes/files", h.listRouteFiles},
		{"GET", "/api/v1/routes/files/{name}", h.loadRouteFile},
		{"PUT", "/api/v1/routes/files/{name}", h.saveRouteFile},
		{"POST", "/api/v1/routes/files", h.createRouteFile},
		{"DELETE", "/api/v1/routes/files/{name}", h.deleteRouteFile},
		{"PUT", "/api/v1/routes/active", h.setActiveRouteFile},
		{"GET", "/api/v1/platform/ips", h.getLocalIPs},
		{"GET", "/api/v1/platform/interfaces", h.getNetworkInterfaces},
	}

	for _, route := range protected {
		mux.HandleFunc(route.method+" "+route.pattern, authMiddleware(auth, route.handler))
	}

	mux.HandleFunc("/ws", wsAuthMiddleware(auth, h.wsEndpoint))
	mux.HandleFunc("GET /api/v1/events", wsAuthMiddleware(auth, h.sseEndpoint))
}
