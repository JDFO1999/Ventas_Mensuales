package main

import (
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ---------- Modelo ----------

type EstaPagoRegistro struct {
	Tienda    string
	Fecha     time.Time
	FormaPago string
	Caja      string
	Factura   string
	Cliente   string
	MontoUSD  float64
	MontoBS   float64
	Cantidad  float64
	MontoUSDD float64
	MontoBSD  float64
	CantidadD float64
}

// ---------- Rutas origen ----------

func rutaEstapagoPath(s Sucursal, year int, modo string) string {
	if modo == "S" {
		base := strings.TrimRight(s.RutaDBF, "\\")
		if base == "" {
			base = fmt.Sprintf("S:\\aBC-Soft\\Data\\%s", s.Codigo)
		}
		return strings.TrimRight(base+"\\Datos\\ESTA_PAGO"+strconv.Itoa(year)+".DBF", " ")
	}
	return fmt.Sprintf("\\\\%s\\Sistema\\aBC-Soft\\Datos\\ESTA_PAGO%d.DBF", s.RutaIP, year)
}

// ---------- Base de datos y tabla única ----------

func CrearBaseForm_PG_Tiendas(db *sql.DB) error {
	_, err := db.Exec(`IF DB_ID('Form_PG_Tiendas') IS NULL BEGIN CREATE DATABASE Form_PG_Tiendas END`)
	return err
}

func ConectarSQL_ESTAPAGO() (*sql.DB, error) {
	connStr := fmt.Sprintf("server=%s;database=Form_PG_Tiendas;user id=%s;password=%s;encrypt=disable;connection timeout=30;timeout=900",
		SQLServer, SQLUser, SQLPassword)
	db, err := sql.Open("mssql", connStr)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(6)
	return db, nil
}

func CrearTablaEstaPago(db *sql.DB) error {
	db.Exec(`IF EXISTS (SELECT 1 FROM sys.indexes WHERE object_id = OBJECT_ID('ESTA_PAGO') AND name = 'IX_ESTA_PAGO_Tienda_Fecha')
		DROP INDEX IX_ESTA_PAGO_Tienda_Fecha ON ESTA_PAGO`)
	_, err := db.Exec(`
		IF OBJECT_ID('ESTA_PAGO','U') IS NULL
		CREATE TABLE ESTA_PAGO (
			Tienda     VARCHAR(6)   NOT NULL,
			DFECHA     DATE         NOT NULL,
			CFORMAPG   VARCHAR(2)   NULL,
			CCAJA      VARCHAR(2)   NULL,
			CFACTURA   VARCHAR(8)   NULL,
			CCLIENTE   VARCHAR(100) NULL,
			NMONTODL   DECIMAL(20,8) NULL,
			NMONTO     DECIMAL(18,2) NULL,
			NCANTIDAD  DECIMAL(18,2) NULL,
			NMONTODLD  DECIMAL(20,8) NULL,
			NMONTOD    DECIMAL(18,2) NULL,
			NCANTIDADD DECIMAL(18,2) NULL,
			FechaCarga DATETIME DEFAULT GETDATE()
		)`)
	if err != nil {
		return err
	}
	_, err = db.Exec(`
		IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE object_id = OBJECT_ID('ESTA_PAGO') AND name = 'IX_ESTA_PAGO_Tienda_Fecha')
		CREATE NONCLUSTERED INDEX IX_ESTA_PAGO_Tienda_Fecha ON ESTA_PAGO(Tienda, DFECHA)`)
	return err
}

// ---------- Lectura ----------

func LeerArchivoEstaPago(path string) ([]EstaPagoRegistro, int, error) {
	dbf, err := OpenDBF(path)
	if err != nil {
		return nil, 0, err
	}
	defer dbf.Close()

	fi := dbf.FieldIndex("DFECHA")
	if fi < 0 {
		return nil, 0, fmt.Errorf("campo DFECHA no encontrado en %s", path)
	}

	getS := func(rec []byte, name string) string {
		if i := dbf.FieldIndex(name); i >= 0 {
			f := dbf.Fields[i]
			if f.Type == 'V' {
				return dbf.GetVarChar(rec, f)
			}
			return dbf.GetString(rec, f)
		}
		return ""
	}
	getF := func(rec []byte, name string) float64 {
		if i := dbf.FieldIndex(name); i >= 0 {
			return dbf.GetFloat(rec, dbf.Fields[i])
		}
		return 0
	}

	var regs []EstaPagoRegistro
	for i := 0; i < dbf.NumRecords; i++ {
		rec, e := dbf.ReadRecord(i)
		if e != nil || len(rec) == 0 {
			continue
		}
		if rec[0] == 0x2A { // registro borrado
			continue
		}

		fecha, ok := dbf.GetDate(rec, dbf.Fields[fi])
		if !ok {
			fecha = time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC)
		}

		regs = append(regs, EstaPagoRegistro{
			Tienda:    "",
			Fecha:     fecha,
			FormaPago: getS(rec, "CFORMAPG"),
			Caja:      getS(rec, "CCAJA"),
			Factura:   getS(rec, "CFACTURA"),
			Cliente:   getS(rec, "CCLIENTE"),
			MontoUSD:  getF(rec, "NMONTODL"),
			MontoBS:   getF(rec, "NMONTO"),
			Cantidad:  getF(rec, "NCANTIDAD"),
			MontoUSDD: getF(rec, "NMONTODLD"),
			MontoBSD:  getF(rec, "NMONTOD"),
			CantidadD: getF(rec, "NCANTIDADD"),
		})
	}
	return regs, 0, nil
}

