@echo off
REM Ventas Mensuales CA - Ejecucion automatica
echo [%date% %time%] CA Iniciando...
"C:\Users\Alkosto\Desktop\excel - automatico\go-ventas\ventas_mensuales.exe" --auto-ca --mode=IP
echo [%date% %time%] CA Completado.
