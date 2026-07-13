package web

import (
	"html/template"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"
)

type Config struct {
	ServiceName string
	Port        string
	JobName     string
	BuildNumber string
	GitCommit   string
	BuildURL    string
	Environment string
}

func ConfigFromEnv() Config {
	return Config{
		ServiceName: env("SERVICE_NAME", "hq-project-demo"),
		Port:        env("PORT", "8080"),
		JobName:     env("JOB_NAME", env("BUILD_JOB_NAME", "local-dev")),
		BuildNumber: env("BUILD_NUMBER", "0"),
		GitCommit:   env("GIT_COMMIT", "unknown"),
		BuildURL:    env("BUILD_URL", ""),
		Environment: env("APP_ENV", "dev"),
	}
}

func (c Config) Version() string {
	shortCommit := c.GitCommit
	if len(shortCommit) > 12 {
		shortCommit = shortCommit[:12]
	}
	return c.JobName + "#" + c.BuildNumber + "@" + shortCommit
}

type Server struct {
	cfg    Config
	logger *slog.Logger
	mux    *http.ServeMux
}

func NewServer(cfg Config, logger *slog.Logger) http.Handler {
	s := &Server{
		cfg:    cfg,
		logger: logger,
		mux:    http.NewServeMux(),
	}
	s.routes()
	return securityHeaders(s.mux)
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /", s.handleIndex)
	s.mux.HandleFunc("POST /logs", s.handleLogSubmit)
	s.mux.HandleFunc("GET /healthz", s.handleHealthz)
	s.mux.HandleFunc("GET /readyz", s.handleHealthz)
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	data := map[string]string{
		"ServiceName": s.cfg.ServiceName,
		"Version":     s.cfg.Version(),
		"JobName":     s.cfg.JobName,
		"BuildNumber": s.cfg.BuildNumber,
		"GitCommit":   s.cfg.GitCommit,
		"BuildURL":    s.cfg.BuildURL,
		"Environment": s.cfg.Environment,
	}
	if err := pageTemplate.Execute(w, data); err != nil {
		s.logger.Error("render index failed", "error", err)
	}
}

func (s *Server) handleLogSubmit(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}

	message := strings.TrimSpace(r.FormValue("message"))
	if message == "" {
		http.Error(w, "message is required", http.StatusBadRequest)
		return
	}
	if len(message) > 512 {
		http.Error(w, "message is too long", http.StatusBadRequest)
		return
	}

	s.logger.Info("user submitted demo log",
		"event", "demo_log_submitted",
		"service", s.cfg.ServiceName,
		"version", s.cfg.Version(),
		"environment", s.cfg.Environment,
		"remote_addr", r.RemoteAddr,
		"message", message,
	)

	http.Redirect(w, r, "/?submitted=1", http.StatusSeeOther)
}

func (s *Server) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

func env(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "same-origin")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'unsafe-inline' 'self'; form-action 'self'")
		next.ServeHTTP(w, r)
	})
}

var pageTemplate = template.Must(template.New("index").Funcs(template.FuncMap{
	"now": func() string { return time.Now().Format(time.RFC3339) },
}).Parse(`<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{.ServiceName}} | CI/CD Demo</title>
  <style>
    :root { color-scheme: light dark; font-family: Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; }
    body { margin: 0; min-height: 100vh; background: #f5f7fb; color: #172033; display: grid; place-items: center; }
    main { width: min(920px, calc(100vw - 32px)); background: #fff; border: 1px solid #d8dee9; border-radius: 8px; box-shadow: 0 16px 40px rgba(21, 34, 50, .08); overflow: hidden; }
    header { padding: 28px 32px; background: #172033; color: #fff; }
    h1 { margin: 0 0 8px; font-size: clamp(26px, 4vw, 38px); letter-spacing: 0; }
    .version { font-size: 15px; color: #c9d6ea; overflow-wrap: anywhere; }
    section { padding: 28px 32px; }
    .grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(220px, 1fr)); gap: 12px; margin-bottom: 26px; }
    .metric { border: 1px solid #e0e6ef; border-radius: 8px; padding: 14px 16px; background: #fbfcff; }
    .label { color: #687386; font-size: 13px; margin-bottom: 5px; }
    .value { font-weight: 700; overflow-wrap: anywhere; }
    form { display: grid; gap: 12px; }
    textarea { width: 100%; min-height: 108px; box-sizing: border-box; resize: vertical; border: 1px solid #cfd8e6; border-radius: 8px; padding: 14px; font: inherit; }
    button { width: fit-content; border: 0; border-radius: 8px; padding: 12px 18px; background: #1264a3; color: #fff; font-weight: 700; cursor: pointer; }
    button:hover { background: #0d568d; }
    footer { padding: 14px 32px; background: #f8fafc; color: #687386; font-size: 13px; border-top: 1px solid #e5eaf2; }
    @media (prefers-color-scheme: dark) {
      body { background: #0f1724; color: #e8edf5; }
      main, .metric { background: #151f2e; border-color: #2c3a50; }
      header { background: #0b1220; }
      textarea { background: #0f1724; color: #e8edf5; border-color: #34445d; }
      footer { background: #101827; border-color: #2c3a50; color: #aab6c8; }
    }
  </style>
</head>
<body>
  <main>
    <header>
      <h1>{{.ServiceName}}</h1>
      <div class="version">当前版本：{{.Version}}</div>
    </header>
    <section>
      <div class="grid">
        <div class="metric"><div class="label">运行环境</div><div class="value">{{.Environment}}</div></div>
        <div class="metric"><div class="label">Jenkins Job</div><div class="value">{{.JobName}}</div></div>
        <div class="metric"><div class="label">Build Number</div><div class="value">{{.BuildNumber}}</div></div>
        <div class="metric"><div class="label">Git Commit</div><div class="value">{{.GitCommit}}</div></div>
      </div>
      <form method="post" action="/logs">
        <label for="message"><strong>提交一条演示日志</strong></label>
        <textarea id="message" name="message" maxlength="512" required placeholder="输入内容后提交，应用会输出 JSON 日志，Promtail 会采集到 Loki。"></textarea>
        <button type="submit">提交日志</button>
      </form>
    </section>
    <footer>Health: /healthz · Ready: /readyz · Server time: {{now}}</footer>
  </main>
</body>
</html>`))
