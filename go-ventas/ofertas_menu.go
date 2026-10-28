package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func menuOfertas() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("\n  *** PANIC OFERTAS: %v ***\n", r)
		}
		fmt.Print("\nPresione Enter para salir...")
		leerLinea()
	}()

	db, sucursales, _, modo := conectarYSucursales("OFERTAS - CARGA DE OFERTAS DE INVENTARIO (ofer001) A SQL")
	if db == nil {
		return
	}
	defer db.Close()

	sucursales = seleccionarTiendasCA(sucursales)

	fmt.Println("\n" + strings.Repeat("-", 40))
	fmt.Printf("  Procesando OFERTAS, %d tiendas (modo %s)...\n", len(sucursales), modo)

	if err := CrearBaseOFERTAS_Tiendas(db); err != nil {
		fmt.Printf("  ERROR creando BD OFERTAS_Tiendas: %v\n", err)
		return
	}

	dbOfer, errC := ConectarSQL_OFERTAS()
	if errC != nil {
		fmt.Printf("  ERROR conexion OFERTAS_Tiendas: %v\n", errC)
		return
	}
	defer dbOfer.Close()

	ProcesarOfertas(db, dbOfer, sucursales, modo, "MANUAL")
}

func ejecutarAutomaticoOfertasManual() {
	cfg, err := CargarConfig("config_ofertas.json")
	if err != nil {
		cfg = Config{
			Modo:       "IP",
			OutputDir:  outputDir,
			HorasFijas: []string{"08:00", "14:00", "20:00"},
		}
	}
	fmt.Printf("\n  Modo lectura: %s\n", cfg.Modo)
	if len(cfg.HorasFijas) > 0 {
		fmt.Printf("  Horarios programados: %s\n", strings.Join(cfg.HorasFijas, ", "))
	}

	procesarAutomaticoOfertas(cfg, false)
}

func estadoTareaOfertas() string {
	psCmd := `$t = Get-ScheduledTask -TaskName 'VentasMensualesOFERTAS' -ErrorAction SilentlyContinue; ` +
		`if ($t) { ` +
		`  $i = Get-ScheduledTaskInfo -TaskName 'VentasMensualesOFERTAS' -ErrorAction SilentlyContinue; ` +
		`  $next = if ($i -and $i.NextRunTime) { $i.NextRunTime.ToString('dd/MM/yyyy HH:mm') } else { '???' }; ` +
		`  Write-Output ('OK|' + $t.State + '|' + $next) ` +
		`} else { ` +
		`  Write-Output 'NO CONFIGURADA' ` +
		`}`

	out, err := exec.Command("powershell", "-NoProfile", "-Command", psCmd).Output()
	if err != nil {
		return "ERROR AL CONSULTAR"
	}
	result := strings.TrimSpace(string(out))
	if result == "NO CONFIGURADA" {
		return result
	}
	parts := strings.SplitN(result, "|", 3)
	if len(parts) >= 3 {
		return fmt.Sprintf("ACTIVA (%s)  |  Proximo: %s", parts[1], parts[2])
	}
	return result
}

