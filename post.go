package main

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// Programas oferecidos na aba "Pós-instalação" (IDs do winget).
type WingetApp struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Cat   string `json:"cat"`
	Basic bool   `json:"basic"`
}

var WingetApps = []WingetApp{
	// navegadores
	{"Google.Chrome", "Google Chrome", "browsers", true},
	{"Mozilla.Firefox", "Mozilla Firefox", "browsers", false},
	{"Brave.Brave", "Brave", "browsers", false},
	// compactadores, mídia e utilitários
	{"7zip.7zip", "7-Zip", "tools", true},
	{"RARLab.WinRAR", "WinRAR", "tools", false},
	{"VideoLAN.VLC", "VLC media player", "media", true},
	{"CodecGuide.K-LiteCodecPack.Standard", "K-Lite Codec Pack", "media", false},
	{"OBSProject.OBSStudio", "OBS Studio", "media", false},
	{"Audacity.Audacity", "Audacity", "media", false},
	{"HandBrake.HandBrake", "HandBrake", "media", false},
	{"IrfanSkiski.IrfanView", "IrfanView", "media", false},
	{"Spotify.Spotify", "Spotify", "media", false},
	{"voidtools.Everything", "Everything (busca de arquivos)", "tools", false},
	{"ShareX.ShareX", "ShareX (capturas de tela)", "tools", false},
	{"qBittorrent.qBittorrent", "qBittorrent", "tools", false},
	{"Microsoft.PowerToys", "Microsoft PowerToys", "tools", false},
	{"Rufus.Rufus", "Rufus", "tools", false},
	{"Ventoy.Ventoy", "Ventoy", "tools", false},
	// escritório e PDF
	{"Adobe.Acrobat.Reader.64-bit", "Adobe Acrobat Reader", "office", true},
	{"SumatraPDF.SumatraPDF", "SumatraPDF", "office", false},
	{"Foxit.FoxitReader", "Foxit PDF Reader", "office", false},
	{"TheDocumentFoundation.LibreOffice", "LibreOffice", "office", false},
	{"ONLYOFFICE.DesktopEditors", "ONLYOFFICE", "office", false},
	{"Google.GoogleDrive", "Google Drive", "office", false},
	// comunicação
	{"WhatsApp.WhatsApp", "WhatsApp", "comm", true},
	{"Telegram.TelegramDesktop", "Telegram", "comm", false},
	{"Discord.Discord", "Discord", "comm", false},
	{"Zoom.Zoom", "Zoom", "comm", false},
	// acesso remoto
	{"AnyDeskSoftwareGmbH.AnyDesk", "AnyDesk", "remote", true},
	{"TeamViewer.TeamViewer", "TeamViewer", "remote", false},
	{"RustDesk.RustDesk", "RustDesk", "remote", false},
	// desenvolvimento
	{"Notepad++.Notepad++", "Notepad++", "dev", true},
	{"Microsoft.VisualStudioCode", "Visual Studio Code", "dev", false},
	{"Git.Git", "Git", "dev", false},
	{"Python.Python.3.12", "Python 3.12", "dev", false},
	{"Microsoft.WindowsTerminal", "Windows Terminal", "dev", false},
	// jogos
	{"Valve.Steam", "Steam", "games", false},
	{"EpicGames.EpicGamesLauncher", "Epic Games Launcher", "games", false},
	// bibliotecas e runtimes
	{"Microsoft.VCRedist.2015+.x64", "Visual C++ Redistributable 2015-2022 (x64)", "runtimes", true},
	{"Microsoft.VCRedist.2015+.x86", "Visual C++ Redistributable 2015-2022 (x86)", "runtimes", true},
	{"Microsoft.DotNet.DesktopRuntime.8", ".NET Desktop Runtime 8", "runtimes", true},
	{"Microsoft.DotNet.DesktopRuntime.6", ".NET Desktop Runtime 6", "runtimes", false},
	{"Microsoft.DirectX", "DirectX (runtime)", "runtimes", false},
	{"Oracle.JavaRuntimeEnvironment", "Java Runtime Environment", "runtimes", false},
	{"Microsoft.EdgeWebView2Runtime", "WebView2 Runtime", "runtimes", false},
	// hardware e segurança
	{"CPUID.CPU-Z", "CPU-Z", "hw", false},
	{"REALiX.HWiNFO", "HWiNFO", "hw", false},
	{"CrystalDewWorld.CrystalDiskInfo", "CrystalDiskInfo", "hw", false},
	{"Malwarebytes.Malwarebytes", "Malwarebytes", "hw", false},
}

