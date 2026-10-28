@echo off
rem Mantem o poster rodando: se o programa cair, sobe de novo em 60 s.
rem O estado (estado.json) faz ele retomar de onde parou. Logs em daemon.log.
cd /d "%~dp0"
:loop
marketplace-poster.exe -daemon >> daemon.log 2>&1
timeout /t 60 /nobreak >nul
goto loop
