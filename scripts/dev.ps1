# InduGate 本地开发（Windows PowerShell）
# 用法: .\scripts\dev.ps1 backend | frontend

param(
    [Parameter(Mandatory = $true)]
    [ValidateSet("backend", "frontend")]
    [string]$Target
)

$Root = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
Set-Location $Root

function Find-Go {
    if (Get-Command go -ErrorAction SilentlyContinue) { return "go" }
    $default = "C:\Program Files\Go\bin\go.exe"
    if (Test-Path $default) { return $default }
    throw "未找到 Go，请安装 Go 1.24+ 并加入 PATH"
}

$env:GOPROXY = "https://goproxy.cn,direct"

switch ($Target) {
    "backend" {
        if (-not (Test-Path "data")) { New-Item -ItemType Directory -Path "data" | Out-Null }
        $go = Find-Go
        Write-Host "启动后端 http://localhost:8080 ..." -ForegroundColor Cyan
        & $go run ./cmd/gateway
    }
    "frontend" {
        $nodeVersion = (node -v) -replace '^v', ''
        $major = [int]($nodeVersion.Split('.')[0])
        if ($major -lt 18) {
            throw "Node.js 版本过低 ($nodeVersion)，Vite 6 需要 Node 18+。请升级 Node 或使用 Docker 启动。"
        }
        Set-Location web
        if (-not (Test-Path "node_modules")) {
            Write-Host "安装前端依赖..." -ForegroundColor Cyan
            npm install
        }
        Write-Host "启动前端 http://localhost:3000 ..." -ForegroundColor Cyan
        npm run dev
    }
}
