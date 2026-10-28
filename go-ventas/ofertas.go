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

// ---------- Modelos ----------

type OfertaRegistro struct {
	NROITEM  string
	DESDE_1  float64
	DESDE_2  float64
	DESDE_3  float64
	HASTA_1  float64
	HASTA_2  float64
	HASTA_3  float64
	PVP_X1   string
	PVP_X2   string
	PVP_X3   string
	ES_DIA   bool
	DIA      int
	FORMULA  string
	FORMULA2 string
	FORMULA3 string
	FECHA_D  *time.Time
	FECHA_H  *time.Time
	HORA_D   string
	HORA_H   string
	DIAS     string
	TIPO_OFE int
	FECHAOFE *time.Time
	DETALLE  string
	AUTORIZA string
	GRUPLIN  string
	DPTO     string
	FAMI     string
	MOTIVO   string
}

// ---------- Rutas ----------

func rutaOferPath(s Sucursal, modo string) string {
	var bases []string
	if modo == "S" {
		bases = append(bases, `S:\aBC-Soft\Data\E001\Inv`)
		if s.Cia != "" {
			bases = append(bases, fmt.Sprintf(`S:\aBC-Soft\Data\E%s\Inv`, s.Cia))
		}
	} else {
		bases = append(bases, fmt.Sprintf(`\\%s\Sistema\aBC-Soft\Data\E001\Inv`, s.RutaIP))
		if s.Cia != "" {
			bases = append(bases, fmt.Sprintf(`\\%s\Sistema\aBC-Soft\Data\E%s\Inv`, s.RutaIP, s.Cia))
		}
	}

	for _, base := range bases {
		pLower := filepath.Join(base, "ofer001.DBF")
		if _, err := os.Stat(pLower); err == nil {
			return pLower
		}
		pUpper := filepath.Join(base, "OFER001.DBF")
		if _, err := os.Stat(pUpper); err == nil {
			return pUpper
		}
	}
	return filepath.Join(bases[0], "ofer001.DBF")
}

// ---------- SQL ----------

func CrearBaseOFERTAS_Tiendas(db *sql.DB) error {
	_, err := db.Exec(`IF DB_ID('OFERTAS_Tiendas') IS NULL BEGIN CREATE DATABASE OFERTAS_Tiendas END`)
	return err
}

func ConectarSQL_OFERTAS() (*sql.DB, error) {
	connStr := fmt.Sprintf("server=%s;database=OFERTAS_Tiendas;user id=%s;password=%s;encrypt=disable;connection timeout=30;timeout=900",
		SQLServer, SQLUser, SQLPassword)
	db, err := sql.Open("mssql", connStr)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(6)
	return db, nil
}

