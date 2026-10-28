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

type EstadRegistro struct {
	Fecha     time.Time
	Mes       string
	Ano       string
	Codigo    string
	Descrip   string
	Cantidad  float64
	CostoPro  float64
	CostoUlt  float64
	Precio    float64
	Ventas    float64
	VentasDol float64
	VtaSiva   float64
	VtaDLsiva float64
	CostoDol  float64
	CodProv   string
	Gruplin   string
	Depto     string
	Familia   string
	IVA       float64
	CantEmp   float64
	Utilidades *float64
}

// ---------- Rutas origen ----------

func rutaEstadPath(s Sucursal, year int, modo string) string {
	if modo == "S" {
		base := strings.TrimRight(s.RutaDBF, "\\")
		if base == "" {
			base = fmt.Sprintf("S:\\aBC-Soft\\Data\\%s", s.Codigo)
		}
		return strings.TrimRight(base+"\\Datos\\ESTAD"+strconv.Itoa(year)+".DBF", " ")
	}
	return fmt.Sprintf("\\\\%s\\Sistema\\aBC-Soft\\Datos\\ESTAD%d.DBF", s.RutaIP, year)
}

// ---------- Base de datos y tablas por tienda ----------

func CrearBaseESTAD_Tiendas(db *sql.DB) error {
	_, err := db.Exec(`IF DB_ID('ESTAD_Tiendas') IS NULL BEGIN CREATE DATABASE ESTAD_Tiendas END`)
	return err
}

func ConectarSQL_ESTAD() (*sql.DB, error) {
	connStr := fmt.Sprintf("server=%s;database=ESTAD_Tiendas;user id=%s;password=%s;encrypt=disable;connection timeout=30;timeout=900",
		SQLServer, SQLUser, SQLPassword)
	db, err := sql.Open("mssql", connStr)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(6)
	return db, nil
}

func CrearTablaEstadTienda(db *sql.DB, codigo string) error {
	table := "ESTAD_" + codigo
	_, err := db.Exec(fmt.Sprintf(`
		IF OBJECT_ID('%s','U') IS NULL
		CREATE TABLE %s (
			Fecha      DATE         NOT NULL,
			Mes        VARCHAR(2)   NULL,
			Ano        VARCHAR(4)   NULL,
			Codigo     VARCHAR(15)  NOT NULL,
			Descrip    VARCHAR(120) NULL,
			Cantidad   DECIMAL(10,3) NULL,
			COSTO_PRO  DECIMAL(18,2) NULL,
			COSTO_ULT  DECIMAL(18,2) NULL,
			PRECIO     DECIMAL(18,2) NULL,
			VENTAS     DECIMAL(18,2) NULL,
			VENTAS_DOL DECIMAL(18,2) NULL,
			VTA_SIVA   DECIMAL(18,2) NULL,
			VTA_DLSIVA DECIMAL(18,2) NULL,
			COSTO_DOL  DECIMAL(20,8) NULL,
			CODPROV    VARCHAR(10) NULL,
			GRUPLIN    VARCHAR(6)  NULL,
			DEPTO      VARCHAR(2)  NULL,
			FAMILIA    VARCHAR(2)  NULL,
			IVA        DECIMAL(8,2) NULL,
			CANTEMP    DECIMAL(16,2) NULL,
			UTILIDADES DECIMAL(18,6) NULL,
			FECHA_CARGA DATETIME DEFAULT GETDATE()
		)`, table, table))
	if err != nil {
		return err
	}
	_, err = db.Exec(fmt.Sprintf(`
		IF COL_LENGTH('%s','UTILIDADES') IS NULL
		ALTER TABLE %s ADD UTILIDADES DECIMAL(18,6) NULL`, table, table))
	if err != nil {
		return err
	}
	_, err = db.Exec(fmt.Sprintf(`
		IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE object_id = OBJECT_ID('%s') AND name = 'IX_ESTAD_Fecha')
		BEGIN
			BEGIN TRY
				CREATE NONCLUSTERED INDEX IX_ESTAD_Fecha ON %s(Fecha, Codigo)
			END TRY
			BEGIN CATCH
				IF ERROR_NUMBER() <> 1913 THROW
			END CATCH
		END`, table, table))
	return err
}

// ---------- Conteo y borrado ----------

