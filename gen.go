package main

import (
	"embed"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"unicode/utf16"
)

//go:embed app/catalog.json app/script.ps1 app/ext_head.txt app/ext_tail.txt app/index.html
var assets embed.FS

// ---------------------------------------------------------------- catálogo

type Item struct {
	ID     string   `json:"id"`
	Group  string   `json:"group"`
	Scope  string   `json:"scope"`
	Label  string   `json:"label"`
	Sub    string   `json:"sub,omitempty"`
	Risk   string   `json:"risk,omitempty"`
	Kind   string   `json:"kind"`
	List   string   `json:"list,omitempty"`
	Ranges [][2]int `json:"ranges"`
	// removido da ferramenta: nunca entra no arquivo gerado
	Removed bool `json:"removed,omitempty"`
}

type CatalogMeta struct {
	TotalLines int               `json:"total_lines"`
	SysStart   int               `json:"sys_start"`
	UsrStart   int               `json:"usr_start"`
	MarkerSys  int               `json:"marker_sys"`
	MarkerUsr  int               `json:"marker_usr"`
	Lists      map[string][2]int `json:"lists"`
}

type Catalog struct {
	Meta  CatalogMeta `json:"meta"`
	Items []Item      `json:"items"`
}

var (
	catalogRaw []byte
	catalog    Catalog
	scriptText string
	extHead    string
	extTail    string
	itemByID   map[string]*Item
)

func loadAssets() error {
	var err error
	catalogRaw, err = assets.ReadFile("app/catalog.json")
	if err != nil {
		return err
	}
	if err = json.Unmarshal(catalogRaw, &catalog); err != nil {
		return fmt.Errorf("catalog.json: %w", err)
	}
	b, err := assets.ReadFile("app/script.ps1")
	if err != nil {
		return err
	}
	scriptText = string(b)
	b, _ = assets.ReadFile("app/ext_head.txt")
	extHead = string(b)
	b, _ = assets.ReadFile("app/ext_tail.txt")
	extTail = string(b)
	itemByID = map[string]*Item{}
	for i := range catalog.Items {
		itemByID[catalog.Items[i].ID] = &catalog.Items[i]
	}
	return nil
}

// ---------------------------------------------------------------- opções

type Options struct {
	Version int    `json:"version,omitempty"`
	Lang    string `json:"lang,omitempty"` // pt-BR | en | es (idioma das mensagens)

	// Instalação (windowsPE)
	ProductKey       string `json:"productKey"`
	BypassTPM        bool   `json:"bypassTPM"`
	BypassSecureBoot bool   `json:"bypassSecureBoot"`
	BypassStorage    bool   `json:"bypassStorage"`
	BypassCPU        bool   `json:"bypassCPU"`
	BypassRAM        bool   `json:"bypassRAM"`
	BypassDisk       bool   `json:"bypassDisk"`
	ShowEditions     bool   `json:"showEditions"`
	WritePID         bool   `json:"writePID"`

	// Disco (perigoso)
	WipeDisk     bool   `json:"wipeDisk"`
	DiskID       int    `json:"diskID"`
	Firmware     string `json:"firmware"`     // uefi | bios
	RecoveryPart bool   `json:"recoveryPart"` // partição de recuperação (WinRE) de 1 GB

	// Drivers da pasta \Drivers do pendrive (WinPE + sistema instalado)
	DriversUSB bool `json:"driversUSB"`

	// Recursos opcionais do Windows ativados na instalação
	FeatWSL     bool `json:"featWSL"`
	FeatHyperV  bool `json:"featHyperV"`
	FeatSandbox bool `json:"featSandbox"`

	// Specialize
	RunScript  bool `json:"runScript"`
	BypassNRO  bool `json:"bypassNRO"`
	DisableNet bool `json:"disableNet"`
	NetFx3     bool `json:"netFx3"`

	// OOBE
	HideEULA        bool   `json:"hideEULA"`
	HideOEM         bool   `json:"hideOEM"`
	HideOnline      bool   `json:"hideOnline"`
	HideWireless    bool   `json:"hideWireless"`
	NetworkLocation string `json:"networkLocation"` // "", Work, Home, Other
	ProtectYourPC   int    `json:"protectYourPC"`   // 0 = não definir

	// Conta e região
	ComputerName  string `json:"computerName"`
	TimeZone      string `json:"timeZone"`
	Locale        string `json:"locale"`
	SetUILanguage bool   `json:"setUILanguage"`
	CreateUser    bool   `json:"createUser"`
	UserName      string `json:"userName"`
	UserDisplay   string `json:"userDisplay"`
	UserPassword  string `json:"userPassword"`
	UserAdmin     bool   `json:"userAdmin"`
	AutoLogon     bool   `json:"autoLogon"`
	AutoLogonN    int    `json:"autoLogonCount"`
	// senha gravada em Base64 (PlainText=false). Só esconde da leitura casual: NÃO é criptografia.
	PasswordBase64 bool        `json:"passwordBase64"`
	ExtraUsers     []ExtraUser `json:"extraUsers"`

	// Avançado
	FirstLogonCmds     string `json:"firstLogonCmds"`
	CustomSystemScript string `json:"customSystemScript"`
	CustomUserScript   string `json:"customUserScript"`

	// Pós-instalação (script winget na área de trabalho)
	PostEnabled bool     `json:"postEnabled"`
	PostApps    []string `json:"postApps"`
	PostCustom  string   `json:"postCustom"`
	PostAutoRun bool     `json:"postAutoRun"`
	PostDelete  bool     `json:"postDelete"`

	// itens do script: só as entradas "false" precisam constar (ausente = ligado)
	Items map[string]bool `json:"items"`
}

