package main

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

func MenuConfig() {
	cfg := Config{
		Modo:             "S",
		OutputDir:        `C:\Users\Alkosto\Desktop\excel - automatico`,
		HoraInicio:       8,
		HoraFin:          20,
		IntervaloMinutos: 120,
	}

	if existing, err := CargarConfig("config.json"); err == nil {
		cfg = existing
	}

	for {
		fmt.Println()
		fmt.Println(strings.Repeat("=", 60))
		fmt.Println("  CONFIGURACION DE AUTOMATIZACION")
		fmt.Println(strings.Repeat("=", 60))
		fmt.Println()
		fmt.Printf("  [1] Hora de inicio:    %02d:00\n", cfg.HoraInicio)
		fmt.Printf("  [2] Hora de fin:       %02d:00\n", cfg.HoraFin)
		fmt.Printf("  [3] Intervalo:         %d minutos (cada %.0f horas)\n",
			cfg.IntervaloMinutos, float64(cfg.IntervaloMinutos)/60)
		fmt.Printf("  [4] Modo de lectura:   %s\n", cfg.Modo)
		fmt.Printf("  [5] Output:            %s\n", cfg.OutputDir)
		fmt.Println()
		fmt.Println("  [6] GUARDAR y crear tarea programada")
		fmt.Println("  [0] Salir sin guardar")
		fmt.Println()

		op := leerEntero("  Seleccione: ")
		switch op {
		case 1:
			fmt.Print("  Hora de inicio (0-23): ")
			h := leerEntero("  > ")
			if h >= 0 && h <= 23 {
				cfg.HoraInicio = h
			}
		case 2:
			fmt.Print("  Hora de fin (0-23): ")
			h := leerEntero("  > ")
			if h >= 0 && h <= 23 && h > cfg.HoraInicio {
				cfg.HoraFin = h
			} else {
				fmt.Println("  ERROR: Debe ser mayor que la hora de inicio.")
			}
		case 3:
			fmt.Println("  Intervalos:")
			fmt.Println("    [1] Cada 30 minutos")
			fmt.Println("    [2] Cada 1 hora")
			fmt.Println("    [3] Cada 2 horas")
			fmt.Println("    [4] Cada 4 horas")
			i := leerEntero("  > ")
			switch i {
			case 1: cfg.IntervaloMinutos = 30
			case 2: cfg.IntervaloMinutos = 60
			case 3: cfg.IntervaloMinutos = 120
			case 4: cfg.IntervaloMinutos = 240
			}
		case 4:
			fmt.Println("  Modo: [1] Servidor S:  [2] Tiendas IP")
			m := leerEntero("  > ")
			if m == 1 {
				cfg.Modo = "S"
			} else {
				cfg.Modo = "IP"
			}
		case 5:
			fmt.Printf("  Output actual: %s\n", cfg.OutputDir)
			fmt.Print("  Nueva ruta (Enter para mantener): ")
			nueva := leerLinea()
			if nueva != "" {
				cfg.OutputDir = nueva
			}
		case 6:
			if err := GuardarConfig(cfg, "config.json"); err != nil {
				fmt.Printf("  ERROR: %v\n", err)
				continue
			}
			fmt.Println("  Configuracion guardada en config.json")
			crearTareaProgramada(cfg)
			fmt.Println()
			fmt.Print("  Presione Enter para salir...")
			leerLinea()
			return
		case 0:
			fmt.Println("  Saliendo sin guardar.")
			return
		}
	}
}