func ContarEstadSQL(db *sql.DB, codigo string, year, mes int) (int, error) {
	table := "ESTAD_" + codigo
	var count int
	var err error
	if mes > 0 {
		err = db.QueryRow(fmt.Sprintf(
			"SELECT COUNT(*) FROM %s WHERE YEAR(Fecha)=? AND MONTH(Fecha)=?", table), year, mes).Scan(&count)
	} else {
		err = db.QueryRow(fmt.Sprintf(
			"SELECT COUNT(*) FROM %s WHERE YEAR(Fecha)=?", table), year).Scan(&count)
	}
	return count, err
}

func ContarEstadDBF(path string, year, mes int) (int, error) {
	dbf, err := OpenDBF(path)
	if err != nil {
		return 0, err
	}
	defer dbf.Close()

	fi := dbf.FieldIndex("FECHA")
	if fi < 0 {
		return 0, nil
	}

	count := 0
	for i := 0; i < dbf.NumRecords; i++ {
		rec, e := dbf.ReadRecord(i)
		if e != nil || len(rec) == 0 {
			continue
		}
		if rec[0] == 0x2A {
			continue
		}
		fecha, ok := dbf.GetDate(rec, dbf.Fields[fi])
		if !ok {
			continue
		}
		if fecha.Year() != year {
			continue
		}
		if mes > 0 && int(fecha.Month()) != mes {
			continue
		}
		count++
	}
	return count, nil
}

func BorrarEstadTienda(db *sql.DB, codigo string, year, mes int) error {
	table := "ESTAD_" + codigo
	var startDate, endDate time.Time
	if mes > 0 {
		startDate = time.Date(year, time.Month(mes), 1, 0, 0, 0, 0, time.UTC)
		endDate = startDate.AddDate(0, 1, 0)
	} else {
		startDate = time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
		endDate = time.Date(year+1, 1, 1, 0, 0, 0, 0, time.UTC)
	}

	dias := int(endDate.Sub(startDate).Hours() / 24)
	for d := 0; d < dias; d++ {
		dia := startDate.AddDate(0, 0, d)
		diaFin := dia.AddDate(0, 0, 1)
		if diaFin.After(endDate) {
			diaFin = endDate
		}

		result, err := db.Exec(fmt.Sprintf("DELETE FROM %s WHERE Fecha >= ? AND Fecha < ?", table), dia, diaFin)
		if err != nil {
			for retry := 0; retry < 2; retry++ {
				time.Sleep(time.Duration(retry+1) * time.Second)
				result, err = db.Exec(fmt.Sprintf("DELETE TOP(50000) FROM %s WHERE Fecha >= ? AND Fecha < ?", table), dia, diaFin)
				if err == nil {
					break
				}
			}
			if err != nil {
				fmt.Printf("\n    WARNING: No se pudo borrar %s dia %s: %v", codigo, dia.Format("02/01"), err)
				continue
			}
		}
		rows, _ := result.RowsAffected()
		if rows > 0 {
			fmt.Printf("\n    %s borrado %s: %d registros", codigo, dia.Format("02/01"), rows)
		}
	}
	return nil
}

// ---------- Lectura ----------

func LeerArchivoEstad(path string, year int, mes int) ([]EstadRegistro, int, error) {
	dbf, err := OpenDBF(path)
	if err != nil {
		return nil, 0, err
	}
	defer dbf.Close()

	fi := dbf.FieldIndex("FECHA")
	ci := dbf.FieldIndex("CODIGO")
	if fi < 0 || ci < 0 {
		return nil, 0, fmt.Errorf("campos FECHA/CODIGO no encontrados en %s", path)
	}

	getS := func(rec []byte, name string) string {
		if i := dbf.FieldIndex(name); i >= 0 {
			return strings.TrimSpace(dbf.GetString(rec, dbf.Fields[i]))
		}
		return ""
	}
	getF := func(rec []byte, name string) float64 {
		if i := dbf.FieldIndex(name); i >= 0 {
			return dbf.GetFloat(rec, dbf.Fields[i])
		}
		return 0
	}

	var regs []EstadRegistro
	invalid := 0
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
			invalid++
			continue
		}
		if fecha.Year() != year {
			continue
		}
		if mes > 0 && int(fecha.Month()) != mes {
			continue
		}
		codigo := strings.TrimSpace(dbf.GetString(rec, dbf.Fields[ci]))
		if codigo == "" {
			invalid++
			continue
		}

		costUlt := getF(rec, "COSTO_ULT")
		ventas := getF(rec, "VENTAS")

		regs = append(regs, EstadRegistro{
			Fecha:     fecha,
			Mes:       fmt.Sprintf("%02d", int(fecha.Month())),
			Ano:       strconv.Itoa(fecha.Year()),
			Codigo:    codigo,
			Descrip:   getS(rec, "DESCRIP"),
			Cantidad:  getF(rec, "CANTIDAD"),
			CostoPro:  getF(rec, "COSTO_PRO"),
			CostoUlt:  costUlt,
			Precio:    getF(rec, "PRECIO"),
			Ventas:    ventas,
			VentasDol: getF(rec, "VENTAS_DOL"),
			VtaSiva:   getF(rec, "VTA_SIVA"),
			VtaDLsiva: getF(rec, "VTA_DLSIVA"),
			CostoDol:  getF(rec, "COSTO_DOL"),
			CodProv:   getS(rec, "CODPROV"),
			Gruplin:   getS(rec, "GRUPLIN"),
			Depto:     getS(rec, "DEPTO"),
			Familia:   getS(rec, "FAMILIA"),
			IVA:       getF(rec, "IVA"),
			CantEmp:   getF(rec, "CANTEMP"),
		})
	}
	return regs, invalid, nil
}

