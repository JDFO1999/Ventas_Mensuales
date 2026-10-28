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

type RCPCabeceraRegistro struct {
	NROPED     string
	NUMERO     string
	NROMOV     string
	FECHAPE    time.Time
	CREDITO    bool
	CODPROV    string
	NOMBRE     string
	DIRECCION  string
	TELEFONO   string
	RIF        string
	NIT        string
	ANULADA    bool
	TIPOORD    string
	STATUS     string
	CONDICION  string
	FACDEV     string
	SUBTOTAL   float64
	HDESCUEN   float64
	HIVA       float64
	SUBTOTALDL float64
	HDESCUENDL float64
	HIVADL     float64
	MONTOSEG   float64
	FLETES     float64
	FLETEEXT   float64
	GASTOSVAR  float64
	ADUANA     float64
	PORGASTOS  float64
	PORSEGURO  float64
	PORADUANA  float64
	NPORCFLETE float64
	NPRINCIPAL float64
	NOTROS     float64
	NPPAGO     float64
	FECHAFAC   time.Time
	FECHAVEN   time.Time
	FECHARE    time.Time
	NROFAC     string
	NROCON     string
	TIPOCOMPRA string
	OPERADOR   string
	TASA       float64
	TASA_V     float64
	IDMONEDA   float64
	NIGTF      float64
	NIGTFDL    float64
	COMPRADOR  string
	TASA_COP   float64
	EXENTA     float64
	DEDUC      float64
}

type RCPLineaRegistro struct {
	NROPED     string
	NUMERO     string
	CODPROV    string
	NROITEM    string
	GRUPLIN    string
	DESCRIP    string
	PRECIOC    float64
	PRECIOUNI  float64
	COSTO_LIN  float64
	UTIL_PVP1  float64
	UTIL_PVP2  float64
	UTIL_PVP3  float64
	PVP1       float64
	PVP2       float64
	PVP3       float64
	PRECIOC_DL float64
	COS_LIN_DL float64
	PVP1_DOL   float64
	DSCTO_1    float64
	DSCTO_2    float64
	DSCTO_3    float64
	CANTIDAP   float64
	CANTIDAR   float64
	CANTIDALL  float64
	CANTIDAV   float64
	FECHAFAC   time.Time
	FECHARE    time.Time
	NROFACT    string
	UNIDAD     string
	EMPAQUE    float64
	PORIVA     float64
	LOCALIZ    string
	COS_DOL    float64
	FACTOR     float64
	FACTOR_V   float64
	FECCADUCA  time.Time
	FACTOR_VTA float64
	COS_DOL1   float64
	COS_DOL2   float64
	COS_DOL3   float64
	UTIL_DOL1  float64
	UTIL_DOL2  float64
	UTIL_DOL3  float64
	FLETE      float64
	COS_DOLP   float64
	PRECIOC_CP float64
	COS_LIN_CP float64
	PVP1_COP   float64
	PVP2_COP   float64
	PVP3_COP   float64
	FACTOR_COP float64
	NMAYOR     float64
	PRECIOFULL float64
}

// ---------- Rutas ----------

func rutaRCPPaths(s Sucursal, modo string) (string, string) {
	var base string
	if modo == "S" {
		base = `S:\aBC-Soft\Data\E001\Inv`
	} else {
		base = fmt.Sprintf(`\\%s\Sistema\aBC-Soft\Data\E001\Inv`, s.RutaIP)
	}
	mrcp := filepath.Join(base, "MRCP001.DBF")
	lrcp := filepath.Join(base, "LRCP001.DBF")
	return mrcp, lrcp
}

// ---------- SQL ----------

func CrearBaseRCP_Tiendas(db *sql.DB) error {
	_, err := db.Exec(`IF DB_ID('RCP_Tiendas') IS NULL BEGIN CREATE DATABASE RCP_Tiendas END`)
	return err
}

