# Start both Python AI and Go Scraper Gateway
Write-Host "==================================================" -ForegroundColor Cyan
Write-Host "  Starting AI Scraper System (Python + Go)" -ForegroundColor Cyan
Write-Host "==================================================" -ForegroundColor Cyan

$WorkspaceRoot = Split-Path -Parent $PSScriptRoot
$PythonExe = Join-Path $WorkspaceRoot "python_ai\.venv\Scripts\python.exe"

if (-not (Test-Path $PythonExe)) {
    $PythonExe = "python"
}

# 1. Start Python AI Microservice
Write-Host "[1/2] Launching Python AI microservice on port 5000..." -ForegroundColor Green
$pyProcess = Start-Process -FilePath $PythonExe -ArgumentList "-m uvicorn python_ai.app.main:app --host 127.0.0.1 --port 5000" -WorkingDirectory $WorkspaceRoot -PassThru

# Give Python a few seconds to initialize
Start-Sleep -Seconds 3

# 2. Check Go
$goExe = (Get-Command go -ErrorAction SilentlyContinue).Source
if (-not $goExe) {
    if (Test-Path "c:\dev\go\bin\go.exe") {
        $goExe = "c:\dev\go\bin\go.exe"
    } elseif (Test-Path "C:\Program Files\Go\bin\go.exe") {
        $goExe = "C:\Program Files\Go\bin\go.exe"
    }
}

if (-not $goExe) {
    Write-Warning "Go executable not found in PATH or standard directory."
    Write-Warning "Python AI service is running with PID: $($pyProcess.Id)"
    exit 1
}

# 3. Start Go Gateway Server
Write-Host "[2/2] Launching Go Gateway Server on port 8080..." -ForegroundColor Green
try {
    & $goExe run ./cmd/scraper-server
}
finally {
    Write-Host "Stopping Python service..." -ForegroundColor Yellow
    Stop-Process -Id $pyProcess.Id -Force -ErrorAction SilentlyContinue
}