// ---------- Inserción ----------

func InsertarEstad(db *sql.DB, regs []EstadRegistro, codigo string) (int, error) {
	const chunkSize = 5000
	table := "ESTAD_" + codigo
	query := fmt.Sprintf(`INSERT INTO %s (Fecha, Mes, Ano, Codigo, Descrip, Cantidad,
		COSTO_PRO, COSTO_ULT, PRECIO, VENTAS, VENTAS_DOL, VTA_SIVA, VTA_DLSIVA, COSTO_DOL,
		CODPROV, GRUPLIN, DEPTO, FAMILIA, IVA, CANTEMP, UTILIDADES)
		VALUES (?,?,?,?,?,?,
			?,?,?,?,?,?,?,?,
			?,?,?,?,?,?,?)`, table)

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
				r.Fecha, r.Mes, r.Ano, r.Codigo, r.Descrip, r.Cantidad,
				r.CostoPro, r.CostoUlt, r.Precio, r.Ventas, r.VentasDol, r.VtaSiva, r.VtaDLsiva, r.CostoDol,
				r.CodProv, r.Gruplin, r.Depto, r.Familia, r.IVA, r.CantEmp, r.Utilidades)
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

func ProcesarTiendaEstad(dbEstad *sql.DB, s Sucursal, year int, mes int, modo string) (filas, insertados, invalidos int, err error) {
	path := rutaEstadPath(s, year, modo)

	if _, errStat := os.Stat(path); errStat != nil {
		return 0, 0, 0, fmt.Errorf("archivo no accesible: %s", path)
	}

	if err := CrearTablaEstadTienda(dbEstad, s.Codigo); err != nil {
		return 0, 0, 0, err
	}

	totalDBF, errContar := ContarEstadDBF(path, year, mes)
	if errContar != nil {
		return 0, 0, 0, fmt.Errorf("error contando DBF: %v", errContar)
	}

	totalSQL, errSQL := ContarEstadSQL(dbEstad, s.Codigo, year, mes)
	if errSQL != nil {
		return 0, 0, 0, fmt.Errorf("error contando SQL: %v", errSQL)
	}

	if totalDBF == totalSQL && totalDBF > 0 {
		return totalDBF, 0, 0, nil
	}

	if totalSQL > totalDBF {
		fmt.Printf("\r  %s SQL>DBF (%d>%d), borrando...", s.Codigo, totalSQL, totalDBF)
		if delErr := BorrarEstadTienda(dbEstad, s.Codigo, year, mes); delErr != nil {
			return 0, 0, 0, fmt.Errorf("error borrando: %v", delErr)
		}
		totalSQL, _ = ContarEstadSQL(dbEstad, s.Codigo, year, mes)
	}

	regs, invalid, errR := LeerArchivoEstad(path, year, mes)
	if errR != nil {
		return 0, 0, 0, errR
	}
	filas = len(regs)
	invalidos = invalid

	forceReload := false
	var nuevos []EstadRegistro

	if !forceReload {
		existing := make(map[string]bool)
		rows, errQ := dbEstad.Query(fmt.Sprintf(
			"SELECT Codigo, CONVERT(VARCHAR(8),Fecha,112), ISNULL(DEPTO,''), ISNULL(FAMILIA,'') FROM %s",
			"ESTAD_"+s.Codigo))
		if errQ == nil {
			for rows.Next() {
				var c, fe, de, fa string
				if rows.Scan(&c, &fe, &de, &fa) == nil {
					existing[c+"|"+fe+"|"+de+"|"+fa] = true
				}
			}
			rows.Close()
		}
		for _, r := range regs {
			key := r.Codigo + "|" + r.Fecha.Format("20060102") + "|" + r.Depto + "|" + r.Familia
			if !existing[key] {
				nuevos = append(nuevos, r)
			}
		}
	} else {
		nuevos = regs
	}

	if len(nuevos) == 0 {
		return filas, 0, invalidos, nil
	}

	ins, errIns := InsertarEstad(dbEstad, nuevos, s.Codigo)
	if errIns != nil {
		return filas, ins, invalidos, errIns
	}
	return filas, ins, invalidos, nil
}

