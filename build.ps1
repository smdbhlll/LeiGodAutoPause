$ErrorActionPreference = "Stop"

$projectDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$distDir = Join-Path $projectDir "dist"
$output = Join-Path $distDir "LeiGodAutoPause.exe"
$icon = Join-Path $projectDir "internal\gui\assets\app.ico"

if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    throw "未找到 Go SDK。请先安装 Go 1.22 或更高版本：https://go.dev/dl/"
}

New-Item -ItemType Directory -Force -Path $distDir | Out-Null
Push-Location $projectDir
try {
    go test -buildvcs=false ./...
    if ($LASTEXITCODE -ne 0) { throw "测试失败" }
    go build -buildvcs=false -trimpath -ldflags "-s -w -H=windowsgui" -o $output .
    if ($LASTEXITCODE -ne 0) { throw "构建失败" }
    go run -buildvcs=false ./tools/iconembed -exe $output -ico $icon
    if ($LASTEXITCODE -ne 0) { throw "写入 EXE 图标失败" }
    Write-Host "构建完成: $output"
}
finally {
    Pop-Location
}
