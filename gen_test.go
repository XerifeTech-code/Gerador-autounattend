package main

import (
	"encoding/base64"
	"encoding/json"
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

func TestUsersAndPasswords(t *testing.T) {
	if err := loadAssets(); err != nil {
		t.Fatal(err)
	}
	o := DefaultOptions()
	o.CreateUser, o.UserName, o.UserPassword, o.AutoLogon = true, "xerife", "abc", true
	o.ExtraUsers = []ExtraUser{{Name: "filho", Password: "123"}, {Name: "  "}}
	o.PasswordBase64 = true
	r, err := Generate(o)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(r.XML, "<Value>"+encodePw("abc", "Password")+"</Value>") || strings.Contains(r.XML, "<Value>abc</Value>") {
		t.Error("senha base64 não aplicada")
	}
	if encodePw("abc", "Password") != "YQBiAGMAUABhAHMAcwB3AG8AcgBkAA==" {
		t.Error("codificação diferente da do Windows SIM")
	}
	if strings.Count(r.XML, "<Name>filho</Name>") != 3 || strings.Count(r.XML, "<LocalAccount ") != 6 {
		t.Errorf("usuários adicionais errados")
	}
	if strings.Count(r.XML, "<PlainText>false</PlainText>") != 9 {
		t.Errorf("PlainText false esperado em 9 lugares, achou %d", strings.Count(r.XML, "<PlainText>false</PlainText>"))
	}
	o.ExtraUsers = []ExtraUser{{Name: "XERIFE"}}
	if _, err := Generate(o); err == nil {
		t.Error("nome repetido deveria falhar")
	}
	o.ExtraUsers = []ExtraUser{{Name: "Convidado"}}
	if _, err := Generate(o); err == nil {
		t.Error("nome reservado deveria falhar")
	}
}

func TestDiskDriversFeatures(t *testing.T) {
	if err := loadAssets(); err != nil {
		t.Fatal(err)
	}
	for _, fw := range []string{"uefi", "bios"} {
		o := DefaultOptions()
		o.WipeDisk, o.Firmware, o.RecoveryPart = true, fw, true
		o.DriversUSB, o.FeatWSL, o.FeatHyperV, o.FeatSandbox = true, true, true, true
		r, err := Generate(o)
		if err != nil {
			t.Fatal(err)
		}
		want := map[string]string{"uefi": "DE94BBA4-06D1-4D40-A16A-BFD50179D6AC", "bios": "<TypeID>0x27</TypeID>"}[fw]
		part := map[string]string{"uefi": "<PartitionID>4</PartitionID>\n          </InstallTo>", "bios": "<PartitionID>3</PartitionID>\n          </InstallTo>"}[fw]
		for _, w := range []string{want, part, "drvload.exe", "pnputil.exe /add-driver", "Microsoft-Windows-Subsystem-Linux", "VirtualMachinePlatform", "Microsoft-Hyper-V-All", "Containers-DisposableClientVM"} {
			if !strings.Contains(r.XML, w) {
				t.Errorf("[%s] falta %q", fw, w)
			}
		}
	}
}

func TestProfileEmbedded(t *testing.T) {
	if err := loadAssets(); err != nil {
		t.Fatal(err)
	}
	o := DefaultOptions()
	o.CreateUser, o.UserName, o.UserPassword, o.ComputerName = true, "ana", "segredo", "PC-ANA"
	o.ExtraUsers = []ExtraUser{{Name: "bia", Password: "outra"}}
	o.Items = map[string]bool{"script-edge": false, "script-bloat": true}
	r, err := Generate(o)
	if err != nil {
		t.Fatal(err)
	}
	i := strings.Index(r.XML, profileMarker)
	if i < 0 {
		t.Fatal("perfil não embutido")
	}
	rest := r.XML[i+len(profileMarker):]
	b64 := rest[:strings.Index(rest, " -->")]
	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		t.Fatal(err)
	}
	var p Options
	if err := json.Unmarshal(raw, &p); err != nil {
		t.Fatal(err)
	}
	if p.UserName != "ana" || p.ComputerName != "PC-ANA" || p.UserPassword != "" || p.ExtraUsers[0].Password != "" || p.ExtraUsers[0].Name != "bia" {
		t.Errorf("perfil errado: %+v", p)
	}
	if v, ok := p.Items["script-edge"]; !ok || v || len(p.Items) != 1 {
		t.Errorf("itens errados: %v", p.Items)
	}
	if strings.Contains(string(raw), "segredo") || strings.Contains(string(raw), "outra") {
		t.Error("senha vazou no perfil")
	}
}

func TestAlwaysWellFormed(t *testing.T) {
	if err := loadAssets(); err != nil {
		t.Fatal(err)
	}
	o := DefaultOptions()
	o.FirstLogonCmds = "echo a & echo <b> > c.txt"
	o.CustomSystemScript = "Write-Host ']]>' # <tag> & coisa"
	o.CustomUserScript = "Write-Host 'ok'"
	o.CreateUser, o.UserName, o.UserPassword, o.UserDisplay = true, "joão", "a<b&c>\"'", "João & Cia"
	o.PostEnabled, o.PostCustom = true, "Foo.Bar"
	o.Locale, o.TimeZone, o.SetUILanguage = "pt-BR", "E. South America Standard Time", true
	r, err := Generate(o)
	if err != nil {
		t.Fatal(err)
	}
	if err := checkXML(r.XML); err != nil {
		t.Fatal(err)
	}
}

func TestVersionLess(t *testing.T) {
	for _, c := range []struct {
		a, b string
		w    bool
	}{{"1.2.0", "1.2.1", true}, {"1.2.0", "1.2.0", false}, {"1.10.0", "1.9.9", false}, {"1.2", "1.2.0.1", true}} {
		if versionLess(c.a, c.b) != c.w {
			t.Errorf("%v", c)
		}
	}
}
