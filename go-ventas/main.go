package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"database/sql"
)

const outputDir = `C:\Users\Administrador\Desktop\Reportes FA`

var (
	logFile *os.File
	logger  *log.Logger
)

func initLog() {
	var err error
	logFilePath := outputDir + "\\go-ventas\\ventas.log"
	logFile, err = os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		logFile = nil
		return
	}
	logger = log.New(io.MultiWriter(os.Stdout, logFile), "", 0)
}

func logInfo(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	line := fmt.Sprintf("[%s] INFO  %s", time.Now().Format("2006-01-02 15:04:05"), msg)
	if logger != nil {
		logger.Println(line)
	} else {
		fmt.Println(msg)
	}
}

func logError(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	line := fmt.Sprintf("[%s] ERROR %s", time.Now().Format("2006-01-02 15:04:05"), msg)
	if logger != nil {
		logger.Println(line)
	} else {
		fmt.Println(msg)
	}
}

func logWarn(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	line := fmt.Sprintf("[%s] WARN  %s", time.Now().Format("2006-01-02 15:04:05"), msg)
	if logger != nil {
		logger.Println(line)
	} else {
		fmt.Println(msg)
	}
}

func logHistorial(msg string) {
	histPath := outputDir + "\\go-ventas\\ventas_historial.log"
	f, err := os.OpenFile(histPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	line := fmt.Sprintf("[%s] %s\n", time.Now().Format("2006-01-02 15:04"), msg)
	f.WriteString(line)
}

func leerEntero(prompt string) int {
	fmt.Print(prompt)
	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	line = strings.TrimSpace(line)
	var n int
	fmt.Sscanf(line, "%d", &n)
	return n
}

func leerLinea() string {
	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	return strings.TrimSpace(line)
}

func main() {
	initLog()
	defer func() {
		if logFile != nil {
			logFile.Close()
		}
	}()

	autoPtr := flag.Bool("auto", false, "Modo automatico sin menu")
	autoCaPtr := flag.Bool("auto-ca", false, "Modo automatico CA")
	caFechaPtr := flag.Bool("ca-fecha", false, "CA por fecha rango (borra y recarga)")
	configPtr := flag.Bool("config", false, "Abrir menu de configuracion")
	autoEstadPtr := flag.Bool("auto-estad", false, "Modo automatico ESTAD")
	autoEstapagoPtr := flag.Bool("auto-estapago", false, "Modo automatico ESTA_PAGO")
	autoIGTFPtr := flag.Bool("auto-igtf", false, "Modo automatico IGTF")
	autoRcpPtr := flag.Bool("auto-rcp", false, "Modo automatico RCP")
	autoSubFamPtr := flag.Bool("auto-subfam", false, "Modo automatico SUBFAM")
	autoCxPPtr := flag.Bool("auto-cxp", false, "Modo automatico CxP")
	autoOfertasPtr := flag.Bool("auto-ofertas", false, "Modo automatico OFERTAS")
	setupOfertasPtr := flag.Bool("setup-ofertas", false, "Configura y registra automaticamente la tarea de ofertas en Windows")
	espejoPtr := flag.Bool("espejo", false, "Cargar espejo DBF a SQL Server")
	espejoSucPtr := flag.String("espejo-sucursal", "", "Sucursal a espejar (ej: VC01)")
	espejoOrigenPtr := flag.String("espejo-origen", "", "Origenes separados por coma: data,datos")
	espejoAnioPtr := flag.Int("espejo-solo-anio", 0, "Solo tablas anuales de ese anio en Datos (0=todos)")
	modePtr := flag.String("mode", "", "Modo: S o IP")
	monthPtr := flag.Int("month", 0, "Mes (1-12)")
	monthFinPtr := flag.Int("month-fin", 0, "Mes fin (1-12), opcional")
	yearPtr := flag.Int("year", 0, "Año")
	fechaInicioPtr := flag.String("fecha-inicio", "", "Fecha inicio DD/MM/YYYY (para --ca-fecha)")
	fechaFinPtr := flag.String("fecha-fin", "", "Fecha fin DD/MM/YYYY (para --ca-fecha)")
	storesPtr := flag.String("stores", "", "Tiendas separadas por coma (ej: AK04,AK05,AK07)")
	flag.Parse()

	if *configPtr {
		MenuConfig()
		return
	}

	if *espejoPtr {
		cfg := cargarConfigEspejo("espejo_config.json")
		if strings.TrimSpace(*espejoSucPtr) != "" {
			cfg.Sucursal = strings.TrimSpace(*espejoSucPtr)
			cfg.BaseDatos = cfg.Sucursal
			cfg.Base = ""
			cfg.Datos = ""
		}
		if strings.TrimSpace(*espejoOrigenPtr) != "" {
			partes := strings.Split(*espejoOrigenPtr, ",")
			cfg.Origenes = cfg.Origenes[:0]
			for _, p := range partes {
				if p = strings.TrimSpace(p); p != "" {
					cfg.Origenes = append(cfg.Origenes, p)
				}
			}
		}
		if *espejoAnioPtr > 0 {
			cfg.SoloAnio = *espejoAnioPtr
		}
		CargarEspejo(cfg)
		return
	}

	if *autoPtr {
		month := *monthPtr
		year := *yearPtr
		mode := *modePtr

		if month == 0 || year == 0 {
			now := time.Now()
			month = int(now.Month())
			year = now.Year()
		}
		if mode == "" {
			cfg, err := CargarConfig("config.json")
			if err != nil {
				fmt.Println("ERROR: Sin config. Ejecute --config primero.")
				return
			}
			mode = cfg.Modo
		}
		cfg := Config{Modo: mode, OutputDir: outputDir}
		procesarAutomatico(year, month, cfg, false, true, "AUTOMATICO")
		return
	}

	if *autoCaPtr {
		month := *monthPtr
		year := *yearPtr
		mode := *modePtr

		if month == 0 || year == 0 {
			now := time.Now()
			month = int(now.Month())
			year = now.Year()
		}
		if mode == "" {
			cfg, err := CargarConfig("config.json")
			if err != nil {
				fmt.Println("ERROR: Sin config. Ejecute --config primero.")
				return
			}
			mode = cfg.Modo
		}
		cfg := Config{Modo: mode, OutputDir: outputDir}
		procesarAutomaticoCA(year, month, cfg, true)
		return
	}

	if *caFechaPtr {
		mode := *modePtr
		if mode == "" {
			cfg, err := CargarConfig("config.json")
			if err != nil {
				fmt.Println("ERROR: Sin config. Ejecute --config primero.")
				return
			}
			mode = cfg.Modo
		}

		fechaInicioStr := *fechaInicioPtr
		fechaFinStr := *fechaFinPtr
		if fechaInicioStr == "" || fechaFinStr == "" {
			fmt.Println("ERROR: --ca-fecha requiere --fecha-inicio y --fecha-fin (DD/MM/YYYY)")
			return
		}
		fechaInicio, err := time.Parse("02/01/2006", fechaInicioStr)
		if err != nil {
			fmt.Printf("ERROR formato fecha inicio: %v\n", err)
			return
		}
		fechaFin, err := time.Parse("02/01/2006", fechaFinStr)
		if err != nil {
			fmt.Printf("ERROR formato fecha fin: %v\n", err)
			return
		}
		fechaFin = fechaFin.AddDate(0, 0, 1)

		fmt.Printf("CA FECHA: %s al %s, modo=%s\n", fechaInicio.Format("02/01/2006"), fechaFin.AddDate(0, 0, -1).Format("02/01/2006"), mode)

		db, err := ConectarSQL()
		if err != nil {
			fmt.Printf("ERROR conexion: %v\n", err)
			return
		}
		defer db.Close()

		allStores, err := ObtenerSucursales(db)
		if err != nil {
			fmt.Printf("ERROR obtener tiendas: %v\n", err)
			return
		}

		storesFilter := *storesPtr
		var sucursales []Sucursal
		if storesFilter != "" {
			selected := strings.Split(storesFilter, ",")
			for _, s := range selected {
				s = strings.TrimSpace(s)
				for _, store := range allStores {
					if strings.EqualFold(store.Codigo, s) {
						sucursales = append(sucursales, store)
						break
					}
				}
			}
			fmt.Printf("Tiendas seleccionadas: %d\n", len(sucursales))
		} else {
			sucursales = allStores
			fmt.Printf("Todas las tiendas: %d\n", len(sucursales))
		}

		ProcesarCAFecha(db, sucursales, fechaInicio.Year(), fechaInicio, fechaFin, mode)
		return
	}

	if *autoEstadPtr {
		year := *yearPtr
		month := *monthPtr
		mode := *modePtr
		if year == 0 {
			year = time.Now().Year()
		}
		if month == 0 {
			month = int(time.Now().Month())
		}
		if mode == "" {
			if cfg, err := CargarConfig("config_estad.json"); err == nil {
				mode = cfg.Modo
			} else {
				mode = "IP"
			}
		}
		cfg := Config{Modo: mode, OutputDir: outputDir}
		procesarAutomaticoEstad(year, month, cfg, true)
		return
	}

	if *autoEstapagoPtr {
		year := *yearPtr
		mode := *modePtr
		if year == 0 {
			year = time.Now().Year()
		}
		if mode == "" {
			if cfg, err := CargarConfig("config_estapago.json"); err == nil {
				mode = cfg.Modo
			} else {
				mode = "IP"
			}
		}
		cfg := Config{Modo: mode, OutputDir: outputDir}
		procesarAutomaticoEstapago(year, cfg, true)
		return
	}

	if *autoIGTFPtr {
		year := *yearPtr
		month := *monthPtr
		monthFin := *monthFinPtr
		if year == 0 {
			year = time.Now().Year()
		}
		if month == 0 {
			month = 1
		}
		if monthFin == 0 {
			monthFin = month
		}
		cfg := Config{Modo: "IP", OutputDir: outputDir}
		for m := month; m <= monthFin; m++ {
			fmt.Printf("\n=== IGTF mes %d de %d ===\n", m, year)
			procesarAutomaticoIGTF(year, m, cfg)
		}
		return
	}

	if *autoRcpPtr {
		year := *yearPtr
		mode := *modePtr
		if year == 0 {
			year = time.Now().Year()
		}
		if mode == "" {
			if cfg, err := CargarConfig("config_rcp.json"); err == nil {
				mode = cfg.Modo
			} else {
				mode = "IP"
			}
		}
		cfg := Config{Modo: mode, OutputDir: outputDir}
		procesarAutomaticoRCP(year, cfg, true)
		return
	}

	if *autoSubFamPtr {
		mode := *modePtr
		if mode == "" {
			if cfg, err := CargarConfig("config_subfam.json"); err == nil {
				mode = cfg.Modo
			} else {
				mode = "IP"
			}
		}
		cfg := Config{Modo: mode, OutputDir: outputDir}
		procesarAutomaticoSubFam(cfg, true)
		return
	}

	if *autoCxPPtr {
		year := *yearPtr
		if year == 0 {
			year = time.Now().Year()
		}
		mode := *modePtr
		if mode == "" {
			if cfg, err := CargarConfig("config_cxp.json"); err == nil {
				mode = cfg.Modo
			} else {
				mode = "IP"
			}
		}
		cfg := Config{Modo: mode, OutputDir: outputDir}
		procesarAutomaticoCxP(year, cfg, true)
		return
	}

	if *autoOfertasPtr {
		mode := *modePtr
		if mode == "" {
			if cfg, err := CargarConfig("config_ofertas.json"); err == nil {
				mode = cfg.Modo
			} else {
				mode = "IP"
			}
		}
		cfg := Config{Modo: mode, OutputDir: outputDir}
		procesarAutomaticoOfertas(cfg, true)
		return
	}

	if *setupOfertasPtr {
		cfg, err := CargarConfig("config_ofertas.json")
		if err != nil {
			cfg = Config{
				Modo:             "IP",
				OutputDir:        outputDir,
				HoraInicio:       0,
				HoraFin:          23,
				IntervaloMinutos: 5,
			}
		}
		if *modePtr != "" {
			cfg.Modo = *modePtr
		}
		crearTareaProgramadaOfertas(cfg)
		return
	}

	menuPrincipal()
}

func menuPrincipal() {
	for {
		fmt.Println()
		fmt.Println("+------------------------------------------------------------+")
		fmt.Println("|                                                            |")
		fmt.Println("|   __     __ _____ _   _ _____  _      __  __               |")
		fmt.Println("|   \\ \\   / /| ____| \\ | |_   _|/ \\    |  \\/  |              |")
		fmt.Println("|    \\ \\ / / |  _| |  \\| | | | / _ \\   | |\\/| |              |")
		fmt.Println("|     \\ V /  | |___| |\\  | | |/ ___ \\  | |  | |              |")
		fmt.Println("|      \\_/   |_____|_| \\_| |_/_/   \\_\\ |_|  |_|              |")
		fmt.Println("|                                                            |")
		fmt.Println("|   __  __ _____ _   _ ____  _   _  _    _____ ____          |")
		fmt.Println("|  |  \\/  | ____| \\ | / ___|| | | |/ \\  | ____/ ___|         |")
		fmt.Println("|  | |\\/| |  _| |  \\| \\___ \\| | | / _ \\|  _| \\___ \\          |")
		fmt.Println("|  | |  | | |___| |\\  |___) | |_|/ ___ \\ |__ ___) |         |")
		fmt.Println("|  |_|  |_|_____|_| \\_|____/ \\___/_/   \\_\\___|____/          |")
		fmt.Println("|                                                            |")
		fmt.Println("|                       A L K O S T O                        |")
		fmt.Println("|                                                            |")
		fmt.Println("+------------------------------------------------------------+")
		fmt.Println()

		fmt.Printf("  TAREA PROGRAMADA FA: %s\n", estadoTarea())
		fmt.Printf("  TAREA PROGRAMADA CA: %s\n", estadoTareaCA())
		fmt.Printf("  TAREA PROGRAMADA ESTAD: %s\n", estadoTareaEstad())
		fmt.Printf("  TAREA PROGRAMADA ESTA_PAGO: %s\n", estadoTareaEstapago())
		fmt.Printf("  TAREA PROGRAMADA RCP: %s\n", estadoTareaRCP())
		fmt.Printf("  TAREA PROGRAMADA SUBFAM: %s\n", estadoTareaSubFam())
		fmt.Printf("  TAREA PROGRAMADA CxP: %s\n", estadoTareaCxP())
		fmt.Printf("  TAREA PROGRAMADA OFERTAS: %s\n", estadoTareaOfertas())
		fmt.Println()
		fmt.Println("  ┌──── FACTURAS (FA) ──────────────────────┐")
		fmt.Println("  │  [1] Reporte manual FA                   │")
		fmt.Println("  │  [2] Configurar automatizacion FA        │")
		fmt.Println("  │  [3] Ejecutar FA modo automatico         │")
		fmt.Println("  │  [14] Insertar IGTF (Formas de pago)     │")
		fmt.Println("  └──────────────────────────────────────────┘")
		fmt.Println()
		fmt.Println("  ┌──── DETALLE PRODUCTOS (CA) ──────────────┐")
		fmt.Println("  │  [4] Reporte manual CA                   │")
		fmt.Println("  │  [5] Ejecutar CA modo automatico         │")
		fmt.Println("  │  [6] Configurar automatizacion CA        │")
		fmt.Println("  │  [21] Corregir CA por fecha              │")
		fmt.Println("  │  [22] Corregir Depto/Familia             │")
		fmt.Println("  └──────────────────────────────────────────┘")
		fmt.Println()
		fmt.Println("  ┌──── ESTADISTICAS VENTAS (ESTAD) ─────────┐")
		fmt.Println("  │  [8] Cargar ESTAD a SQL                  │")
		fmt.Println("  │  [9] Ejecutar ESTAD modo automatico      │")
		fmt.Println("  │  [10] Configurar automatizacion ESTAD   │")
		fmt.Println("  └──────────────────────────────────────────┘")
		fmt.Println()
		fmt.Println("  ┌──── FORMAS DE PAGO (ESTA_PAGO) ──────────┐")
		fmt.Println("  │  [11] Cargar ESTA_PAGO a SQL             │")
		fmt.Println("  │  [12] Ejecutar ESTA_PAGO automatico      │")
		fmt.Println("  │  [13] Configurar automatizacion ESTA_PAGO│")
		fmt.Println("  └──────────────────────────────────────────┘")
		fmt.Println()
		fmt.Println("  ┌──── RECEPCION DE COMPRA (RCP) ───────────┐")
		fmt.Println("  │  [15] Cargar RCP a SQL                   │")
		fmt.Println("  │  [16] Ejecutar RCP modo automatico       │")
		fmt.Println("  │  [17] Configurar automatizacion RCP      │")
		fmt.Println("  └──────────────────────────────────────────┘")
		fmt.Println()
		fmt.Println("  ┌──── CATEGORIZACION DE ARTICULOS ──────────┐")
		fmt.Println("  │  [18] Cargar SUBFAM a SQL                 │")
		fmt.Println("  │  [19] Ejecutar SUBFAM modo automatico     │")
		fmt.Println("  │  [20] Configurar automatizacion SUBFAM    │")
		fmt.Println("  └──────────────────────────────────────────┘")
		fmt.Println()
		fmt.Println("  ┌──── CUENTAS POR PAGAR (CxP) ──────────────┐")
		fmt.Println("  │  [23] Cargar CxP a SQL                    │")
		fmt.Println("  │  [24] Ejecutar CxP modo automatico        │")
		fmt.Println("  │  [25] Configurar automatizacion CxP       │")
		fmt.Println("  └───────────────────────────────────────────┘")
		fmt.Println()
		fmt.Println("  ┌──── ESPEJO DBF → SQL SERVER ──────────────┐")
		fmt.Println("  │  [26] Cargar espejo (Data + Datos)        │")
		fmt.Println("  └───────────────────────────────────────────┘")
		fmt.Println()
		fmt.Println("  ┌──── OFERTAS (ofer001) ────────────────────┐")
		fmt.Println("  │  [27] Cargar OFERTAS a SQL                │")
		fmt.Println("  │  [28] Ejecutar OFERTAS modo automatico    │")
		fmt.Println("  │  [29] Configurar automatizacion OFERTAS   │")
		fmt.Println("  └───────────────────────────────────────────┘")
		fmt.Println()
		fmt.Println("  [7] Ver historial de ejecuciones")
		fmt.Println("  [0] Salir")
		fmt.Println()

		op := leerEntero("  Seleccione: ")
		switch op {
		case 1:
			menuNormal()
		case 2:
			MenuConfig()
		case 3:
			ejecutarAutomaticoManual()
		case 4:
			menuCA()
		case 5:
			ejecutarAutomaticoCAManual()
		case 6:
			MenuConfigCA()
		case 7:
			verHistorial()
		case 8:
			menuEstad()
		case 9:
			ejecutarAutomaticoEstadManual()
		case 10:
			MenuConfigEstad()
		case 11:
			menuEstapago()
		case 12:
			ejecutarAutomaticoEstapagoManual()
		case 13:
			MenuConfigEstapago()
		case 14:
			menuIGTF()
		case 15:
			menuRCP()
		case 16:
			ejecutarAutomaticoRCPManual()
		case 17:
			MenuConfigRCP()
		case 18:
			menuCategorizacion()
		case 19:
			ejecutarAutomaticoSubFamManual()
		case 20:
			MenuConfigSubFam()
		case 21:
			menuCA_Fecha()
		case 22:
			menuCorregirDepto()
		case 23:
			menuCxP()
		case 24:
			ejecutarAutomaticoCxPManual()
		case 25:
			MenuConfigCxP()
		case 26:
			menuEspejo()
		case 27:
			menuOfertas()
		case 28:
			ejecutarAutomaticoOfertasManual()
		case 29:
			MenuConfigOfertas()
		case 0:
			fmt.Println("  Hasta luego.")
			return
		}
	}
}

func ejecutarAutomaticoManual() {
	now := time.Now()
	mes := int(now.Month())
	anio := now.Year()

	fmt.Printf("\n  Modo automatico: %s %d\n", MesesES[mes], anio)

	cfg, err := CargarConfig("config.json")
	if err != nil {
		cfg = Config{Modo: "S", OutputDir: outputDir}
	}
	fmt.Printf("  Modo lectura: %s | Output: %s\n\n", cfg.Modo, cfg.OutputDir)

	procesarAutomatico(anio, mes, cfg, true, false, "MANUAL")
}

func ejecutarAutomaticoCAManual() {
	now := time.Now()
	mes := int(now.Month())
	anio := now.Year()

	fmt.Printf("\n  Modo automatico CA: %s %d\n", MesesES[mes], anio)

	cfg, err := CargarConfig("config.json")
	if err != nil {
		cfg = Config{Modo: "S", OutputDir: outputDir}
	}
	fmt.Printf("  Modo lectura: %s | Output: %s\n\n", cfg.Modo, cfg.OutputDir)

	fmt.Println("\nConectando a SQL Server...")
	db, err := ConectarSQL()
	if err != nil {
		fmt.Printf("  ERROR: %v\n", err)
		fmt.Print("\nPresione Enter para salir...")
		leerLinea()
		return
	}
	defer db.Close()

	fmt.Println("\nObteniendo lista de sucursales...")
	sucursales, err := ObtenerSucursales(db)
	if err != nil {
		fmt.Printf("  ERROR: %v\n", err)
		fmt.Print("\nPresione Enter para salir...")
		leerLinea()
		return
	}
	fmt.Printf("  %d sucursales activas.\n", len(sucursales))

	procesarAutomaticoCA(anio, 0, Config{Modo: cfg.Modo, OutputDir: cfg.OutputDir}, false)
	fmt.Print("\nPresione Enter para salir...")
	leerLinea()
}

func menuNormal() {
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("  GENERADOR DE VENTAS MENSUALES DESDE POS (Go)")
	fmt.Println(strings.Repeat("=", 60))

	fmt.Println("\nConectando a SQL Server...")
	db, err := ConectarSQL()
	if err != nil {
		fmt.Printf("  ERROR de conexion: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()
	fmt.Println("  Conexion exitosa.")

	fmt.Println("\nObteniendo lista de sucursales...")
	sucursales, err := ObtenerSucursales(db)
	if err != nil {
		fmt.Printf("  ERROR: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("  %d sucursales activas encontradas.\n", len(sucursales))

	fmt.Println("\n" + strings.Repeat("-", 40))
	mes := leerEntero("Ingrese el numero de MES (1-12): ")
	if mes < 1 || mes > 12 {
		fmt.Println("  ERROR: Mes fuera de rango.")
		os.Exit(1)
	}
	anio := leerEntero("Ingrese el Ano (ej. 2026): ")
	if anio < 2000 || anio > 2100 {
		fmt.Println("  ERROR: Ano fuera de rango.")
		os.Exit(1)
	}

	nombreMes := MesesES[mes]
	fmt.Printf("\n  Procesando: %s %d\n", nombreMes, anio)
	fmt.Println(strings.Repeat("-", 40))

	fmt.Println("\nModo de lectura:")
	fmt.Println("  [1] Servidor S: (S:\\aBC-Soft\\Data\\{codigo}\\CIERRE_POS)")
	fmt.Println("  [2] Tiendas directo (\\\\IP\\Sistema\\aBC-Soft\\Cierre_POS)")
	modoInput := leerEntero("Seleccione (1/2): ")

	modo := "IP"
	if modoInput == 1 {
		modo = "S"
		fmt.Println("  Modo: Servidor S:")
	} else {
		fmt.Println("  Modo: Tiendas IP")
	}

	fmt.Println("\nSeleccion de tiendas:")
	fmt.Println("  [1] Todas las tiendas")
	fmt.Println("  [2] Elegir manualmente")
	selInput := leerEntero("Seleccione (1/2): ")

	if selInput == 2 {
		fmt.Printf("\n  Tiendas disponibles (%d):\n", len(sucursales))
		for i, s := range sucursales {
			fmt.Printf("    [%2d] %-6s - %s\n", i+1, s.Codigo, s.Nombre)
		}
		fmt.Print("  Ingrese numeros separados por comas (ej: 1,3,5-7): ")
		seleccion := leerLinea()

		var indices []int
		for _, parte := range strings.Split(seleccion, ",") {
			parte = strings.TrimSpace(parte)
			if strings.Contains(parte, "-") {
				var ini, fin int
				fmt.Sscanf(parte, "%d-%d", &ini, &fin)
				for j := ini; j <= fin; j++ {
					indices = append(indices, j)
				}
			} else {
				var n int
				fmt.Sscanf(parte, "%d", &n)
				if n > 0 && n <= len(sucursales) {
					indices = append(indices, n)
				}
			}
		}
		var seleccionadas []Sucursal
		for _, idx := range indices {
			if idx >= 1 && idx <= len(sucursales) {
				seleccionadas = append(seleccionadas, sucursales[idx-1])
			}
		}
		sucursales = seleccionadas
		fmt.Printf("\n  Seleccionadas: %d tiendas.\n", len(sucursales))
	} else {
		fmt.Printf("\n  Procesando todas: %d tiendas.\n", len(sucursales))
	}

	fmt.Println(strings.Repeat("-", 40))
	procesarNormal(db, sucursales, anio, mes, modo)
}

func procesarNormal(db *sql.DB, sucursales []Sucursal, anio, mes int, modo string) {
	tStart := time.Now()

	CrearTablaPosVentas(db)
	CrearTablaH_Tiendas(db)

	dataSQL, err := LeerDatosDesdeSQL(db, anio, mes)
	if err != nil {
		dataSQL = make(map[string]map[int]VentaPorHora)
	}

	total := len(sucursales)
	resultados := make([]ResultadoTienda, total)
	var tasks []tareaTienda

	// Releer el DBF de TODAS las tiendas y reconciliar con SQL
	for i, s := range sucursales {
		sqlAntes, _ := ContarFASQL(db, s.Codigo, anio, mes)
		tasks = append(tasks, tareaTienda{idx: i, suc: s, sqlAntes: sqlAntes})
	}

	if len(tasks) > 0 {
		dbfResultados := procesarConReintentos(tasks, total, db, anio, mes, modo, true)
		for _, t := range tasks {
			r := dbfResultados[t.idx]
			if r.Clientes == 0 && r.Tienda == "" {
				InsertarH_Tiendas(db, t.suc.Codigo, anio, mes, 0, t.sqlAntes, 0, "SIN_DATOS", 0, modo, "MANUAL", "FA")
				if tiendaData, ok := dataSQL[t.suc.Codigo]; ok {
					resultados[t.idx] = resultadoDesdeSQL(tiendaData, t.suc.Codigo)
				}
				continue
			}

			resultados[t.idx] = r

			sqlDespues, errCnt := ContarFASQL(db, t.suc.Codigo, anio, mes)
			if errCnt != nil {
				InsertarH_Tiendas(db, t.suc.Codigo, anio, mes, r.Clientes, t.sqlAntes, 0, "ERROR", 0, modo, "MANUAL", "FA")
				continue
			}
			insertados := sqlDespues - t.sqlAntes
			if insertados < 0 {
				insertados = 0
			}
			estado := "SALTADO"
			if insertados > 0 {
				estado = "OK"
			}
			InsertarH_Tiendas(db, t.suc.Codigo, anio, mes, r.Clientes, t.sqlAntes, insertados, estado, 0, modo, "MANUAL", "FA")
			fmt.Printf("\n  %s: DBF %d facts | $%.2f | SQL %d | nuevas %d | %s",
				t.suc.Codigo, r.Clientes, r.Total, sqlDespues, insertados, estado)
		}
	}

	totalFact := 0
	for _, r := range resultados {
		totalFact += r.Clientes
	}
	if totalFact == 0 {
		fmt.Println("\nADVERTENCIA: Ninguna tienda tiene datos.")
		fmt.Print("\nPresione Enter para salir...")
		leerLinea()
		return
	}

	fmt.Println("\nGenerando archivo Excel...")
	if err := GenerarExcel(resultados, anio, mes, outputDir); err != nil {
		fmt.Printf("  ERROR: %v\n", err)
	}

	fmt.Printf("Tiempo total: %.1f minutos.\n", time.Since(tStart).Minutes())
	fmt.Print("\nPresione Enter para salir...")
	leerLinea()
}

func resultadoDesdeSQL(tiendaData map[int]VentaPorHora, codigo string) ResultadoTienda {
	conteo := make(map[int]int)
	totalV := 0.0
	clientes := 0
	for h, v := range tiendaData {
		conteo[h] = v.Facturas
		totalV += v.TotalUSD
		clientes += v.Facturas
	}
	promedio := 0.0
	if clientes > 0 {
		promedio = totalV / float64(clientes)
	}
	hmKey, cm := maxKey(conteo)
	hmiKey, cmi := minKey(conteo)
	return ResultadoTienda{
		Tienda: strings.ToUpper(string(codigo[0])), PromedioFactura: promedio,
		Clientes: clientes, Total: totalV,
		HoraMayor: formatoHora(hmKey), ClientesMayor: cm,
		HoraMenor: formatoHora(hmiKey), ClientesMenor: cmi,
	}
}

func menuCA() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("\n  *** PANIC: %v ***\n", r)
		}
		fmt.Print("\nPresione Enter para salir...")
		leerLinea()
	}()

	db, sucursales, anio, modo := conectarYSucursales("REPORTE CA - DETALLE DE PRODUCTOS")
	if db == nil {
		return
	}
	defer db.Close()

	sucursales = seleccionarTiendasCA(sucursales)

	fmt.Println("\nRango de meses a procesar:")
	mesInicio := leerEntero("  Mes inicio (1-12): ")
	mesFin := leerEntero("  Mes fin (1-12): ")

	fmt.Println("\n" + strings.Repeat("-", 40))
	fmt.Println("  Procesando CA (insertando datos faltantes)...")
	if err := ProcesarCA(db, sucursales, anio, mesInicio, mesFin, modo, "MANUAL"); err != nil {
		fmt.Printf("  ERROR: %v\n", err)
	}

	fmt.Print("\nPresione Enter para salir...")
	leerLinea()
}

func menuCA_Fecha() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("\n  *** PANIC: %v ***\n", r)
		}
		fmt.Print("\nPresione Enter para salir...")
		leerLinea()
	}()

	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("  CORREGIR CA POR FECHA")
	fmt.Println(strings.Repeat("=", 60))

	fmt.Println("\nConectando a SQL Server...")
	db, err := ConectarSQL()
	if err != nil {
		fmt.Printf("  ERROR: %v\n", err)
		return
	}
	defer db.Close()
	fmt.Println("  Conexion exitosa.")

	fmt.Println("\nObteniendo lista de sucursales...")
	sucursales, err := ObtenerSucursales(db)
	if err != nil {
		fmt.Printf("  ERROR: %v\n", err)
		return
	}
	fmt.Printf("  %d sucursales activas.\n", len(sucursales))

	sucursales = seleccionarTiendasCA(sucursales)

	fmt.Println("\nModo de lectura:")
	fmt.Println("  [1] Servidor S:")
	fmt.Println("  [2] Tiendas IP")
	modo := "IP"
	if leerEntero("Seleccione: ") == 1 {
		modo = "S"
	}

	fmt.Println("\nFecha inicio (DD/MM/YYYY): ")
	fechaInicioStr := leerLinea()
	fechaInicio, err := time.Parse("02/01/2006", fechaInicioStr)
	if err != nil {
		fmt.Printf("  ERROR formato fecha: %v\n", err)
		return
	}

	fmt.Println("Fecha fin (DD/MM/YYYY): ")
	fechaFinStr := leerLinea()
	fechaFin, err := time.Parse("02/01/2006", fechaFinStr)
	if err != nil {
		fmt.Printf("  ERROR formato fecha: %v\n", err)
		return
	}

	fechaFin = fechaFin.AddDate(0, 0, 1)

	if fechaFin.Before(fechaInicio) || fechaFin.Equal(fechaInicio) {
		fmt.Println("  ERROR: La fecha fin debe ser mayor a la fecha inicio.")
		return
	}

	dias := int(fechaFin.Sub(fechaInicio).Hours() / 24)
	fmt.Printf("\n  Rango: %s al %s (%d dias)\n",
		fechaInicio.Format("02/01/2006"), fechaFin.AddDate(0, 0, -1).Format("02/01/2006"), dias)
	fmt.Printf("  Tiendas: %d | Modo: %s\n", len(sucursales), modo)
	fmt.Println(strings.Repeat("-", 40))
	fmt.Println("  Procesando CA por fecha...")

	ProcesarCAFecha(db, sucursales, fechaInicio.Year(), fechaInicio, fechaFin, modo)
}

