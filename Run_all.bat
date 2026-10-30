@echo off
echo ===================================================
echo   MENJALANKAN HT MONITORING SYSTEM (ONE-CLICK)
echo ===================================================

:: 1. Memastikan MySQL Service Berjalan
net start MySQL80 >nul 2>&1

:: 2. Jalankan Python STT Service di Jendela Baru
start "Python STT Service (Port 8000)" cmd /k "cd /d "%~dp0python-sst" && "%~dp0ht-monitoring-backend\venv\Scripts\activate.bat" && python -m uvicorn main:app --reload --port 8000"

:: 3. Jalankan Go Backend API di Jendela Baru
start "Go Backend API (Port 8080)" cmd /k "cd /d "%~dp0go-backend" && set "GOTMPDIR=%~dp0go-backend" && go run main.go"

echo.
echo Semua service berhasil dipanggil!
echo - Python STT : http://localhost:8000
echo - Go API     : http://localhost:8080/api/incidents
echo ===================================================
pause