func BasicPostApps() []string {
	var r []string
	for _, a := range WingetApps {
		if a.Basic {
			r = append(r, a.ID)
		}
	}
	return r
}

var reWingetID = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._+\-]{1,99}$`)

const (
	postBatPath = `C:\Users\Public\Desktop\Pos-instalacao.bat`
	postPs1Path = `C:\ProgramData\Winhance\Unattend\Scripts\PosInstalacao.ps1`
)

type postApp struct{ ID, Name string }

// resolve a seleção: programas marcados (na ordem do catálogo) + IDs digitados
func postSelection(o *Options) ([]postApp, error) {
	sel := map[string]bool{}
	for _, id := range o.PostApps {
		sel[id] = true
	}
	var out []postApp
	seen := map[string]bool{}
	for _, a := range WingetApps {
		if sel[a.ID] {
			out = append(out, postApp{a.ID, a.Name})
			seen[a.ID] = true
		}
	}
	for _, ln := range strings.Split(strings.ReplaceAll(o.PostCustom, "\r", ""), "\n") {
		if i := strings.Index(ln, "#"); i >= 0 {
			ln = ln[:i]
		}
		ln = strings.TrimSpace(ln)
		if ln == "" {
			continue
		}
		if !reWingetID.MatchString(ln) {
			return nil, fmt.Errorf(tr(o.Lang, "ID do winget inválido: %q (use o formato Empresa.Programa)"), ln)
		}
		if !seen[ln] {
			seen[ln] = true
			out = append(out, postApp{ln, ln})
		}
	}
	return out, nil
}

type postText struct {
	Title, NoAdmin, Count, WaitWinget, NoWinget, Enter, Done, Failed, Retry string
}

var postTexts = map[string]postText{
	"pt-BR": {"Pós-instalação — instalando programas com o winget", "Não foi possível pedir permissão de administrador; algumas instalações podem falhar.", "Serão instalados {0} programas. Aguarde, isso pode demorar.", "Aguardando o winget (Instalador de Aplicativo) ficar disponível… precisa de internet.", "O winget não está disponível neste Windows. Atualize o \"Instalador de Aplicativo\" na Microsoft Store e execute este script de novo.", "Pressione Enter para fechar", "Concluído: {0} instalado(s) ou já presente(s), {1} com falha.", "Programas que falharam:", "Verifique a internet e execute o script novamente: ele só instala o que falta."},
	"en":    {"Post-install — installing programs with winget", "Could not ask for administrator permission; some installs may fail.", "{0} programs will be installed. Please wait, this can take a while.", "Waiting for winget (App Installer) to become available… internet is required.", "winget is not available on this Windows. Update \"App Installer\" in the Microsoft Store and run this script again.", "Press Enter to close", "Done: {0} installed or already present, {1} failed.", "Programs that failed:", "Check your internet connection and run the script again: it only installs what is missing."},
	"es":    {"Posinstalación — instalando programas con winget", "No se pudo pedir permiso de administrador; algunas instalaciones pueden fallar.", "Se instalarán {0} programas. Espere, puede tardar.", "Esperando a que winget (Instalador de aplicación) esté disponible… se necesita internet.", "winget no está disponible en este Windows. Actualice el \"Instalador de aplicación\" en Microsoft Store y ejecute este script de nuevo.", "Pulse Enter para cerrar", "Terminado: {0} instalado(s) o ya presente(s), {1} con error.", "Programas con error:", "Compruebe internet y ejecute el script de nuevo: solo instala lo que falta."},
}

const postPs1Body = `$ErrorActionPreference = 'Continue'
try { [Console]::OutputEncoding = [System.Text.Encoding]::UTF8 } catch { }
$Host.UI.RawUI.WindowTitle = '@@TITLE@@'

$isAdmin = ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
if (-not $isAdmin) {
    try {
        Start-Process -FilePath 'powershell.exe' -Verb RunAs -ArgumentList @('-NoProfile', '-ExecutionPolicy', 'Bypass', '-File', ('"{0}"' -f $PSCommandPath))
        exit
    } catch {
        Write-Host '@@NOADMIN@@' -ForegroundColor Yellow
    }
}

