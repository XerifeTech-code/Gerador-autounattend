package main

// Criação de uma ISO do Windows já com o autounattend.xml dentro.
// Usa o oscdimg.exe do Windows ADK (Microsoft). Passos: monta a ISO original, copia para uma pasta
// temporária, grava o autounattend.xml na raiz, gera a nova ISO (BIOS + UEFI) e limpa tudo.

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
	"unicode/utf16"
)

type isoJob struct {
	mu      sync.Mutex
	Running bool   `json:"running"`
	Step    string `json:"step"`
	Done    bool   `json:"done"`
	Error   string `json:"error"`
	Out     string `json:"out"`
	Started int64  `json:"started"`
}

var job isoJob

func (j *isoJob) set(f func(*isoJob)) {
	j.mu.Lock()
	f(j)
	j.mu.Unlock()
}

func (j *isoJob) snapshot() map[string]any {
	j.mu.Lock()
	defer j.mu.Unlock()
	return map[string]any{"running": j.Running, "step": j.Step, "done": j.Done, "error": j.Error, "out": j.Out, "started": j.Started}
}

// findOscdimg procura o oscdimg.exe: ao lado do programa, no PATH e nas pastas do Windows ADK.
func findOscdimg() string {
	var c []string
	if p, err := os.Executable(); err == nil {
		c = append(c, filepath.Join(filepath.Dir(p), "oscdimg.exe"))
	}
	if p, err := exec.LookPath("oscdimg.exe"); err == nil {
		c = append(c, p)
	}
	for _, base := range []string{os.Getenv("ProgramFiles(x86)"), os.Getenv("ProgramFiles")} {
		if base == "" {
			continue
		}
		for _, a := range []string{"amd64", "arm64", "x86"} {
			c = append(c, filepath.Join(base, `Windows Kits\10\Assessment and Deployment Kit\Deployment Tools`, a, `Oscdimg\oscdimg.exe`))
		}
	}
	for _, p := range c {
		if fileExists(p) {
			return p
		}
	}
	return ""
}

func fileExists(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && !fi.IsDir()
}

const isoScript = `
$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = New-Object System.Text.UTF8Encoding($false)
$src = $env:GA_SRC; $out = $env:GA_OUT; $xml = $env:GA_XML; $osc = $env:GA_OSC
$work = Join-Path $env:TEMP ('GA-ISO-' + [guid]::NewGuid().ToString('N'))
$mounted = $false
try {
    Write-Output 'STEP mount'
    $img = Mount-DiskImage -ImagePath $src -PassThru
    $mounted = $true
    $letter = $null
    for ($i = 0; $i -lt 20 -and -not $letter; $i++) { Start-Sleep -Milliseconds 500; $letter = ($img | Get-Volume).DriveLetter }
    if (-not $letter) { throw 'A ISO foi montada, mas o Windows não deu uma letra de unidade para ela.' }
    $vol = Get-Volume -DriveLetter $letter
    $label = ($vol.FileSystemLabel -replace '[^A-Za-z0-9_]', '_')
    if (-not $label) { $label = 'WIN_XERIFE' }
    if ($label.Length -gt 32) { $label = $label.Substring(0, 32) }
    if (-not (Test-Path ($letter + ':\sources'))) { throw 'Esta ISO não parece ser uma mídia de instalação do Windows (falta a pasta sources).' }
    Write-Output 'STEP copy'
    $null = New-Item -ItemType Directory -Path $work -Force
    & robocopy.exe ($letter + ':\') $work /E /NFL /NDL /NJH /NJS /NP /R:1 /W:1 | Out-Null
    if ($LASTEXITCODE -ge 8) { throw ('Falha ao copiar os arquivos da ISO (robocopy ' + $LASTEXITCODE + ').') }
    Dismount-DiskImage -ImagePath $src | Out-Null
    $mounted = $false
    Copy-Item -LiteralPath $xml -Destination (Join-Path $work 'autounattend.xml') -Force
    Write-Output 'STEP build'
    $etfs = Join-Path $work 'boot\etfsboot.com'
    $efi = Join-Path $work 'efi\microsoft\boot\efisys.bin'
    if (Test-Path $etfs) { $boot = '-bootdata:2#p0,e,b' + $etfs + '#pEF,e,b' + $efi }
    else { $boot = '-bootdata:1#pEF,e,b' + $efi }
    $log = Join-Path $env:TEMP 'GA-oscdimg.log'
    & $osc -m -o -u2 -udfver102 ('-l' + $label) $boot $work $out *> $log
    if ($LASTEXITCODE -ne 0) { $t = (Get-Content $log -Tail 3) -join ' '; throw ('O oscdimg falhou (' + $LASTEXITCODE + '): ' + $t) }
    Write-Output 'DONE'
} catch {
    Write-Output ('ERR ' + $_.Exception.Message)
    exit 1
} finally {
    if ($mounted) { try { Dismount-DiskImage -ImagePath $src | Out-Null } catch { } }
    if (Test-Path $work) { Remove-Item -LiteralPath $work -Recurse -Force -ErrorAction SilentlyContinue }
}
`

