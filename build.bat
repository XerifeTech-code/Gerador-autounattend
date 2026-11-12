@echo off
rem ============================================================================
rem  Gerador de Autounattend - Criado por Xerife Tech
rem  Compila os .exe (x64, x86 e ARM64), testa, coloca o icone e gera os hashes SHA-256.
rem  Requisito: Go 1.24+  (se nao tiver, este script instala sozinho)
rem  Uso: dois cliques neste arquivo, ou "build.bat" no terminal.
rem ============================================================================
setlocal EnableExtensions
chcp 65001 >nul
cd /d "%~dp0"

call :garantir_go
if errorlevel 1 goto :fim_erro
go version

set "VER="
for /f "tokens=4 delims= " %%v in ('findstr /b /c:"const appVersion" main.go') do set "VER=%%~v"
if not defined VER (
  echo [ERRO] Nao achei a versao em main.go.
  goto :fim_erro
)
set "OUT=dist\v%VER%"
echo.
echo  Gerador de Autounattend v%VER% - Xerife Tech
echo  Saida: %OUT%
echo.
if not exist "%OUT%" mkdir "%OUT%"

rem os .syso da pasta recursos sao so para o build.sh (Linux); aqui o icone e colocado depois
del /q rsrc_windows_*.syso >nul 2>nul

echo [1/4] Compilando...
set "CGO_ENABLED=0"
set "GOOS=windows"
set "FLAGS=-s -w -H=windowsgui -buildid="
set "GOARCH=amd64"
go build -trimpath -ldflags "%FLAGS%" -o "%OUT%\GeradorAutounattend.exe" .
if errorlevel 1 goto :compilar_falhou
set "GOARCH=386"
go build -trimpath -ldflags "%FLAGS%" -o "%OUT%\GeradorAutounattend_x86.exe" .
if errorlevel 1 goto :compilar_falhou
set "GOARCH=arm64"
go build -trimpath -ldflags "%FLAGS%" -o "%OUT%\GeradorAutounattend_arm64.exe" .
if errorlevel 1 goto :compilar_falhou
set "GOARCH="
set "GOOS="
set "CGO_ENABLED="
set "EXES="%OUT%\GeradorAutounattend.exe" "%OUT%\GeradorAutounattend_x86.exe" "%OUT%\GeradorAutounattend_arm64.exe""

echo [2/4] Testando o programa neste PC...
call :testar
if errorlevel 1 (
  echo [ERRO] O Windows nao deixou executar o .exe compilado.
  echo        Quase sempre e o antivirus bloqueando programas novos sem assinatura.
  echo        Adicione esta pasta como excecao no antivirus e rode o build.bat de novo:
  echo        %CD%
  goto :fim_erro
)

echo [3/4] Colocando o icone e os detalhes da Xerife Tech...
go build -o "%OUT%\winres.exe" ./tools/winres
if errorlevel 1 goto :sem_icone
for %%E in (%EXES%) do copy /y "%%~E" "%%~E.bak" >nul
"%OUT%\winres.exe" -ico app_icon.ico -version %VER% %EXES%
if errorlevel 1 goto :restaurar
call :testar
if not errorlevel 1 goto :icone_ok
echo  [aviso] Com os detalhes de versao o .exe nao abriu. Tentando so com o icone...
for %%E in (%EXES%) do copy /y "%%~E.bak" "%%~E" >nul
"%OUT%\winres.exe" -ico app_icon.ico -version %VER% -noversion %EXES%
if errorlevel 1 goto :restaurar
call :testar
if not errorlevel 1 goto :icone_ok
:restaurar
for %%E in (%EXES%) do copy /y "%%~E.bak" "%%~E" >nul
:sem_icone
echo  [aviso] Nao foi possivel colocar o icone; os .exe ficaram sem icone. Me mande um print desta janela.
goto :limpar
:icone_ok
"%OUT%\winres.exe" -check %EXES%
:limpar
for %%E in (%EXES%) do del /q "%%~E.bak" >nul 2>nul
del /q "%OUT%\winres.exe" "%OUT%\teste.xml" >nul 2>nul

echo [4/4] Hashes SHA-256...
powershell -NoProfile -ExecutionPolicy Bypass -Command ^
  "$o='%OUT%'; $v='%VER%';" ^
  "$h = Get-ChildItem (Join-Path $o 'GeradorAutounattend*.exe') | Sort-Object Name | Get-FileHash -Algorithm SHA256;" ^
  "$h | ForEach-Object { $_.Hash + ' *' + (Split-Path $_.Path -Leaf) } | Set-Content -Encoding ASCII (Join-Path $o 'SHA256SUMS.txt');" ^
  "$t = @('Gerador de Autounattend v' + $v + ' - Criado por Xerife Tech','https://www.youtube.com/@xerifetech','','Hashes SHA-256 (confira antes de usar):','');" ^
  "foreach ($x in $h) { $t += (Split-Path $x.Path -Leaf); $t += ('  ' + $x.Hash); $t += '' };" ^
  "$t += 'Como conferir no Windows (PowerShell):'; $t += '  Get-FileHash .\GeradorAutounattend.exe -Algorithm SHA256'; $t += 'O mesmo codigo aparece no programa, em Sobre (botao i).';" ^
  "$t | Set-Content -Encoding UTF8 (Join-Path $o ('HASHES-v' + $v + '.txt'))"
