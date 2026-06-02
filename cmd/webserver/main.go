package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"gitee.com/jiuhuidalan1/goproxy/internal/webapi"
)

func main() {
	configPath := flag.String("config", "config.yaml", "path to YAML config file")
	listenAddr := flag.String("listen", "", "web panel listen address (overrides config)")
	staticDir := flag.String("static", "", "frontend static files directory (default: ./frontend/dist next to executable)")
	writeDefault := flag.Bool("write-default", false, "write the default config and exit")
	flag.Parse()

	if *writeDefault {
		mgr := resolvePath(*configPath)
		if err := os.MkdirAll(filepath.Dir(mgr), 0o755); err != nil {
			log.Fatalf("create config dir: %v", err)
		}
		if err := os.WriteFile(mgr, []byte(defaultConfigContent()), 0o644); err != nil {
			log.Fatalf("write default config: %v", err)
		}
		log.Printf("default config written to %s", mgr)
		return
	}

	cp := resolvePath(*configPath)
	lp := filepath.Join(filepath.Dir(cp), "logs", "proxy-server.log")

	app, err := webapi.NewWebApp(cp, lp)
	if err != nil {
		log.Fatalf("create web app: %v", err)
	}
	defer app.Close()

	mux := http.NewServeMux()
	webapi.RegisterRoutes(mux, app)

	sd := resolveStaticDir(*staticDir)
	spaHandler := newSPAHandler(http.FileServer(http.Dir(sd)), sd)
	mux.Handle("/", spaHandler)

	addr := *listenAddr
	if addr == "" {
		addr = app.GetConfig().Web.Listen
	}
	if addr == "" {
		addr = "0.0.0.0:9090"
	}

	var handler http.Handler = mux
	handler = webapi.SecurityHeaders(handler)
	handler = webapi.Logging(handler)
	handler = webapi.CORS(handler)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	srv := &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	go func() {
		<-ctx.Done()
		log.Print("shutting down web server")
		shutCtx, cancel := context.WithTimeout(context.Background(), 5)
		defer cancel()
		_ = srv.Shutdown(shutCtx)
	}()

	go app.StartStatsLoop(ctx)

	log.Printf("GoProxy web server starting on %s", addr)
	log.Printf("static files served from %s", sd)

	cfg := app.GetConfig()
	if cfg.Web.TLSEnabled && cfg.Web.TLSCert != "" && cfg.Web.TLSKey != "" {
		err = srv.ListenAndServeTLS(cfg.Web.TLSCert, cfg.Web.TLSKey)
	} else {
		err = srv.ListenAndServe()
	}
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("web server error: %v", err)
	}
}

func resolvePath(p string) string {
	if filepath.IsAbs(p) {
		return p
	}
	execPath, err := os.Executable()
	if err != nil {
		return p
	}
	return filepath.Join(filepath.Dir(execPath), p)
}

func resolveStaticDir(override string) string {
	if override != "" {
		if filepath.IsAbs(override) {
			return override
		}
		return resolvePath(override)
	}
	execPath, err := os.Executable()
	if err != nil {
		return "frontend/dist"
	}
	return filepath.Join(filepath.Dir(execPath), "frontend", "dist")
}

func defaultConfigContent() string {
	return `server:
  socks5:
    enabled: true
    host: "0.0.0.0"
    port: 1080
  http:
    enabled: true
    host: "0.0.0.0"
    port: 8080

auth:
  enabled: false
  users: []

relay:
  dial_timeout_sec: 10
  read_timeout_sec: 30
  max_connections: 1000
  keepalive_sec: 15

log:
  level: info
  max_size_mb: 50
  max_backups: 3
  output: both

ui:
  theme: dark
  language: zh-CN
  start_minimized: false
  auto_start_proxy: false
  show_tray_icon: false
  close_to_tray: false
  tray_status_and_ip: false

route:
  enabled: false
  active_file: default.rule

web:
  enabled: true
  listen: "0.0.0.0:9090"
  username: admin
  jwt_expire_hours: 24
  tls_enabled: false
`
}

type spaHandler struct {
	fileServer http.Handler
	root       string
}

func newSPAHandler(fs http.Handler, root string) *spaHandler {
	return &spaHandler{fileServer: fs, root: root}
}

func (s *spaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	filePath := filepath.Join(s.root, path)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		r.URL.Path = "/"
	}
	s.fileServer.ServeHTTP(w, r)
}
