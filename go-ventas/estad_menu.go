package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

func menuEstad() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("\n  *** PANIC ESTAD: %v ***\n", r)
		}
		fmt.Print("\nPresione Enter para salir...")
		leerLinea()
	}()

	db, sucursales, anio, modo := conectarYSucursales("ESTAD - CARGA DE ESTADISTICAS DE VENTAS A SQL")
	if db == nil {
		return
	}
	defer db.Close()

	sucursales = seleccionarTiendasCA(sucursales)

	fmt.Println("\nRango de meses a procesar:")
	mesInicio := leerEntero("  Mes inicio (1-12): ")
	mesFin := leerEntero("  Mes fin (1-12): ")
	if mesInicio < 1 || mesInicio > 12 || mesFin < 1 || mesFin > 12 || mesFin < mesInicio {
		fmt.Println("  ERROR: Rango de meses invalido.")
		return
	}

	fmt.Println("\n" + strings.Repeat("-", 40))
	fmt.Printf("  Procesando ESTAD %d, meses %d-%d, %d tiendas (modo %s)...\n",
		anio, mesInicio, mesFin, len(sucursales), modo)

	CrearBaseESTAD_Tiendas(db)

	dbEstad, errC := ConectarSQL_ESTAD()
	if errC != nil {
		fmt.Printf("  ERROR conexion ESTAD_Tiendas: %v\n", errC)
		return
	}
	defer dbEstad.Close()

	fmt.Print("\nRecalcular UTILIDADES (por producto: 1-COSTO_ULT/VENTAS)? [S/N] (S): ")
	recalcular := true
	if resp := strings.ToUpper(strings.TrimSpace(leerLinea())); resp == "N" {
		recalcular = false
	}

	for mes := mesInicio; mes <= mesFin; mes++ {
		fmt.Printf("\n=== Procesando mes %d de %d ===\n", mes, anio)
		ProcesarEstad(db, dbEstad, sucursales, anio, mes, modo, "MANUAL", recalcular)
	}
}

func ejecutarAutomaticoEstadManual() {
	anio := leerEntero("Ano a procesar (ej. 2026): ")
	if anio < 2000 || anio > 2100 {
		fmt.Println("  ERROR: Ano fuera de rango.")
		fmt.Print("\nPresione Enter para salir...")
		leerLinea()
		return
	}
	mes := leerEntero("Mes a procesar (1-12): ")
	if mes < 1 || mes > 12 {
		fmt.Println("  ERROR: Mes fuera de rango.")
		fmt.Print("\nPresione Enter para salir...")
		leerLinea()
		return
	}

	cfg, err := CargarConfig("config_estad.json")
	if err != nil {
		cfg = Config{Modo: "IP", OutputDir: outputDir}
	}
	fmt.Printf("\n  Modo lectura: %s\n", cfg.Modo)

	procesarAutomaticoEstad(anio, mes, cfg, false)
}

func estadoTareaEstad() string {
	psCmd := `$t = Get-ScheduledTask -TaskName 'VentasMensualesESTAD' -ErrorAction SilentlyContinue; ` +
		`if ($t) { ` +
		`  $i = Get-ScheduledTaskInfo -TaskName 'VentasMensualesESTAD' -ErrorAction SilentlyContinue; ` +
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

func MenuConfigEstad() {
	cfg := Config{
		Modo:             "IP",
		OutputDir:        `C:\Users\Administrador\Desktop\Reportes FA`,
		HoraInicio:       6,
		HoraFin:          6,
		IntervaloMinutos: 1440,
	}

	if existing, err := CargarConfig("config_estad.json"); err == nil {
		cfg = existing
	}

	for {
		fmt.Println()
		fmt.Println(strings.Repeat("=", 60))
		fmt.Println("  CONFIGURACION AUTOMATIZACION ESTAD")
		fmt.Println(strings.Repeat("=", 60))
		fmt.Println()
		fmt.Printf("  [1] Hora de inicio:    %02d:00\n", cfg.HoraInicio)
		fmt.Printf("  [2] Hora de fin:       %02d:00\n", cfg.HoraFin)
		fmt.Printf("  [3] Intervalo:         %d minutos (cada %.0f horas)\n",
			cfg.IntervaloMinutos, float64(cfg.IntervaloMinutos)/60)
		fmt.Printf("  [4] Modo de lectura:   %s\n", cfg.Modo)
		fmt.Println()
		fmt.Println("  [5] GUARDAR y crear tarea programada ESTAD")
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
			if err := GuardarConfig(cfg, "config_estad.json"); err != nil {
				fmt.Printf("  ERROR: %v\n", err)
				continue
			}
			fmt.Println("  Configuracion guardada en config_estad.json")
			crearTareaProgramadaEstad(cfg)
			fmt.Println()
			fmt.Print("  Presione Enter para salir...")
			leerLinea()
			return
		case 0:
			return
		}
	}
}

func crearTareaProgramadaEstad(cfg Config) {
	exeDir, _ := os.Getwd()
	cmdPath := exeDir + "\\run_ventas_estad.cmd"

	cmd := fmt.Sprintf(`@echo off
REM Ventas Mensuales ESTAD - Ejecucion automatica
for /f "tokens=2 delims==" %%%%I in ('wmic os get localdatetime /value') do set datetime=%%%%I
set MES=%%datetime:~4,2%%
set ANIO=%%datetime:~0,4%%
if "%%MES:~0,1%%"=="0" set MES=%%MES:~1,1%%
echo [%%date%% %%time%%] ESTAD Iniciando...
"%s\ventas_mensuales.exe" --auto-estad --year=%%ANIO%% --month=%%MES%% --mode=%s
	echo [%%date%% %%time%%] ESTAD Completado.
	pause
`, exeDir, cfg.Modo)
	os.WriteFile(cmdPath, []byte(cmd), 0644)

	psPath := exeDir + "\\setup_scheduler_estad.ps1"
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
		psMessage = fmt.Sprintf(`Write-Host "Tarea ESTAD creada: $taskName (diaria %02d:00)"`, cfg.HoraInicio)
	} else {
		psMessage = fmt.Sprintf(`Write-Host "Tarea ESTAD creada: $taskName (%02d:00 a %02d:00, cada %d min)"`,
			cfg.HoraInicio, cfg.HoraFin, cfg.IntervaloMinutos)
	}

	ps := fmt.Sprintf(`
$taskName = "VentasMensualesESTAD"
$action = New-ScheduledTaskAction -Execute "%s\ventas_mensuales.exe" -Argument '--auto-estad --mode=%s'
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

	fmt.Println("  Intentando registrar tarea ESTAD...")
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
