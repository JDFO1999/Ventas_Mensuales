package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// ---------- Modelo ----------

type SubFamRegistro struct {
	SUBCODIGO string
	CODFAMI   string
	CODDEPTO  string
	NOMBRE    string
}

// ---------- Rutas ----------

func rutaSubFamPath(s Sucursal, modo string) string {
	var base string
	if modo == "S" {
		base = `S:\aBC-Soft\Data\E001\Inv`
	} else {
		base = fmt.Sprintf(`\\%s\Sistema\aBC-Soft\Data\E001\Inv`, s.RutaIP)
	}
	return filepath.Join(base, "SUBFAMI001.DBF")
}

// ---------- SQL ----------

func CrearBaseSUBFAM_Tiendas(db *sql.DB) error {
	_, err := db.Exec(`IF DB_ID('SUBFAM_Tiendas') IS NULL BEGIN CREATE DATABASE SUBFAM_Tiendas END`)
	return err
}

func ConectarSQL_SUBFAM() (*sql.DB, error) {
	connStr := fmt.Sprintf("server=%s;database=SUBFAM_Tiendas;user id=%s;password=%s;encrypt=disable;connection timeout=30;timeout=900",
		SQLServer, SQLUser, SQLPassword)
	db, err := sql.Open("mssql", connStr)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(6)
	return db, nil
}

func CrearTablaSubFam(db *sql.DB, codigo string) error {
	table := "SUBFAMI_" + codigo
	_, err := db.Exec(fmt.Sprintf(`
		IF OBJECT_ID('%s','U') IS NULL
		CREATE TABLE %s (
			SUBCODIGO VARCHAR(2) NOT NULL,
			CODFAMI   VARCHAR(2) NOT NULL,
			CODDEPTO  VARCHAR(2) NOT NULL,
			NOMBRE    VARCHAR(60) NULL,
			FechaCarga DATETIME DEFAULT GETDATE(),
			CONSTRAINT PK_%s PRIMARY KEY (SUBCODIGO, CODFAMI, CODDEPTO)
		)`, table, table, table))
	if err != nil {
		return err
	}
	_, err = db.Exec(fmt.Sprintf(`
		IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE object_id = OBJECT_ID('%s') AND name = 'IX_%s_Codigo')
		CREATE NONCLUSTERED INDEX IX_%s_Codigo ON %s(CODFAMI, CODDEPTO)`, table, table, table, table))
	return err
}

// ---------- Lectura ----------

func LeerArchivoSubFam(path string) ([]SubFamRegistro, int, error) {
	dbf, err := OpenDBF(path)
	if err != nil {
		return nil, 0, err
	}
	defer dbf.Close()

	getS := func(rec []byte, name string) string {
		if i := dbf.FieldIndex(name); i >= 0 {
			return dbf.GetString(rec, dbf.Fields[i])
		}
		return ""
	}

	var regs []SubFamRegistro
	invalid := 0
	for i := 0; i < dbf.NumRecords; i++ {
		rec, e := dbf.ReadRecord(i)
		if e != nil || len(rec) == 0 || rec[0] == 0x2A {
			continue
		}
		regs = append(regs, SubFamRegistro{
			SUBCODIGO: getS(rec, "SUBCODIGO"),
			CODFAMI:   getS(rec, "CODFAMI"),
			CODDEPTO:  getS(rec, "CODDEPTO"),
			NOMBRE:    getS(rec, "NOMBRE"),
		})
	}
	return regs, invalid, nil
}

// ---------- Inserción ----------

func InsertarSubFam(db *sql.DB, codigo string, regs []SubFamRegistro) (int, error) {
	if len(regs) == 0 {
		return 0, nil
	}
	table := "SUBFAMI_" + codigo
	query := fmt.Sprintf(`INSERT INTO %s (SUBCODIGO, CODFAMI, CODDEPTO, NOMBRE)
		VALUES (?,?,?,?)`, table)

	const chunkSize = 5000
	inserted := 0
	for i := 0; i < len(regs); i += chunkSize {
		end := i + chunkSize
		if end > len(regs) {
			end = len(regs)
		}
		chunk := regs[i:end]
		tx, err := db.Begin()
		if err != nil {
			return inserted, err
		}
		for _, r := range chunk {
			_, err = tx.Exec(query, r.SUBCODIGO, r.CODFAMI, r.CODDEPTO, r.NOMBRE)
			if err != nil {
				tx.Rollback()
				return inserted, err
			}
			inserted++
		}
		if err := tx.Commit(); err != nil {
			return inserted, err
		}
	}
	return inserted, nil
}

// ---------- Procesamiento ----------

