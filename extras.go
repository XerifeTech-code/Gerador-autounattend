package main

// Funções extras da versão 1.2: janela Sobre (com SHA-256 do próprio .exe), abrir o canal só uma vez
// por usuário/computador, links externos seguros, gravar direto no pendrive e criar ISO.

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

const (
	vendorName = "Xerife Tech"
	productDir = "GeradorAutounattend"
	// endereço de um JSON {"version":"1.2.0","url":"https://..."} para avisar de versão nova.
	// Vazio = verificação de atualização desligada.
	updateURL = ""
)

// ---------------------------------------------------------------- canal: só na primeira vez

// channelMarker fica em %LocalAppData%\Xerife Tech\GeradorAutounattend (por usuário e por computador).
func channelMarker() string {
	base, err := os.UserCacheDir()
	if err != nil || base == "" {
		return ""
	}
	return filepath.Join(base, vendorName, productDir, "canal-aberto.txt")
}

// firstChannelOpen devolve true só na primeira vez e grava a marca. Se não conseguir gravar a marca,
// não abre o canal (para nunca abrir em toda execução).
func firstChannelOpen() bool {
	m := channelMarker()
	if m == "" {
		return false
	}
	if _, err := os.Stat(m); err == nil {
		return false
	}
	if err := os.MkdirAll(filepath.Dir(m), 0o755); err != nil {
		return false
	}
	txt := "Canal da Xerife Tech já foi aberto em " + time.Now().Format("2006-01-02 15:04") + ".\r\nApague este arquivo para ver de novo.\r\n"
	return os.WriteFile(m, []byte(txt), 0o644) == nil
}

// ---------------------------------------------------------------- Sobre

var (
	selfHashOnce sync.Once
	selfHash     string
)

func exeHash() string {
	selfHashOnce.Do(func() {
		p, err := os.Executable()
		if err != nil {
			return
		}
		f, err := os.Open(p)
		if err != nil {
			return
		}
		defer f.Close()
		h := sha256.New()
		if _, err := io.Copy(h, f); err == nil {
			selfHash = strings.ToUpper(hex.EncodeToString(h.Sum(nil)))
		}
	})
	return selfHash
}

func (s *server) handleAbout(w http.ResponseWriter, r *http.Request) {
	name := ""
	if p, err := os.Executable(); err == nil {
		name = filepath.Base(p)
	}
	writeJSON(w, 200, map[string]any{
		"version": appVersion, "vendor": vendorName, "channel": channelURL,
		"exe": name, "sha256": exeHash(), "updateURL": updateURL != "",
	})
}

// ---------------------------------------------------------------- links externos (só o canal)

const adkURL = "https://learn.microsoft.com/windows-hardware/get-started/adk-install"

var reSafeURL = regexp.MustCompile(`^https://(www\.)?youtube\.com/@xerifetech(/[A-Za-z0-9_\-/?=&%.+]*)?$`)

func (s *server) handleOpenURL(w http.ResponseWriter, r *http.Request) {
	var q struct {
		URL string `json:"url"`
	}
	_ = json.NewDecoder(r.Body).Decode(&q)
	if !reSafeURL.MatchString(q.URL) && q.URL != adkURL {
		writeJSON(w, 400, map[string]string{"error": "url"})
		return
	}
	openURL(q.URL)
	writeJSON(w, 200, map[string]bool{"ok": true})
}

// ---------------------------------------------------------------- verificação de atualização

func (s *server) handleUpdate(w http.ResponseWriter, r *http.Request) {
	if updateURL == "" {
		writeJSON(w, 200, map[string]any{"enabled": false})
		return
	}
	c := &http.Client{Timeout: 8 * time.Second}
	resp, err := c.Get(updateURL)
	if err != nil {
		writeJSON(w, 200, map[string]any{"enabled": true, "error": err.Error()})
		return
	}
	defer resp.Body.Close()
	var v struct {
		Version string `json:"version"`
		URL     string `json:"url"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 64<<10)).Decode(&v); err != nil {
		writeJSON(w, 200, map[string]any{"enabled": true, "error": "json"})
		return
	}
	writeJSON(w, 200, map[string]any{"enabled": true, "latest": v.Version, "url": v.URL, "newer": versionLess(appVersion, v.Version)})
}

func versionLess(a, b string) bool {
	pa, pb := strings.Split(a, "."), strings.Split(b, ".")
	for i := 0; i < len(pa) || i < len(pb); i++ {
		var x, y int
		if i < len(pa) {
			fmt.Sscanf(pa[i], "%d", &x)
		}
		if i < len(pb) {
			fmt.Sscanf(pb[i], "%d", &y)
		}
		if x != y {
			return x < y
		}
	}
	return false
}

// ---------------------------------------------------------------- pendrive

type Drive struct {
	Root      string `json:"root"`
	Label     string `json:"label"`
	Removable bool   `json:"removable"`
	Installer bool   `json:"installer"` // tem sources\install.wim/esd ou boot.wim
	Ventoy    bool   `json:"ventoy"`
	Free      uint64 `json:"free"`
	Total     uint64 `json:"total"`
	HasXML    bool   `json:"hasXML"`
}

func isInstaller(root string) bool {
	for _, f := range []string{`sources\install.wim`, `sources\install.esd`, `sources\install.swm`, `sources\boot.wim`} {
		if _, err := os.Stat(filepath.Join(root, f)); err == nil {
			return true
		}
	}
	return false
}

func isVentoy(root string) bool {
	_, err := os.Stat(filepath.Join(root, "ventoy"))
	return err == nil
}

func (s *server) handleDrives(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]any{"drives": listDrives()})
}

var reRoot = regexp.MustCompile(`^[A-Za-z]:\\$`)

func (s *server) handleSaveDrive(w http.ResponseWriter, r *http.Request) {
	var q struct {
		Root    string `json:"root"`
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&q); err != nil || !reRoot.MatchString(q.Root) || q.Content == "" {
		writeJSON(w, 400, map[string]string{"error": "Pedido inválido"})
		return
	}
	ok := false
	for _, d := range listDrives() {
		if strings.EqualFold(d.Root, q.Root) {
			ok = true
		}
	}
	if !ok {
		writeJSON(w, 400, map[string]string{"error": "drive"})
		return
	}
	dst := filepath.Join(q.Root, "autounattend.xml")
	backup := ""
	if _, err := os.Stat(dst); err == nil {
		backup = filepath.Join(q.Root, "autounattend-backup-"+time.Now().Format("20060102-150405")+".xml")
		if err := os.Rename(dst, backup); err != nil {
			writeJSON(w, 500, map[string]string{"error": err.Error()})
			return
		}
	}
	if err := os.WriteFile(dst, []byte(q.Content), 0o644); err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, 200, map[string]string{"path": dst, "backup": backup})
}
