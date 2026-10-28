package main

import (
	"database/sql"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"
)

func rnd8(v float64) float64 {
	return math.Round(v*1e8) / 1e8
}

func menuIGTF() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("\n  *** PANIC IGTF: %v ***\n", r)
		}
		fmt.Print("\nPresione Enter para salir...")
		leerLinea()
	}()

	db, sucursales, anio, _ := conectarYSucursales("INSERTAR IGTF A FORMAS DE PAGO (XX)")
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
	fmt.Printf("  Procesando IGTF %d, meses %d-%d, %d tiendas...\n",
		anio, mesInicio, mesFin, len(sucursales))

	dbPago, errC := ConectarSQL_ESTAPAGO()
	if errC != nil {
		fmt.Printf("  ERROR conexion Form_PG_Tiendas: %v\n", errC)
		return
	}
	defer dbPago.Close()

	for mes := mesInicio; mes <= mesFin; mes++ {
		fmt.Printf("\n=== Procesando mes %d de %d ===\n", mes, anio)
		ProcesarIGTF(db, dbPago, sucursales, anio, mes, "MANUAL")
	}
}

func ProcesarTiendaIGTF(db *sql.DB, dbPago *sql.DB, codigo string, year, month int) (filas, insertados int, err error) {
	startDate := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	endDate := startDate.AddDate(0, 1, 0)

	rows, err := db.Query(
		"SELECT Fecha, Caja, Numero, ISNULL(CodCli,''), IGTF, TasaDOL FROM PosVentas WHERE Tienda=? AND Fecha >= ? AND Fecha < ? AND IGTF > 0",
		codigo, startDate, endDate)
	if err != nil {
		return 0, 0, err
	}
	defer rows.Close()

	var regs []EstaPagoRegistro
	for rows.Next() {
		var f time.Time
		var caja int
		var numero, cliente string
		var igtf, tasa float64
		if err := rows.Scan(&f, &caja, &numero, &cliente, &igtf, &tasa); err != nil {
			return 0, 0, err
		}
		usd := 0.0
		if tasa > 0 {
			usd = igtf / tasa
		}
		num := numero
		if len(num) > 8 {
			num = num[:8]
		}
		regs = append(regs, EstaPagoRegistro{
			Tienda:    codigo,
			Fecha:     f,
			FormaPago: "XX",
			Caja:      fmt.Sprintf("%02d", caja),
			Factura:   num,
			Cliente:   cliente,
			MontoUSD:  usd,
			MontoBS:   igtf,
			Cantidad:  1,
		})
	}
	filas = len(regs)

	existing := make(map[string]bool)
	rowsE, errQ := dbPago.Query(
		"SELECT DFECHA, ISNULL(CCAJA,''), ISNULL(CFACTURA,''), ISNULL(CCLIENTE,''), " +
			"ISNULL(NMONTODL,0), ISNULL(NMONTO,0), ISNULL(NCANTIDAD,0), ISNULL(NMONTODLD,0), ISNULL(NMONTOD,0), ISNULL(NCANTIDADD,0) " +
			"FROM ESTA_PAGO WHERE Tienda=? AND CFORMAPG='XX'",
		codigo)
	if errQ == nil {
		for rowsE.Next() {
			var f time.Time
			var cj, fa, cl string
			var m1, m2, c1, m3, m4, c2 float64
			if rowsE.Scan(&f, &cj, &fa, &cl, &m1, &m2, &c1, &m3, &m4, &c2) == nil {
				existing[fmt.Sprintf("%s|%s|XX|%s|%s|%s|%g|%g|%g|%g|%g|%g",
					codigo, f.Format("20060102"), cj, fa, cl, rnd8(m1), rnd8(m2), rnd8(c1), rnd8(m3), rnd8(m4), rnd8(c2))] = true
			}
		}
		rowsE.Close()
	}

	var nuevos []EstaPagoRegistro
	for _, r := range regs {
		key := fmt.Sprintf("%s|%s|XX|%s|%s|%s|%g|%g|%g|%g|%g|%g",
			codigo, r.Fecha.Format("20060102"), r.Caja, r.Factura, r.Cliente,
			rnd8(r.MontoUSD), rnd8(r.MontoBS), rnd8(r.Cantidad), rnd8(r.MontoUSDD), rnd8(r.MontoBSD), rnd8(r.CantidadD))
		if !existing[key] {
			nuevos = append(nuevos, r)
		}
	}

	if len(nuevos) == 0 {
		return filas, 0, nil
	}

	ins, errIns := InsertarEstaPago(dbPago, nuevos)
	if errIns != nil {
		return filas, ins, errIns
	}
	return filas, ins, nil
}

