package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"time"
)

const appVersion = "1.1.0"

// canal exibido ao encerrar o programa
const channelURL = "https://www.youtube.com/@xerifetech?sub_confirmation=1"

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "erro:", err)
	// no Windows (subsistema GUI) não há console: registra num arquivo
	if runtime.GOOS == "windows" {
		_ = os.WriteFile(filepath.Join(os.TempDir(), "GeradorAutounattend-erro.txt"), []byte(err.Error()), 0o644)
	}
	os.Exit(1)
}

func main() {
	if err := loadAssets(); err != nil {
		fatal(err)
	}
	args := os.Args[1:]
	if len(args) > 0 && args[0] == "--cli" {
		runCLI(args[1:])
		return
	}
	noOpen := false
	for _, a := range args {
		if a == "--no-open" {
			noOpen = true
		}
	}
	runGUI(noOpen)
}

// --cli [perfil.json] saida.xml
func runCLI(a []string) {
	o := DefaultOptions()
	var out string
	switch len(a) {
	case 1:
		out = a[0]
	case 2:
		b, err := os.ReadFile(a[0])
		if err != nil {
			fatal(err)
		}
		if err := json.Unmarshal(b, o); err != nil {
			fatal(fmt.Errorf("perfil inválido: %w", err))
		}
		out = a[1]
	default:
		fatal(fmt.Errorf("uso: --cli [perfil.json] saida.xml"))
	}
	r, err := Generate(o)
	if err != nil {
		fatal(err)
	}
	if err := os.WriteFile(out, []byte(r.XML), 0o644); err != nil {
		fatal(err)
	}
	for _, w := range r.Warnings {
		fmt.Println("aviso:", w)
	}
	fmt.Printf("ok: %s (%d bytes)\n", out, len(r.XML))
}

// ---------------------------------------------------------------- servidor local

type server struct {
	token    string
	host     string
	start    time.Time
	lastPing atomic.Int64
	byeAt    atomic.Int64
	seen     atomic.Bool
	quit     chan struct{}
	lastDir  atomic.Value
}

func newToken() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func runGUI(noOpen bool) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fatal(err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	s := &server{token: newToken(), host: fmt.Sprintf("127.0.0.1:%d", port), start: time.Now(), quit: make(chan struct{})}
	s.lastDir.Store("")

	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/api/init", s.api(s.handleInit))
	mux.HandleFunc("/api/generate", s.api(s.handleGenerate))
	mux.HandleFunc("/api/save", s.api(s.handleSave))
	mux.HandleFunc("/api/open", s.api(s.handleOpen))
	mux.HandleFunc("/api/reveal", s.api(s.handleReveal))
	mux.HandleFunc("/api/ping", s.api(func(w http.ResponseWriter, r *http.Request) {
		s.touch()
		writeJSON(w, 200, map[string]bool{"ok": true})
	}))
	mux.HandleFunc("/api/bye", s.api(func(w http.ResponseWriter, r *http.Request) {
		s.byeAt.Store(time.Now().UnixNano())
		writeJSON(w, 200, map[string]bool{"ok": true})
	}))
	mux.HandleFunc("/api/quit", s.api(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]bool{"ok": true})
		go func() { time.Sleep(150 * time.Millisecond); close(s.quit) }()
	}))

	srv := &http.Server{Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	go func() { _ = srv.Serve(ln) }()

	url := "http://" + s.host + "/"
	var cmd *exec.Cmd
	if noOpen {
		fmt.Println("Interface em:", url)
	} else {
		cmd = launchWindow(url)
	}

	go s.watchdog()
	<-s.quit
	killWindow(cmd)
	if !noOpen && s.seen.Load() {
		openURL(channelURL)
	}
}

func (s *server) touch() {
	s.seen.Store(true)
	s.lastPing.Store(time.Now().UnixNano())
}

// encerra quando a janela some (sem ping) ou nunca abriu
func (s *server) watchdog() {
	for {
		time.Sleep(2 * time.Second)
		now := time.Now().UnixNano()
		if !s.seen.Load() {
			if time.Since(s.start) > 2*time.Minute {
				close(s.quit)
				return
			}
			continue
		}
		last := s.lastPing.Load()
		bye := s.byeAt.Load()
		if bye > 0 && last < bye && now-bye > int64(8*time.Second) {
			close(s.quit)
			return
		}
		if now-last > int64(5*time.Minute) {
			close(s.quit)
			return
		}
	}
}

func (s *server) hostOK(r *http.Request) bool {
	return r.Host == s.host || r.Host == strings.Replace(s.host, "127.0.0.1", "localhost", 1)
}