func menuCorregirDepto() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("\n  *** PANIC: %v ***\n", r)
		}
		fmt.Print("\nPresione Enter para salir...")
		leerLinea()
	}()

	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("  CORREGIR DEPTO/FAMILIA EN POS_CA")
	fmt.Println(strings.Repeat("=", 60))

	fmt.Println("\nConectando a SQL Server...")
	db, err := ConectarSQL()
	if err != nil {
		fmt.Printf("  ERROR: %v\n", err)
		return
	}
	defer db.Close()
	fmt.Println("  Conexion exitosa.")

	fmt.Println("\nObteniendo lista de sucursales...")
	sucursales, err := ObtenerSucursales(db)
	if err != nil {
		fmt.Printf("  ERROR: %v\n", err)
		return
	}
	fmt.Printf("  %d sucursales activas.\n", len(sucursales))

	sucursales = seleccionarTiendasCA(sucursales)

	fmt.Println("\n" + strings.Repeat("-", 40))
	fmt.Println("  Corrigiendo Depto/Familia...")

	totalActualizados := 0
	totalPendientes := 0
	for _, s := range sucursales {
		fmt.Printf("\n  %s:\n", s.Codigo)
		act, pend, err := CorregirDeptoFamilia(db, s.Codigo)
		if err != nil {
			fmt.Printf("    ERROR: %v\n", err)
			continue
		}
		totalActualizados += act
		totalPendientes += pend
		fmt.Printf("    Resultado: %d actualizados, %d pendientes\n", act, pend)
	}

	fmt.Println(strings.Repeat("-", 40))
	fmt.Printf("  TOTAL: %d actualizados, %d pendientes\n", totalActualizados, totalPendientes)
}