func ConectarSQL_RCP() (*sql.DB, error) {
	connStr := fmt.Sprintf("server=%s;database=RCP_Tiendas;user id=%s;password=%s;encrypt=disable;connection timeout=30;timeout=900",
		SQLServer, SQLUser, SQLPassword)
	db, err := sql.Open("mssql", connStr)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(6)
	return db, nil
}

func CrearTablasRCP(db *sql.DB, codigo string) error {
	mTable := "MRCP_" + codigo
	lTable := "LRCP_" + codigo

	// Maestro
	_, err := db.Exec(fmt.Sprintf(`
		IF OBJECT_ID('%s','U') IS NULL
		CREATE TABLE %s (
			NROPED     VARCHAR(8) NULL,
			NUMERO     VARCHAR(8) NOT NULL,
			NROMOV     VARCHAR(8) NULL,
			FECHAPE    DATE NULL,
			CREDITO    BIT NULL,
			CODPROV    VARCHAR(10) NULL,
			NOMBRE     VARCHAR(120) NULL,
			DIRECCION  VARCHAR(200) NULL,
			TELEFONO   VARCHAR(30) NULL,
			RIF        VARCHAR(15) NULL,
			NIT        VARCHAR(15) NULL,
			ANULADA    BIT NULL,
			TIPOORD    VARCHAR(2) NULL,
			STATUS     VARCHAR(1) NULL,
			CONDICION  VARCHAR(30) NULL,
			FACDEV     VARCHAR(1) NULL,
			SUBTOTAL   DECIMAL(18,2) NULL,
			HDESCUEN   DECIMAL(18,2) NULL,
			HIVA       DECIMAL(18,2) NULL,
			SUBTOTALDL DECIMAL(18,2) NULL,
			HDESCUENDL DECIMAL(18,2) NULL,
			HIVADL     DECIMAL(18,2) NULL,
			MONTOSEG   DECIMAL(18,4) NULL,
			FLETES     DECIMAL(18,4) NULL,
			FLETEEXT   DECIMAL(18,4) NULL,
			GASTOSVAR  DECIMAL(18,4) NULL,
			ADUANA     DECIMAL(18,4) NULL,
			PORGASTOS  DECIMAL(18,8) NULL,
			PORSEGURO  DECIMAL(18,8) NULL,
			PORADUANA  DECIMAL(18,8) NULL,
			NPORCFLETE DECIMAL(18,8) NULL,
			NPRINCIPAL DECIMAL(18,8) NULL,
			NOTROS     DECIMAL(18,8) NULL,
			NPPAGO     DECIMAL(18,8) NULL,
			FECHAFAC   DATE NULL,
			FECHAVEN   DATE NULL,
			FECHARE    DATE NULL,
			NROFAC     VARCHAR(25) NULL,
			NROCON     VARCHAR(14) NULL,
			TIPOCOMPRA VARCHAR(2) NULL,
			OPERADOR   VARCHAR(20) NULL,
			TASA       DECIMAL(18,6) NULL,
			TASA_V     DECIMAL(18,6) NULL,
			IDMONEDA   DECIMAL(10,0) NULL,
			NIGTF      DECIMAL(18,2) NULL,
			NIGTFDL    DECIMAL(18,2) NULL,
			COMPRADOR  VARCHAR(6) NULL,
			TASA_COP   DECIMAL(18,6) NULL,
			EXENTA     DECIMAL(18,4) NULL,
			DEDUC      DECIMAL(18,2) NULL,
			FechaCarga DATETIME DEFAULT GETDATE(),
			CONSTRAINT PK_%s PRIMARY KEY (NUMERO)
		)`, mTable, mTable, mTable))
	if err != nil {
		return err
	}

	// Detalle
	_, err = db.Exec(fmt.Sprintf(`
		IF OBJECT_ID('%s','U') IS NULL
		CREATE TABLE %s (
			NROPED     VARCHAR(8) NULL,
			NUMERO     VARCHAR(8) NOT NULL,
			CODPROV    VARCHAR(10) NULL,
			NROITEM    VARCHAR(15) NOT NULL,
			GRUPLIN    VARCHAR(6) NULL,
			DESCRIP    VARCHAR(120) NULL,
			PRECIOC    DECIMAL(18,4) NULL,
			PRECIOUNI  DECIMAL(18,4) NULL,
			COSTO_LIN  DECIMAL(18,4) NULL,
			UTIL_PVP1  DECIMAL(18,2) NULL,
			UTIL_PVP2  DECIMAL(18,2) NULL,
			UTIL_PVP3  DECIMAL(18,2) NULL,
			PVP1       DECIMAL(18,4) NULL,
			PVP2       DECIMAL(18,4) NULL,
			PVP3       DECIMAL(18,4) NULL,
			PRECIOC_DL DECIMAL(18,4) NULL,
			COS_LIN_DL DECIMAL(18,4) NULL,
			PVP1_DOL   DECIMAL(18,3) NULL,
			DSCTO_1    DECIMAL(18,2) NULL,
			DSCTO_2    DECIMAL(18,2) NULL,
			DSCTO_3    DECIMAL(18,2) NULL,
			CANTIDAP   DECIMAL(18,4) NULL,
			CANTIDAR   DECIMAL(18,4) NULL,
			CANTIDALL  DECIMAL(18,4) NULL,
			CANTIDAV   DECIMAL(18,4) NULL,
			FECHAFAC   DATE NULL,
			FECHARE    DATE NULL,
			NROFACT    VARCHAR(25) NULL,
			UNIDAD     VARCHAR(3) NULL,
			EMPAQUE    DECIMAL(18,2) NULL,
			PORIVA     DECIMAL(18,2) NULL,
			LOCALIZ    VARCHAR(11) NULL,
			COS_DOL    DECIMAL(20,8) NULL,
			FACTOR     DECIMAL(18,6) NULL,
			FACTOR_V   DECIMAL(18,6) NULL,
			FECCADUCA  DATE NULL,
			FACTOR_VTA DECIMAL(18,4) NULL,
			COS_DOL1   DECIMAL(18,4) NULL,
			COS_DOL2   DECIMAL(18,4) NULL,
			COS_DOL3   DECIMAL(18,4) NULL,
			UTIL_DOL1  DECIMAL(18,4) NULL,
			UTIL_DOL2  DECIMAL(18,4) NULL,
			UTIL_DOL3  DECIMAL(18,4) NULL,
			FLETE      DECIMAL(18,4) NULL,
			COS_DOLP   DECIMAL(20,8) NULL,
			PRECIOC_CP DECIMAL(18,4) NULL,
			COS_LIN_CP DECIMAL(18,2) NULL,
			PVP1_COP   DECIMAL(18,3) NULL,
			PVP2_COP   DECIMAL(18,3) NULL,
			PVP3_COP   DECIMAL(18,3) NULL,
			FACTOR_COP DECIMAL(18,6) NULL,
			NMAYOR     DECIMAL(18,3) NULL,
			PRECIOFULL DECIMAL(18,4) NULL,
			FechaCarga DATETIME DEFAULT GETDATE()
		)`, lTable, lTable))
	if err != nil {
		return err
	}

	_, err = db.Exec(fmt.Sprintf(`
		IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE object_id = OBJECT_ID('%s') AND name = 'IX_%s_Numero')
		CREATE NONCLUSTERED INDEX IX_%s_Numero ON %s(NUMERO)`, lTable, lTable, lTable, lTable))
	if err != nil {
		return err
	}

	return nil
}

