@echo off
echo Starting server and agent...

:: Собираем сервер
echo Building server...
go build -o .\cmd\gophermart\server.exe .\cmd\gophermart\

:: Собираем агента
@REM echo Building agent...
@REM go build -o .\cmd\agent\agent.exe .\cmd\agent\


set "DATABASE_URI=postgres://postgres:pass@localhost:5432/postgres?sslmode=disable"
set "JWT_SECRET=secret"
:: Запускаем сервер в отдельном окне
start "Server" cmd /k ".\cmd\gophermart\server.exe"
set SERVER_PID=%ERRORLEVEL%

:: Ждем 2 секунды, чтобы сервер успел запуститься
timeout /t 2

:: Запускаем агента в отдельном окне
@REM start "Agent" cmd /k ".\cmd\agent\agent.exe -k=test"
@REM set AGENT_PID=%ERRORLEVEL%

echo Server and Agent are running...
echo Server window shows server logs
echo Agent window shows agent logs
echo Press any key to stop all processes...

:: Ждем нажатия любой клавиши
pause > nul

:: Завершаем процессы
echo.
echo Stopping processes...
taskkill /F /FI "WINDOWTITLE eq Server" 2>nul
taskkill /F /FI "WINDOWTITLE eq Agent" 2>nul

echo All processes stopped.