package main

import (
	"encoding/base64"
	"errors"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"unicode/utf16"
)

var errNoDialog = errors.New("dialog-unavailable")

const psDialogScript = `
$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = New-Object System.Text.UTF8Encoding($false)
Add-Type -AssemblyName System.Windows.Forms
$f = New-Object System.Windows.Forms.Form
$f.TopMost = $true; $f.ShowInTaskbar = $false; $f.Opacity = 0
$f.StartPosition = 'CenterScreen'; $f.Width = 1; $f.Height = 1
$f.Show()
if ($env:GA_KIND -eq 'save') { $d = New-Object System.Windows.Forms.SaveFileDialog; $d.OverwritePrompt = $true }
else { $d = New-Object System.Windows.Forms.OpenFileDialog; $d.CheckFileExists = $true }
$d.Title = $env:GA_TITLE
$d.Filter = $env:GA_FILTER
if ($env:GA_NAME) { $d.FileName = $env:GA_NAME }
if ($env:GA_DIR -and (Test-Path $env:GA_DIR)) { $d.InitialDirectory = $env:GA_DIR }
else { $d.InitialDirectory = [Environment]::GetFolderPath('Desktop') }
$r = $d.ShowDialog($f)
if ($r -eq [System.Windows.Forms.DialogResult]::OK) { [Console]::Out.Write($d.FileName) }
$f.Close()
`

func psDialog(kind, title, filter, name, dir string) (string, error) {
	if runtime.GOOS != "windows" {
		return "", errNoDialog
	}
	u := utf16.Encode([]rune(psDialogScript))
	buf := make([]byte, 0, len(u)*2)
	for _, c := range u {
		buf = append(buf, byte(c), byte(c>>8))
	}
	enc := base64.StdEncoding.EncodeToString(buf)
	cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-STA", "-ExecutionPolicy", "Bypass", "-EncodedCommand", enc)
	cmd.Env = append(os.Environ(),
		"GA_KIND="+kind, "GA_TITLE="+title, "GA_FILTER="+filter, "GA_NAME="+name, "GA_DIR="+dir)
	hideWindow(cmd)
	out, err := cmd.Output()
	if err != nil {
		return "", errNoDialog
	}
	return strings.TrimSpace(string(out)), nil
}