if errorlevel 1 goto :fim_erro
echo.
type "%OUT%\SHA256SUMS.txt"
echo.
echo  Pronto! Arquivos em: %CD%\%OUT%
rem atualiza o cache de icones do Explorer (senao ele pode mostrar o icone antigo)
ie4uinit.exe -show >nul 2>nul
explorer "%OUT%"
goto :fim

rem ============================================================================
:testar
rem ---- abre o .exe deste PC (escondido) e confere se ele gera um autounattend.xml de teste
set "TESTE=GeradorAutounattend.exe"
if /i "%PROCESSOR_ARCHITECTURE%"=="x86" if not defined PROCESSOR_ARCHITEW6432 set "TESTE=GeradorAutounattend_x86.exe"
del /q "%OUT%\teste.xml" >nul 2>nul
powershell -NoProfile -ExecutionPolicy Bypass -Command ^
  "try { $p = Start-Process -FilePath '%CD%\%OUT%\%TESTE%' -ArgumentList '--cli','%OUT%\teste.xml' -Wait -PassThru -WindowStyle Hidden; if ($p.ExitCode -ne 0) { Write-Host ('   o programa saiu com codigo ' + $p.ExitCode); exit 2 } } catch { Write-Host ('   ' + $_.Exception.Message); exit 1 }"
if errorlevel 1 exit /b 1
if not exist "%OUT%\teste.xml" (
  echo    O programa abriu, mas nao gerou o XML de teste.
  exit /b 1
)
del /q "%OUT%\teste.xml" >nul 2>nul
echo    OK: o programa abriu e gerou um autounattend.xml de teste.
exit /b 0

:compilar_falhou
set "GOARCH="
set "GOOS="
echo [ERRO] O Go nao conseguiu compilar. Veja a mensagem acima.
goto :fim_erro

rem ============================================================================
:garantir_go
rem ---- procura o Go; se nao achar, instala (winget e, se falhar, versao portatil de go.dev)
where go >nul 2>nul && exit /b 0
if exist "%ProgramFiles%\Go\bin\go.exe" set "PATH=%ProgramFiles%\Go\bin;%PATH%"
if exist "%LOCALAPPDATA%\XerifeTech\go\bin\go.exe" set "PATH=%LOCALAPPDATA%\XerifeTech\go\bin;%PATH%"
where go >nul 2>nul && exit /b 0
echo.
echo  O Go nao esta instalado neste computador. Instalando automaticamente...
where winget >nul 2>nul
if errorlevel 1 goto :go_portatil
echo  [winget] Instalando o Go oficial (pode aparecer um pedido de permissao)...
winget install -e --id GoLang.Go --silent --accept-package-agreements --accept-source-agreements
if exist "%ProgramFiles%\Go\bin\go.exe" set "PATH=%ProgramFiles%\Go\bin;%PATH%"
where go >nul 2>nul && exit /b 0
echo  O winget nao conseguiu. Tentando a versao portatil...
:go_portatil
echo  [go.dev] Baixando o Go portatil (uns 80 MB, nao precisa de administrador)...
powershell -NoProfile -ExecutionPolicy Bypass -Command ^
  "$ErrorActionPreference='Stop'; $ProgressPreference='SilentlyContinue';" ^
  "[Net.ServicePointManager]::SecurityProtocol=[Net.SecurityProtocolType]::Tls12;" ^
  "$v=((Invoke-WebRequest 'https://go.dev/VERSION?m=text' -UseBasicParsing).Content -split [char]10)[0].Trim();" ^
  "$a='amd64'; if($env:PROCESSOR_ARCHITECTURE -eq 'ARM64' -or $env:PROCESSOR_ARCHITEW6432 -eq 'ARM64'){$a='arm64'} elseif($env:PROCESSOR_ARCHITECTURE -eq 'x86' -and -not $env:PROCESSOR_ARCHITEW6432){$a='386'};" ^
  "$u='https://go.dev/dl/'+$v+'.windows-'+$a+'.zip'; Write-Host ('   '+$u);" ^
  "$d=Join-Path $env:LOCALAPPDATA 'XerifeTech'; New-Item -ItemType Directory -Force -Path $d | Out-Null;" ^
  "$z=Join-Path $env:TEMP ($v+'.zip'); Invoke-WebRequest $u -OutFile $z -UseBasicParsing;" ^
  "if(Test-Path (Join-Path $d 'go')){Remove-Item (Join-Path $d 'go') -Recurse -Force};" ^
  "Write-Host '   Extraindo...'; if(Get-Command tar.exe -ErrorAction SilentlyContinue){tar.exe -xf $z -C $d} else {Expand-Archive $z -DestinationPath $d -Force};" ^
  "Remove-Item $z -Force"
if exist "%LOCALAPPDATA%\XerifeTech\go\bin\go.exe" set "PATH=%LOCALAPPDATA%\XerifeTech\go\bin;%PATH%"
where go >nul 2>nul && exit /b 0
echo [ERRO] Nao foi possivel instalar o Go automaticamente. Verifique a internet
echo        ou instale manualmente em https://go.dev/dl/ e rode o build.bat de novo.
exit /b 1

:fim_erro
echo.
echo  A compilacao falhou. Veja a mensagem acima.
:fim
echo.
pause
endlocal