func conectarYSucursales(titulo string) (*sql.DB, []Sucursal, int, string) {
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("  " + titulo)
	fmt.Println(strings.Repeat("=", 60))

	fmt.Println("\nConectando a SQL Server...")
	db, err := ConectarSQL()
	if err != nil {
		fmt.Printf("  ERROR: %v\n", err)
		return nil, nil, 0, ""
	}
	fmt.Println("  Conexion exitosa.")

	fmt.Println("\nObteniendo lista de sucursales...")
	sucursales, err := ObtenerSucursales(db)
	if err != nil {
		fmt.Printf("  ERROR: %v\n", err)
		return nil, nil, 0, ""
	}
	fmt.Printf("  %d sucursales activas.\n", len(sucursales))

	anio := leerEntero("Ano (ej. 2026): ")
	if anio < 2000 || anio > 2100 {
		fmt.Println("  ERROR: Ano fuera de rango.")
		return nil, nil, 0, ""
	}

	fmt.Println("\nModo de lectura:")
	fmt.Println("  [1] Servidor S:")
	fmt.Println("  [2] Tiendas IP")
	modo := "IP"
	if leerEntero("Seleccione: ") == 1 {
		modo = "S"
	}
	return db, sucursales, anio, modo
}

func seleccionarTiendasCA(sucursales []Sucursal) []Sucursal {
	fmt.Println("\nSeleccion de tiendas:")
	fmt.Println("  [1] Todas las tiendas")
	fmt.Println("  [2] Elegir manualmente")
	if leerEntero("Seleccione (1/2): ") == 2 {
		fmt.Printf("\n  Tiendas disponibles (%d):\n", len(sucursales))
		for i, s := range sucursales {
			fmt.Printf("    [%2d] %-6s - %s\n", i+1, s.Codigo, s.Nombre)
		}
		fmt.Print("  Ingrese numeros separados por comas (ej: 1,3,5-7): ")
		seleccion := leerLinea()
		var indices []int
		for _, parte := range strings.Split(seleccion, ",") {
			parte = strings.TrimSpace(parte)
			if strings.Contains(parte, "-") {
				var ini, fin int
				fmt.Sscanf(parte, "%d-%d", &ini, &fin)
				for j := ini; j <= fin; j++ {
					indices = append(indices, j)
				}
			} else {
				var n int
				fmt.Sscanf(parte, "%d", &n)
				if n > 0 && n <= len(sucursales) {
					indices = append(indices, n)
				}
			}
		}
		var sel []Sucursal
		for _, idx := range indices {
			if idx >= 1 && idx <= len(sucursales) {
				sel = append(sel, sucursales[idx-1])
			}
		}
		fmt.Printf("\n  Seleccionadas: %d tiendas.\n", len(sel))
		return sel
	}
	return sucursales
}