func DefaultOptions() *Options {
	return &Options{
		Version:          1,
		ProductKey:       "00000-00000-00000-00000-00000",
		BypassTPM:        true,
		BypassSecureBoot: true,
		BypassStorage:    true,
		BypassCPU:        true,
		BypassRAM:        true,
		BypassDisk:       true,
		ShowEditions:     true,
		WritePID:         true,
		Firmware:         "uefi",
		RunScript:        true,
		BypassNRO:        true,
		NetFx3:           true,
		HideEULA:         true,
		HideOEM:          true,
		HideOnline:       true,
		NetworkLocation:  "Work",
		ProtectYourPC:    3,
		UserAdmin:        true,
		AutoLogonN:       1,
		PostApps:         BasicPostApps(),
		Items:            map[string]bool{},
	}
}

// usuários locais adicionais (criados junto com o usuário principal)
type ExtraUser struct {
	Name     string `json:"name"`
	Display  string `json:"display"`
	Password string `json:"password"`
	Admin    bool   `json:"admin"`
}

// Chaves genéricas de INSTALAÇÃO publicadas pela Microsoft: escolhem a edição e
// pulam a pergunta da chave, mas NÃO ativam o Windows.
type Edition struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Key  string `json:"key"`
}

var Editions = []Edition{
	{"home", "Windows Home", "TX9XD-98N7V-6WMQ6-BX7FG-H8Q99"},
	{"home-n", "Windows Home N", "3KHY7-WNT83-DGQKR-F7HPR-844BM"},
	{"home-sl", "Windows Home Single Language", "7HNRX-D7KGG-3K4RQ-4WPJ4-YTDFH"},
	{"pro", "Windows Pro", "VK7JG-NPHTM-C97JM-9MPGT-3V66T"},
	{"pro-n", "Windows Pro N", "2B87N-8KFHP-DKV6R-Y2C8J-PKCKT"},
	{"pro-ws", "Windows Pro for Workstations", "DXG7C-N36C4-C4HTG-X4T3X-2YV77"},
	{"pro-edu", "Windows Pro Education", "8PTT6-RNW4C-6V7J2-C2D3X-MHBPB"},
	{"edu", "Windows Education", "YNMGQ-8RYV3-4PGQ3-C8XTP-7CFBY"},
	{"ent", "Windows Enterprise", "XGVPP-NMH47-7TTHJ-W3FW7-8HV2C"},
}

type Locale struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Input string `json:"input"`
	Tag   string `json:"tag"`
}

var Locales = []Locale{
	{"pt-BR", "Português (Brasil) — teclado ABNT2", "0416:00010416", "pt-BR"},
	{"pt-BR-us", "Português (Brasil) — teclado US", "0416:00000409", "pt-BR"},
	{"pt-PT", "Português (Portugal)", "0816:00000816", "pt-PT"},
	{"en-US", "English (United States)", "0409:00000409", "en-US"},
	{"en-GB", "English (United Kingdom)", "0809:00000809", "en-GB"},
	{"es-ES", "Español (España)", "0c0a:0000040a", "es-ES"},
	{"es-MX", "Español (México)", "080a:0000080a", "es-MX"},
	{"fr-FR", "Français (France)", "040c:0000040c", "fr-FR"},
	{"de-DE", "Deutsch (Deutschland)", "0407:00000407", "de-DE"},
	{"it-IT", "Italiano (Italia)", "0410:00000410", "it-IT"},
}

type TZ struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

var TimeZones = []TZ{
	{"E. South America Standard Time", "(UTC-03:00) Brasília"},
	{"Central Brazilian Standard Time", "(UTC-04:00) Cuiabá / Campo Grande"},
	{"SA Western Standard Time", "(UTC-04:00) Manaus / La Paz"},
	{"SA Pacific Standard Time", "(UTC-05:00) Rio Branco / Bogotá / Lima"},
	{"Argentina Standard Time", "(UTC-03:00) Buenos Aires"},
	{"UTC", "(UTC) Tempo Universal Coordenado"},
	{"GMT Standard Time", "(UTC+00:00) Lisboa / Londres"},
	{"W. Europe Standard Time", "(UTC+01:00) Berlim / Roma / Amsterdã"},
	{"Romance Standard Time", "(UTC+01:00) Paris / Madri"},
	{"Eastern Standard Time", "(UTC-05:00) Hora do Leste (EUA e Canadá)"},
	{"Central Standard Time", "(UTC-06:00) Hora Central (EUA e Canadá)"},
	{"Mountain Standard Time", "(UTC-07:00) Hora das Montanhas (EUA e Canadá)"},
	{"Pacific Standard Time", "(UTC-08:00) Hora do Pacífico (EUA e Canadá)"},
	{"Central Standard Time (Mexico)", "(UTC-06:00) Cidade do México"},
	{"Pacific SA Standard Time", "(UTC-04:00) Santiago"},
	{"Paraguay Standard Time", "(UTC-03:00) Assunção"},
	{"Montevideo Standard Time", "(UTC-03:00) Montevidéu"},
	{"Venezuela Standard Time", "(UTC-04:00) Caracas"},
	{"Atlantic Standard Time", "(UTC-04:00) Hora do Atlântico (Canadá)"},
	{"China Standard Time", "(UTC+08:00) Pequim / Hong Kong"},
	{"Tokyo Standard Time", "(UTC+09:00) Tóquio"},
	{"India Standard Time", "(UTC+05:30) Chennai / Nova Délhi"},
}

// ---------------------------------------------------------------- geração

type Stats struct {
	ItemsTotal  int  `json:"itemsTotal"`
	ItemsOn     int  `json:"itemsOn"`
	ScriptLines int  `json:"scriptLines"`
	Bytes       int  `json:"bytes"`
	PackagesOn  int  `json:"packagesOn"`
	PackagesAll int  `json:"packagesAll"`
	HasScript   bool `json:"hasScript"`
}

type Result struct {
	XML      string   `json:"xml"`
	Warnings []string `json:"warnings"`
	Stats    Stats    `json:"stats"`
}

var archs = []string{"x86", "arm64", "amd64"}

