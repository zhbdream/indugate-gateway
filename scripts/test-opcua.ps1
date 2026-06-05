# OPC UA 自动化测试脚本（PowerShell）
# 用法: powershell -File scripts/test-opcua.ps1

param(
    [string]$BaseUrl = "http://localhost:8080"
)

$ErrorActionPreference = "Stop"
$Base = "$BaseUrl/api/v1"
$passed = 0
$failed = 0

function Test-Step($name, $script) {
    Write-Host "`n[$name]" -ForegroundColor Cyan
    try {
        $result = & $script
        Write-Host "  PASS" -ForegroundColor Green
        $script:passed++
        return $result
    } catch {
        Write-Host "  FAIL: $($_.Exception.Message)" -ForegroundColor Red
        $script:failed++
        return $null
    }
}

Test-Step "Health" { (Invoke-RestMethod "$BaseUrl/health" -TimeoutSec 5).data | Out-Null }

$sim = Test-Step "Start Simulator" {
    (Invoke-RestMethod "$Base/simulators/opcua/start" -Method POST -TimeoutSec 15).data
}
if (-not $sim) { exit 1 }

$tempNode = ($sim.nodes | Where-Object { $_ -match 'Temperature' } | Select-Object -First 1)
$pressureNode = ($sim.nodes | Where-Object { $_ -match 'Pressure' } | Select-Object -First 1)
Write-Host "  Nodes: $($sim.nodes -join ', ')"

$dev = Test-Step "Create Device" {
    (Invoke-RestMethod "$Base/devices" -Method POST -ContentType "application/json" `
        -Body '{"name":"Local Sim","protocol":"opcua","address":"opc.tcp://127.0.0.1:4840"}' -TimeoutSec 10).data
}
$id = $dev.id

Test-Step "Connect" {
    $r = (Invoke-RestMethod "$Base/devices/$id/connect" -Method POST -TimeoutSec 15).data
    if ($r.status -ne "connected") { throw "status=$($r.status)" }
}

Test-Step "Browse Nodes" {
    (Invoke-RestMethod "$Base/devices/$id/nodes?node=ns=1;i=85&depth=1&children_only=true" -TimeoutSec 15).data | Out-Null
}

$enc = [uri]::EscapeDataString($tempNode)
Test-Step "Read" {
    (Invoke-RestMethod "$Base/devices/$id/data/$enc" -TimeoutSec 10).data | Out-Null
}

Test-Step "Write" {
    (Invoke-RestMethod "$Base/devices/$id/data/$enc" -Method POST -ContentType "application/json" `
        -Body '{"value":55.5}' -TimeoutSec 10).data | Out-Null
}

Test-Step "Read After Write" {
    $v = (Invoke-RestMethod "$Base/devices/$id/data/$enc" -TimeoutSec 10).data.value
    if ($v -ne 55.5) { throw "expected 55.5, got $v" }
}

$sub = Test-Step "Subscribe" {
    $body = (@{ node_ids = @($tempNode, $pressureNode); interval_ms = 500 } | ConvertTo-Json -Compress)
    (Invoke-RestMethod "$Base/devices/$id/subscribe" -Method POST -ContentType "application/json" `
        -Body $body -TimeoutSec 15).data
}

Start-Sleep -Seconds 4
Test-Step "Poll Events" {
    $events = (Invoke-RestMethod "$Base/devices/$id/subscriptions/$($sub.id)/events" -TimeoutSec 10).data
    if ($events.Count -lt 1) { throw "no events received" }
    Write-Host "  Events: $($events.Count)"
}

Test-Step "Disconnect" {
    (Invoke-RestMethod "$Base/devices/$id/disconnect" -Method POST -TimeoutSec 10).data | Out-Null
}

Test-Step "Stop Simulator" {
    (Invoke-RestMethod "$Base/simulators/opcua/stop" -Method POST -TimeoutSec 10).data | Out-Null
}

Write-Host "`n========================================" -ForegroundColor Yellow
Write-Host "Passed: $passed  Failed: $failed" -ForegroundColor Yellow
Write-Host "========================================" -ForegroundColor Yellow
if ($failed -gt 0) { exit 1 }
