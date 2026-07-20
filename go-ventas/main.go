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

const outputDir = `C:\Users\Alkosto\Desktop\excel - automatico`

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
	configPtr := flag.Bool("config", false, "Abrir menu de configuracion")
	modePtr := flag.String("mode", "", "Modo: S o IP")
	monthPtr := flag.Int("month", 0, "Mes (1-12)")
	yearPtr := flag.Int("year", 0, "Año")
	flag.Parse()

	if *configPtr {
		MenuConfig()
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
		procesarAutomatico(year, month, cfg, false, true)
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
		fmt.Println()
		fmt.Println("  ┌──── FACTURAS (FA) ──────────────────────┐")
		fmt.Println("  │  [1] Reporte manual FA                   │")
		fmt.Println("  │  [2] Configurar automatizacion FA        │")
		fmt.Println("  │  [3] Ejecutar FA modo automatico         │")
		fmt.Println("  └──────────────────────────────────────────┘")
		fmt.Println()
		fmt.Println("  ┌──── DETALLE PRODUCTOS (CA) ──────────────┐")
		fmt.Println("  │  [4] Reporte manual CA                   │")
		fmt.Println("  │  [5] Ejecutar CA modo automatico         │")
		fmt.Println("  │  [6] Configurar automatizacion CA        │")
		fmt.Println("  └──────────────────────────────────────────┘")
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

	procesarAutomatico(anio, mes, cfg, true, false)
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

	dataSQL, err := LeerDatosDesdeSQL(db, anio, mes)
	if err != nil {
		dataSQL = make(map[string]map[int]VentaPorHora)
	}
	if len(dataSQL) > 0 {
		fmt.Printf("  %d tiendas con datos en SQL (se saltara DBF)\n", len(dataSQL))
	}

	total := len(sucursales)
	resultados := make([]ResultadoTienda, total)
	var tasks []tareaTienda

	for i, s := range sucursales {
		codigo := s.Codigo
		tiendaLetra := strings.ToUpper(string(codigo[0]))

		if tiendaData, ok := dataSQL[codigo]; ok {
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

			fmt.Printf("\n[%d/%d] %s - %s", i+1, total, codigo, s.Nombre)
			if clientes > 0 {
				fmt.Printf("\n  SQL: %d facts | Total: $%.2f | Pico: %s (%d)\n",
					clientes, totalV, formatoHora(hmKey), cm)
			}
			resultados[i] = ResultadoTienda{
				Tienda: tiendaLetra, PromedioFactura: promedio,
				Clientes: clientes, Total: totalV,
				HoraMayor: formatoHora(hmKey), ClientesMayor: cm,
				HoraMenor: formatoHora(hmiKey), ClientesMenor: cmi,
			}
		} else {
			tasks = append(tasks, tareaTienda{idx: i, suc: s})
		}
	}

	if len(tasks) > 0 {
		dbfResultados := procesarConReintentos(tasks, total, db, anio, mes, modo, true)
		for i, r := range dbfResultados {
			if r.Clientes > 0 || r.Tienda != "" {
				resultados[i] = r
			}
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

	forceFull := false
	fmt.Print("\n  Reprocesar todo desde cero? (S/N) [N]: ")
	resp := strings.ToUpper(leerLinea())
	if resp == "S" {
		forceFull = true
		fmt.Println("  Modo: reprocesado completo (borra y reinserta todo)")
	} else {
		fmt.Println("  Modo: incremental (solo inserta datos faltantes)")
	}

	fmt.Println("\n" + strings.Repeat("-", 40))
	fmt.Println("  Procesando CA (insertando datos faltantes)...")
	if err := ProcesarCA(db, sucursales, anio, mesInicio, mesFin, modo, forceFull); err != nil {
		fmt.Printf("  ERROR: %v\n", err)
	}

	fmt.Print("\nPresione Enter para salir...")
	leerLinea()
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