// ---------- Inserción (tabla única) ----------

func InsertarEstaPago(db *sql.DB, regs []EstaPagoRegistro) (int, error) {
	const chunkSize = 5000
	query := `INSERT INTO ESTA_PAGO (Tienda, DFECHA, CFORMAPG, CCAJA, CFACTURA, CCLIENTE,
		NMONTODL, NMONTO, NCANTIDAD, NMONTODLD, NMONTOD, NCANTIDADD)
		VALUES (?,?,?,?,?,?,
			?,?,?,?,?,?)`

	insertados := 0
	for i := 0; i < len(regs); i += chunkSize {
		end := i + chunkSize
		if end > len(regs) {
			end = len(regs)
		}
		chunk := regs[i:end]

		tx, err := db.Begin()
		if err != nil {
			return insertados, err
		}
		for _, r := range chunk {
			_, err = tx.Exec(query,
				r.Tienda, r.Fecha, r.FormaPago, r.Caja, r.Factura, r.Cliente,
				r.MontoUSD, r.MontoBS, r.Cantidad, r.MontoUSDD, r.MontoBSD, r.CantidadD)
			if err != nil {
				tx.Rollback()
				return insertados, err
			}
			insertados++
		}
		if err := tx.Commit(); err != nil {
			return insertados, err
		}
		fmt.Printf("\r  SQL Progreso: %d/%d (%d%%)", end, len(regs), 100*end/len(regs))
	}
	fmt.Println()
	return insertados, nil
}

// ---------- Procesamiento por tienda ----------

func ProcesarTiendaEstapago(dbPago *sql.DB, s Sucursal, year int, modo string) (filas, insertados int, err error) {
	path := rutaEstapagoPath(s, year, modo)

	if _, errStat := os.Stat(path); errStat != nil {
		return 0, 0, fmt.Errorf("archivo no accesible: %s", path)
	}

	regs, _, errR := LeerArchivoEstaPago(path)
	if errR != nil {
		return 0, 0, errR
	}
	filas = len(regs)

	// asignar tienda por origen de la ruta
	codigo := s.Codigo

	// dedupe por clave natural completa
	existing := make(map[string]bool)
	rows, errQ := dbPago.Query(
		"SELECT DFECHA, ISNULL(CFORMAPG,''), ISNULL(CCAJA,''), ISNULL(CFACTURA,''), ISNULL(CCLIENTE,''), " +
			"ISNULL(NMONTODL,0), ISNULL(NMONTO,0), ISNULL(NCANTIDAD,0), ISNULL(NMONTODLD,0), ISNULL(NMONTOD,0), ISNULL(NCANTIDADD,0) FROM ESTA_PAGO WHERE Tienda=?",
		codigo)
	if errQ == nil {
		for rows.Next() {
			var f time.Time
			var fp, cj, fa, cl string
			var m1, m2, c1, m3, m4, c2 float64
			if rows.Scan(&f, &fp, &cj, &fa, &cl, &m1, &m2, &c1, &m3, &m4, &c2) == nil {
				existing[fmt.Sprintf("%s|%s|%s|%s|%s|%s|%g|%g|%g|%g|%g|%g",
					codigo, f.Format("20060102"), fp, cj, fa, cl, m1, m2, c1, m3, m4, c2)] = true
			}
		}
		rows.Close()
	}

	var nuevos []EstaPagoRegistro
	for _, r := range regs {
		key := fmt.Sprintf("%s|%s|%s|%s|%s|%s|%g|%g|%g|%g|%g|%g",
			codigo, r.Fecha.Format("20060102"), r.FormaPago, r.Caja, r.Factura, r.Cliente,
			r.MontoUSD, r.MontoBS, r.Cantidad, r.MontoUSDD, r.MontoBSD, r.CantidadD)
		if !existing[key] {
			r.Tienda = codigo
			nuevos = append(nuevos, r)
		}
	}

	if len(nuevos) == 0 {
		return filas, 0, nil // todo ya cargado
	}

	ins, errIns := InsertarEstaPago(dbPago, nuevos)
	if errIns != nil {
		return filas, ins, errIns
	}
	return filas, ins, nil
}