func encodePS(script string) string {
	u := utf16.Encode([]rune(script))
	buf := make([]byte, 0, len(u)*2)
	for _, c := range u {
		buf = append(buf, byte(c), byte(c>>8))
	}
	return base64.StdEncoding.EncodeToString(buf)
}

func (s *server) handleISOStart(w http.ResponseWriter, r *http.Request) {
	var q struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&q); err != nil || q.Content == "" {
		writeJSON(w, 400, map[string]string{"error": "Pedido inválido"})
		return
	}
	if runtime.GOOS != "windows" {
		writeJSON(w, 501, map[string]string{"error": "windows-only"})
		return
	}
	if job.snapshot()["running"].(bool) {
		writeJSON(w, 409, map[string]string{"error": "busy"})
		return
	}
	osc := findOscdimg()
	if osc == "" {
		writeJSON(w, 200, map[string]any{"noOscdimg": true})
		return
	}
	dir, _ := s.lastDir.Load().(string)
	src, err := psDialog("open", "Escolha a ISO original do Windows", "Imagem ISO (*.iso)|*.iso", "", dir)
	if err != nil || src == "" {
		writeJSON(w, 200, map[string]any{"cancelled": true})
		return
	}
	name := strings.TrimSuffix(filepath.Base(src), filepath.Ext(src)) + "-autounattend.iso"
	out, err := psDialog("save", "Salvar a nova ISO", "Imagem ISO (*.iso)|*.iso", name, filepath.Dir(src))
	if err != nil || out == "" {
		writeJSON(w, 200, map[string]any{"cancelled": true})
		return
	}
	if strings.EqualFold(filepath.Clean(src), filepath.Clean(out)) {
		writeJSON(w, 400, map[string]string{"error": "same"})
		return
	}
	tmp := filepath.Join(os.TempDir(), "GA-autounattend-"+newToken()[:8]+".xml")
	if err := os.WriteFile(tmp, []byte(q.Content), 0o644); err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	job.set(func(j *isoJob) {
		*j = isoJob{Running: true, Step: "start", Out: out, Started: time.Now().Unix()}
	})
	go runISO(osc, src, out, tmp)
	writeJSON(w, 200, map[string]any{"started": true, "out": out})
}

func runISO(osc, src, out, xmlPath string) {
	defer os.Remove(xmlPath)
	cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-EncodedCommand", encodePS(isoScript))
	cmd.Env = append(os.Environ(), "GA_SRC="+src, "GA_OUT="+out, "GA_XML="+xmlPath, "GA_OSC="+osc)
	hideWindow(cmd)
	pipe, err := cmd.StdoutPipe()
	if err == nil {
		err = cmd.Start()
	}
	if err != nil {
		job.set(func(j *isoJob) { j.Running, j.Error = false, err.Error() })
		return
	}
	sc := bufio.NewScanner(pipe)
	var errMsg string
	done := false
	for sc.Scan() {
		ln := strings.TrimSpace(sc.Text())
		switch {
		case strings.HasPrefix(ln, "STEP "):
			st := strings.TrimPrefix(ln, "STEP ")
			job.set(func(j *isoJob) { j.Step = st })
		case strings.HasPrefix(ln, "ERR "):
			errMsg = strings.TrimPrefix(ln, "ERR ")
		case ln == "DONE":
			done = true
		}
	}
	werr := cmd.Wait()
	job.set(func(j *isoJob) {
		j.Running = false
		j.Done = done && werr == nil
		if !j.Done {
			if errMsg == "" && werr != nil {
				errMsg = werr.Error()
			}
			if errMsg == "" {
				errMsg = "falha desconhecida"
			}
			j.Error = errMsg
		}
	})
}

func (s *server) handleISOStatus(w http.ResponseWriter, r *http.Request) {
	s.touch()
	writeJSON(w, 200, job.snapshot())
}
