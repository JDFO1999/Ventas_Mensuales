package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

func menuCxP() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("\n  *** PANIC CxP: %v ***\n", r)
		}
		fmt.Print("\nPresione Enter para salir...")
		leerLinea()
	}()

	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("  CXP - CUENTAS POR PAGAR")
	fmt.Println(strings.Repeat("=", 50))

	cfg, err := CargarConfig("config_cxp.json")
	modo := "IP"
	if err == nil {
		modo = cfg.Modo
	}

	anio := time.Now().Year()
	fmt.Printf("  Ano actual: %d\n", anio)
	fmt.Print("  Ano a procesar (Enter = actual): ")
	linea := leerLinea()
	if linea != "" {
		if n, err := fmt.Sscanf(linea, "%d", &anio); err == nil && n == 1 && anio >= 2000 && anio <= 2100 {
			// OK
		} else {
			fmt.Println("  Ano invalido, usando ano actual.")
			anio = time.Now().Year()
		}
	}

	fmt.Printf("  Modo: %s\n", modo)
	fmt.Println()

	db, err := ConectarSQL()
	if err != nil {
		fmt.Printf("  ERROR conexion Sistemas: %v\n", err)
		return
	}
	defer db.Close()

	CrearTablaH_Tiendas(db)

	if err := CrearBasePDOC(db); err != nil {
		fmt.Printf("  ERROR creando BD PDOC_Tiendas: %v\n", err)
		return
	}
	if err := CrearBasePTRA(db); err != nil {
		fmt.Printf("  ERROR creando BD PTRA_Tiendas: %v\n", err)
		return
	}
	if err := CrearBaseMPRO(db); err != nil {
		fmt.Printf("  ERROR creando BD MPRO_Tiendas: %v\n", err)
		return
	}

	ProcesarCxP(db, anio, modo, "MANUAL")
}

func ejecutarAutomaticoCxPManual() {
	anio := time.Now().Year()

	cfg, err := CargarConfig("config_cxp.json")
	modo := "IP"
	if err == nil {
		modo = cfg.Modo
	}

	fmt.Printf("\n  CxP automatico: ano %d, modo %s\n", anio, modo)
	procesarAutomaticoCxP(anio, cfg, false)
}

func procesarAutomaticoCxP(year int, cfg Config, headless bool) {
	defer func() {
		if r := recover(); r != nil {
			logError("CxP PANIC: %v", r)
			if !headless {
				fmt.Printf("\n  *** PANIC CxP: %v ***\n", r)
			}
		}
	}()

	modo := cfg.Modo
	if modo == "" {
		modo = "IP"
	}

	tStart := time.Now()
	logHistorial(fmt.Sprintf("CxP: Iniciado %d", year))

	db, err := ConectarSQL()
	if err != nil {
		logError("CxP: ERROR conexion Sistemas: %v", err)
		if !headless {
			fmt.Printf("ERROR conexion Sistemas: %v\n", err)
		}
		return
	}
	defer db.Close()

	CrearTablaH_Tiendas(db)

	if err := CrearBasePDOC(db); err != nil {
		logError("CxP: ERROR creando BD PDOC_Tiendas: %v", err)
		return
	}
	if err := CrearBasePTRA(db); err != nil {
		logError("CxP: ERROR creando BD PTRA_Tiendas: %v", err)
		return
	}
	if err := CrearBaseMPRO(db); err != nil {
		logError("CxP: ERROR creando BD MPRO_Tiendas: %v", err)
		return
	}

	ProcesarCxP(db, year, modo, "AUTOMATICO")

	fmt.Printf("[%s] CxP Completado.\n", time.Now().Format("15:04:05"))
	logHistorial(fmt.Sprintf("CxP: Completado en %.1f min", time.Since(tStart).Minutes()))
}