// Procesa las sucursales seleccionadas en paralelo
func ProcesarEstapago(db *sql.DB, dbPago *sql.DB, sucursales []Sucursal, year int, modo, tipoEjecucion string) {
	tStart := time.Now()
	logInfo("ESTA_PAGO: Iniciando %d tiendas, año %d, modo=%s", len(sucursales), year, modo)

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
					logError("ESTA_PAGO: PANIC en %s: %v", sucursales[i].Codigo, r)
				}
			}()

			s := sucursales[i]
			mStart := time.Now()
			filas, insertados, err := ProcesarTiendaEstapago(dbPago, s, year, modo)
			duracion := int(time.Since(mStart).Seconds())

			mu.Lock()
			defer mu.Unlock()

			if err != nil {
				logError("ESTA_PAGO: %s %d FALLO: %v", s.Codigo, year, err)
				InsertarH_Tiendas(db, s.Codigo, year, 0, 0, 0, 0, "ERROR", duracion, modo, tipoEjecucion, "ESTA_PAGO")
				errGlobal++
				return
			}

			estado := "OK"
			if insertados == 0 {
				estado = "SALTADO"
			}
			logInfo("ESTA_PAGO: %s %d (%s) DBF=%d insertados=%d", s.Codigo, year, estado, filas, insertados)
			InsertarH_Tiendas(db, s.Codigo, year, 0, filas, insertados, insertados, estado, duracion, modo, tipoEjecucion, "ESTA_PAGO")
		}()
	}

	wg.Wait()
	fmt.Printf("\n  ESTA_PAGO listo. Tiempo: %.1f min. (%d errores)\n", time.Since(tStart).Minutes(), errGlobal)
	logInfo("ESTA_PAGO: Completado en %.1f min, %d errores", time.Since(tStart).Minutes(), errGlobal)
}

// ---------- Modo automático ----------

func procesarAutomaticoEstapago(year int, cfg Config, headless bool) {
	if !headless {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("\n  *** PANIC ESTA_PAGO: %v ***\n", r)
			}
			fmt.Print("\nPresione Enter para salir...")
			leerLinea()
		}()
	}

	tStart := time.Now()
	logHistorial(fmt.Sprintf("ESTA_PAGO: Iniciado %d", year))

	db, err := ConectarSQL()
	if err != nil {
		fmt.Printf("ERROR SQL: %v\n", err)
		return
	}
	defer db.Close()

	if err := CrearBaseForm_PG_Tiendas(db); err != nil {
		fmt.Printf("ERROR creando BD Form_PG_Tiendas: %v\n", err)
		return
	}

	dbPago, err := ConectarSQL_ESTAPAGO()
	if err != nil {
		fmt.Printf("ERROR conexion Form_PG_Tiendas: %v\n", err)
		return
	}
	defer dbPago.Close()

	if err := CrearTablaEstaPago(dbPago); err != nil {
		fmt.Printf("ERROR creando tabla ESTA_PAGO: %v\n", err)
		return
	}

	sucursales, err := ObtenerSucursales(db)
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
		return
	}

	ProcesarEstapago(db, dbPago, sucursales, year, cfg.Modo, "AUTOMATICO")

	fmt.Printf("[%s] ESTA_PAGO Completado.\n", time.Now().Format("15:04:05"))
	logHistorial(fmt.Sprintf("ESTA_PAGO: Completado en %.1f min", time.Since(tStart).Minutes()))
}