// ---------- Lectura ----------

func LeerArchivoMRCP(path string, year int) ([]RCPCabeceraRegistro, int, error) {
	dbf, err := OpenDBF(path)
	if err != nil {
		return nil, 0, err
	}
	defer dbf.Close()

	fiFac := dbf.FieldIndex("FECHAFAC")
	if fiFac < 0 {
		return nil, 0, fmt.Errorf("campo FECHAFAC no encontrado en %s", path)
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
	getD := func(rec []byte, name string) time.Time {
		if i := dbf.FieldIndex(name); i >= 0 {
			d, _ := dbf.GetDate(rec, dbf.Fields[i])
			return d
		}
		return time.Time{}
	}
	getB := func(rec []byte, name string) bool {
		if i := dbf.FieldIndex(name); i >= 0 {
			return dbf.GetBool(rec, dbf.Fields[i])
		}
		return false
	}

	var regs []RCPCabeceraRegistro
	invalid := 0
	for i := 0; i < dbf.NumRecords; i++ {
		rec, e := dbf.ReadRecord(i)
		if e != nil || len(rec) == 0 || rec[0] == 0x2A {
			continue
		}

		fechaFac, ok := dbf.GetDate(rec, dbf.Fields[fiFac])
		if !ok || fechaFac.Year() != year {
			continue
		}

		regs = append(regs, RCPCabeceraRegistro{
			NROPED:     getS(rec, "NROPED"),
			NUMERO:     getS(rec, "NUMERO"),
			NROMOV:     getS(rec, "NROMOV"),
			FECHAPE:    getD(rec, "FECHAPE"),
			CREDITO:    getB(rec, "CREDITO"),
			CODPROV:    getS(rec, "CODPROV"),
			NOMBRE:     getS(rec, "NOMBRE"),
			DIRECCION:  getS(rec, "DIRECCION"),
			TELEFONO:   getS(rec, "TELEFONO"),
			RIF:        getS(rec, "RIF"),
			NIT:        getS(rec, "NIT"),
			ANULADA:    getB(rec, "ANULADA"),
			TIPOORD:    getS(rec, "TIPOORD"),
			STATUS:     getS(rec, "STATUS"),
			CONDICION:  getS(rec, "CONDICION"),
			FACDEV:     getS(rec, "FACDEV"),
			SUBTOTAL:   getF(rec, "SUBTOTAL"),
			HDESCUEN:   getF(rec, "HDESCUEN"),
			HIVA:       getF(rec, "HIVA"),
			SUBTOTALDL: getF(rec, "SUBTOTALDL"),
			HDESCUENDL: getF(rec, "HDESCUENDL"),
			HIVADL:     getF(rec, "HIVADL"),
			MONTOSEG:   getF(rec, "MONTOSEG"),
			FLETES:     getF(rec, "FLETES"),
			FLETEEXT:   getF(rec, "FLETEEXT"),
			GASTOSVAR:  getF(rec, "GASTOSVAR"),
			ADUANA:     getF(rec, "ADUANA"),
			PORGASTOS:  getF(rec, "PORGASTOS"),
			PORSEGURO:  getF(rec, "PORSEGURO"),
			PORADUANA:  getF(rec, "PORADUANA"),
			NPORCFLETE: getF(rec, "NPORCFLETE"),
			NPRINCIPAL: getF(rec, "NPRINCIPAL"),
			NOTROS:     getF(rec, "NOTROS"),
			NPPAGO:     getF(rec, "NPPAGO"),
			FECHAFAC:   fechaFac,
			FECHAVEN:   getD(rec, "FECHAVEN"),
			FECHARE:    getD(rec, "FECHARE"),
			NROFAC:     getS(rec, "NROFAC"),
			NROCON:     getS(rec, "NROCON"),
			TIPOCOMPRA: getS(rec, "TIPOCOMPRA"),
			OPERADOR:   getS(rec, "OPERADOR"),
			TASA:       getF(rec, "TASA"),
			TASA_V:     getF(rec, "TASA_V"),
			IDMONEDA:   getF(rec, "IDMONEDA"),
			NIGTF:      getF(rec, "NIGTF"),
			NIGTFDL:    getF(rec, "NIGTFDL"),
			COMPRADOR:  getS(rec, "COMPRADOR"),
			TASA_COP:   getF(rec, "TASA_COP"),
			EXENTA:     getF(rec, "EXENTA"),
			DEDUC:      getF(rec, "DEDUC"),
		})
	}
	return regs, invalid, nil
}

func LeerArchivoLRCP(path string, year int) ([]RCPLineaRegistro, int, error) {
	dbf, err := OpenDBF(path)
	if err != nil {
		return nil, 0, err
	}
	defer dbf.Close()

	fiFac := dbf.FieldIndex("FECHAFAC")
	if fiFac < 0 {
		return nil, 0, fmt.Errorf("campo FECHAFAC no encontrado en %s", path)
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
	getD := func(rec []byte, name string) time.Time {
		if i := dbf.FieldIndex(name); i >= 0 {
			d, _ := dbf.GetDate(rec, dbf.Fields[i])
			return d
		}
		return time.Time{}
	}

	var regs []RCPLineaRegistro
	invalid := 0
	for i := 0; i < dbf.NumRecords; i++ {
		rec, e := dbf.ReadRecord(i)
		if e != nil || len(rec) == 0 || rec[0] == 0x2A {
			continue
		}

		fechaFac, ok := dbf.GetDate(rec, dbf.Fields[fiFac])
		if !ok || fechaFac.Year() != year {
			continue
		}

		regs = append(regs, RCPLineaRegistro{
			NROPED:     getS(rec, "NROPED"),
			NUMERO:     getS(rec, "NUMERO"),
			CODPROV:    getS(rec, "CODPROV"),
			NROITEM:    getS(rec, "NROITEM"),
			GRUPLIN:    getS(rec, "GRUPLIN"),
			DESCRIP:    getS(rec, "DESCRIP"),
			PRECIOC:    getF(rec, "PRECIOC"),
			PRECIOUNI:  getF(rec, "PRECIOUNI"),
			COSTO_LIN:  getF(rec, "COSTO_LIN"),
			UTIL_PVP1:  getF(rec, "UTIL_PVP1"),
			UTIL_PVP2:  getF(rec, "UTIL_PVP2"),
			UTIL_PVP3:  getF(rec, "UTIL_PVP3"),
			PVP1:       getF(rec, "PVP1"),
			PVP2:       getF(rec, "PVP2"),
			PVP3:       getF(rec, "PVP3"),
			PRECIOC_DL: getF(rec, "PRECIOC_DL"),
			COS_LIN_DL: getF(rec, "COS_LIN_DL"),
			PVP1_DOL:   getF(rec, "PVP1_DOL"),
			DSCTO_1:    getF(rec, "DSCTO_1"),
			DSCTO_2:    getF(rec, "DSCTO_2"),
			DSCTO_3:    getF(rec, "DSCTO_3"),
			CANTIDAP:   getF(rec, "CANTIDAP"),
			CANTIDAR:   getF(rec, "CANTIDAR"),
			CANTIDALL:  getF(rec, "CANTIDALL"),
			CANTIDAV:   getF(rec, "CANTIDAV"),
			FECHAFAC:   fechaFac,
			FECHARE:    getD(rec, "FECHARE"),
			NROFACT:    getS(rec, "NROFACT"),
			UNIDAD:     getS(rec, "UNIDAD"),
			EMPAQUE:    getF(rec, "EMPAQUE"),
			PORIVA:     getF(rec, "PORIVA"),
			LOCALIZ:    getS(rec, "LOCALIZ"),
			COS_DOL:    getF(rec, "COS_DOL"),
			FACTOR:     getF(rec, "FACTOR"),
			FACTOR_V:   getF(rec, "FACTOR_V"),
			FECCADUCA:  getD(rec, "FECCADUCA"),
			FACTOR_VTA: getF(rec, "FACTOR_VTA"),
			COS_DOL1:   getF(rec, "COS_DOL1"),
			COS_DOL2:   getF(rec, "COS_DOL2"),
			COS_DOL3:   getF(rec, "COS_DOL3"),
			UTIL_DOL1:  getF(rec, "UTIL_DOL1"),
			UTIL_DOL2:  getF(rec, "UTIL_DOL2"),
			UTIL_DOL3:  getF(rec, "UTIL_DOL3"),
			FLETE:      getF(rec, "FLETE"),
			COS_DOLP:   getF(rec, "COS_DOLP"),
			PRECIOC_CP: getF(rec, "PRECIOC_CP"),
			COS_LIN_CP: getF(rec, "COS_LIN_CP"),
			PVP1_COP:   getF(rec, "PVP1_COP"),
			PVP2_COP:   getF(rec, "PVP2_COP"),
			PVP3_COP:   getF(rec, "PVP3_COP"),
			FACTOR_COP: getF(rec, "FACTOR_COP"),
			NMAYOR:     getF(rec, "NMAYOR"),
			PRECIOFULL: getF(rec, "PRECIOFULL"),
		})
	}
	return regs, invalid, nil
}

// ---------- Inserción ----------

func InsertarMRCP(db *sql.DB, codigo string, regs []RCPCabeceraRegistro) (int, error) {
	if len(regs) == 0 {
		return 0, nil
	}
	table := "MRCP_" + codigo
	query := fmt.Sprintf(`INSERT INTO %s (NROPED, NUMERO, NROMOV, FECHAPE, CREDITO, CODPROV, NOMBRE, DIRECCION, TELEFONO, RIF, NIT, ANULADA, TIPOORD, STATUS, CONDICION, FACDEV, SUBTOTAL, HDESCUEN, HIVA, SUBTOTALDL, HDESCUENDL, HIVADL, MONTOSEG, FLETES, FLETEEXT, GASTOSVAR, ADUANA, PORGASTOS, PORSEGURO, PORADUANA, NPORCFLETE, NPRINCIPAL, NOTROS, NPPAGO, FECHAFAC, FECHAVEN, FECHARE, NROFAC, NROCON, TIPOCOMPRA, OPERADOR, TASA, TASA_V, IDMONEDA, NIGTF, NIGTFDL, COMPRADOR, TASA_COP, EXENTA, DEDUC)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, table)

	const chunkSize = 2000
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
			_, err = tx.Exec(query, r.NROPED, r.NUMERO, r.NROMOV, r.FECHAPE, r.CREDITO, r.CODPROV, r.NOMBRE, r.DIRECCION, r.TELEFONO, r.RIF, r.NIT, r.ANULADA, r.TIPOORD, r.STATUS, r.CONDICION, r.FACDEV, r.SUBTOTAL, r.HDESCUEN, r.HIVA, r.SUBTOTALDL, r.HDESCUENDL, r.HIVADL, r.MONTOSEG, r.FLETES, r.FLETEEXT, r.GASTOSVAR, r.ADUANA, r.PORGASTOS, r.PORSEGURO, r.PORADUANA, r.NPORCFLETE, r.NPRINCIPAL, r.NOTROS, r.NPPAGO, r.FECHAFAC, r.FECHAVEN, r.FECHARE, r.NROFAC, r.NROCON, r.TIPOCOMPRA, r.OPERADOR, r.TASA, r.TASA_V, r.IDMONEDA, r.NIGTF, r.NIGTFDL, r.COMPRADOR, r.TASA_COP, r.EXENTA, r.DEDUC)
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

func InsertarLRCP(db *sql.DB, codigo string, regs []RCPLineaRegistro) (int, error) {
	if len(regs) == 0 {
		return 0, nil
	}
	table := "LRCP_" + codigo
	query := fmt.Sprintf(`INSERT INTO %s (NROPED, NUMERO, CODPROV, NROITEM, GRUPLIN, DESCRIP, PRECIOC, PRECIOUNI, COSTO_LIN, UTIL_PVP1, UTIL_PVP2, UTIL_PVP3, PVP1, PVP2, PVP3, PRECIOC_DL, COS_LIN_DL, PVP1_DOL, DSCTO_1, DSCTO_2, DSCTO_3, CANTIDAP, CANTIDAR, CANTIDALL, CANTIDAV, FECHAFAC, FECHARE, NROFACT, UNIDAD, EMPAQUE, PORIVA, LOCALIZ, COS_DOL, FACTOR, FACTOR_V, FECCADUCA, FACTOR_VTA, COS_DOL1, COS_DOL2, COS_DOL3, UTIL_DOL1, UTIL_DOL2, UTIL_DOL3, FLETE, COS_DOLP, PRECIOC_CP, COS_LIN_CP, PVP1_COP, PVP2_COP, PVP3_COP, FACTOR_COP, NMAYOR, PRECIOFULL)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, table)

	const chunkSize = 2000
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
			_, err = tx.Exec(query, r.NROPED, r.NUMERO, r.CODPROV, r.NROITEM, r.GRUPLIN, r.DESCRIP, r.PRECIOC, r.PRECIOUNI, r.COSTO_LIN, r.UTIL_PVP1, r.UTIL_PVP2, r.UTIL_PVP3, r.PVP1, r.PVP2, r.PVP3, r.PRECIOC_DL, r.COS_LIN_DL, r.PVP1_DOL, r.DSCTO_1, r.DSCTO_2, r.DSCTO_3, r.CANTIDAP, r.CANTIDAR, r.CANTIDALL, r.CANTIDAV, r.FECHAFAC, r.FECHARE, r.NROFACT, r.UNIDAD, r.EMPAQUE, r.PORIVA, r.LOCALIZ, r.COS_DOL, r.FACTOR, r.FACTOR_V, r.FECCADUCA, r.FACTOR_VTA, r.COS_DOL1, r.COS_DOL2, r.COS_DOL3, r.UTIL_DOL1, r.UTIL_DOL2, r.UTIL_DOL3, r.FLETE, r.COS_DOLP, r.PRECIOC_CP, r.COS_LIN_CP, r.PVP1_COP, r.PVP2_COP, r.PVP3_COP, r.FACTOR_COP, r.NMAYOR, r.PRECIOFULL)
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

func ProcesarTiendaRCP(db *sql.DB, dbRCP *sql.DB, s Sucursal, year int, modo string) (mF, mIns, lF, lIns int, err error) {
	mPath, lPath := rutaRCPPaths(s, modo)

	if _, errStat := os.Stat(mPath); errStat != nil {
		return 0, 0, 0, 0, fmt.Errorf("MRCP no accesible: %s", mPath)
	}
	if _, errStat := os.Stat(lPath); errStat != nil {
		return 0, 0, 0, 0, fmt.Errorf("LRCP no accesible: %s", lPath)
	}

	if err := CrearTablasRCP(dbRCP, s.Codigo); err != nil {
		return 0, 0, 0, 0, err
	}

	// 1. Leer MRCP
	mRegs, _, errM := LeerArchivoMRCP(mPath, year)
	if errM != nil {
		return 0, 0, 0, 0, errM
	}
	mF = len(mRegs)

	// Dedupe MRCP (NUMERO)
	mExisting := make(map[string]bool)
	mRows, _ := dbRCP.Query(fmt.Sprintf("SELECT NUMERO FROM MRCP_%s WHERE YEAR(FECHAFAC)=?", s.Codigo), year)
	if mRows != nil {
		for mRows.Next() {
			var n string
			if mRows.Scan(&n) == nil {
				mExisting[strings.TrimSpace(n)] = true
			}
		}
		mRows.Close()
	}
	var mNuevos []RCPCabeceraRegistro
	for _, r := range mRegs {
		if !mExisting[strings.TrimSpace(r.NUMERO)] {
			mNuevos = append(mNuevos, r)
		}
	}
	if len(mNuevos) > 0 {
		mIns, err = InsertarMRCP(dbRCP, s.Codigo, mNuevos)
		if err != nil {
			return mF, mIns, 0, 0, err
		}
	}

	// 2. Leer LRCP
	lRegs, _, errL := LeerArchivoLRCP(lPath, year)
	if errL != nil {
		return mF, mIns, 0, 0, errL
	}
	lF = len(lRegs)

	// Dedupe LRCP (NUMERO + NROITEM)
	lExisting := make(map[string]bool)
	lRows, _ := dbRCP.Query(fmt.Sprintf("SELECT NUMERO, NROITEM FROM LRCP_%s WHERE YEAR(FECHAFAC)=?", s.Codigo), year)
	if lRows != nil {
		for lRows.Next() {
			var n, it string
			if lRows.Scan(&n, &it) == nil {
				lExisting[strings.TrimSpace(n)+"|"+strings.TrimSpace(it)] = true
			}
		}
		lRows.Close()
	}
	var lNuevos []RCPLineaRegistro
	for _, r := range lRegs {
		key := strings.TrimSpace(r.NUMERO) + "|" + strings.TrimSpace(r.NROITEM)
		if !lExisting[key] {
			lNuevos = append(lNuevos, r)
		}
	}
	if len(lNuevos) > 0 {
		lIns, err = InsertarLRCP(dbRCP, s.Codigo, lNuevos)
		if err != nil {
			return mF, mIns, lF, lIns, err
		}
	}

	return mF, mIns, lF, lIns, nil
}

func ProcesarRCP(db *sql.DB, dbRCP *sql.DB, sucursales []Sucursal, year int, modo, tipoEjecucion string) {
	tStart := time.Now()
	logInfo("RCP: Iniciando %d tiendas, año %d, modo=%s", len(sucursales), year, modo)

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
					logError("RCP: PANIC en %s: %v", sucursales[i].Codigo, r)
				}
			}()

			s := sucursales[i]
			mStart := time.Now()
			mF, mIns, lF, lIns, err := ProcesarTiendaRCP(db, dbRCP, s, year, modo)
			duracion := int(time.Since(mStart).Seconds())

			mu.Lock()
			defer mu.Unlock()

			if err != nil {
				logError("RCP: %s %d FALLO: %v", s.Codigo, year, err)
				InsertarH_Tiendas(db, s.Codigo, year, 0, 0, 0, 0, "ERROR", duracion, modo, tipoEjecucion, "RCP")
				errGlobal++
				return
			}

			estado := "OK"
			if mIns == 0 && lIns == 0 {
				estado = "SALTADO"
			}
			logInfo("RCP: %s %d (%s) MRCP=%d (+%d) LRCP=%d (+%d)",
				s.Codigo, year, estado, mF, mIns, lF, lIns)
			// En H_Tiendas guardamos mIns + lIns como insertados? O solo mIns? 
			// Los demas modulos guardan un solo conteo. Aqui sumamos ambos.
			InsertarH_Tiendas(db, s.Codigo, year, 0, mF+lF, mF+lF-mIns-lIns, mIns+lIns, estado, duracion, modo, tipoEjecucion, "RCP")
		}()
	}

	wg.Wait()
	fmt.Printf("\n  RCP listo. Tiempo: %.1f min. (%d errores)\n", time.Since(tStart).Minutes(), errGlobal)
	logInfo("RCP: Completado en %.1f min, %d errores", time.Since(tStart).Minutes(), errGlobal)
}

