$WorkspaceRoot = Split-Path -Parent $PSScriptRoot
$PythonExe = Join-Path $WorkspaceRoot "python_ai\.venv\Scripts\python.exe"

if (-not (Test-Path $PythonExe)) {
    $PythonExe = "python"
}

Write-Host "Starting Python service on port 5000..."
$pyProcess = Start-Process -FilePath $PythonExe -ArgumentList "-m uvicorn python_ai.app.main:app --host 127.0.0.1 --port 5000" -WorkingDirectory $WorkspaceRoot -PassThru

Start-Sleep -Seconds 2

$goExe = (Get-Command go -ErrorAction SilentlyContinue).Source
if (-not $goExe) {
    if (Test-Path "c:\dev\go\bin\go.exe") {
        $goExe = "c:\dev\go\bin\go.exe"
    } elseif (Test-Path "C:\Program Files\Go\bin\go.exe") {
        $goExe = "C:\Program Files\Go\bin\go.exe"
    }
}

if (-not $goExe) {
    Write-Warning "Go executable not found."
    exit 1
}

Write-Host "Starting Go server on port 8080..."
try {
    & $goExe run ./cmd/scraper-server
}
finally {
    Stop-Process -Id $pyProcess.Id -Force -ErrorAction SilentlyContinue
}
