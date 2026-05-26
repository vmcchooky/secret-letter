@echo off
title Khoi dong Du an Secret Letter
echo ==============================================
echo  DANG KHOI DONG HE THONG (COSMIC THEME)
echo ==============================================
echo.

echo [+] Dang khoi dong Go Backend API tren port 8080...
start "Go Backend API" cmd /k "go run ./backend/cmd/api"

echo [+] Dang khoi dong Vite Frontend Dev Server tren port 5173...
start "Vite Dev Server" cmd /k "cd frontend/web-app && npm run dev"

echo.
echo ==============================================
echo  DA KHOI DONG THANH CONG!
echo  - Frontend: http://localhost:5173
echo  - Backend: http://localhost:8080
echo.
echo  De tat du an, hay chay file: stop.bat
echo ==============================================
pause