func ProcesarIGTF(db *sql.DB, dbPago *sql.DB, sucursales []Sucursal, year, month int, tipoEjecucion string) {
	tStart := time.Now()
	logInfo("IGTF: Iniciando %d tiendas, %s %d", len(sucursales), MesesES[month], year)

	CrearTablaH_Tiendas(db)

	var wg sync.WaitGroup
	sem := make(chan struct{}, 6)
	errGlobal := 0
	var mu sync.Mutex

	for i := 0; i < len(sucursales); i++ {
		i := i
		sem <- struct{}{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			defer func() {
				if r := recover(); r != nil {
					logError("IGTF: PANIC en %s: %v", sucursales[i].Codigo, r)
				}
			}()

			s := sucursales[i]
			mStart := time.Now()
			filas, insertados, err := ProcesarTiendaIGTF(db, dbPago, s.Codigo, year, month)
			duracion := int(time.Since(mStart).Seconds())

			mu.Lock()
			defer mu.Unlock()

			if err != nil {
				logError("IGTF: %s %d-%d FALLO: %v", s.Codigo, year, month, err)
				InsertarH_Tiendas(db, s.Codigo, year, month, 0, 0, 0, "ERROR", duracion, "IP", tipoEjecucion, "IGTF")
				errGlobal++
				return
			}

			estado := "OK"
			if insertados == 0 {
				estado = "SALTADO"
			}
			logInfo("IGTF: %s %d-%d (%s) facturas=%d insertados=%d", s.Codigo, year, month, estado, filas, insertados)
			InsertarH_Tiendas(db, s.Codigo, year, month, 0, filas, insertados, estado, duracion, "IP", tipoEjecucion, "IGTF")
		}()
	}

	wg.Wait()
	fmt.Printf("\n  IGTF listo. Tiempo: %.1f min. (%d errores)\n", time.Since(tStart).Minutes(), errGlobal)
	logInfo("IGTF: Completado en %.1f min, %d errores", time.Since(tStart).Minutes(), errGlobal)
}

func procesarAutomaticoIGTF(year, month int, cfg Config) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("\n  *** PANIC IGTF: %v ***\n", r)
		}
		fmt.Print("\nPresione Enter para salir...")
		leerLinea()
	}()

	tStart := time.Now()
	logHistorial(fmt.Sprintf("IGTF: Iniciado %d-%d", year, month))

	db, err := ConectarSQL()
	if err != nil {
		fmt.Printf("ERROR SQL: %v\n", err)
		return
	}
	defer db.Close()

	sucursales, err := ObtenerSucursales(db)
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
		return
	}

	dbPago, err := ConectarSQL_ESTAPAGO()
	if err != nil {
		fmt.Printf("ERROR conexion Form_PG_Tiendas: %v\n", err)
		return
	}
	defer dbPago.Close()

	ProcesarIGTF(db, dbPago, sucursales, year, month, "AUTOMATICO")

	fmt.Printf("[%s] IGTF Completado.\n", time.Now().Format("15:04:05"))
	logHistorial(fmt.Sprintf("IGTF: Completado en %.1f min", time.Since(tStart).Minutes()))
}