func CrearTablaOfertas(db *sql.DB, codigo string) error {
	table := "OFERTAS_" + codigo
	pkName := "PK_" + table

	_, err := db.Exec(fmt.Sprintf(`
		IF OBJECT_ID('%s','U') IS NULL
		CREATE TABLE %s (
			NROITEM    VARCHAR(15) NOT NULL,
			DESDE_1    DECIMAL(18,2) NULL,
			DESDE_2    DECIMAL(18,2) NULL,
			DESDE_3    DECIMAL(18,2) NULL,
			HASTA_1    DECIMAL(18,2) NULL,
			HASTA_2    DECIMAL(18,2) NULL,
			HASTA_3    DECIMAL(18,2) NULL,
			PVP_X1     VARCHAR(4) NULL,
			PVP_X2     VARCHAR(4) NULL,
			PVP_X3     VARCHAR(4) NULL,
			ES_DIA     BIT NULL,
			DIA        SMALLINT NULL,
			FORMULA    VARCHAR(100) NULL,
			FORMULA2   VARCHAR(100) NULL,
			FORMULA3   VARCHAR(100) NULL,
			FECHA_D    DATE NULL,
			FECHA_H    DATE NULL,
			HORA_D     VARCHAR(8) NULL,
			HORA_H     VARCHAR(8) NULL,
			DIAS       VARCHAR(7) NULL,
			TIPO_OFE   SMALLINT NULL,
			FECHAOFE   DATE NULL,
			DETALLE    VARCHAR(50) NULL,
			AUTORIZA   VARCHAR(50) NULL,
			GRUPLIN    VARCHAR(10) NULL,
			DPTO       VARCHAR(5) NULL,
			FAMI       VARCHAR(5) NULL,
			MOTIVO     VARCHAR(100) NULL,
			FechaCarga DATETIME DEFAULT GETDATE(),
			CONSTRAINT %s PRIMARY KEY (NROITEM)
		)`, table, table, pkName))
	if err != nil {
		return err
	}

	// Si la tabla ya existía sin Primary Key, eliminar posibles duplicados y agregar la PK
	_, _ = db.Exec(fmt.Sprintf(`
		IF OBJECT_ID('%s','PK') IS NULL
		BEGIN
			;WITH CTE AS (
				SELECT NROITEM, ROW_NUMBER() OVER (PARTITION BY NROITEM ORDER BY FechaCarga DESC) AS rn
				FROM %s
			)
			DELETE FROM CTE WHERE rn > 1;

			ALTER TABLE %s ALTER COLUMN NROITEM VARCHAR(15) NOT NULL;
			ALTER TABLE %s ADD CONSTRAINT %s PRIMARY KEY (NROITEM);
		END`, pkName, table, table, table, pkName))

	_, err = db.Exec(fmt.Sprintf(`
		IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE object_id = OBJECT_ID('%s') AND name = 'IX_%s_Item')
		CREATE NONCLUSTERED INDEX IX_%s_Item ON %s(FECHA_D, FECHA_H)`, table, table, table, table))
	return err
}

// ---------- Lectura ----------

func LeerArchivoOfertas(path string) ([]OfertaRegistro, error) {
	dbf, err := OpenDBF(path)
	if err != nil {
		return nil, err
	}
	defer dbf.Close()

	getS := func(rec []byte, name string) string {
		if i := dbf.FieldIndex(name); i >= 0 {
			f := dbf.Fields[i]
			if f.Type == 'V' {
				return strings.TrimSpace(dbf.GetVarChar(rec, f))
			}
			return strings.TrimSpace(dbf.GetString(rec, f))
		}
		return ""
	}

	getF := func(rec []byte, name string) float64 {
		if i := dbf.FieldIndex(name); i >= 0 {
			return dbf.GetFloat(rec, dbf.Fields[i])
		}
		return 0
	}

	getI := func(rec []byte, name string) int {
		s := getS(rec, name)
		if s == "" {
			return 0
		}
		var val int
		fmt.Sscanf(s, "%d", &val)
		return val
	}

	getB := func(rec []byte, name string) bool {
		if i := dbf.FieldIndex(name); i >= 0 {
			return dbf.GetBool(rec, dbf.Fields[i])
		}
		return false
	}

	getD := func(rec []byte, name string) *time.Time {
		if i := dbf.FieldIndex(name); i >= 0 {
			if d, ok := dbf.GetDate(rec, dbf.Fields[i]); ok && !d.IsZero() && d.Year() >= 1900 {
				return &d
			}
		}
		return nil
	}

	var regs []OfertaRegistro
	seen := make(map[string]bool)
	for i := 0; i < dbf.NumRecords; i++ {
		rec, e := dbf.ReadRecord(i)
		// Saltar registros eliminados o vacíos
		if e != nil || len(rec) == 0 || rec[0] == 0x2A {
			continue
		}

		item := getS(rec, "NROITEM")
		if item == "" {
			continue
		}

		// Evitar duplicados de producto
		if seen[item] {
			continue
		}
		seen[item] = true

		regs = append(regs, OfertaRegistro{
			NROITEM:  item,
			DESDE_1:  getF(rec, "DESDE_1"),
			DESDE_2:  getF(rec, "DESDE_2"),
			DESDE_3:  getF(rec, "DESDE_3"),
			HASTA_1:  getF(rec, "HASTA_1"),
			HASTA_2:  getF(rec, "HASTA_2"),
			HASTA_3:  getF(rec, "HASTA_3"),
			PVP_X1:   getS(rec, "PVP_X1"),
			PVP_X2:   getS(rec, "PVP_X2"),
			PVP_X3:   getS(rec, "PVP_X3"),
			ES_DIA:   getB(rec, "ES_DIA"),
			DIA:      getI(rec, "DIA"),
			FORMULA:  getS(rec, "FORMULA"),
			FORMULA2: getS(rec, "FORMULA2"),
			FORMULA3: getS(rec, "FORMULA3"),
			FECHA_D:  getD(rec, "FECHA_D"),
			FECHA_H:  getD(rec, "FECHA_H"),
			HORA_D:   getS(rec, "HORA_D"),
			HORA_H:   getS(rec, "HORA_H"),
			DIAS:     getS(rec, "DIAS"),
			TIPO_OFE: getI(rec, "TIPO_OFE"),
			FECHAOFE: getD(rec, "FECHAOFE"),
			DETALLE:  getS(rec, "DETALLE"),
			AUTORIZA: getS(rec, "AUTORIZA"),
			GRUPLIN:  getS(rec, "GRUPLIN"),
			DPTO:     getS(rec, "DPTO"),
			FAMI:     getS(rec, "FAMI"),
			MOTIVO:   getS(rec, "MOTIVO"),
		})
	}

	return regs, nil
}