func ActualizarUtilidadesEstad(dbEstad *sql.DB, codigo string) error {
	table := "ESTAD_" + codigo
	if _, err := dbEstad.Exec(fmt.Sprintf(
		`IF COL_LENGTH('%s','UTILIDADES') IS NULL ALTER TABLE %s ADD UTILIDADES DECIMAL(18,6) NULL`,
		table, table)); err != nil {
		return err
	}
	_, err := dbEstad.Exec(fmt.Sprintf(`
		UPDATE %s SET UTILIDADES = ROUND(
			CASE WHEN VENTAS IS NULL OR VENTAS = 0 THEN 0
			     ELSE (1 - COSTO_ULT / NULLIF(VENTAS, 0)) * 100 END, 2)`, table))
	return err
}

// Procesa un lote de sucursales en paralelo y retorna las que fallaron
func procesarLoteEstad(db *sql.DB, dbEstad *sql.DB, sucursales []Sucursal, year int, mes int, modo, tipoEjecucion string, errGlobal *int) []Sucursal {
	var wg sync.WaitGroup
	sem := make(chan struct{}, 4)
	var mu sync.Mutex
	var fallidas []Sucursal

	for i := 0; i < len(sucursales); i++ {
		i := i
		sem <- struct{}{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			defer func() {
				if r := recover(); r != nil {
					logError("ESTAD: PANIC en %s: %v", sucursales[i].Codigo, r)
				}
			}()

			s := sucursales[i]
			mStart := time.Now()
			filas, insertados, invalidos, err := ProcesarTiendaEstad(dbEstad, s, year, mes, modo)
			duracion := int(time.Since(mStart).Seconds())

			mu.Lock()
			defer mu.Unlock()

			if err != nil {
				logError("ESTAD: %s %d FALLO: %v", s.Codigo, year, err)
				InsertarH_Tiendas(db, s.Codigo, year, 0, 0, 0, 0, "ERROR", duracion, modo, tipoEjecucion, "ESTAD")
				*errGlobal++
				fallidas = append(fallidas, s)
				return
			}

			finalCount, _ := ContarEstadSQL(dbEstad, s.Codigo, year, mes)
			estado := "OK"
			if filas > 0 && finalCount != filas {
				estado = "REVISAR"
				logWarn("ESTAD: %s %d REVISAR (DBF=%d SQL=%d diferencia=%d)", s.Codigo, year, filas, finalCount, filas-finalCount)
			} else if insertados == 0 {
				estado = "SALTADO"
			}
			logInfo("ESTAD: %s %d (%s) DBF=%d SQL=%d insertados=%d invalidos=%d",
				s.Codigo, year, estado, filas, finalCount, insertados, invalidos)
			InsertarH_Tiendas(db, s.Codigo, year, 0, filas, finalCount, insertados, estado, duracion, modo, tipoEjecucion, "ESTAD")
		}()
	}

	wg.Wait()
	return fallidas
}

