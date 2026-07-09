package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"database/sql"
)

const outputDir = `C:\Users\Alkosto\Desktop\excel - automatico`

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
	autoPtr := flag.Bool("auto", false, "Modo automatico sin menu")
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
		procesarAutomatico(year, month, cfg, false)
		return
	}

	menuPrincipal()
}

func menuPrincipal() {
	for {
		fmt.Println()
		fmt.Println(strings.Repeat("=", 60))
		fmt.Println("  GENERADOR DE VENTAS MENSUALES DESDE POS (Go)")
		fmt.Println(strings.Repeat("=", 60))
		fmt.Println()
		fmt.Println("  [1] Generar reporte manual")
		fmt.Println("  [2] Configurar automatizacion")
		fmt.Println("  [3] Ejecutar modo automatico")
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

	procesarAutomatico(anio, mes, cfg, true)
	fmt.Print("\nPresione Enter para continuar...")
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
					fmt.Printf("\n[%d/%d] %s - PANIC: %v\n", idx+1, len(sucursales), suc.Codigo, r)
					resultChan <- storeResult{index: idx}
				}
			}()

			codigo := suc.Codigo
			tiendaLetra := strings.ToUpper(string(codigo[0]))

			if tiendaData, ok := dataSQL[codigo]; ok {
				conteo := make(map[int]int)
				total := 0.0
				clientes := 0
				for h, v := range tiendaData {
					conteo[h] = v.Facturas
					total += v.TotalUSD
					clientes += v.Facturas
				}
				promedio := 0.0
				if clientes > 0 {
					promedio = total / float64(clientes)
				}
				hmKey, cm := maxKey(conteo)
				hmiKey, cmi := minKey(conteo)

				fmt.Printf("\n[%d/%d] %s - %s", idx+1, len(sucursales), suc.Codigo, suc.Nombre)
				if clientes > 0 {
					fmt.Printf("\n  SQL: %d facts | Total: $%.2f | Pico: %s (%d)\n",
						clientes, total, formatoHora(hmKey), cm)
				}
				resultChan <- storeResult{index: idx, res: ResultadoTienda{
					Tienda: tiendaLetra, PromedioFactura: promedio,
					Clientes: clientes, Total: total,
					HoraMayor: formatoHora(hmKey), ClientesMayor: cm,
					HoraMenor: formatoHora(hmiKey), ClientesMenor: cmi,
				}}
				return
			}

			ch := ProcesarTienda(suc, db, anio, mes, modo, true)
			select {
			case msg := <-ch:
				fmt.Printf("\n[%d/%d] %s - %s", idx+1, len(sucursales), suc.Codigo, suc.Nombre)
				if msg.Err != nil {
					fmt.Printf("\n  %v\n", msg.Err)
				} else if msg.Resultado.Clientes > 0 {
					fmt.Printf("\n  => %d facts | Total: $%.2f | Prom: $%.2f | Pico: %s (%d)\n",
						msg.Resultado.Clientes, msg.Resultado.Total,
						msg.Resultado.PromedioFactura,
						msg.Resultado.HoraMayor, msg.Resultado.ClientesMayor)
				} else {
					fmt.Printf("\n  Sin datos de ventas.\n")
				}
				resultChan <- storeResult{index: idx, res: msg.Resultado}
			case <-time.After(30 * time.Minute):
				fmt.Printf("\n[%d/%d] %s - TIMEOUT\n", idx+1, len(sucursales), suc.Codigo)
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
