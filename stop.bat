@echo off
title Tat Du an Secret Letter
echo ==============================================
echo  DANG DUNG CAC DICH VU DANG CHAY
echo ==============================================
echo.

echo [+] Dang tim va giai phong port 8080 (Go Backend)...
for /f "tokens=5" %%a in ('netstat -aon ^| findstr :8080 ^| findstr LISTENING') do (
    taskkill /F /PID %%a >nul 2>&1
)

echo [+] Dang tim va giai phong port 5173 (Vite Frontend)...
for /f "tokens=5" %%a in ('netstat -aon ^| findstr :5173 ^| findstr LISTENING') do (
    taskkill /F /PID %%a >nul 2>&1
)

echo.
echo ==============================================
echo  DA TAT CAC DICH VU THANH CONG!
echo ==============================================
timeout /t 3 >nul
