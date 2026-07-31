@echo off
REM Ventas Mensuales CA - Ejecucion automatica
echo [%date% %time%] CA Iniciando...
"C:\Users\Administrador\Desktop\Reportes FA\go-ventas\ventas_mensuales.exe" --auto-ca --mode=IP
echo [%date% %time%] CA Completado.
pause
