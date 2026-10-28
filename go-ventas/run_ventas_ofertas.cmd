@echo off
cd /d "%~dp0"
echo [%%date%% %%time%%] OFERTAS Iniciando...
"%~dp0ventas_mensuales.exe" --auto-ofertas --mode=IP
echo [%%date%% %%time%%] OFERTAS Completado.