// ---------- Inserción / Actualización ----------

func InsertarOfertas(db *sql.DB, codigo string, regs []OfertaRegistro) (int, error) {
	table := "OFERTAS_" + codigo

	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}

	// Al ser ofertas sincronizadas periódicamente (3 veces al día),
	// se limpia la tabla dentro de la transacción para mantener el estado fiel actual.
	if _, err := tx.Exec(fmt.Sprintf("TRUNCATE TABLE %s", table)); err != nil {
		// Fallback a DELETE si TRUNCATE falla por permisos
		if _, errDel := tx.Exec(fmt.Sprintf("DELETE FROM %s", table)); errDel != nil {
			tx.Rollback()
			return 0, fmt.Errorf("error limpiando tabla previa: %w", errDel)
		}
	}

	if len(regs) == 0 {
		return 0, tx.Commit()
	}

	const batchSize = 50
	cols := "NROITEM,DESDE_1,DESDE_2,DESDE_3,HASTA_1,HASTA_2,HASTA_3,PVP_X1,PVP_X2,PVP_X3,ES_DIA,DIA,FORMULA,FORMULA2,FORMULA3,FECHA_D,FECHA_H,HORA_D,HORA_H,DIAS,TIPO_OFE,FECHAOFE,DETALLE,AUTORIZA,GRUPLIN,DPTO,FAMI,MOTIVO"
	singlePH := "(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)"

	inserted := 0
	for i := 0; i < len(regs); i += batchSize {
		end := i + batchSize
		if end > len(regs) {
			end = len(regs)
		}
		chunk := regs[i:end]

		placeholders := make([]string, len(chunk))
		args := make([]interface{}, 0, len(chunk)*28)
		for j, r := range chunk {
			placeholders[j] = singlePH
			var fD, fH, fO interface{}
			if r.FECHA_D != nil {
				fD = *r.FECHA_D
			}
			if r.FECHA_H != nil {
				fH = *r.FECHA_H
			}
			if r.FECHAOFE != nil {
				fO = *r.FECHAOFE
			}

			args = append(args,
				r.NROITEM, r.DESDE_1, r.DESDE_2, r.DESDE_3, r.HASTA_1, r.HASTA_2, r.HASTA_3,
				r.PVP_X1, r.PVP_X2, r.PVP_X3, r.ES_DIA, r.DIA,
				r.FORMULA, r.FORMULA2, r.FORMULA3,
				fD, fH, r.HORA_D, r.HORA_H, r.DIAS,
				r.TIPO_OFE, fO, r.DETALLE, r.AUTORIZA,
				r.GRUPLIN, r.DPTO, r.FAMI, r.MOTIVO,
			)
		}

		query := fmt.Sprintf("INSERT INTO %s (%s) VALUES %s", table, cols, strings.Join(placeholders, ","))
		if _, err := tx.Exec(query, args...); err != nil {
			tx.Rollback()
			return inserted, fmt.Errorf("error insertando lote %d-%d: %w", i, end, err)
		}
		inserted += len(chunk)
	}

	if err := tx.Commit(); err != nil {
		return inserted, err
	}
	return inserted, nil
}