func MenuConfigOfertas() {
	cfg := Config{
		Modo:             "IP",
		OutputDir:        outputDir,
		HoraInicio:       0,
		HoraFin:          23,
		IntervaloMinutos: 5,
		HorasFijas:       nil,
	}

	if existing, err := CargarConfig("config_ofertas.json"); err == nil {
		cfg = existing
	}

	for {
		fmt.Println()
		fmt.Println(strings.Repeat("=", 60))
		fmt.Println("  CONFIGURACION AUTOMATIZACION OFERTAS (ofer001)")
		fmt.Println(strings.Repeat("=", 60))
		fmt.Println()

		if len(cfg.HorasFijas) > 0 {
			fmt.Printf("  Frecuencia actual:     3 veces al dia (Horas fijas: %s)\n", strings.Join(cfg.HorasFijas, ", "))
		} else if cfg.IntervaloMinutos < 60 {
			fmt.Printf("  Frecuencia actual:     Cada %d minutos (Continuo 24/7)\n", cfg.IntervaloMinutos)
		} else {
			fmt.Printf("  Frecuencia actual:     Cada %d min (%02d:00 a %02d:00)\n",
				cfg.IntervaloMinutos, cfg.HoraInicio, cfg.HoraFin)
		}
		fmt.Printf("  Modo de lectura:       %s\n", cfg.Modo)
		fmt.Println()
		fmt.Println("  [1] Ejecutar cada 5 minutos (Recomendado)")
		fmt.Println("  [2] Ejecutar cada 10 minutos")
		fmt.Println("  [3] Ejecutar cada 15 minutos")
		fmt.Println("  [4] Ejecutar cada 30 minutos")
		fmt.Println("  [5] Ejecutar 3 veces al dia (Horas fijas)")
		fmt.Println("  [6] Intervalo personalizado (Hora inicio, fin, minutos)")
		fmt.Println("  [7] Modo de lectura: [IP] o [S]")
		fmt.Println()
		fmt.Println("  [8] GUARDAR y registrar tarea programada OFERTAS")
		fmt.Println("  [0] Salir sin guardar")
		fmt.Println()

		op := leerEntero("  Seleccione: ")
		switch op {
		case 1:
			cfg.IntervaloMinutos = 5
			cfg.HoraInicio = 0
			cfg.HoraFin = 23
			cfg.HorasFijas = nil
			fmt.Println("  -> Seleccionado: Cada 5 minutos")
		case 2:
			cfg.IntervaloMinutos = 10
			cfg.HoraInicio = 0
			cfg.HoraFin = 23
			cfg.HorasFijas = nil
			fmt.Println("  -> Seleccionado: Cada 10 minutos")
		case 3:
			cfg.IntervaloMinutos = 15
			cfg.HoraInicio = 0
			cfg.HoraFin = 23
			cfg.HorasFijas = nil
			fmt.Println("  -> Seleccionado: Cada 15 minutos")
		case 4:
			cfg.IntervaloMinutos = 30
			cfg.HoraInicio = 0
			cfg.HoraFin = 23
			cfg.HorasFijas = nil
			fmt.Println("  -> Seleccionado: Cada 30 minutos")
		case 5:
			fmt.Println("\n  Configurando 3 horas fijas al dia (formato HH:mm, ej: 08:00):")
			h1Default := "08:00"
			h2Default := "14:00"
			h3Default := "20:00"
			if len(cfg.HorasFijas) >= 3 {
				h1Default = cfg.HorasFijas[0]
				h2Default = cfg.HorasFijas[1]
				h3Default = cfg.HorasFijas[2]
			}
			fmt.Printf("  Hora 1 (Enter para '%s'): ", h1Default)
			h1 := leerLinea()
			if h1 == "" {
				h1 = h1Default
			}
			fmt.Printf("  Hora 2 (Enter para '%s'): ", h2Default)
			h2 := leerLinea()
			if h2 == "" {
				h2 = h2Default
			}
			fmt.Printf("  Hora 3 (Enter para '%s'): ", h3Default)
			h3 := leerLinea()
			if h3 == "" {
				h3 = h3Default
			}
			cfg.HorasFijas = []string{h1, h2, h3}
			fmt.Printf("  -> Configurado: %s, %s, %s\n", h1, h2, h3)
		case 6:
			fmt.Println("\n  Horario por intervalo:")
			fmt.Print("  Hora de inicio (0-23): ")
			hIni := leerEntero("  > ")
			fmt.Print("  Hora de fin (0-23): ")
			hFin := leerEntero("  > ")
			fmt.Print("  Intervalo en minutos (ej: 5, 10, 60): ")
			min := leerEntero("  > ")
			if min > 0 && hIni >= 0 && hIni <= 23 && hFin >= 0 && hFin <= 23 {
				cfg.HoraInicio = hIni
				cfg.HoraFin = hFin
				cfg.IntervaloMinutos = min
				cfg.HorasFijas = nil
				fmt.Printf("  -> Configurado: %02d:00 a %02d:00 cada %d min\n", hIni, hFin, min)
			} else {
				fmt.Println("  ERROR: Valores invalidos.")
			}
		case 7:
			fmt.Println("  Modo: [1] Servidor S:  [2] Tiendas IP")
			m := leerEntero("  > ")
			if m == 1 {
				cfg.Modo = "S"
			} else {
				cfg.Modo = "IP"
			}
		case 8:
			if err := GuardarConfig(cfg, "config_ofertas.json"); err != nil {
				fmt.Printf("  ERROR: %v\n", err)
				continue
			}
			fmt.Println("  Configuracion guardada en config_ofertas.json")
			crearTareaProgramadaOfertas(cfg)
			fmt.Println()
			fmt.Print("  Presione Enter para salir...")
			leerLinea()
			return
		case 0:
			return
		}
	}
}

func obtenerDirectorioEjecutable() string {
	exePath, err := os.Executable()
	if err != nil {
		dir, _ := os.Getwd()
		return dir
	}
	resolved, err := filepath.EvalSymlinks(exePath)
	if err == nil {
		return filepath.Dir(resolved)
	}
	return filepath.Dir(exePath)
}