// Procesa las sucursales seleccionadas con reintentos automáticos
func ProcesarEstad(db *sql.DB, dbEstad *sql.DB, sucursales []Sucursal, year int, mes int, modo, tipoEjecucion string, recalcular bool) {
	tStart := time.Now()
	logInfo("ESTAD: Iniciando %d tiendas, año %d, mes %d, modo=%s", len(sucursales), year, mes, modo)

	CrearTablaH_Tiendas(db)

	errGlobal := 0
	pendientes := sucursales

	for intento := 1; intento <= 3; intento++ {
		if len(pendientes) == 0 {
			break
		}
		if intento > 1 {
			fmt.Printf("\n  Reintentando %d tiendas fallidas (intento %d/3)...\n", len(pendientes), intento)
		}
		removidas := procesarLoteEstad(db, dbEstad, pendientes, year, mes, modo, tipoEjecucion, &errGlobal)
		pendientes = removidas
	}

	// backfill UTILIDADES secuencial (evita saturar SQL Server en paralelo)
	if recalcular {
		fmt.Println("\n  Calculando UTILIDADES (por producto: 1 - COSTO_ULT/VENTAS)...")
		for i := range sucursales {
			codigo := sucursales[i].Codigo
			if err := ActualizarUtilidadesEstad(dbEstad, codigo); err != nil {
				errBackfill := err
				reintento := 0
				for reintento = 1; reintento <= 2; reintento++ {
					logWarn("ESTAD: %s backfill intento %d: %v", codigo, reintento, errBackfill)
					if err := ActualizarUtilidadesEstad(dbEstad, codigo); err == nil {
						errBackfill = nil
						break
					} else {
						errBackfill = err
					}
				}
				if errBackfill != nil {
					logError("ESTAD: %s backfill UTILIDADES FALLO: %v", codigo, errBackfill)
					errGlobal++
					continue
				}
			}
			logInfo("ESTAD: %s UTILIDADES actualizadas", codigo)
		}
	} else {
		fmt.Println("\n  Omitiendo recalculo de UTILIDADES.")
	}

	resumenUtilidadesEstad(dbEstad, sucursales)

	fmt.Printf("\n  ESTAD listo. Tiempo: %.1f min. (%d errores)\n", time.Since(tStart).Minutes(), errGlobal)
	logInfo("ESTAD: Completado en %.1f min, %d errores", time.Since(tStart).Minutes(), errGlobal)
}

func resumenUtilidadesEstad(dbEstad *sql.DB, sucursales []Sucursal) {
	var totV, totC float64
	hayDatos := false
	fmt.Println()
	for i := range sucursales {
		codigo := sucursales[i].Codigo
		table := "ESTAD_" + codigo
		rows, err := dbEstad.Query(fmt.Sprintf(`
			SELECT YEAR(Fecha), MONTH(Fecha), SUM(VENTAS), SUM(COSTO_ULT)
			FROM %s GROUP BY YEAR(Fecha), MONTH(Fecha)
			ORDER BY YEAR(Fecha), MONTH(Fecha)`, table))
		if err != nil {
			continue
		}
		var tV, tC float64
		for rows.Next() {
			var a, m int
			var v, c float64
			if err := rows.Scan(&a, &m, &v, &c); err != nil {
				continue
			}
			tV += v
			tC += c
			util := 0.0
			if v > 0 {
				util = (1 - c/v) * 100
			}
			hayDatos = true
			fmt.Printf("  %s %d/%02d  ventas=%14.2f  costo=%14.2f  utilidad=%.2f%%\n",
				codigo, a, m, v, c, util)
		}
		rows.Close()
		totV += tV
		totC += tC
	}
	if hayDatos {
		fmt.Println("  " + strings.Repeat("-", 62))
		util := 0.0
		if totV > 0 {
			util = (1 - totC/totV) * 100
		}
		fmt.Printf("  TOTAL:       ventas=%14.2f  costo=%14.2f  utilidad=%.2f%%\n", totV, totC, util)
	}
}

// ---------- Modo automático ----------

func procesarAutomaticoEstad(year int, mes int, cfg Config, headless bool) {
	if !headless {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("\n  *** PANIC ESTAD: %v ***\n", r)
			}
			fmt.Print("\nPresione Enter para salir...")
			leerLinea()
		}()
	}

	tStart := time.Now()
	logHistorial(fmt.Sprintf("ESTAD: Iniciado %d/%d", mes, year))

	db, err := ConectarSQL()
	if err != nil {
		fmt.Printf("ERROR SQL: %v\n", err)
		return
	}
	defer db.Close()

	if err := CrearBaseESTAD_Tiendas(db); err != nil {
		fmt.Printf("ERROR creando BD ESTAD_Tiendas: %v\n", err)
		return
	}

	dbEstad, err := ConectarSQL_ESTAD()
	if err != nil {
		fmt.Printf("ERROR conexion ESTAD_Tiendas: %v\n", err)
		return
	}
	defer dbEstad.Close()

	sucursales, err := ObtenerSucursales(db)
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
		return
	}

	ProcesarEstad(db, dbEstad, sucursales, year, mes, cfg.Modo, "AUTOMATICO", true)

	fmt.Printf("[%s] ESTAD Completado.\n", time.Now().Format("15:04:05"))
	logHistorial(fmt.Sprintf("ESTAD: Completado en %.1f min", time.Since(tStart).Minutes()))
}