func procesarAutomaticoRCP(year int, cfg Config, headless bool) {
	if !headless {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("\n  *** PANIC RCP: %v ***\n", r)
			}
			fmt.Print("\nPresione Enter para salir...")
			leerLinea()
		}()
	}

	tStart := time.Now()
	logHistorial(fmt.Sprintf("RCP: Iniciado %d", year))

	db, err := ConectarSQL()
	if err != nil {
		fmt.Printf("ERROR SQL: %v\n", err)
		return
	}
	defer db.Close()

	if err := CrearBaseRCP_Tiendas(db); err != nil {
		fmt.Printf("ERROR creando BD RCP_Tiendas: %v\n", err)
		return
	}

	dbRCP, err := ConectarSQL_RCP()
	if err != nil {
		fmt.Printf("ERROR conexion RCP_Tiendas: %v\n", err)
		return
	}
	defer dbRCP.Close()

	sucursales, err := ObtenerSucursales(db)
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
		return
	}

	ProcesarRCP(db, dbRCP, sucursales, year, cfg.Modo, "AUTOMATICO")

	fmt.Printf("[%s] RCP Completado.\n", time.Now().Format("15:04:05"))
	logHistorial(fmt.Sprintf("RCP: Completado en %.1f min", time.Since(tStart).Minutes()))
}
