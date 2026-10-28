package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

func menuEstapago() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("\n  *** PANIC ESTA_PAGO: %v ***\n", r)
		}
		fmt.Print("\nPresione Enter para salir...")
		leerLinea()
	}()

	db, sucursales, anio, modo := conectarYSucursales("ESTA_PAGO - CARGA DE FORMAS DE PAGO A SQL")
	if db == nil {
		return
	}
	defer db.Close()

	sucursales = seleccionarTiendasCA(sucursales)

	fmt.Println("\n" + strings.Repeat("-", 40))
	fmt.Printf("  Procesando ESTA_PAGO %d, %d tiendas (modo %s)...\n", anio, len(sucursales), modo)

	if err := CrearBaseForm_PG_Tiendas(db); err != nil {
		fmt.Printf("  ERROR creando BD Form_PG_Tiendas: %v\n", err)
		return
	}

	dbPago, errC := ConectarSQL_ESTAPAGO()
	if errC != nil {
		fmt.Printf("  ERROR conexion Form_PG_Tiendas: %v\n", errC)
		return
	}
	defer dbPago.Close()

	if err := CrearTablaEstaPago(dbPago); err != nil {
		fmt.Printf("  ERROR creando tabla ESTA_PAGO: %v\n", err)
		return
	}

	ProcesarEstapago(db, dbPago, sucursales, anio, modo, "MANUAL")
}

func ejecutarAutomaticoEstapagoManual() {
	anio := leerEntero("Ano a procesar (ej. 2026): ")
	if anio < 2000 || anio > 2100 {
		fmt.Println("  ERROR: Ano fuera de rango.")
		fmt.Print("\nPresione Enter para salir...")
		leerLinea()
		return
	}

	cfg, err := CargarConfig("config_estapago.json")
	if err != nil {
		cfg = Config{Modo: "IP", OutputDir: outputDir}
	}
	fmt.Printf("\n  Modo lectura: %s\n", cfg.Modo)

	procesarAutomaticoEstapago(anio, cfg, false)
}

func estadoTareaEstapago() string {
	psCmd := `$t = Get-ScheduledTask -TaskName 'VentasMensualesESTA_PAGO' -ErrorAction SilentlyContinue; ` +
		`if ($t) { ` +
		`  $i = Get-ScheduledTaskInfo -TaskName 'VentasMensualesESTA_PAGO' -ErrorAction SilentlyContinue; ` +
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

func MenuConfigEstapago() {
	cfg := Config{
		Modo:             "IP",
		OutputDir:        `C:\Users\Administrador\Desktop\Reportes FA`,
		HoraInicio:       6,
		HoraFin:          6,
		IntervaloMinutos: 1440,
	}

	if existing, err := CargarConfig("config_estapago.json"); err == nil {
		cfg = existing
	}

	for {
		fmt.Println()
		fmt.Println(strings.Repeat("=", 60))
		fmt.Println("  CONFIGURACION AUTOMATIZACION ESTA_PAGO")
		fmt.Println(strings.Repeat("=", 60))
		fmt.Println()
		fmt.Printf("  [1] Hora de inicio:    %02d:00\n", cfg.HoraInicio)
		fmt.Printf("  [2] Hora de fin:       %02d:00\n", cfg.HoraFin)
		fmt.Printf("  [3] Intervalo:         %d minutos (cada %.0f horas)\n",
			cfg.IntervaloMinutos, float64(cfg.IntervaloMinutos)/60)
		fmt.Printf("  [4] Modo de lectura:   %s\n", cfg.Modo)
		fmt.Println()
		fmt.Println("  [5] GUARDAR y crear tarea programada ESTA_PAGO")
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
			if err := GuardarConfig(cfg, "config_estapago.json"); err != nil {
				fmt.Printf("  ERROR: %v\n", err)
				continue
			}
			fmt.Println("  Configuracion guardada en config_estapago.json")
			crearTareaProgramadaEstapago(cfg)
			fmt.Println()
			fmt.Print("  Presione Enter para salir...")
			leerLinea()
			return
		case 0:
			return
		}
	}
}

func crearTareaProgramadaEstapago(cfg Config) {
	exeDir, _ := os.Getwd()
	cmdPath := exeDir + "\\run_ventas_estapago.cmd"

	cmd := fmt.Sprintf(`@echo off
REM Ventas Mensuales ESTA_PAGO - Ejecucion automatica
for /f "tokens=2 delims==" %%%%I in ('wmic os get localdatetime /value') do set datetime=%%%%I
set ANIO=%%datetime:~0,4%%
echo [%%date%% %%time%%] ESTA_PAGO Iniciando...
"%s\ventas_mensuales.exe" --auto-estapago --year=%%ANIO%% --mode=%s
	echo [%%date%% %%time%%] ESTA_PAGO Completado.
	pause
`, exeDir, cfg.Modo)
	os.WriteFile(cmdPath, []byte(cmd), 0644)

	psPath := exeDir + "\\setup_scheduler_estapago.ps1"
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
		psMessage = fmt.Sprintf(`Write-Host "Tarea ESTA_PAGO creada: $taskName (diaria %02d:00)"`, cfg.HoraInicio)
	} else {
		psMessage = fmt.Sprintf(`Write-Host "Tarea ESTA_PAGO creada: $taskName (%02d:00 a %02d:00, cada %d min)"`,
			cfg.HoraInicio, cfg.HoraFin, cfg.IntervaloMinutos)
	}

	ps := fmt.Sprintf(`
$taskName = "VentasMensualesESTA_PAGO"
$action = New-ScheduledTaskAction -Execute "%s\ventas_mensuales.exe" -Argument '--auto-estapago --mode=%s'
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

	fmt.Println("  Intentando registrar tarea ESTA_PAGO...")
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

var _ = time.Now