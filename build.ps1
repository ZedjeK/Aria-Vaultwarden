# Aria Vault : compile Aria-Vault.exe.
# Normalement lancé par la CI (.github/workflows/build.yml), qui compile
# Vaultwarden depuis le source officiel dani-garcia/vaultwarden et récupère
# l'interface web officielle vérifiée par checksum, puis appelle ce script.
#
# Usage : powershell -ExecutionPolicy Bypass -File .\build.ps1 -Vaultwarden <dossier> [-VaultwardenVersion 1.37.4] [-Version 0.2.0]
#   <dossier> contient vaultwarden.exe et web-vault\
# Prérequis : Go 1.26 ou plus, Node.js LTS.

param(
    [Parameter(Mandatory = $true)] [string] $Vaultwarden,
    [string] $VaultwardenVersion = '',
    [string] $Version = '0.7.1'
)

$ErrorActionPreference = 'Stop'

$Root     = $PSScriptRoot
$Src      = Join-Path $Root 'src'
$Frontend = Join-Path $Src 'frontend'
$Assets   = Join-Path $Src 'internal\vaultwarden\assets'
$WinresVersion = 'v0.3.3'

function Step($msg) { Write-Host "`n==> $msg" -ForegroundColor Cyan }

function Invoke-Native {
    param([string]$Exe, [string[]]$Arguments)
    & $Exe @Arguments
    if ($LASTEXITCODE -ne 0) { throw "$Exe a échoué (code $LASTEXITCODE)" }
}

Step 'Vérification des outils'
if (-not (Get-Command 'go' -ErrorAction SilentlyContinue)) { throw 'go introuvable. Installe Go 1.26 ou plus : https://go.dev/dl/' }
if (-not (Get-Command 'npm' -ErrorAction SilentlyContinue)) { throw 'npm introuvable. Installe Node.js LTS : https://nodejs.org/' }
if (-not (Get-Command 'tar.exe' -ErrorAction SilentlyContinue)) { throw 'tar.exe introuvable (fourni avec Windows 10 et 11).' }
Write-Host ("  " + (go version))
Write-Host ("  node " + (node --version))

Step "Vaultwarden à embarquer depuis $Vaultwarden"
$exe = Join-Path $Vaultwarden 'vaultwarden.exe'
$web = Join-Path $Vaultwarden 'web-vault'
if (-not (Test-Path $exe)) { throw "$exe introuvable. Indique le bon dossier avec -Vaultwarden." }
if (-not (Test-Path (Join-Path $web 'index.html'))) { throw "$web\index.html introuvable : l'interface web doit être à côté de vaultwarden.exe." }
Write-Host ("  vaultwarden.exe SHA-256 : " + (Get-FileHash $exe -Algorithm SHA256).Hash.ToLower())

New-Item -ItemType Directory -Force -Path $Assets | Out-Null

# vaultwarden.exe -> vaultwarden.exe.gz
$gzPath = Join-Path $Assets 'vaultwarden.exe.gz'
$in  = [IO.File]::OpenRead((Resolve-Path $exe))
$out = [IO.File]::Create($gzPath)
try {
    $gz = New-Object IO.Compression.GZipStream($out, [IO.Compression.CompressionLevel]::Optimal)
    try { $in.CopyTo($gz) } finally { $gz.Dispose() }
} finally {
    $in.Dispose()
    $out.Dispose()
}
Write-Host '  vaultwarden.exe.gz prêt'

# web-vault\ -> web-vault.tar.gz (dossier racine « web-vault »)
Invoke-Native 'tar.exe' @('-czf', (Join-Path $Assets 'web-vault.tar.gz'), '-C', $Vaultwarden, 'web-vault')
Write-Host '  web-vault.tar.gz prêt'

# Version affichée dans l'interface (sans retour à la ligne final)
[IO.File]::WriteAllText((Join-Path $Assets 'version.txt'), $VaultwardenVersion)

Step 'Interface (Svelte)'
Push-Location $Frontend
try {
    Invoke-Native 'npm' @('ci', '--no-audit', '--no-fund')
    Invoke-Native 'npm' @('run', 'build')
} finally {
    Pop-Location
}

Step 'Ressources Windows (icône, manifeste, version)'
Push-Location $Src
try {
    Invoke-Native 'go' @('run', "github.com/tc-hib/go-winres@$WinresVersion", 'simply',
        '--arch', 'amd64',
        '--manifest', 'gui',
        '--icon', 'build\icon.ico',
        '--product-name', 'Aria Vault',
        '--file-description', 'Aria Vault',
        '--product-version', $Version,
        '--file-version', $Version,
        '--copyright', '© 2026 ZedjeK · licence MIT',
        '--original-filename', 'Aria-Vault.exe',
        '--out', 'rsrc')

    Step 'Compilation'
    $env:GOOS = 'windows'
    $env:GOARCH = 'amd64'
    Invoke-Native 'go' @('build', '-trimpath', '-ldflags', "-H windowsgui -s -w -X main.version=$Version", '-o', (Join-Path $Root 'Aria-Vault.exe'), '.')
} finally {
    Remove-Item Env:GOOS, Env:GOARCH -ErrorAction SilentlyContinue
    Pop-Location
}

Step 'Terminé'
Write-Host "  Exe prêt : $(Join-Path $Root 'Aria-Vault.exe')" -ForegroundColor Green
