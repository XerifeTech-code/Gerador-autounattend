#!/usr/bin/env bash
# Compila o Gerador de Autounattend (Xerife Tech) para Windows x64, x86 e ARM64
# e gera os hashes SHA-256 para publicar junto com a versão.
# Uso: ./build.sh            (saída na pasta dist/vX.Y.Z)
set -euo pipefail
cd "$(dirname "$0")"

VER=$(sed -n 's/^const appVersion = "\(.*\)"/\1/p' main.go)
OUT="dist/v$VER"
mkdir -p "$OUT"

go test ./...   # roda antes de copiar os recursos
python3 tools/make_syso.py app_icon.ico recursos/completo --version "$VER" --no-manifest
python3 tools/make_syso.py app_icon.ico recursos/icone --version "$VER" --no-manifest --no-version
cp recursos/completo/rsrc_windows_*.syso .
trap 'rm -f rsrc_windows_*.syso' EXIT

build() { # GOARCH nome
  CGO_ENABLED=0 GOOS=windows GOARCH="$1" go build -trimpath -ldflags "-s -w -H=windowsgui -buildid=" -o "$OUT/$2" .
  echo "ok $OUT/$2"
}
build amd64 GeradorAutounattend.exe
build 386   GeradorAutounattend_x86.exe
build arm64 GeradorAutounattend_arm64.exe

# código-fonte (sem a pasta dist)
git archive --format=zip -o "$OUT/GeradorAutounattend-fonte-v$VER.zip" HEAD 2>/dev/null || \
  zip -qr "$OUT/GeradorAutounattend-fonte-v$VER.zip" . -x 'dist/*' '.git/*' '*.syso' 'tools/__pycache__/*'

(
  cd "$OUT"
  sha256sum GeradorAutounattend*.exe | awk '{print toupper($1) " *" $2}' > SHA256SUMS.txt
  {
    echo "Gerador de Autounattend v$VER - Criado por Xerife Tech"
    echo "https://www.youtube.com/@xerifetech"
    echo
    echo "Hashes SHA-256 (confira antes de usar):"
    echo
    sha256sum GeradorAutounattend*.exe | awk '{printf "%s\n  %s\n\n", $2, toupper($1)}'
    echo "Como conferir no Windows (PowerShell):"
    echo "  Get-FileHash .\\GeradorAutounattend.exe -Algorithm SHA256"
    echo "O mesmo código aparece no programa, em Sobre (botão i)."
  } > "HASHES-v$VER.txt"
)
echo "pronto: $OUT"
