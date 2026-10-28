//go:build windows

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
)

func browserCandidates() []string {
	pf := os.Getenv("ProgramFiles")
	pf86 := os.Getenv("ProgramFiles(x86)")
	la := os.Getenv("LocalAppData")
	var c []string
	add := func(base, rel string) {
		if base != "" {
			c = append(c, filepath.Join(base, rel))
		}
	}
	add(pf86, `Microsoft\Edge\Application\msedge.exe`)
	add(pf, `Microsoft\Edge\Application\msedge.exe`)
	add(pf, `Google\Chrome\Application\chrome.exe`)
	add(pf86, `Google\Chrome\Application\chrome.exe`)
	add(la, `Google\Chrome\Application\chrome.exe`)
	add(pf, `BraveSoftware\Brave-Browser\Application\brave.exe`)
	add(la, `BraveSoftware\Brave-Browser\Application\brave.exe`)
	add(pf, `Vivaldi\Application\vivaldi.exe`)
	add(la, `Vivaldi\Application\vivaldi.exe`)
	return c
}

// launchWindow abre a interface numa janela própria (modo app) do Edge/Chrome.
// Se nenhum navegador compatível for encontrado, usa o navegador padrão.
func launchWindow(url string) *exec.Cmd {
	profile := filepath.Join(os.TempDir(), "GeradorAutounattend-ui")
	for _, p := range browserCandidates() {
		if _, err := os.Stat(p); err != nil {
			continue
		}
		cmd := exec.Command(p,
			"--app="+url,
			"--window-size=1280,880",
			"--user-data-dir="+profile,
			"--no-first-run",
			"--no-default-browser-check",
			"--disable-sync",
		)
		if err := cmd.Start(); err == nil {
			return cmd
		}
	}
	cmd := exec.Command("rundll32.exe", "url.dll,FileProtocolHandler", url)
	_ = cmd.Start()
	return nil
}

func killWindow(cmd *exec.Cmd) {
	if cmd != nil && cmd.Process != nil {
		k := exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(cmd.Process.Pid))
		hideWindow(k)
		_ = k.Run()
	}
	_ = os.RemoveAll(filepath.Join(os.TempDir(), "GeradorAutounattend-ui"))
}

func revealFile(path string) {
	cmd := exec.Command("explorer.exe", "/select,"+path)
	_ = cmd.Start()
}

// openURL abre o endereço no navegador padrão do usuário.
func openURL(url string) {
	cmd := exec.Command("rundll32.exe", "url.dll,FileProtocolHandler", url)
	hideWindow(cmd)
	_ = cmd.Start()
}