func verHistorial() {
	histPath := outputDir + "\\go-ventas\\ventas_historial.log"
	data, err := os.ReadFile(histPath)
	if err != nil {
		fmt.Println("\n  No hay historial todavia.")
		fmt.Print("\nPresione Enter para salir...")
		leerLinea()
		return
	}

	lines := strings.Split(string(data), "\n")
	var validas []string
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l != "" {
			validas = append(validas, l)
		}
	}

	fmt.Println("\n" + strings.Repeat("=", 70))
	fmt.Println("  HISTORIAL DE EJECUCIONES")
	fmt.Println(strings.Repeat("=", 70))

	if len(validas) == 0 {
		fmt.Println("  (sin registros)")
	} else {
		start := 0
		if len(validas) > 30 {
			start = len(validas) - 30
		}
		for _, l := range validas[start:] {
			fmt.Printf("  %s\n", l)
		}
		fmt.Printf("\n  Mostrando %d de %d registros.\n", len(validas)-start, len(validas))
	}

	logPath := outputDir + "\\go-ventas\\ventas.log"
	if errData, _ := os.ReadFile(logPath); len(errData) > 0 {
		errLines := strings.Split(string(errData), "\n")
		var errores []string
		for _, l := range errLines {
			if strings.Contains(l, "ERROR") || strings.Contains(l, "WARN") {
				errores = append(errores, l)
			}
		}
		if len(errores) > 0 {
			fmt.Println("\n  --- ULTIMOS ERRORES ---")
			start := 0
			if len(errores) > 10 {
				start = len(errores) - 10
			}
			for _, e := range errores[start:] {
				fmt.Printf("  %s\n", e)
			}
		}
	}

	fmt.Print("\nPresione Enter para salir...")
	leerLinea()
}