func crearTareaProgramada(cfg Config) {
	exeDir, _ := os.Getwd()
	cmdPath := exeDir + "\\run_ventas.cmd"

	cmd := fmt.Sprintf(`@echo off
REM Ventas Mensuales - Ejecucion automatica
for /f "tokens=2 delims==" %%%%I in ('wmic os get localdatetime /value') do set datetime=%%%%I
set MES=%%datetime:~4,2%%
set ANIO=%%datetime:~0,4%%
if "%%MES:~0,1%%"=="0" set MES=%%MES:~1,1%%

echo [%%date%% %%time%%] Iniciando...
"%s\ventas_mensuales.exe" --auto --month=%%MES%% --year=%%ANIO%% --mode=%s
echo [%%date%% %%time%%] Completado.
`, exeDir, cfg.Modo)
	os.WriteFile(cmdPath, []byte(cmd), 0644)

	psPath := exeDir + "\\setup_scheduler.ps1"
	duration := cfg.HoraFin - cfg.HoraInicio
	ps := fmt.Sprintf(`
$taskName = "VentasMensuales"
$action = New-ScheduledTaskAction -Execute "cmd.exe" -Argument "/c %s"
$trigger = New-ScheduledTaskTrigger -Daily -At "%02d:00" -RepetitionInterval (New-TimeSpan -Minutes %d) -RepetitionDuration (New-TimeSpan -Hours %d)
$settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries -StartWhenAvailable

try {
    Unregister-ScheduledTask -TaskName $taskName -Confirm:$false -ErrorAction SilentlyContinue
    Register-ScheduledTask -TaskName $taskName -Trigger $trigger -Action $action -Settings $settings -RunLevel Highest
    Write-Host "Tarea creada: $taskName (%02d:00 a %02d:00, cada %d min)"
} catch {
    Write-Host "ERROR: $_"
}
`, cmdPath, cfg.HoraInicio, cfg.IntervaloMinutos, duration, cfg.HoraInicio, cfg.HoraFin, cfg.IntervaloMinutos)
	os.WriteFile(psPath, []byte(ps), 0644)

	fmt.Println("\n  Archivos creados:")
	fmt.Printf("    %s\n    %s\n", cmdPath, psPath)
	fmt.Println()
	fmt.Println("  EJECUTE COMO ADMINISTRADOR:")
	fmt.Printf("    powershell -File \"%s\"\n", psPath)
}

func procesarAutomatico(year, month int, cfg Config, conProgreso bool) {
	tStart := time.Now()

	db, err := ConectarSQL()
	if err != nil {
		fmt.Printf("ERROR SQL: %v\n", err)
		return
	}
	defer db.Close()

	fmt.Printf("[%s] Iniciando: %s %d\n", time.Now().Format("15:04:05"), MesesES[month], year)

	sucursales, err := ObtenerSucursales(db)
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
		return
	}

	CrearTablaPosVentas(db)

	fmt.Printf("  %d tiendas a procesar\n", len(sucursales))

	resultados := make([]ResultadoTienda, len(sucursales))
	var wg sync.WaitGroup
	type storeResult struct {
		index int
		res   ResultadoTienda
	}
	resultChan := make(chan storeResult, len(sucursales))

	for i, s := range sucursales {
		wg.Add(1)
		go func(idx int, suc Sucursal) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					resultChan <- storeResult{index: idx}
				}
			}()

			ch := ProcesarTienda(suc, db, year, month, cfg.Modo, true)
			select {
			case msg := <-ch:
				if conProgreso && msg.Resultado.Clientes > 0 {
					fmt.Printf("[%d/%d] %s - %d facts | Total: $%.2f\n",
						idx+1, len(sucursales), suc.Codigo,
						msg.Resultado.Clientes, msg.Resultado.Total)
				}
				resultChan <- storeResult{index: idx, res: msg.Resultado}
			case <-time.After(30 * time.Minute):
				if conProgreso {
					fmt.Printf("[%d/%d] %s - TIMEOUT\n", idx+1, len(sucursales), suc.Codigo)
				}
				resultChan <- storeResult{index: idx}
			}
		}(i, s)
	}

	wg.Wait()
	close(resultChan)
	for msg := range resultChan {
		resultados[msg.index] = msg.res
	}

	totalFact := 0
	for _, r := range resultados {
		totalFact += r.Clientes
	}
	if totalFact == 0 {
		fmt.Println("ADVERTENCIA: Sin datos.")
		return
	}

	fmt.Printf("[%s] Generando Excel...\n", time.Now().Format("15:04:05"))
	if err := GenerarExcel(resultados, year, month, cfg.OutputDir); err != nil {
		fmt.Printf("ERROR Excel: %v\n", err)
		return
	}

	fmt.Printf("[%s] Completado en %.1f min.\n",
		time.Now().Format("15:04:05"), time.Since(tStart).Minutes())
}