func crearTareaProgramadaOfertas(cfg Config) {
	exeDir := obtenerDirectorioEjecutable()
	cmdPath := filepath.Join(exeDir, "run_ventas_ofertas.cmd")
	exePath := filepath.Join(exeDir, "ventas_mensuales.exe")

	cmd := fmt.Sprintf(`@echo off
cd /d "%%~dp0"
echo [%%date%% %%time%%] OFERTAS Iniciando...
"%%~dp0ventas_mensuales.exe" --auto-ofertas --mode=%s
echo [%%date%% %%time%%] OFERTAS Completado.
`, cfg.Modo)
	os.WriteFile(cmdPath, []byte(cmd), 0644)

	psPath := filepath.Join(exeDir, "setup_scheduler_ofertas.ps1")

	var psTriggerDef string
	var psMessage string

	if len(cfg.HorasFijas) > 0 {
		var triggerLines []string
		for _, h := range cfg.HorasFijas {
			triggerLines = append(triggerLines, fmt.Sprintf(`(New-ScheduledTaskTrigger -Daily -At "%s")`, h))
		}
		psTriggerDef = fmt.Sprintf(`$triggers = @(%s)`, strings.Join(triggerLines, ", "))
		psMessage = fmt.Sprintf(`Write-Host "Tarea OFERTAS creada: $taskName (Horas fijas: %s)"`, strings.Join(cfg.HorasFijas, ", "))
	} else if cfg.IntervaloMinutos < 60 || (cfg.HoraInicio == 0 && cfg.HoraFin >= 23) {
		// Intervalo continuo (ej. cada 5 o 10 minutos 24/7)
		psTriggerDef = fmt.Sprintf(`$triggers = @(New-ScheduledTaskTrigger -Once -At '00:00' -RepetitionInterval (New-TimeSpan -Minutes %d) -RepetitionDuration (New-TimeSpan -Days 3650))`, cfg.IntervaloMinutos)
		psMessage = fmt.Sprintf(`Write-Host "Tarea OFERTAS creada: $taskName (cada %d minutos, 24/7)"`, cfg.IntervaloMinutos)
	} else {
		duration := cfg.HoraFin - cfg.HoraInicio
		if duration <= 0 {
			psTriggerDef = fmt.Sprintf(`$triggers = @(New-ScheduledTaskTrigger -Daily -At "%02d:00")`, cfg.HoraInicio)
			psMessage = fmt.Sprintf(`Write-Host "Tarea OFERTAS creada: $taskName (diaria %02d:00)"`, cfg.HoraInicio)
		} else {
			psTriggerDef = fmt.Sprintf(
				`$triggers = @(New-ScheduledTaskTrigger -Once -At "%02d:00" -RepetitionInterval (New-TimeSpan -Minutes %d) -RepetitionDuration (New-TimeSpan -Hours %d))`,
				cfg.HoraInicio, cfg.IntervaloMinutos, duration)
			psMessage = fmt.Sprintf(`Write-Host "Tarea OFERTAS creada: $taskName (%02d:00 a %02d:00, cada %d min)"`,
				cfg.HoraInicio, cfg.HoraFin, cfg.IntervaloMinutos)
		}
	}

	ps := fmt.Sprintf(`
$taskName = "VentasMensualesOFERTAS"
$exePath = "%s"
$workDir = "%s"
$action = New-ScheduledTaskAction -Execute $exePath -Argument '--auto-ofertas --mode=%s' -WorkingDirectory $workDir
%s
$settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries -StartWhenAvailable -MultipleInstances IgnoreNew
$loggedUser = [System.Security.Principal.WindowsIdentity]::GetCurrent().Name
if (-not $loggedUser) { $loggedUser = (Get-CimInstance -ClassName Win32_ComputerSystem).UserName }
$principal = New-ScheduledTaskPrincipal -UserId $loggedUser -LogonType Interactive -RunLevel Highest

try {
    Unregister-ScheduledTask -TaskName $taskName -Confirm:$false -ErrorAction SilentlyContinue
    Register-ScheduledTask -TaskName $taskName -Principal $principal -Trigger $triggers -Action $action -Settings $settings
    %s
} catch {
    Write-Host "ERROR: $_"
}
`, exePath, exeDir, cfg.Modo, psTriggerDef, psMessage)
	os.WriteFile(psPath, []byte(ps), 0644)

	fmt.Println("\n  Archivos generados:")
	fmt.Printf("    CMD: %s\n", cmdPath)
	fmt.Printf("    PS1: %s\n", psPath)
	fmt.Printf("    Directorio de trabajo (WorkingDirectory): %s\n", exeDir)
	fmt.Println()

	fmt.Println("  Configurando y registrando tarea en Windows Task Scheduler...")
	regCmd := exec.Command("powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", psPath)
	out, err := regCmd.CombinedOutput()
	outStr := strings.TrimSpace(string(out))
	if err != nil || strings.Contains(outStr, "Access is denied") || strings.Contains(outStr, "Acceso denegado") || strings.Contains(outStr, "ERROR:") {
		fmt.Printf("  AVISO: %s\n", outStr)
		fmt.Println("  Intentando registrar elevando permisos de Administrador...")
		elevateCmd := exec.Command("powershell", "-NoProfile", "-Command", fmt.Sprintf(`Start-Process powershell -Verb RunAs -ArgumentList '-NoProfile -ExecutionPolicy Bypass -File "%s"'`, psPath))
		if errElev := elevateCmd.Run(); errElev == nil {
			fmt.Println("  -> Se solicito autorizacion de Administrador de Windows (UAC).")
		} else {
			fmt.Println("  EJECUTE COMO ADMINISTRADOR MANUALMENTE:")
			fmt.Printf("    powershell -File \"%s\"\n", psPath)
		}
	} else {
		fmt.Printf("  %s\n", outStr)
	}
}