func (s *server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if !s.hostOK(r) || r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	b, _ := assets.ReadFile("app/index.html")
	html := strings.ReplaceAll(string(b), "__TOKEN__", s.token)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self' 'unsafe-inline'; script-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'self'; frame-ancestors 'none'")
	_, _ = io.WriteString(w, html)
}

func (s *server) api(h func(http.ResponseWriter, *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !s.hostOK(r) || r.Header.Get("X-Token") != s.token {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 16<<20)
		h(w, r)
	}
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func (s *server) handleInit(w http.ResponseWriter, r *http.Request) {
	s.touch()
	writeJSON(w, 200, map[string]any{
		"catalog":   json.RawMessage(catalogRaw),
		"defaults":  DefaultOptions(),
		"winget":    WingetApps,
		"locales":   Locales,
		"timezones": TimeZones,
		"os":        runtime.GOOS,
		"version":   appVersion,
	})
}

func (s *server) readOptions(r *http.Request) (*Options, error) {
	o := DefaultOptions()
	b, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	if len(strings.TrimSpace(string(b))) > 0 {
		if err := json.Unmarshal(b, o); err != nil {
			return nil, fmt.Errorf("opções inválidas: %w", err)
		}
	}
	return o, nil
}

func (s *server) handleGenerate(w http.ResponseWriter, r *http.Request) {
	o, err := s.readOptions(r)
	if err != nil {
		writeJSON(w, 400, map[string]string{"error": err.Error()})
		return
	}
	res, err := Generate(o)
	if err != nil {
		writeJSON(w, 400, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, 200, res)
}

type saveReq struct {
	Kind    string `json:"kind"` // xml | profile
	Name    string `json:"name"`
	Content string `json:"content"`
}

func (s *server) handleSave(w http.ResponseWriter, r *http.Request) {
	var q saveReq
	if err := json.NewDecoder(r.Body).Decode(&q); err != nil {
		writeJSON(w, 400, map[string]string{"error": err.Error()})
		return
	}
	title, filter, name := "Salvar autounattend.xml", "Arquivo XML (*.xml)|*.xml|Todos os arquivos (*.*)|*.*", q.Name
	if name == "" {
		name = "autounattend.xml"
	}
	if q.Kind == "profile" {
		title, filter = "Salvar perfil de configuração", "Perfil (*.json)|*.json|Todos os arquivos (*.*)|*.*"
		if q.Name == "" {
			name = "perfil-autounattend.json"
		}
	}
	dir, _ := s.lastDir.Load().(string)
	p, err := psDialog("save", title, filter, name, dir)
	if err != nil {
		writeJSON(w, 501, map[string]string{"error": "dialog-unavailable"})
		return
	}
	if p == "" {
		writeJSON(w, 200, map[string]any{"cancelled": true})
		return
	}
	if err := os.WriteFile(p, []byte(q.Content), 0o644); err != nil {
		writeJSON(w, 500, map[string]string{"error": "Não foi possível gravar o arquivo: " + err.Error()})
		return
	}
	s.lastDir.Store(filepath.Dir(p))
	writeJSON(w, 200, map[string]any{"path": p})
}

func (s *server) handleOpen(w http.ResponseWriter, r *http.Request) {
	dir, _ := s.lastDir.Load().(string)
	p, err := psDialog("open", "Abrir perfil de configuração", "Perfil (*.json)|*.json|Todos os arquivos (*.*)|*.*", "", dir)
	if err != nil {
		writeJSON(w, 501, map[string]string{"error": "dialog-unavailable"})
		return
	}
	if p == "" {
		writeJSON(w, 200, map[string]any{"cancelled": true})
		return
	}
	fi, err := os.Stat(p)
	if err != nil || fi.Size() > 8<<20 {
		writeJSON(w, 400, map[string]string{"error": "Arquivo inválido ou muito grande"})
		return
	}
	b, err := os.ReadFile(p)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	s.lastDir.Store(filepath.Dir(p))
	writeJSON(w, 200, map[string]any{"path": p, "content": string(b)})
}

func (s *server) handleReveal(w http.ResponseWriter, r *http.Request) {
	var q struct {
		Path string `json:"path"`
	}
	_ = json.NewDecoder(r.Body).Decode(&q)
	if q.Path != "" {
		if _, err := os.Stat(q.Path); err == nil {
			revealFile(q.Path)
		}
	}
	writeJSON(w, 200, map[string]bool{"ok": true})
}