func ProcesarTiendaSubFam(db *sql.DB, dbSub *sql.DB, s Sucursal, modo string) (filas, insertados int, err error) {
	path := rutaSubFamPath(s, modo)

	if _, errStat := os.Stat(path); errStat != nil {
		return 0, 0, fmt.Errorf("archivo no accesible: %s", path)
	}

	if err := CrearTablaSubFam(dbSub, s.Codigo); err != nil {
		return 0, 0, err
	}

	regs, _, errR := LeerArchivoSubFam(path)
	if errR != nil {
		return 0, 0, errR
	}
	filas = len(regs)

	// dedupe por clave natural
	existing := make(map[string]bool)
	rows, errQ := dbSub.Query(fmt.Sprintf(
		"SELECT SUBCODIGO, CODFAMI, CODDEPTO FROM %s", "SUBFAMI_"+s.Codigo))
	if errQ == nil {
		for rows.Next() {
			var sc, cf, cd string
			if rows.Scan(&sc, &cf, &cd) == nil {
				existing[strings.TrimSpace(sc)+"|"+strings.TrimSpace(cf)+"|"+strings.TrimSpace(cd)] = true
			}
		}
		rows.Close()
	}

	var nuevos []SubFamRegistro
	for _, r := range regs {
		key := strings.TrimSpace(r.SUBCODIGO) + "|" + strings.TrimSpace(r.CODFAMI) + "|" + strings.TrimSpace(r.CODDEPTO)
		if !existing[key] {
			nuevos = append(nuevos, r)
		}
	}

	if len(nuevos) == 0 {
		return filas, 0, nil // todo ya cargado
	}

	ins, errIns := InsertarSubFam(dbSub, s.Codigo, nuevos)
	if errIns != nil {
		return filas, ins, errIns
	}
	return filas, ins, nil
}

func ProcesarSubFam(db *sql.DB, dbSub *sql.DB, sucursales []Sucursal, modo, tipoEjecucion string) {
	tStart := time.Now()
	logInfo("SUBFAM: Iniciando %d tiendas, modo=%s", len(sucursales), modo)

	CrearTablaH_Tiendas(db)

	var wg sync.WaitGroup
	errGlobal := 0
	var mu sync.Mutex
	sem := make(chan struct{}, 4)

	for i := range sucursales {
		i := i
		sem <- struct{}{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			defer func() {
				if r := recover(); r != nil {
					logError("SUBFAM: PANIC en %s: %v", sucursales[i].Codigo, r)
				}
			}()

			s := sucursales[i]
			mStart := time.Now()
			filas, insertados, err := ProcesarTiendaSubFam(db, dbSub, s, modo)
			duracion := int(time.Since(mStart).Seconds())

			mu.Lock()
			defer mu.Unlock()

			if err != nil {
				logError("SUBFAM: %s FALLO: %v", s.Codigo, err)
				InsertarH_Tiendas(db, s.Codigo, time.Now().Year(), 0, 0, 0, 0, "ERROR", duracion, modo, tipoEjecucion, "SUBFAM")
				errGlobal++
				return
			}

			estado := "OK"
			if insertados == 0 {
				estado = "SALTADO"
			}
			logInfo("SUBFAM: %s (%s) DBF=%d insertados=%d", s.Codigo, estado, filas, insertados)
			InsertarH_Tiendas(db, s.Codigo, time.Now().Year(), 0, filas, filas-insertados, insertados, estado, duracion, modo, tipoEjecucion, "SUBFAM")
		}()
	}

	wg.Wait()
	fmt.Printf("\n  SUBFAM listo. Tiempo: %.1f min. (%d errores)\n", time.Since(tStart).Minutes(), errGlobal)
	logInfo("SUBFAM: Completado en %.1f min, %d errores", time.Since(tStart).Minutes(), errGlobal)
}

func procesarAutomaticoSubFam(cfg Config, headless bool) {
	if !headless {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("\n  *** PANIC SUBFAM: %v ***\n", r)
			}
			fmt.Print("\nPresione Enter para salir...")
			leerLinea()
		}()
	}

	tStart := time.Now()
	logHistorial(fmt.Sprintf("SUBFAM: Iniciado %d", time.Now().Year()))

	db, err := ConectarSQL()
	if err != nil {
		fmt.Printf("ERROR SQL: %v\n", err)
		return
	}
	defer db.Close()

	if err := CrearBaseSUBFAM_Tiendas(db); err != nil {
		fmt.Printf("ERROR creando BD SUBFAM_Tiendas: %v\n", err)
		return
	}

	dbSub, err := ConectarSQL_SUBFAM()
	if err != nil {
		fmt.Printf("ERROR conexion SUBFAM_Tiendas: %v\n", err)
		return
	}
	defer dbSub.Close()

	sucursales, err := ObtenerSucursales(db)
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
		return
	}

	ProcesarSubFam(db, dbSub, sucursales, cfg.Modo, "AUTOMATICO")

	fmt.Printf("[%s] SUBFAM Completado.\n", time.Now().Format("15:04:05"))
	logHistorial(fmt.Sprintf("SUBFAM: Completado en %.1f min", time.Since(tStart).Minutes()))
}