$apps = @(
@@APPS@@
)

Write-Host ''
Write-Host '@@TITLE@@' -ForegroundColor Cyan
Write-Host ('@@COUNT@@' -f $apps.Count)
Write-Host ''

function Find-Winget { Get-Command winget.exe -ErrorAction SilentlyContinue }
$tries = 0
while (-not (Find-Winget) -and $tries -lt 20) {
    if ($tries -eq 0) { Write-Host '@@WAITWINGET@@' -ForegroundColor Yellow }
    try { Add-AppxPackage -RegisterByFamilyName -MainPackage 'Microsoft.DesktopAppInstaller_8wekyb3d8bbwe' -ErrorAction Stop } catch { }
    Start-Sleep -Seconds 15
    $tries++
}
if (-not (Find-Winget)) {
    Write-Host '@@NOWINGET@@' -ForegroundColor Red
    Read-Host '@@ENTER@@' | Out-Null
    exit 1
}

$ok = @()
$fail = @()
$i = 0
foreach ($a in $apps) {
    $i++
    Write-Host ('[{0}/{1}] {2}' -f $i, $apps.Count, $a.Name) -ForegroundColor Cyan
    $code = 1
    for ($try = 1; $try -le 2 -and $code -ne 0; $try++) {
        & winget install --id $a.Id -e --source winget --silent --accept-package-agreements --accept-source-agreements
        $code = $LASTEXITCODE
        if ($code -eq -1978335135 -or $code -eq -1978335189) { $code = 0 }
        if ($code -ne 0 -and $try -lt 2) { Start-Sleep -Seconds 10 }
    }
    if ($code -eq 0) { $ok += $a.Name } else { $fail += $a.Name }
}

Write-Host ''
Write-Host ('@@DONE@@' -f $ok.Count, $fail.Count) -ForegroundColor Green
if ($fail.Count -gt 0) {
    Write-Host '@@FAILED@@' -ForegroundColor Yellow
    foreach ($f in $fail) { Write-Host (' - ' + $f) -ForegroundColor Yellow }
    Write-Host '@@RETRY@@'
}
@@DELETE@@Read-Host '@@ENTER@@' | Out-Null
`

func psq(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }

func buildPostScript(o *Options, apps []postApp) string {
	tx, ok := postTexts[o.Lang]
	if !ok {
		tx = postTexts["pt-BR"]
	}
	var al []string
	for _, a := range apps {
		al = append(al, fmt.Sprintf("    @{ Id = %s; Name = %s }", psq(a.ID), psq(a.Name)))
	}
	del := ""
	if o.PostDelete {
		del = "if ($fail.Count -eq 0) { Remove-Item -LiteralPath '" + postBatPath + "' -Force -ErrorAction SilentlyContinue; Remove-Item -LiteralPath $PSCommandPath -Force -ErrorAction SilentlyContinue }\n"
	}
	q := func(s string) string { return strings.ReplaceAll(s, "'", "''") }
	r := strings.NewReplacer(
		"@@TITLE@@", q(tx.Title), "@@NOADMIN@@", q(tx.NoAdmin), "@@COUNT@@", q(tx.Count),
		"@@WAITWINGET@@", q(tx.WaitWinget), "@@NOWINGET@@", q(tx.NoWinget), "@@ENTER@@", q(tx.Enter),
		"@@DONE@@", q(tx.Done), "@@FAILED@@", q(tx.Failed), "@@RETRY@@", q(tx.Retry),
		"@@APPS@@", strings.Join(al, "\n"), "@@DELETE@@", del,
	)
	return strings.ReplaceAll(r.Replace(postPs1Body), "\r", "")
}

const postBat = "@echo off\npowershell.exe -NoProfile -ExecutionPolicy Bypass -File \"" + postPs1Path + "\"\n"

// trechos <File> adicionais do bloco Extensions
func postFiles(o *Options, apps []postApp) string {
	ps := strings.ReplaceAll(buildPostScript(o, apps), "]]>", "]]]]><![CDATA[>")
	return "    <File path=\"" + postPs1Path + "\"><![CDATA[\n" + ps + "]]></File>\n" +
		"    <File path=\"" + postBatPath + "\"><![CDATA[\n" + postBat + "]]></File>\n"
}

var errNoApps = errors.New("no apps")
