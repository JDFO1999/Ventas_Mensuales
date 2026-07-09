@echo off
REM Ventas Mensuales - Ejecucion automatica
for /f "tokens=2 delims==" %%I in ('wmic os get localdatetime /value') do set datetime=%%I
set MES=%datetime:~4,2%
set ANIO=%datetime:~0,4%
if "%MES:~0,1%"=="0" set MES=%MES:~1,1%

echo [%date% %time%] Iniciando...
"C:\Users\Alkosto\Desktop\excel - automatico\go-ventas\ventas_mensuales.exe" --auto --month=%MES% --year=%ANIO% --mode=IP
echo [%date% %time%] Completado.