func estadoTareaCxP() string {
	psCmd := `$t = Get-ScheduledTask -TaskName 'VentasMensualesCxP' -ErrorAction SilentlyContinue; ` +
		`if ($t) { ` +
		`  $i = Get-ScheduledTaskInfo -TaskName 'VentasMensualesCxP' -ErrorAction SilentlyContinue; ` +
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

func MenuConfigCxP() {
	cfg := Config{
		Modo:             "IP",
		OutputDir:        `C:\Users\Administrador\Desktop\Reportes FA`,
		HoraInicio:       6,
		HoraFin:          6,
		IntervaloMinutos: 1440,
	}

	if existing, err := CargarConfig("config_cxp.json"); err == nil {
		cfg = existing
	}

	for {
		fmt.Println()
		fmt.Println(strings.Repeat("=", 60))
		fmt.Println("  CONFIGURACION AUTOMATIZACION CxP")
		fmt.Println(strings.Repeat("=", 60))
		fmt.Println()
		fmt.Printf("  [1] Hora de inicio:    %02d:00\n", cfg.HoraInicio)
		fmt.Printf("  [2] Hora de fin:       %02d:00\n", cfg.HoraFin)
		fmt.Printf("  [3] Intervalo:         %d minutos (cada %.0f horas)\n",
			cfg.IntervaloMinutos, float64(cfg.IntervaloMinutos)/60)
		fmt.Printf("  [4] Modo de lectura:   %s\n", cfg.Modo)
		fmt.Println()
		fmt.Println("  [5] GUARDAR y crear tarea programada CxP")
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
			fmt.Println("  Veces que se repite al dia:")
			fmt.Println("    [1] Cada 30 minutos")
			fmt.Println("    [2] Cada 1 hora")
			fmt.Println("    [3] Cada 2 horas")
			fmt.Println("    [4] Cada 4 horas")
			fmt.Println("    [5] Una sola vez al dia")
			i := leerEntero("  > ")
			switch i {
			case 1: cfg.IntervaloMinutos = 30
			case 2: cfg.IntervaloMinutos = 60
			case 3: cfg.IntervaloMinutos = 120
			case 4: cfg.IntervaloMinutos = 240
			case 5:
				cfg.IntervaloMinutos = 1440
				cfg.HoraFin = cfg.HoraInicio
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
			if err := GuardarConfig(cfg, "config_cxp.json"); err != nil {
				fmt.Printf("  ERROR: %v\n", err)
				continue
			}
			fmt.Println("  Configuracion guardada en config_cxp.json")
			crearTareaProgramadaCxP(cfg)
			fmt.Println()
			fmt.Print("  Presione Enter para salir...")
			leerLinea()
			return
		case 0:
			return
		}
	}
}

func crearTareaProgramadaCxP(cfg Config) {
	exeDir, _ := os.Getwd()
	cmdPath := exeDir + "\\run_ventas_cxp.cmd"

	cmd := fmt.Sprintf(`@echo off
REM Ventas Mensuales CxP - Ejecucion automatica
for /f "tokens=2 delims==" %%%%I in ('wmic os get localdatetime /value') do set datetime=%%%%I
set ANIO=%%datetime:~0,4%%
echo [%%date%% %%time%%] CxP Iniciando...
"%s\ventas_mensuales.exe" --auto-cxp --year=%%ANIO%% --mode=%s
	echo [%%date%% %%time%%] CxP Completado.
	pause
`, exeDir, cfg.Modo)
	os.WriteFile(cmdPath, []byte(cmd), 0644)

	psPath := exeDir + "\\setup_scheduler_cxp.ps1"
	duration := cfg.HoraFin - cfg.HoraInicio

	var psTrigger string
	if duration <= 0 {
		psTrigger = fmt.Sprintf(`$trigger = New-ScheduledTaskTrigger -Daily -At "%02d:00"`, cfg.HoraInicio)
	} else {
		psTrigger = fmt.Sprintf(
			`$trigger = New-ScheduledTaskTrigger -Once -At "%02d:00" -RepetitionInterval (New-TimeSpan -Minutes %d) -RepetitionDuration (New-TimeSpan -Hours %d)`,
			cfg.HoraInicio, cfg.IntervaloMinutos, duration)
	}

	var psMessage string
	if duration <= 0 {
		psMessage = fmt.Sprintf(`Write-Host "Tarea CxP creada: $taskName (diaria %02d:00)"`, cfg.HoraInicio)
	} else {
		psMessage = fmt.Sprintf(`Write-Host "Tarea CxP creada: $taskName (%02d:00 a %02d:00, cada %d min)"`,
			cfg.HoraInicio, cfg.HoraFin, cfg.IntervaloMinutos)
	}

	ps := fmt.Sprintf(`
$taskName = "VentasMensualesCxP"
$action = New-ScheduledTaskAction -Execute "%s\ventas_mensuales.exe" -Argument '--auto-cxp --year=<YEAR> --mode=%s'
%s
$settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries -StartWhenAvailable
$loggedUser = (Get-CimInstance -ClassName Win32_ComputerSystem).UserName
if (-not $loggedUser) { $loggedUser = [System.Security.Principal.WindowsIdentity]::GetCurrent().Name }
$principal = New-ScheduledTaskPrincipal -UserId $loggedUser -LogonType Interactive -RunLevel Highest

try {
    Unregister-ScheduledTask -TaskName $taskName -Confirm:$false -ErrorAction SilentlyContinue
    Register-ScheduledTask -TaskName $taskName -Principal $principal -Trigger $trigger -Action $action -Settings $settings
    %s
} catch {
    Write-Host "ERROR: $_"
}
`, exeDir, cfg.Modo, psTrigger, psMessage)
	os.WriteFile(psPath, []byte(ps), 0644)

	fmt.Println("\n  Archivos creados:")
	fmt.Printf("    %s\n", cmdPath)
	fmt.Printf("    %s\n", psPath)
	fmt.Println()

	fmt.Println("  Intentando registrar tarea CxP...")
	regCmd := exec.Command("powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", psPath)
	if out, err := regCmd.CombinedOutput(); err != nil {
		fmt.Printf("  NO SE PUDO REGISTRAR (requiere admin): %s\n", strings.TrimSpace(string(out)))
		fmt.Println()
		fmt.Println("  EJECUTE COMO ADMINISTRADOR MANUALMENTE:")
		fmt.Printf("    powershell -File \"%s\"\n", psPath)
	} else {
		fmt.Printf("  %s\n", strings.TrimSpace(string(out)))
	}
}
