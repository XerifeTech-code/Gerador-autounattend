package main

import (
	"os"
	"strings"
	"testing"
)

func TestDefaultsMatchGolden(t *testing.T) {
	if err := loadAssets(); err != nil {
		t.Fatal(err)
	}
	g, err := os.ReadFile("golden_default.xml")
	if err != nil {
		t.Skip("golden_default.xml ausente")
	}
	r, err := Generate(DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	if r.XML != string(g) {
		t.Fatalf("XML padrão mudou (len %d vs %d)", len(r.XML), len(g))
	}
}

func TestNoNetworkOrShortcutStuff(t *testing.T) {
	if err := loadAssets(); err != nil {
		t.Fatal(err)
	}
	o := DefaultOptions()
	o.DisableNet, o.HideWireless = true, true // mesmo que venham de um perfil antigo
	r, err := Generate(o)
	if err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{"Disable-NetAdapter", "Enable-NetAdapter", "Install Winhance", "WinhanceInstall", "HideWirelessSetupInOOBE",
		"TcpAckFrequency", "TCPNoDelay", "NetworkThrottlingIndex", `Services\RasMan`, `Services\RasAuto`, `Services\icssvc`, `Services\BITS`,
		"AllowWiFiHotSpotReporting", "AllowAutoConnectToWiFiSenseHotspots", "DODownloadMode", "CreateShortcut"} {
		if strings.Contains(r.XML, bad) {
			t.Errorf("o XML ainda contém %q", bad)
		}
	}
}

func TestPostInstall(t *testing.T) {
	if err := loadAssets(); err != nil {
		t.Fatal(err)
	}
	for _, lang := range []string{"pt-BR", "en", "es"} {
		for _, run := range []bool{true, false} {
			o := DefaultOptions()
			o.Lang = lang
			o.RunScript = run
			o.PostEnabled = true
			o.PostAutoRun = true
			o.PostDelete = true
			o.PostCustom = "Foo.Bar # comentario\n\nMicrosoft.VCRedist.2015+.x64\n"
			r, err := Generate(o)
			if err != nil {
				t.Fatal(err)
			}
			for _, want := range []string{postBatPath, postPs1Path, "Id = 'Google.Chrome'", "Id = 'Foo.Bar'", "winget install --id", `start "" "C:\Users\Public\Desktop\Pos-instalacao.bat"`, "Remove-Item -LiteralPath"} {
				if !strings.Contains(r.XML, want) {
					t.Errorf("[%s run=%v] falta %q", lang, run, want)
				}
			}
			if strings.Count(r.XML, "Id = 'Microsoft.VCRedist.2015+.x64'") != 1 {
				t.Errorf("duplicado ou ausente: VCRedist")
			}
			if run != strings.Contains(r.XML, "Winhancements.ps1") {
				t.Errorf("Winhancements.ps1 inconsistente com RunScript=%v", run)
			}
			if !strings.HasSuffix(r.XML, "  </Extensions>\n</unattend>\n") {
				t.Errorf("fim do XML inesperado")
			}
		}
	}
	o := DefaultOptions()
	o.PostEnabled = true
	o.PostCustom = "x y; calc"
	if _, err := Generate(o); err == nil {
		t.Errorf("ID inválido deveria falhar")
	}
	o = DefaultOptions()
	o.PostEnabled = true
	o.PostApps = nil
	r, _ := Generate(o)
	if r == nil || len(r.Warnings) == 0 {
		t.Errorf("esperava aviso de lista vazia")
	}
}