const pkt = `publicKeyToken="31bf3856ad364e35" language="neutral" versionScope="nonSxS"`

func esc(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

type rsCmd struct{ desc, path string }

var (
	reKey      = regexp.MustCompile(`^[A-Za-z0-9]{5}(-[A-Za-z0-9]{5}){4}$`)
	reCompName = regexp.MustCompile(`^[A-Za-z0-9-]{1,15}$`)
	reAllDigit = regexp.MustCompile(`^[0-9]+$`)
)

func enabled(o *Options, id string) bool {
	if it := itemByID[id]; it != nil && it.Removed {
		return false
	}
	v, ok := o.Items[id]
	if !ok {
		return true
	}
	return v
}

func validate(o *Options) error {
	if o.ProductKey != "" && !reKey.MatchString(o.ProductKey) {
		return errors.New(tr(o.Lang, "Chave do produto inválida: use o formato XXXXX-XXXXX-XXXXX-XXXXX-XXXXX"))
	}
	if o.ComputerName != "" && o.ComputerName != "*" {
		if !reCompName.MatchString(o.ComputerName) || reAllDigit.MatchString(o.ComputerName) {
			return errors.New(tr(o.Lang, "Nome do computador inválido: até 15 caracteres (letras, números e hífen), sem ser só números"))
		}
	}
	if o.CreateUser {
		u := strings.TrimSpace(o.UserName)
		if u == "" {
			return errors.New(tr(o.Lang, "Informe o nome do usuário local"))
		}
		if err := checkUserName(o.Lang, u); err != nil {
			return err
		}
		seen := map[string]bool{strings.ToLower(u): true}
		for _, x := range o.ExtraUsers {
			n := strings.TrimSpace(x.Name)
			if n == "" {
				continue
			}
			if err := checkUserName(o.Lang, n); err != nil {
				return err
			}
			if seen[strings.ToLower(n)] {
				return fmt.Errorf(tr(o.Lang, "O usuário %q aparece mais de uma vez"), n)
			}
			seen[strings.ToLower(n)] = true
		}
		if len(o.ExtraUsers) > 20 {
			return errors.New(tr(o.Lang, "No máximo 20 usuários adicionais"))
		}
		if o.AutoLogon && (o.AutoLogonN < 1 || o.AutoLogonN > 999) {
			return errors.New(tr(o.Lang, "Quantidade de logons automáticos deve estar entre 1 e 999"))
		}
	}
	if o.WipeDisk && (o.DiskID < 0 || o.DiskID > 32) {
		return errors.New(tr(o.Lang, "Número do disco inválido"))
	}
	return nil
}

func checkUserName(lang, u string) error {
	if len([]rune(u)) > 20 || strings.ContainsAny(u, `/\[]:;|=,+*?<>"@`) || strings.HasSuffix(u, ".") {
		return errors.New(tr(lang, "Nome de usuário inválido: até 20 caracteres e sem / \\ [ ] : ; | = , + * ? < > \" @"))
	}
	switch strings.ToLower(u) {
	case "administrator", "administrador", "guest", "convidado", "defaultaccount", "system", "wdagutilityaccount":
		return fmt.Errorf(tr(lang, "O nome %q é reservado pelo Windows; escolha outro"), u)
	}
	return nil
}

// account é um usuário local já validado
type account struct {
	name, display, password string
	admin                   bool
}

func accounts(o *Options) []account {
	if !o.CreateUser {
		return nil
	}
	name := strings.TrimSpace(o.UserName)
	a := []account{{name, strings.TrimSpace(o.UserDisplay), o.UserPassword, o.UserAdmin}}
	for _, x := range o.ExtraUsers {
		n := strings.TrimSpace(x.Name)
		if n == "" {
			continue
		}
		a = append(a, account{n, strings.TrimSpace(x.Display), x.Password, x.Admin})
	}
	return a
}

// encodePw codifica a senha como o Windows SIM faz: Base64 de UTF-16LE(senha + sufixo)
func encodePw(pw, suffix string) string {
	u := utf16.Encode([]rune(pw + suffix))
	b := make([]byte, 0, len(u)*2)
	for _, c := range u {
		b = append(b, byte(c), byte(c>>8))
	}
	return base64.StdEncoding.EncodeToString(b)
}

func writePw(b *strings.Builder, indent, pw string, enc bool) {
	if enc {
		fmt.Fprintf(b, "%s<Password>\n%s  <Value>%s</Value>\n%s  <PlainText>false</PlainText>\n%s</Password>\n", indent, indent, encodePw(pw, "Password"), indent, indent)
		return
	}
	fmt.Fprintf(b, "%s<Password>\n%s  <Value>%s</Value>\n%s  <PlainText>true</PlainText>\n%s</Password>\n", indent, indent, esc(pw), indent, indent)
}

// profileComment guarda as escolhas (sem senhas) dentro do XML para poder reabrir o arquivo na ferramenta.
func profileComment(o *Options) string {
	c := *o
	c.Lang = ""
	c.UserPassword = ""
	c.ExtraUsers = nil
	for _, x := range o.ExtraUsers {
		x.Password = ""
		c.ExtraUsers = append(c.ExtraUsers, x)
	}
	c.Items = map[string]bool{}
	for k, v := range o.Items {
		if !v {
			c.Items[k] = false
		}
	}
	j, err := json.Marshal(&c)
	if err != nil {
		return ""
	}
	return "  <!-- " + profileMarker + base64.StdEncoding.EncodeToString(j) + " -->\n"
}

const profileMarker = "XERIFE-PROFILE:"

// checkXML garante que o arquivo gerado é um XML bem formado.
func checkXML(s string) error {
	d := xml.NewDecoder(strings.NewReader(s))
	for {
		_, err := d.Token()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
	}
}

// Path de um comando RunSynchronous em PowerShell que nunca devolve erro (não trava a instalação)
func psCmd(script string) string {
	return esc(`powershell.exe -NoProfile -WindowStyle Hidden -ExecutionPolicy Bypass -Command "` + script + `; exit 0"`)
}

func featureCmd(feature string) string {
	return psCmd(`dism.exe /Online /Enable-Feature /FeatureName:` + feature + ` /All /NoRestart /Quiet | Out-Null`)
}

func Generate(o *Options) (*Result, error) {
	if err := validate(o); err != nil {
		return nil, err
	}
	// opções que mexiam na rede/Wi-Fi foram removidas da ferramenta
	o.DisableNet, o.HideWireless = false, false
	var postApps []postApp
	if o.PostEnabled {
		var err error
		if postApps, err = postSelection(o); err != nil {
			return nil, err
		}
	}
	var warns []string
	var b strings.Builder
	W := b.WriteString

	W(`<?xml version="1.0" encoding="utf-8"?>
<unattend xmlns="urn:schemas-microsoft-com:unattend" xmlns:wcm="http://schemas.microsoft.com/WMIConfig/2002/State">
  <!-- Gerado pelo Gerador de Autounattend v` + appVersion + ` - Criado por Xerife Tech - https://www.youtube.com/@xerifetech -->
  <!-- Ajustes baseados no Winhance (MIT) https://github.com/memstechtips/Winhance -->
`)
	W(profileComment(o))
	W(`  <settings pass="offlineServicing"></settings>
  <settings pass="windowsPE">
`)

	// ---- windowsPE / Microsoft-Windows-Setup
	key := o.ProductKey
	for _, arch := range archs {
		fmt.Fprintf(&b, "    <component name=\"Microsoft-Windows-Setup\" processorArchitecture=\"%s\" %s>\n", arch, pkt)
		if o.WipeDisk {
			writeDisk(&b, o)
		}
		W("      <UserData>\n")
		if key != "" {
			fmt.Fprintf(&b, "        <ProductKey>\n          <Key>%s</Key>\n        </ProductKey>\n", esc(key))
		}
		W("        <AcceptEula>true</AcceptEula>\n      </UserData>\n")

		var cmds []rsCmd
		hw := []struct {
			on   bool
			name string
		}{
			{o.BypassTPM, "BypassTPMCheck"}, {o.BypassSecureBoot, "BypassSecureBootCheck"},
			{o.BypassStorage, "BypassStorageCheck"}, {o.BypassCPU, "BypassCPUCheck"},
			{o.BypassRAM, "BypassRAMCheck"}, {o.BypassDisk, "BypassDiskCheck"},
		}
		start := len(cmds) + 1
		var hwc []rsCmd
		for _, h := range hw {
			if h.on {
				hwc = append(hwc, rsCmd{"", `reg.exe add "HKLM\SYSTEM\Setup\LabConfig" /v ` + h.name + ` /t REG_DWORD /d 1 /f`})
			}
		}
		if len(hwc) > 0 {
			hwc[0].desc = orderDesc(start, len(hwc), "Skip Windows 11 Hardware Requirement Checks")
			cmds = append(cmds, hwc...)
		}
		if o.DriversUSB {
			cmds = append(cmds, rsCmd{orderDesc(len(cmds)+1, 1, "Loads drivers from the Drivers folder of the installation media (Xerife Tech)"),
				`cmd.exe /c for %d in (C D E F G H I J K L M N O P Q R S T U V W Y Z) do @if exist "%d:\Drivers\" for /r "%d:\Drivers" %i in (*.inf) do @drvload.exe "%i"`})
		}
		if o.ShowEditions {
			ed := []rsCmd{
				{"", `cmd.exe /c del /f /q X:\Sources\ei.cfg`},
				{"", `cmd.exe /c echo [Channel] > X:\Sources\ei.cfg`},
				{"", `cmd.exe /c echo _Default >> X:\Sources\ei.cfg`},
				{"", `cmd.exe /c echo [VL] >> X:\Sources\ei.cfg`},
				{"", `cmd.exe /c echo 0 >> X:\Sources\ei.cfg`},
			}
			ed[0].desc = orderDesc(len(cmds)+1, len(ed), "Prevents auto detection of Windows Edition and forces Windows Setup to show all available Editions of Windows during setup")
			cmds = append(cmds, ed...)
		}
		if arch == "amd64" && o.WritePID {
			pk := key
			if pk == "" {
				pk = "00000-00000-00000-00000-00000"
			}
			cmds = append(cmds,
				rsCmd{"", `cmd.exe /c echo [PID] > X:\Sources\PID.txt`},
				rsCmd{"", `cmd.exe /c echo Value=` + esc(pk) + ` >> X:\Sources\PID.txt`})
		}
		if len(cmds) > 0 {
			W("      <RunSynchronous>\n")
			writeRS(&b, cmds, "RunSynchronousCommand", true)
			W("      </RunSynchronous>\n")
		}
		W("    </component>\n")
	}
	if o.Locale != "" {
		if loc := findLocale(o.Locale); loc != nil {
			for _, arch := range archs {
				fmt.Fprintf(&b, "    <component name=\"Microsoft-Windows-International-Core-WinPE\" processorArchitecture=\"%s\" %s>\n", arch, pkt)
				if o.SetUILanguage {
					fmt.Fprintf(&b, "      <SetupUILanguage>\n        <UILanguage>%s</UILanguage>\n      </SetupUILanguage>\n", loc.Tag)
				}
				fmt.Fprintf(&b, "      <InputLocale>%s</InputLocale>\n      <SystemLocale>%s</SystemLocale>\n", loc.Input, loc.Tag)
				if o.SetUILanguage {
					fmt.Fprintf(&b, "      <UILanguage>%s</UILanguage>\n", loc.Tag)
				}
				fmt.Fprintf(&b, "      <UserLocale>%s</UserLocale>\n    </component>\n", loc.Tag)
			}
			if o.SetUILanguage {
				warns = append(warns, tr(o.Lang, "O idioma de exibição só funciona se a ISO do Windows contiver esse idioma. Se a mídia não tiver, a instalação para com erro."))
			}
		}
	}
	W("  </settings>\n  <settings pass=\"generalize\"></settings>\n")

	// ---- specialize
	type spec struct {
		cond bool
		c    rsCmd
	}
	depl := []spec{
		{o.PostEnabled, rsCmd{"Creates the Desktop folder for the post-install script", `cmd.exe /c if not exist "C:\Users\Public\Desktop" mkdir "C:\Users\Public\Desktop"`}},
		{o.RunScript || o.PostEnabled, rsCmd{"Loads Scripts in this XML File", `powershell.exe -NoProfile -WindowStyle Hidden -Command "$xml = [xml]::new(); $xml.Load('C:\Windows\Panther\unattend.xml'); $sb = [scriptblock]::Create( $xml.unattend.Extensions.ExtractScript ); Invoke-Command -ScriptBlock $sb -ArgumentList $xml;"`}},
		{o.BypassNRO, rsCmd{"Registry Entry to Create a Local Account during OOBE", `reg.exe add "HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\OOBE" /v BypassNRO /t REG_DWORD /d 1 /f`}},
		{o.DisableNet, rsCmd{"Disable All Network Adapters Temporarily so Windows Doesn't Update During OOBE and to Allow Local Account Creation", `powershell.exe -NoProfile -WindowStyle Hidden -ExecutionPolicy Bypass -Command "Get-NetAdapter | Disable-NetAdapter -Confirm:$false"`}},
		{o.DriversUSB, rsCmd{"Installs drivers from the Drivers folder of the installation media (Xerife Tech)", psCmd(`foreach($d in Get-PSDrive -PSProvider FileSystem){$p=Join-Path $d.Root 'Drivers';if(Test-Path $p){pnputil.exe /add-driver (Join-Path $p '*.inf') /subdirs /install | Out-Null}}`)}},
		{o.FeatWSL, rsCmd{"Enables Windows Subsystem for Linux (Xerife Tech)", featureCmd("Microsoft-Windows-Subsystem-Linux")}},
		{o.FeatWSL, rsCmd{"Enables Virtual Machine Platform for WSL 2 (Xerife Tech)", featureCmd("VirtualMachinePlatform")}},
		{o.FeatHyperV, rsCmd{"Enables Hyper-V (Xerife Tech)", featureCmd("Microsoft-Hyper-V-All")}},
		{o.FeatSandbox, rsCmd{"Enables Windows Sandbox (Xerife Tech)", featureCmd("Containers-DisposableClientVM")}},
		{o.NetFx3, rsCmd{"Enables .NET Framework 3.5 from Windows Installation Media", `powershell.exe -NoProfile -WindowStyle Hidden -Command "foreach($d in 'C','D','E','F','G','H','I','J','K'){$src=Join-Path ($d+':') 'sources\sxs';if(Test-Path $src\*.cab){dism /Online /Enable-Feature /FeatureName:NetFx3 /All /LimitAccess /Source:$src;break}}"`}},
		{o.RunScript, rsCmd{"Runs Winhance Script System Wide Operations like App Uninstalls, HKLM Registry entries etc.", `powershell.exe -NoProfile -WindowStyle Hidden -ExecutionPolicy Bypass -File "C:\ProgramData\Winhance\Unattend\Scripts\Winhancements.ps1"`}},
	}
	var dcmds []rsCmd
	for _, d := range depl {
		if d.cond {
			dcmds = append(dcmds, d.c)
		}
	}
	compName := strings.TrimSpace(o.ComputerName)
	if len(dcmds) == 0 && compName == "" {
		W("  <settings pass=\"specialize\"></settings>\n")
	} else {
		W("  <settings pass=\"specialize\">\n")
		for _, arch := range archs {
			if len(dcmds) > 0 {
				fmt.Fprintf(&b, "    <component name=\"Microsoft-Windows-Deployment\" processorArchitecture=\"%s\" %s>\n      <RunSynchronous>\n", arch, pkt)
				writeRS(&b, dcmds, "RunSynchronousCommand", false)
				W("      </RunSynchronous>\n    </component>\n")
			}
			if compName != "" {
				fmt.Fprintf(&b, "    <component name=\"Microsoft-Windows-Shell-Setup\" processorArchitecture=\"%s\" %s>\n      <ComputerName>%s</ComputerName>\n    </component>\n", arch, pkt, esc(compName))
			}
		}
		W("  </settings>\n")
	}
	W("  <settings pass=\"auditSystem\"></settings>\n  <settings pass=\"auditUser\"></settings>\n")

	// ---- oobeSystem
	var flc []rsCmd
	if o.PostEnabled && o.PostAutoRun {
		flc = append(flc, rsCmd{"Runs the post-install (winget) script", `cmd.exe /c start "" "` + postBatPath + `"`})
	}
	if o.DisableNet {
		flc = append(flc, rsCmd{"Enables Network Adapters After OOBE Completes", `powershell.exe -NoProfile -WindowStyle Hidden -ExecutionPolicy Bypass -Command "Get-NetAdapter | Enable-NetAdapter -Confirm:$false"`})
	}
	for i, ln := range strings.Split(strings.ReplaceAll(o.FirstLogonCmds, "\r", ""), "\n") {
		ln = strings.TrimSpace(ln)
		if ln == "" {
			continue
		}
		flc = append(flc, rsCmd{fmt.Sprintf("Custom command %d", i+1), esc(ln)})
	}
	oobe := o.HideEULA || o.HideOEM || o.HideOnline || o.HideWireless || o.NetworkLocation != "" || o.ProtectYourPC != 0
	hasOOBE := oobe || o.CreateUser || o.TimeZone != "" || len(flc) > 0
	locOK := o.Locale != "" && findLocale(o.Locale) != nil
	if !hasOOBE && !locOK {
		W("  <settings pass=\"oobeSystem\"></settings>\n")
	} else {
		W("  <settings pass=\"oobeSystem\">\n")
		for _, arch := range archs {
			if hasOOBE {
				fmt.Fprintf(&b, "    <component name=\"Microsoft-Windows-Shell-Setup\" processorArchitecture=\"%s\" %s>\n", arch, pkt)
				if o.CreateUser {
					writeUser(&b, o)
				}
				if oobe {
					W("      <OOBE>\n")
					if o.HideEULA {
						W("        <HideEULAPage>true</HideEULAPage>\n")
					}
					if o.HideOEM {
						W("        <HideOEMRegistrationScreen>true</HideOEMRegistrationScreen>\n")
					}
					if o.HideOnline {
						W("        <HideOnlineAccountScreens>true</HideOnlineAccountScreens>\n")
					}
					if o.HideWireless {
						W("        <HideWirelessSetupInOOBE>true</HideWirelessSetupInOOBE>\n")
					}
					if o.NetworkLocation != "" {
						fmt.Fprintf(&b, "        <NetworkLocation>%s</NetworkLocation>\n", esc(o.NetworkLocation))
					}
					if o.ProtectYourPC != 0 {
						fmt.Fprintf(&b, "        <ProtectYourPC>%d</ProtectYourPC>\n", o.ProtectYourPC)
					}
					W("      </OOBE>\n")
				}
				if o.TimeZone != "" {
					fmt.Fprintf(&b, "      <TimeZone>%s</TimeZone>\n", esc(o.TimeZone))
				}
				if len(flc) > 0 {
					W("      <FirstLogonCommands>\n")
					for i, c := range flc {
						fmt.Fprintf(&b, "        <SynchronousCommand>\n        <Order>%d</Order>\n        <Description>%s</Description>\n        <CommandLine>%s</CommandLine>\n        </SynchronousCommand>\n", i+1, esc(c.desc), c.path)
					}
					W("    </FirstLogonCommands>\n")
				}
				W("    </component>\n")
			}
			if locOK {
				loc := findLocale(o.Locale)
				fmt.Fprintf(&b, "    <component name=\"Microsoft-Windows-International-Core\" processorArchitecture=\"%s\" %s>\n      <InputLocale>%s</InputLocale>\n      <SystemLocale>%s</SystemLocale>\n", arch, pkt, loc.Input, loc.Tag)
				if o.SetUILanguage {
					fmt.Fprintf(&b, "      <UILanguage>%s</UILanguage>\n", loc.Tag)
				}
				fmt.Fprintf(&b, "      <UserLocale>%s</UserLocale>\n    </component>\n", loc.Tag)
			}
		}
		W("  </settings>\n")
	}

	// ---- script embutido
	st := Stats{HasScript: o.RunScript}
	for i := range catalog.Items {
		if catalog.Items[i].Removed {
			continue
		}
		st.ItemsTotal++
		if enabled(o, catalog.Items[i].ID) {
			st.ItemsOn++
		}
		if catalog.Items[i].Kind == "package" {
			st.PackagesAll++
			if enabled(o, catalog.Items[i].ID) {
				st.PackagesOn++
			}
		}
	}
	if o.RunScript || o.PostEnabled {
		fi := strings.Index(extHead, "    <File path=")
		ci := strings.Index(extTail, "  </Extensions>")
		W(extHead[:fi])
		if o.RunScript {
			script, nl := buildScript(o)
			st.ScriptLines = nl
			W(extHead[fi:])
			W(strings.ReplaceAll(script, "]]>", "]]]]><![CDATA[>"))
			W(extTail[:ci])
			warns = append(warns, scriptWarnings(o)...)
		}
		if o.PostEnabled {
			W(postFiles(o, postApps))
			if len(postApps) == 0 {
				warns = append(warns, tr(o.Lang, "Pós-instalação ligada, mas nenhum programa foi escolhido: o script será criado vazio."))
			}
		}
		W(extTail[ci:])
	} else {
		W("</unattend>\n")
	}
	if !o.RunScript {
		warns = append(warns, tr(o.Lang, "O script de ajustes (Winhancements.ps1) está desligado: nenhuma remoção de apps ou ajuste de registro será aplicado."))
	}

	hasPw := false
	for _, a := range accounts(o) {
		if a.password != "" {
			hasPw = true
		}
	}
	if hasPw && o.PasswordBase64 {
		warns = append(warns, tr(o.Lang, "A senha fica apenas CODIFICADA (Base64) no XML, não criptografada: quem tiver o arquivo consegue ler. Não compartilhe nem publique o arquivo."))
	} else if hasPw {
		warns = append(warns, tr(o.Lang, "A senha do usuário fica em TEXTO PURO dentro do XML. Não compartilhe nem publique o arquivo."))
	}
	if !o.CreateUser {
		warns = append(warns, tr(o.Lang, "Nenhum usuário local definido. Nas versões novas do Windows 11 (25H2 em diante) a Microsoft removeu os atalhos de conta local (BypassNRO); o jeito garantido é ligar \"Criar um usuário local\" na aba Conta e região."))
	}
	if o.FeatHyperV || o.FeatSandbox {
		warns = append(warns, tr(o.Lang, "Hyper-V e Área Restrita do Windows só existem nas edições Pro, Education e Enterprise; no Home o comando é ignorado."))
	}
	if o.WipeDisk {
		warns = append(warns, fmt.Sprintf(tr(o.Lang, "ATENÇÃO: a instalação vai APAGAR o disco %d sem perguntar. Use somente em máquina que pode ser formatada."), o.DiskID))
	}

	out := b.String()
	if err := checkXML(out); err != nil {
		return nil, fmt.Errorf(tr(o.Lang, "O XML gerado ficou inválido (%v). Revise os comandos e scripts próprios."), err)
	}
	st.Bytes = len(out)
	return &Result{XML: out, Warnings: append([]string{}, dedupe(warns)...), Stats: st}, nil
}

func dedupe(s []string) []string {
	seen := map[string]bool{}
	var o []string
	for _, x := range s {
		if !seen[x] {
			seen[x] = true
			o = append(o, x)
		}
	}
	return o
}

func orderDesc(start, n int, text string) string {
	if n == 1 {
		return fmt.Sprintf("Order %d: %s", start, text)
	}
	return fmt.Sprintf("Order %d-%d: %s", start, start+n-1, text)
}

func writeRS(b *strings.Builder, cmds []rsCmd, tag string, _ bool) {
	for i, c := range cmds {
		fmt.Fprintf(b, "        <%s wcm:action=\"add\">\n          <Order>%d</Order>\n", tag, i+1)
		if c.desc != "" {
			fmt.Fprintf(b, "          <Description>%s</Description>\n", esc(c.desc))
		}
		fmt.Fprintf(b, "          <Path>%s</Path>\n        </%s>\n", c.path, tag)
	}
}

func findLocale(id string) *Locale {
	for i := range Locales {
		if Locales[i].ID == id {
			return &Locales[i]
		}
	}
	return nil
}

func writeDisk(b *strings.Builder, o *Options) {
	fmt.Fprintf(b, "      <DiskConfiguration>\n        <Disk wcm:action=\"add\">\n          <DiskID>%d</DiskID>\n          <WillWipeDisk>true</WillWipeDisk>\n          <CreatePartitions>\n", o.DiskID)
	var installPart int
	if o.RecoveryPart {
		writeDiskRecovery(b, o)
		return
	}
	if o.Firmware == "bios" {
		installPart = 2
		b.WriteString(`            <CreatePartition wcm:action="add">
              <Order>1</Order>
              <Type>Primary</Type>
              <Size>100</Size>
            </CreatePartition>
            <CreatePartition wcm:action="add">
              <Order>2</Order>
              <Type>Primary</Type>
              <Extend>true</Extend>
            </CreatePartition>
          </CreatePartitions>
          <ModifyPartitions>
            <ModifyPartition wcm:action="add">
              <Order>1</Order>
              <PartitionID>1</PartitionID>
              <Active>true</Active>
              <Format>NTFS</Format>
              <Label>System Reserved</Label>
            </ModifyPartition>
            <ModifyPartition wcm:action="add">
              <Order>2</Order>
              <PartitionID>2</PartitionID>
              <Format>NTFS</Format>
              <Label>Windows</Label>
            </ModifyPartition>
          </ModifyPartitions>
`)
	} else {
		installPart = 3
		b.WriteString(`            <CreatePartition wcm:action="add">
              <Order>1</Order>
              <Type>EFI</Type>
              <Size>300</Size>
            </CreatePartition>
            <CreatePartition wcm:action="add">
              <Order>2</Order>
              <Type>MSR</Type>
              <Size>16</Size>
            </CreatePartition>
            <CreatePartition wcm:action="add">
              <Order>3</Order>
              <Type>Primary</Type>
              <Extend>true</Extend>
            </CreatePartition>
          </CreatePartitions>
          <ModifyPartitions>
            <ModifyPartition wcm:action="add">
              <Order>1</Order>
              <PartitionID>1</PartitionID>
              <Format>FAT32</Format>
              <Label>System</Label>
            </ModifyPartition>
            <ModifyPartition wcm:action="add">
              <Order>2</Order>
              <PartitionID>2</PartitionID>
            </ModifyPartition>
            <ModifyPartition wcm:action="add">
              <Order>3</Order>
              <PartitionID>3</PartitionID>
              <Format>NTFS</Format>
              <Label>Windows</Label>
            </ModifyPartition>
          </ModifyPartitions>
`)
	}
	fmt.Fprintf(b, "        </Disk>\n      </DiskConfiguration>\n      <ImageInstall>\n        <OSImage>\n          <InstallTo>\n            <DiskID>%d</DiskID>\n            <PartitionID>%d</PartitionID>\n          </InstallTo>\n        </OSImage>\n      </ImageInstall>\n", o.DiskID, installPart)
}

func writeUser(b *strings.Builder, o *Options) {
	accs := accounts(o)
	b.WriteString("      <UserAccounts>\n        <LocalAccounts>\n")
	for _, a := range accs {
		disp := a.display
		if disp == "" {
			disp = a.name
		}
		grp := "Users"
		if a.admin {
			grp = "Administrators"
		}
		fmt.Fprintf(b, "          <LocalAccount wcm:action=\"add\">\n            <Name>%s</Name>\n            <DisplayName>%s</DisplayName>\n            <Group>%s</Group>\n", esc(a.name), esc(disp), grp)
		if a.password != "" {
			writePw(b, "            ", a.password, o.PasswordBase64)
		}
		b.WriteString("          </LocalAccount>\n")
	}
	b.WriteString("        </LocalAccounts>\n      </UserAccounts>\n")
	if o.AutoLogon && len(accs) > 0 {
		b.WriteString("      <AutoLogon>\n")
		writePw(b, "        ", accs[0].password, o.PasswordBase64)
		fmt.Fprintf(b, "        <Enabled>true</Enabled>\n        <LogonCount>%d</LogonCount>\n        <Username>%s</Username>\n      </AutoLogon>\n", o.AutoLogonN, esc(accs[0].name))
	}
}

// writeDiskRecovery: igual ao layout padrão, com uma partição de recuperação (WinRE) de 1 GB antes do Windows
func writeDiskRecovery(b *strings.Builder, o *Options) {
	var installPart int
	if o.Firmware == "bios" {
		installPart = 3
		b.WriteString(`            <CreatePartition wcm:action="add">
              <Order>1</Order>
              <Type>Primary</Type>
              <Size>100</Size>
            </CreatePartition>
            <CreatePartition wcm:action="add">
              <Order>2</Order>
              <Type>Primary</Type>
              <Size>1024</Size>
            </CreatePartition>
            <CreatePartition wcm:action="add">
              <Order>3</Order>
              <Type>Primary</Type>
              <Extend>true</Extend>
            </CreatePartition>
          </CreatePartitions>
          <ModifyPartitions>
            <ModifyPartition wcm:action="add">
              <Order>1</Order>
              <PartitionID>1</PartitionID>
              <Active>true</Active>
              <Format>NTFS</Format>
              <Label>System Reserved</Label>
            </ModifyPartition>
            <ModifyPartition wcm:action="add">
              <Order>2</Order>
              <PartitionID>2</PartitionID>
              <Format>NTFS</Format>
              <Label>Recovery</Label>
              <TypeID>0x27</TypeID>
            </ModifyPartition>
            <ModifyPartition wcm:action="add">
              <Order>3</Order>
              <PartitionID>3</PartitionID>
              <Format>NTFS</Format>
              <Label>Windows</Label>
            </ModifyPartition>
          </ModifyPartitions>
`)
	} else {
		installPart = 4
		b.WriteString(`            <CreatePartition wcm:action="add">
              <Order>1</Order>
              <Type>EFI</Type>
              <Size>300</Size>
            </CreatePartition>
            <CreatePartition wcm:action="add">
              <Order>2</Order>
              <Type>MSR</Type>
              <Size>16</Size>
            </CreatePartition>
            <CreatePartition wcm:action="add">
              <Order>3</Order>
              <Type>Primary</Type>
              <Size>1024</Size>
            </CreatePartition>
            <CreatePartition wcm:action="add">
              <Order>4</Order>
              <Type>Primary</Type>
              <Extend>true</Extend>
            </CreatePartition>
          </CreatePartitions>
          <ModifyPartitions>
            <ModifyPartition wcm:action="add">
              <Order>1</Order>
              <PartitionID>1</PartitionID>
              <Format>FAT32</Format>
              <Label>System</Label>
            </ModifyPartition>
            <ModifyPartition wcm:action="add">
              <Order>2</Order>
              <PartitionID>2</PartitionID>
            </ModifyPartition>
            <ModifyPartition wcm:action="add">
              <Order>3</Order>
              <PartitionID>3</PartitionID>
              <Format>NTFS</Format>
              <Label>Recovery</Label>
              <TypeID>DE94BBA4-06D1-4D40-A16A-BFD50179D6AC</TypeID>
            </ModifyPartition>
            <ModifyPartition wcm:action="add">
              <Order>4</Order>
              <PartitionID>4</PartitionID>
              <Format>NTFS</Format>
              <Label>Windows</Label>
            </ModifyPartition>
          </ModifyPartitions>
`)
	}
	fmt.Fprintf(b, "        </Disk>\n      </DiskConfiguration>\n      <ImageInstall>\n        <OSImage>\n          <InstallTo>\n            <DiskID>%d</DiskID>\n            <PartitionID>%d</PartitionID>\n          </InstallTo>\n        </OSImage>\n      </ImageInstall>\n", o.DiskID, installPart)
}

// buildScript aplica as opções ao Winhancements.ps1: remove as faixas de linhas dos
// itens desligados, corrige vírgulas finais de listas e insere scripts próprios.
func buildScript(o *Options) (string, int) {
	lines := strings.Split(scriptText, "\n")
	removed := make([]bool, len(lines)+2)
	for i := range catalog.Items {
		it := &catalog.Items[i]
		if enabled(o, it.ID) {
			continue
		}
		for _, r := range it.Ranges {
			for ln := r[0]; ln <= r[1] && ln < len(removed); ln++ {
				removed[ln] = true
			}
		}
	}
	// vírgula final em listas @( ... , ... )
	fix := map[int]bool{}
	for _, rg := range catalog.Meta.Lists {
		last := 0
		for ln := rg[0]; ln <= rg[1]; ln++ {
			if !removed[ln] {
				last = ln
			}
		}
		if last != 0 && last != rg[1] {
			fix[last] = true
		}
	}
	out := make([]string, 0, len(lines))
	for idx, ln := range lines {
		n := idx + 1
		if removed[n] {
			continue
		}
		if fix[n] {
			t := strings.TrimRight(ln, " \t")
			if strings.HasSuffix(t, ",") {
				ln = strings.TrimSuffix(t, ",")
			}
		}
		out = append(out, ln)
		if n == catalog.Meta.MarkerSys && strings.TrimSpace(o.CustomSystemScript) != "" {
			out = append(out, strings.Split(strings.ReplaceAll(strings.TrimRight(o.CustomSystemScript, "\r\n \t"), "\r", ""), "\n")...)
		}
		if n == catalog.Meta.MarkerUsr && strings.TrimSpace(o.CustomUserScript) != "" {
			out = append(out, strings.Split(strings.ReplaceAll(strings.TrimRight(o.CustomUserScript, "\r\n \t"), "\r", ""), "\n")...)
		}
	}
	return strings.Join(out, "\n"), len(out)
}

func scriptWarnings(o *Options) []string {
	var w []string
	usrOn := 0
	for i := range catalog.Items {
		it := &catalog.Items[i]
		if it.Scope == "usr" && it.Kind != "master" && enabled(o, it.ID) {
			usrOn++
		}
	}
	if !enabled(o, "usertask") && (usrOn > 0 || strings.TrimSpace(o.CustomUserScript) != "") {
		w = append(w, tr(o.Lang, "O ajuste \"Aplicar ajustes de usuário no primeiro logon\" está desligado: os ajustes de usuário (HKCU) e o script de usuário NÃO serão aplicados."))
	}
	if enabled(o, "script-edge") {
		w = append(w, tr(o.Lang, "O Microsoft Edge será removido. Tenha outro navegador (instale após o primeiro logon)."))
	}
	if enabled(o, "sys-privacy-consentpromptbehavioradmin") {
		w = append(w, tr(o.Lang, "O UAC (aviso de elevação) será desativado pelo ajuste de privacidade e segurança."))
	}
	return w
}