// ---------- Procesamiento ----------

func ProcesarTiendaOfertas(db *sql.DB, dbOfer *sql.DB, s Sucursal, modo string) (filas, insertados int, err error) {
	path := rutaOferPath(s, modo)

	if _, errStat := os.Stat(path); errStat != nil {
		return 0, 0, fmt.Errorf("archivo no accesible: %s", path)
	}

	if err := CrearTablaOfertas(dbOfer, s.Codigo); err != nil {
		return 0, 0, fmt.Errorf("error asegurando tabla: %w", err)
	}

	regs, errR := LeerArchivoOfertas(path)
	if errR != nil {
		return 0, 0, fmt.Errorf("error leyendo DBF: %w", errR)
	}
	filas = len(regs)

	ins, errIns := InsertarOfertas(dbOfer, s.Codigo, regs)
	if errIns != nil {
		return filas, ins, fmt.Errorf("error actualizando ofertas: %w", errIns)
	}
	return filas, ins, nil
}

func ProcesarOfertas(db *sql.DB, dbOfer *sql.DB, sucursales []Sucursal, modo, tipoEjecucion string) {
	tStart := time.Now()
	logInfo("OFERTAS: Iniciando %d tiendas, modo=%s", len(sucursales), modo)

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
					logError("OFERTAS: PANIC en %s: %v", sucursales[i].Codigo, r)
				}
			}()

			s := sucursales[i]
			mStart := time.Now()
			filas, insertados, err := ProcesarTiendaOfertas(db, dbOfer, s, modo)
			duracion := int(time.Since(mStart).Seconds())

			mu.Lock()
			defer mu.Unlock()

			if err != nil {
				logError("OFERTAS: %s FALLO: %v", s.Codigo, err)
				InsertarH_Tiendas(db, s.Codigo, time.Now().Year(), int(time.Now().Month()), 0, 0, 0, "ERROR", duracion, modo, tipoEjecucion, "OFERTAS")
				errGlobal++
				return
			}

			estado := "OK"
			if insertados == 0 {
				estado = "VACIO"
			}
			logInfo("OFERTAS: %s (%s) DBF=%d ofertas actualizadas=%d", s.Codigo, estado, filas, insertados)
			InsertarH_Tiendas(db, s.Codigo, time.Now().Year(), int(time.Now().Month()), filas, 0, insertados, estado, duracion, modo, tipoEjecucion, "OFERTAS")
		}()
	}

	wg.Wait()
	fmt.Printf("\n  OFERTAS listo. Tiempo: %.1f min. (%d errores)\n", time.Since(tStart).Minutes(), errGlobal)
	logInfo("OFERTAS: Completado en %.1f min, %d errores", time.Since(tStart).Minutes(), errGlobal)
}

func procesarAutomaticoOfertas(cfg Config, headless bool) {
	if !headless {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("\n  *** PANIC OFERTAS: %v ***\n", r)
			}
			fmt.Print("\nPresione Enter para salir...")
			leerLinea()
		}()
	}

	tStart := time.Now()
	logHistorial(fmt.Sprintf("OFERTAS: Iniciado (%s)", time.Now().Format("15:04")))

	db, err := ConectarSQL()
	if err != nil {
		fmt.Printf("ERROR SQL: %v\n", err)
		return
	}
	defer db.Close()

	if err := CrearBaseOFERTAS_Tiendas(db); err != nil {
		fmt.Printf("ERROR creando BD OFERTAS_Tiendas: %v\n", err)
		return
	}

	dbOfer, err := ConectarSQL_OFERTAS()
	if err != nil {
		fmt.Printf("ERROR conexion OFERTAS_Tiendas: %v\n", err)
		return
	}
	defer dbOfer.Close()

	sucursales, err := ObtenerSucursales(db)
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
		return
	}

	ProcesarOfertas(db, dbOfer, sucursales, cfg.Modo, "AUTOMATICO")

	fmt.Printf("[%s] OFERTAS Completado.\n", time.Now().Format("15:04:05"))
	logHistorial(fmt.Sprintf("OFERTAS: Completado en %.1f min", time.Since(tStart).Minutes()))
}
