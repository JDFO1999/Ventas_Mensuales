package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func sqlDate(d time.Time) sql.NullTime {
	if d.IsZero() || d.Year() < 1753 || d.Year() > 9999 {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: d, Valid: true}
}

// ---------- Modelos ----------

type PDOCRegistro struct {
	FCODCLI    string
	FCODCLIT   string
	TIPOTRAN   string
	FTP        string
	FTIPOD     string
	FNRODOC    string
	FNROCON    string
	FNRODOCLAR string
	FNROAFE    string
	DIASVEN    float64
	FFECHAD    sql.NullTime
	FFECHAV    sql.NullTime
	FECHAEXT   sql.NullTime
	FECHAR     sql.NullTime
	FMONTO     float64
	NIGTF      float64
	FMONTO_DL  float64
	FDESCRIPD  string
	FMONTOPG   float64
	FAPUNT     float64
	FFLAG      string
	CUEFEC     string
	NROPED     string
	FRECIBO    string
	FDEPOSITO  string
	FCODBAN    string
	DESCUENTO  float64
	MONTOIVA   float64
	EXENTA     float64
	DEDUC      float64
	MON_RET    float64
	POR_RET    float64
	NRORET     string
	MONTOOBJ   float64
	BS_DOLLAR  float64
	FECHAGRA   sql.NullTime
	TIPOGAS    string
	OPERADOR   string
	PORIVA     float64
	NUMRELA    string
	PORMUNI    float64
	MTOMUNI    float64
	SERIAL_F   string
	CLIBROCOM  string
	CODACTECO  string
	NSUJETO    float64
	CLIBROGAS  string
	RUTAIMAGEN string
	CCODIGOALC string
	LAUDITADO  bool
	OPERADORA  string
	FECHAAUDIT sql.NullTime
	CCIA       string
	DES_LARG   string
	FCODCLI2   string
	TIPOTRAN2  string
	FNRODOC2   string
	FNROCON2   string
	FFECHAD2   sql.NullTime
	FECHAR2    sql.NullTime
	FMONTO2    float64
	FMONTO_DL2 float64
}

type PTRARegistro struct {
	TCODCLI    string
	TTIPOD     string
	TTP        string
	TIPOTRAN   string
	TNROMOV    string
	TNRODOC    string
	NRO_NC     string
	TMONTOPG   float64
	TMONTOPGDL float64
	TNC75PG    float64
	TFECHAP    sql.NullTime
	TDESCRIP   string
	RECIBO     string
	DEPOSITO   string
	TBANCO     string
	TCOBRADOR  string
	BS_DOLLAR  float64
	PORCEN     float64
	MTOOBJIM   float64
	NOMBRE     string
	FECHAGRA   sql.NullTime
	OPERADOR   string
	NUMRELA    string
	NDNC       string
	NRODED     string
	DES_LARG   string
}

type MPRORegistro struct {
	CODCLI     string
	ZONA       string
	NOMCLI     string
	CEDCLI     string
	NITCLI     string
	DIRCLI1    string
	DIRCLI2    string
	DIRTRAB1   string
	DIRTRAB2   string
	CIUDAD1    string
	CIUDAD2    string
	TELEFONO   string
	TELFTRAB   string
	ATENCION   string
	REFERIDO   string
	CEDFIAD    string
	RIM        string
	NOMFIAD    string
	DIRFIAD1   string
	DIRFIAD2   string
	DIRFIAD3   string
	DIRFIAD4   string
	CIUDADF1   string
	CIUDADF2   string
	TELFFIAD1  string
	TELFFIAD2  string
	SALCLI     float64
	PLAZO      float64
	LCREDCLI   float64
	CODCUEN    string
	CODEFEC    string
	CODCAJ     string
	SERIE      string
	NATJU      string
	DOLLAR     string
	INIACT     string
	REPLEG     string
	FECREG     sql.NullTime
	ACTCOM     string
	LOGMON     string
	ESTADO1    string
	PAIS       string
	CLOCK      bool
	IM         bool
	SIN_RET    bool
	TBANCARIA  bool
	PORCENT    float64
	CODCXC     string
	CODND      string
	CONTACTO   string
	TELEFONOC  string
	OBSERVA    string
	TPLAZO     float64
	POR_IMP    float64
	POR_DST1   float64
	POR_DST2   float64
	POR_DST3   float64
	POR_DSTPP  float64
	COD_BAN1   string
	COD_BAN2   string
	COD_BAN3   string
	COD_BAN4   string
	COD_BAN5   string
	COD_BAN6   string
	CTABAN1    string
	CTABAN2    string
	CTABAN3    string
	CTABAN4    string
	CTABAN5    string
	CTABAN6    string
	DEPNOMBRE1 string
	DEPNOMBRE2 string
	DEPNOMBRE3 string
	DEPNOMBRE4 string
	DEPNOMBRE5 string
	DEPNOMBRE6 string
	PAG_WEB    string
	EMAILP     string
	EMAILC     string
	EMAILR     string
	REPRE      string
	TELEFONOR  string
	POR_DSTP2  float64
	POR_DSTP3  float64
	LDEPOSITAR bool
	DETAFIJO   string
	NSUJETO    float64
	ZONAP      string
	CODACTECO  string
	DOLLARP    string
	ID_PROVEE  int
	CTIPOTASA  string
	AREAAD     string
	TELEFONOAD string
	CONTACTOAD string
	EMAILAD    string
	COD_BANT1  string
	NOMCLIT1   string
	CEDCLIT1   string
	CCUENTAT1  string
	COD_BANT2  string
	NOMCLIT2   string
	CEDCLIT2   string
	CCUENTAT2  string
	TIPO_CTA1  string
	TIPO_CTA2  string
	TIPO_CTA3  string
	TIPO_CTA4  string
	TIPO_CTA1T string
	TIPO_CTA2T string
	LREFCAMBI  bool
}

// ---------- Rutas ----------

func rutaCxPPath() string {
	return `\\10.10.10.30\Sistema\aBC-Soft\Data\E001\CxP`
}

func rutaPDOC() string {
	return filepath.Join(rutaCxPPath(), "PDOC001.DBF")
}

func rutaPTRA() string {
	return filepath.Join(rutaCxPPath(), "PTRA001.DBF")
}

func rutaMPRO() string {
	return filepath.Join(rutaCxPPath(), "MPRO001.DBF")
}

// ---------- Conexiones SQL ----------

func ConectarSQL_PDOC() (*sql.DB, error) {
	connStr := fmt.Sprintf("server=%s;database=PDOC_Tiendas;user id=%s;password=%s;encrypt=disable;connection timeout=30;timeout=900",
		SQLServer, SQLUser, SQLPassword)
	db, err := sql.Open("mssql", connStr)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(4)
	return db, nil
}

func ConectarSQL_PTRA() (*sql.DB, error) {
	connStr := fmt.Sprintf("server=%s;database=PTRA_Tiendas;user id=%s;password=%s;encrypt=disable;connection timeout=30;timeout=900",
		SQLServer, SQLUser, SQLPassword)
	db, err := sql.Open("mssql", connStr)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(4)
	return db, nil
}

func ConectarSQL_MPRO() (*sql.DB, error) {
	connStr := fmt.Sprintf("server=%s;database=MPRO_Tiendas;user id=%s;password=%s;encrypt=disable;connection timeout=30;timeout=900",
		SQLServer, SQLUser, SQLPassword)
	db, err := sql.Open("mssql", connStr)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(4)
	return db, nil
}

func CrearBasePDOC(db *sql.DB) error {
	_, err := db.Exec(`IF DB_ID('PDOC_Tiendas') IS NULL BEGIN CREATE DATABASE PDOC_Tiendas END`)
	return err
}

func CrearBasePTRA(db *sql.DB) error {
	_, err := db.Exec(`IF DB_ID('PTRA_Tiendas') IS NULL BEGIN CREATE DATABASE PTRA_Tiendas END`)
	return err
}

func CrearBaseMPRO(db *sql.DB) error {
	_, err := db.Exec(`IF DB_ID('MPRO_Tiendas') IS NULL BEGIN CREATE DATABASE MPRO_Tiendas END`)
	return err
}

// ---------- Crear Tablas ----------

func CrearTablaPDOC(db *sql.DB) error {
	_, err := db.Exec(`
		IF OBJECT_ID('PDOC_Datos','U') IS NULL
		CREATE TABLE PDOC_Datos (
			FCODCLI     VARCHAR(10) NULL,
			FCODCLIT    VARCHAR(10) NULL,
			TIPOTRAN    VARCHAR(2) NULL,
			FTP         VARCHAR(2) NULL,
			FTIPOD      VARCHAR(2) NULL,
			FNRODOC     VARCHAR(25) NOT NULL,
			FNROCON     VARCHAR(14) NULL,
			FNRODOCLAR  VARCHAR(25) NULL,
			FNROAFE     VARCHAR(25) NULL,
			DIASVEN     DECIMAL(5,0) NULL,
			FFECHAD     DATE NULL,
			FFECHAV     DATE NULL,
			FECHAEXT    DATE NULL,
			FECHAR      DATE NULL,
			FMONTO      DECIMAL(18,2) NULL,
			NIGTF       DECIMAL(18,2) NULL,
			FMONTO_DL   DECIMAL(18,2) NULL,
			FDESCRIPD   VARCHAR(100) NULL,
			FMONTOPG    DECIMAL(18,2) NULL,
			FAPUNT      DECIMAL(5,0) NULL,
			FFLAG       VARCHAR(1) NULL,
			CUEFEC      VARCHAR(1) NULL,
			NROPED      VARCHAR(8) NULL,
			FRECIBO     VARCHAR(8) NULL,
			FDEPOSITO   VARCHAR(12) NULL,
			FCODBAN     VARCHAR(2) NULL,
			DESCUENTO   DECIMAL(18,2) NULL,
			MONTOIVA    DECIMAL(18,2) NULL,
			EXENTA      DECIMAL(18,2) NULL,
			DEDUC       DECIMAL(9,2) NULL,
			MON_RET     DECIMAL(18,2) NULL,
			POR_RET     DECIMAL(6,2) NULL,
			NRORET      VARCHAR(10) NULL,
			MONTOOBJ    DECIMAL(18,2) NULL,
			BS_DOLLAR   DECIMAL(20,6) NULL,
			FECHAGRA    DATETIME NULL,
			TIPOGAS     VARCHAR(6) NULL,
			OPERADOR    VARCHAR(20) NULL,
			PORIVA      DECIMAL(8,2) NULL,
			NUMRELA     VARCHAR(10) NULL,
			PORMUNI     DECIMAL(6,2) NULL,
			MTOMUNI     DECIMAL(18,2) NULL,
			SERIAL_F    VARCHAR(20) NULL,
			CLIBROCOM   VARCHAR(1) NULL,
			CODACTECO   VARCHAR(20) NULL,
			NSUJETO     DECIMAL(1,0) NULL,
			CLIBROGAS   VARCHAR(1) NULL,
			RUTAIMAGEN  VARCHAR(200) NULL,
			CCODIGOALC  VARCHAR(4) NULL,
			LAUDITADO   BIT NULL,
			OPERADORA   VARCHAR(20) NULL,
			FECHAAUDIT  DATETIME NULL,
			CCIA        VARCHAR(3) NULL,
			DES_LARG    VARCHAR(MAX) NULL,
			FCODCLI2    VARCHAR(10) NULL,
			TIPOTRAN2   VARCHAR(2) NULL,
			FNRODOC2    VARCHAR(25) NULL,
			FNROCON2    VARCHAR(14) NULL,
			FFECHAD2    DATE NULL,
			FECHAR2     DATE NULL,
			FMONTO2     DECIMAL(18,2) NULL,
			FMONTO_DL2  DECIMAL(18,2) NULL,
			FechaCarga  DATETIME DEFAULT GETDATE(),
			CONSTRAINT PK_PDOC PRIMARY KEY (FNRODOC)
		)`)
	return err
}

func CrearTablaPTRA(db *sql.DB) error {
	_, err := db.Exec(`
		IF OBJECT_ID('PTRA_Datos','U') IS NULL
		CREATE TABLE PTRA_Datos (
			TCODCLI     VARCHAR(10) NULL,
			TTIPOD      VARCHAR(2) NULL,
			TTP         VARCHAR(2) NULL,
			TIPOTRAN    VARCHAR(2) NULL,
			TNROMOV     VARCHAR(10) NOT NULL,
			TNRODOC     VARCHAR(25) NOT NULL,
			NRO_NC      VARCHAR(25) NULL,
			TMONTOPG    DECIMAL(18,2) NULL,
			TMONTOPGDL  DECIMAL(18,2) NULL,
			TNC75PG     DECIMAL(18,2) NULL,
			TFECHAP     DATE NULL,
			TDESCRIP    VARCHAR(100) NULL,
			RECIBO      VARCHAR(8) NULL,
			DEPOSITO    VARCHAR(12) NULL,
			TBANCO      VARCHAR(2) NULL,
			TCOBRADOR   VARCHAR(2) NULL,
			BS_DOLLAR   DECIMAL(20,6) NULL,
			PORCEN      DECIMAL(10,2) NULL,
			MTOOBJIM    DECIMAL(18,2) NULL,
			NOMBRE      VARCHAR(120) NULL,
			FECHAGRA    DATETIME NULL,
			OPERADOR    VARCHAR(20) NULL,
			NUMRELA     VARCHAR(10) NULL,
			NDNC        VARCHAR(2) NULL,
			NRODED      VARCHAR(25) NULL,
			DES_LARG    VARCHAR(MAX) NULL,
			FechaCarga  DATETIME DEFAULT GETDATE(),
			CONSTRAINT PK_PTRA PRIMARY KEY (TNRODOC, TNROMOV)
		)`)
	if err != nil {
		return err
	}
	_, err = db.Exec(`
		IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE object_id = OBJECT_ID('PTRA_Datos') AND name = 'IX_PTRA_TCODCLI')
		CREATE NONCLUSTERED INDEX IX_PTRA_TCODCLI ON PTRA_Datos(TCODCLI)`)
	return err
}

func CrearTablaMPRO(db *sql.DB) error {
	_, err := db.Exec(`
		IF OBJECT_ID('MPRO_Datos','U') IS NULL
		CREATE TABLE MPRO_Datos (
			CODCLI      VARCHAR(10) NOT NULL,
			ZONA        VARCHAR(20) NULL,
			NOMCLI      VARCHAR(255) NULL,
			CEDCLI      VARCHAR(255) NULL,
			NITCLI      VARCHAR(255) NULL,
			DIRCLI1     VARCHAR(255) NULL,
			DIRCLI2     VARCHAR(255) NULL,
			DIRTRAB1    VARCHAR(255) NULL,
			DIRTRAB2    VARCHAR(255) NULL,
			CIUDAD1     VARCHAR(20) NULL,
			CIUDAD2     VARCHAR(20) NULL,
			TELEFONO    VARCHAR(255) NULL,
			TELFTRAB    VARCHAR(255) NULL,
			ATENCION    VARCHAR(255) NULL,
			REFERIDO    VARCHAR(255) NULL,
			CEDFIAD     VARCHAR(255) NULL,
			RIM         VARCHAR(255) NULL,
			NOMFIAD     VARCHAR(255) NULL,
			DIRFIAD1    VARCHAR(255) NULL,
			DIRFIAD2    VARCHAR(255) NULL,
			DIRFIAD3    VARCHAR(255) NULL,
			DIRFIAD4    VARCHAR(255) NULL,
			CIUDADF1    VARCHAR(20) NULL,
			CIUDADF2    VARCHAR(20) NULL,
			TELFFIAD1   VARCHAR(255) NULL,
			TELFFIAD2   VARCHAR(255) NULL,
			SALCLI      DECIMAL(18,2) NULL,
			PLAZO       DECIMAL(5,0) NULL,
			LCREDCLI    DECIMAL(18,2) NULL,
			CODCUEN     VARCHAR(255) NULL,
			CODEFEC     VARCHAR(20) NULL,
			CODCAJ      VARCHAR(20) NULL,
			SERIE       VARCHAR(20) NULL,
			NATJU       VARCHAR(20) NULL,
			DOLLAR      VARCHAR(10) NULL,
			INIACT      VARCHAR(20) NULL,
			REPLEG      VARCHAR(255) NULL,
			FECREG      DATE NULL,
			ACTCOM      VARCHAR(20) NULL,
			LOGMON      VARCHAR(10) NULL,
			ESTADO1     VARCHAR(10) NULL,
			PAIS        VARCHAR(20) NULL,
			CLOCK       BIT NULL,
			IM          BIT NULL,
			SIN_RET     BIT NULL,
			TBANCARIA   BIT NULL,
			PORCENT     DECIMAL(8,2) NULL,
			CODCXC      VARCHAR(20) NULL,
			CODND       VARCHAR(20) NULL,
			CONTACTO    VARCHAR(255) NULL,
			TELEFONOC   VARCHAR(255) NULL,
			OBSERVA     VARCHAR(255) NULL,
			TPLAZO      DECIMAL(5,0) NULL,
			POR_IMP     DECIMAL(8,2) NULL,
			POR_DST1    DECIMAL(8,2) NULL,
			POR_DST2    DECIMAL(8,2) NULL,
			POR_DST3    DECIMAL(8,2) NULL,
			POR_DSTPP   DECIMAL(8,2) NULL,
			COD_BAN1    VARCHAR(20) NULL,
			COD_BAN2    VARCHAR(20) NULL,
			COD_BAN3    VARCHAR(20) NULL,
			COD_BAN4    VARCHAR(20) NULL,
			COD_BAN5    VARCHAR(20) NULL,
			COD_BAN6    VARCHAR(20) NULL,
			CTABAN1     VARCHAR(255) NULL,
			CTABAN2     VARCHAR(255) NULL,
			CTABAN3     VARCHAR(255) NULL,
			CTABAN4     VARCHAR(255) NULL,
			CTABAN5     VARCHAR(255) NULL,
			CTABAN6     VARCHAR(255) NULL,
			DEPNOMBRE1  VARCHAR(255) NULL,
			DEPNOMBRE2  VARCHAR(255) NULL,
			DEPNOMBRE3  VARCHAR(255) NULL,
			DEPNOMBRE4  VARCHAR(255) NULL,
			DEPNOMBRE5  VARCHAR(255) NULL,
			DEPNOMBRE6  VARCHAR(255) NULL,
			PAG_WEB     VARCHAR(255) NULL,
			EMAILP      VARCHAR(255) NULL,
			EMAILC      VARCHAR(255) NULL,
			EMAILR      VARCHAR(255) NULL,
			REPRE       VARCHAR(255) NULL,
			TELEFONOR   VARCHAR(255) NULL,
			POR_DSTP2   DECIMAL(8,2) NULL,
			POR_DSTP3   DECIMAL(8,2) NULL,
			LDEPOSITAR  BIT NULL,
			DETAFIJO    VARCHAR(20) NULL,
			NSUJETO     DECIMAL(1,0) NULL,
			ZONAP       VARCHAR(20) NULL,
			CODACTECO   VARCHAR(255) NULL,
			DOLLARP     VARCHAR(10) NULL,
			ID_PROVEE   INT NULL,
			CTIPOTASA   VARCHAR(20) NULL,
			AREAAD      VARCHAR(20) NULL,
			TELEFONOAD  VARCHAR(255) NULL,
			CONTACTOAD  VARCHAR(255) NULL,
			EMAILAD     VARCHAR(255) NULL,
			COD_BANT1   VARCHAR(20) NULL,
			NOMCLIT1    VARCHAR(255) NULL,
			CEDCLIT1    VARCHAR(255) NULL,
			CCUENTAT1   VARCHAR(255) NULL,
			COD_BANT2   VARCHAR(20) NULL,
			NOMCLIT2    VARCHAR(255) NULL,
			CEDCLIT2    VARCHAR(255) NULL,
			CCUENTAT2   VARCHAR(255) NULL,
			TIPO_CTA1   VARCHAR(20) NULL,
			TIPO_CTA2   VARCHAR(20) NULL,
			TIPO_CTA3   VARCHAR(20) NULL,
			TIPO_CTA4   VARCHAR(20) NULL,
			TIPO_CTA1T  VARCHAR(20) NULL,
			TIPO_CTA2T  VARCHAR(20) NULL,
			LREFCAMBI   BIT NULL,
			FechaCarga  DATETIME DEFAULT GETDATE(),
			CONSTRAINT PK_MPRO PRIMARY KEY (CODCLI)
		)`)
	return err
}

// ---------- Lectura DBF ----------

func LeerPDOC(path string) ([]PDOCRegistro, error) {
	dbf, err := OpenDBF(path)
	if err != nil {
		return nil, err
	}
	defer dbf.Close()

	if dbf.NumRecords == 0 {
		return nil, nil
	}

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
	getD := func(rec []byte, name string) time.Time {
		if i := dbf.FieldIndex(name); i >= 0 {
			d, _ := dbf.GetDate(rec, dbf.Fields[i])
			return d
		}
		return time.Time{}
	}
	getT := func(rec []byte, name string) time.Time {
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
	getM := func(rec []byte, name string) string {
		if i := dbf.FieldIndex(name); i >= 0 {
			return dbf.GetString(rec, dbf.Fields[i])
		}
		return ""
	}

	var regs []PDOCRegistro
	seen := make(map[string]bool)
	for i := 0; i < dbf.NumRecords; i++ {
		rec, err := dbf.ReadRecord(i)
		if err != nil || len(rec) == 0 {
			continue
		}
		if rec[0] == 0x2A {
			continue
		}

		fnrodoc := getS(rec, "FNRODOC")
		if fnrodoc == "" {
			continue
		}
		if seen[fnrodoc] {
			continue
		}
		seen[fnrodoc] = true

		regs = append(regs, PDOCRegistro{
			FCODCLI:    getS(rec, "FCODCLI"),
			FCODCLIT:   getS(rec, "FCODCLIT"),
			TIPOTRAN:   getS(rec, "TIPOTRAN"),
			FTP:        getS(rec, "FTP"),
			FTIPOD:     getS(rec, "FTIPOD"),
			FNRODOC:    getS(rec, "FNRODOC"),
			FNROCON:    getS(rec, "FNROCON"),
			FNRODOCLAR: getS(rec, "FNRODOCLAR"),
			FNROAFE:    getS(rec, "FNROAFE"),
			DIASVEN:    getF(rec, "DIASVEN"),
			FFECHAD:    sqlDate(getD(rec, "FFECHAD")),
			FFECHAV:    sqlDate(getD(rec, "FFECHAV")),
			FECHAEXT:   sqlDate(getD(rec, "FECHAEXT")),
			FECHAR:     sqlDate(getD(rec, "FECHAR")),
			FMONTO:     getF(rec, "FMONTO"),
			NIGTF:      getF(rec, "NIGTF"),
			FMONTO_DL:  getF(rec, "FMONTO_DL"),
			FDESCRIPD:  getS(rec, "FDESCRIPD"),
			FMONTOPG:   getF(rec, "FMONTOPG"),
			FAPUNT:     getF(rec, "FAPUNT"),
			FFLAG:      getS(rec, "FFLAG"),
			CUEFEC:     getS(rec, "CUEFEC"),
			NROPED:     getS(rec, "NROPED"),
			FRECIBO:    getS(rec, "FRECIBO"),
			FDEPOSITO:  getS(rec, "FDEPOSITO"),
			FCODBAN:    getS(rec, "FCODBAN"),
			DESCUENTO:  getF(rec, "DESCUENTO"),
			MONTOIVA:   getF(rec, "MONTOIVA"),
			EXENTA:     getF(rec, "EXENTA"),
			DEDUC:      getF(rec, "DEDUC"),
			MON_RET:    getF(rec, "MON_RET"),
			POR_RET:    getF(rec, "POR_RET"),
			NRORET:     getS(rec, "NRORET"),
			MONTOOBJ:   getF(rec, "MONTOOBJ"),
			BS_DOLLAR:  getF(rec, "BS_DOLLAR"),
			FECHAGRA:   sqlDate(getT(rec, "FECHAGRA")),
			TIPOGAS:    getS(rec, "TIPOGAS"),
			OPERADOR:   getS(rec, "OPERADOR"),
			PORIVA:     getF(rec, "PORIVA"),
			NUMRELA:    getS(rec, "NUMRELA"),
			PORMUNI:    getF(rec, "PORMUNI"),
			MTOMUNI:    getF(rec, "MTOMUNI"),
			SERIAL_F:   getS(rec, "SERIAL_F"),
			CLIBROCOM:  getS(rec, "CLIBROCOM"),
			CODACTECO:  getS(rec, "CODACTECO"),
			NSUJETO:    getF(rec, "NSUJETO"),
			CLIBROGAS:  getS(rec, "CLIBROGAS"),
			RUTAIMAGEN: getS(rec, "RUTAIMAGEN"),
			CCODIGOALC: getS(rec, "CCODIGOALC"),
			LAUDITADO:  getB(rec, "LAUDITADO"),
			OPERADORA:  getS(rec, "OPERADORA"),
			FECHAAUDIT: sqlDate(getT(rec, "FECHAAUDIT")),
			CCIA:       getS(rec, "CCIA"),
			DES_LARG:   getM(rec, "DES_LARG"),
			FCODCLI2:   getS(rec, "FCODCLI2"),
			TIPOTRAN2:  getS(rec, "TIPOTRAN2"),
			FNRODOC2:   getS(rec, "FNRODOC2"),
			FNROCON2:   getS(rec, "FNROCON2"),
			FFECHAD2:   sqlDate(getD(rec, "FFECHAD2")),
			FECHAR2:    sqlDate(getD(rec, "FECHAR2")),
			FMONTO2:    getF(rec, "FMONTO2"),
			FMONTO_DL2: getF(rec, "FMONTO_DL2"),
		})
	}
	return regs, nil
}

func LeerPTRA(path string) ([]PTRARegistro, error) {
	dbf, err := OpenDBF(path)
	if err != nil {
		return nil, err
	}
	defer dbf.Close()

	if dbf.NumRecords == 0 {
		return nil, nil
	}

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
	getD := func(rec []byte, name string) time.Time {
		if i := dbf.FieldIndex(name); i >= 0 {
			d, _ := dbf.GetDate(rec, dbf.Fields[i])
			return d
		}
		return time.Time{}
	}
	getT := func(rec []byte, name string) time.Time {
		if i := dbf.FieldIndex(name); i >= 0 {
			d, _ := dbf.GetDate(rec, dbf.Fields[i])
			return d
		}
		return time.Time{}
	}
	getM := func(rec []byte, name string) string {
		if i := dbf.FieldIndex(name); i >= 0 {
			return dbf.GetString(rec, dbf.Fields[i])
		}
		return ""
	}

	var regs []PTRARegistro
	for i := 0; i < dbf.NumRecords; i++ {
		rec, err := dbf.ReadRecord(i)
		if err != nil || len(rec) == 0 {
			continue
		}
		if rec[0] == 0x2A {
			continue
		}

		tnrodoc := getS(rec, "TNRODOC")
		tnromov := getS(rec, "TNROMOV")
		if tnrodoc == "" && tnromov == "" {
			continue
		}

		regs = append(regs, PTRARegistro{
			TCODCLI:    getS(rec, "TCODCLI"),
			TTIPOD:     getS(rec, "TTIPOD"),
			TTP:        getS(rec, "TTP"),
			TIPOTRAN:   getS(rec, "TIPOTRAN"),
			TNROMOV:    getS(rec, "TNROMOV"),
			TNRODOC:    getS(rec, "TNRODOC"),
			NRO_NC:     getS(rec, "NRO_NC"),
			TMONTOPG:   getF(rec, "TMONTOPG"),
			TMONTOPGDL: getF(rec, "TMONTOPGDL"),
			TNC75PG:    getF(rec, "TNC75PG"),
			TFECHAP:    sqlDate(getD(rec, "TFECHAP")),
			TDESCRIP:   getS(rec, "TDESCRIP"),
			RECIBO:     getS(rec, "RECIBO"),
			DEPOSITO:   getS(rec, "DEPOSITO"),
			TBANCO:     getS(rec, "TBANCO"),
			TCOBRADOR:  getS(rec, "TCOBRADOR"),
			BS_DOLLAR:  getF(rec, "BS_DOLLAR"),
			PORCEN:     getF(rec, "PORCEN"),
			MTOOBJIM:   getF(rec, "MTOOBJIM"),
			NOMBRE:     getS(rec, "NOMBRE"),
			FECHAGRA:   sqlDate(getT(rec, "FECHAGRA")),
			OPERADOR:   getS(rec, "OPERADOR"),
			NUMRELA:    getS(rec, "NUMRELA"),
			NDNC:       getS(rec, "NDNC"),
			NRODED:     getS(rec, "NRODED"),
			DES_LARG:   getM(rec, "DES_LARG"),
		})
	}
	return regs, nil
}

// ---------- Inserción SQL ----------

func InsertarPDOC(db *sql.DB, regs []PDOCRegistro) (int, error) {
	if len(regs) == 0 {
		return 0, nil
	}
	const batchSize = 33
	total := 0
	cols := "FCODCLI,FCODCLIT,TIPOTRAN,FTP,FTIPOD,FNRODOC,FNROCON,FNRODOCLAR,FNROAFE,DIASVEN,FFECHAD,FFECHAV,FECHAEXT,FECHAR,FMONTO,NIGTF,FMONTO_DL,FDESCRIPD,FMONTOPG,FAPUNT,FFLAG,CUEFEC,NROPED,FRECIBO,FDEPOSITO,FCODBAN,DESCUENTO,MONTOIVA,EXENTA,DEDUC,MON_RET,POR_RET,NRORET,MONTOOBJ,BS_DOLLAR,FECHAGRA,TIPOGAS,OPERADOR,PORIVA,NUMRELA,PORMUNI,MTOMUNI,SERIAL_F,CLIBROCOM,CODACTECO,NSUJETO,CLIBROGAS,RUTAIMAGEN,CCODIGOALC,LAUDITADO,OPERADORA,FECHAAUDIT,CCIA,DES_LARG,FCODCLI2,TIPOTRAN2,FNRODOC2,FNROCON2,FFECHAD2,FECHAR2,FMONTO2,FMONTO_DL2"
	singlePH := "(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)"

	for i := 0; i < len(regs); i += batchSize {
		end := i + batchSize
		if end > len(regs) {
			end = len(regs)
		}
		chunk := regs[i:end]

		placeholders := make([]string, len(chunk))
		args := make([]interface{}, 0, len(chunk)*62)
		for j, r := range chunk {
			placeholders[j] = singlePH
			args = append(args,
				r.FCODCLI, r.FCODCLIT, r.TIPOTRAN, r.FTP, r.FTIPOD, r.FNRODOC, r.FNROCON, r.FNRODOCLAR, r.FNROAFE, r.DIASVEN,
				r.FFECHAD, r.FFECHAV, r.FECHAEXT, r.FECHAR, r.FMONTO, r.NIGTF, r.FMONTO_DL, r.FDESCRIPD, r.FMONTOPG, r.FAPUNT,
				r.FFLAG, r.CUEFEC, r.NROPED, r.FRECIBO, r.FDEPOSITO, r.FCODBAN, r.DESCUENTO, r.MONTOIVA, r.EXENTA, r.DEDUC,
				r.MON_RET, r.POR_RET, r.NRORET, r.MONTOOBJ, r.BS_DOLLAR, r.FECHAGRA, r.TIPOGAS, r.OPERADOR, r.PORIVA, r.NUMRELA,
				r.PORMUNI, r.MTOMUNI, r.SERIAL_F, r.CLIBROCOM, r.CODACTECO, r.NSUJETO, r.CLIBROGAS, r.RUTAIMAGEN, r.CCODIGOALC, r.LAUDITADO,
				r.OPERADORA, r.FECHAAUDIT, r.CCIA, r.DES_LARG, r.FCODCLI2, r.TIPOTRAN2, r.FNRODOC2, r.FNROCON2, r.FFECHAD2, r.FECHAR2,
				r.FMONTO2, r.FMONTO_DL2,
			)
		}

		query := fmt.Sprintf("INSERT INTO PDOC_Datos (%s) VALUES %s", cols, strings.Join(placeholders, ","))
		_, err := db.Exec(query, args...)
		if err != nil {
			for _, r := range chunk {
				singleArgs := []interface{}{
					r.FCODCLI, r.FCODCLIT, r.TIPOTRAN, r.FTP, r.FTIPOD, r.FNRODOC, r.FNROCON, r.FNRODOCLAR, r.FNROAFE, r.DIASVEN,
					r.FFECHAD, r.FFECHAV, r.FECHAEXT, r.FECHAR, r.FMONTO, r.NIGTF, r.FMONTO_DL, r.FDESCRIPD, r.FMONTOPG, r.FAPUNT,
					r.FFLAG, r.CUEFEC, r.NROPED, r.FRECIBO, r.FDEPOSITO, r.FCODBAN, r.DESCUENTO, r.MONTOIVA, r.EXENTA, r.DEDUC,
					r.MON_RET, r.POR_RET, r.NRORET, r.MONTOOBJ, r.BS_DOLLAR, r.FECHAGRA, r.TIPOGAS, r.OPERADOR, r.PORIVA, r.NUMRELA,
					r.PORMUNI, r.MTOMUNI, r.SERIAL_F, r.CLIBROCOM, r.CODACTECO, r.NSUJETO, r.CLIBROGAS, r.RUTAIMAGEN, r.CCODIGOALC, r.LAUDITADO,
					r.OPERADORA, r.FECHAAUDIT, r.CCIA, r.DES_LARG, r.FCODCLI2, r.TIPOTRAN2, r.FNRODOC2, r.FNROCON2, r.FFECHAD2, r.FECHAR2,
					r.FMONTO2, r.FMONTO_DL2,
				}
				singleQuery := fmt.Sprintf("INSERT INTO PDOC_Datos (%s) VALUES %s", cols, singlePH)
				if _, errSingle := db.Exec(singleQuery, singleArgs...); errSingle == nil {
					total++
				}
			}
		} else {
			total += len(chunk)
		}
		if total%5000 == 0 {
			fmt.Printf("\r  PDOC: %d/%d insertados...    ", total, len(regs))
		}
	}
	fmt.Printf("\n")
	return total, nil
}

func InsertarPTRA(db *sql.DB, regs []PTRARegistro) (int, error) {
	if len(regs) == 0 {
		return 0, nil
	}
	const batchSize = 50
	total := 0
	cols := "TCODCLI,TTIPOD,TTP,TIPOTRAN,TNROMOV,TNRODOC,NRO_NC,TMONTOPG,TMONTOPGDL,TNC75PG,TFECHAP,TDESCRIP,RECIBO,DEPOSITO,TBANCO,TCOBRADOR,BS_DOLLAR,PORCEN,MTOOBJIM,NOMBRE,FECHAGRA,OPERADOR,NUMRELA,NDNC,NRODED,DES_LARG"
	singlePH := "(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)"

	for i := 0; i < len(regs); i += batchSize {
		end := i + batchSize
		if end > len(regs) {
			end = len(regs)
		}
		chunk := regs[i:end]

		placeholders := make([]string, len(chunk))
		args := make([]interface{}, 0, len(chunk)*26)
		for j, r := range chunk {
			placeholders[j] = singlePH
			args = append(args,
				r.TCODCLI, r.TTIPOD, r.TTP, r.TIPOTRAN, r.TNROMOV, r.TNRODOC, r.NRO_NC,
				r.TMONTOPG, r.TMONTOPGDL, r.TNC75PG, r.TFECHAP, r.TDESCRIP, r.RECIBO, r.DEPOSITO,
				r.TBANCO, r.TCOBRADOR, r.BS_DOLLAR, r.PORCEN, r.MTOOBJIM, r.NOMBRE, r.FECHAGRA,
				r.OPERADOR, r.NUMRELA, r.NDNC, r.NRODED, r.DES_LARG,
			)
		}

		query := fmt.Sprintf("INSERT INTO PTRA_Datos (%s) VALUES %s", cols, strings.Join(placeholders, ","))
		_, err := db.Exec(query, args...)
		if err != nil {
			for _, r := range chunk {
				singleArgs := []interface{}{
					r.TCODCLI, r.TTIPOD, r.TTP, r.TIPOTRAN, r.TNROMOV, r.TNRODOC, r.NRO_NC,
					r.TMONTOPG, r.TMONTOPGDL, r.TNC75PG, r.TFECHAP, r.TDESCRIP, r.RECIBO, r.DEPOSITO,
					r.TBANCO, r.TCOBRADOR, r.BS_DOLLAR, r.PORCEN, r.MTOOBJIM, r.NOMBRE, r.FECHAGRA,
					r.OPERADOR, r.NUMRELA, r.NDNC, r.NRODED, r.DES_LARG,
				}
				singleQuery := fmt.Sprintf("INSERT INTO PTRA_Datos (%s) VALUES %s", cols, singlePH)
				if _, errSingle := db.Exec(singleQuery, singleArgs...); errSingle == nil {
					total++
				}
			}
		} else {
			total += len(chunk)
		}
		if total%5000 == 0 {
			fmt.Printf("\r  PTRA: %d/%d insertados...    ", total, len(regs))
		}
	}
	fmt.Printf("\n")
	return total, nil
}

func LeerMPRO(path string) ([]MPRORegistro, error) {
	dbf, err := OpenDBF(path)
	if err != nil {
		return nil, err
	}
	defer dbf.Close()

	if dbf.NumRecords == 0 {
		return nil, nil
	}

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
	getI := func(rec []byte, name string) int {
		if i := dbf.FieldIndex(name); i >= 0 {
			return int(dbf.GetFloat(rec, dbf.Fields[i]))
		}
		return 0
	}

	var regs []MPRORegistro
	seen := make(map[string]bool)
	for i := 0; i < dbf.NumRecords; i++ {
		rec, err := dbf.ReadRecord(i)
		if err != nil || len(rec) == 0 {
			continue
		}
		if rec[0] == 0x2A {
			continue
		}

		codcli := getS(rec, "CODCLI")
		if codcli == "" {
			continue
		}
		if seen[codcli] {
			continue
		}
		seen[codcli] = true

		regs = append(regs, MPRORegistro{
			CODCLI:     codcli,
			ZONA:       getS(rec, "ZONA"),
			NOMCLI:     getS(rec, "NOMCLI"),
			CEDCLI:     getS(rec, "CEDCLI"),
			NITCLI:     getS(rec, "NITCLI"),
			DIRCLI1:    getS(rec, "DIRCLI1"),
			DIRCLI2:    getS(rec, "DIRCLI2"),
			DIRTRAB1:   getS(rec, "DIRTRAB1"),
			DIRTRAB2:   getS(rec, "DIRTRAB2"),
			CIUDAD1:    getS(rec, "CIUDAD1"),
			CIUDAD2:    getS(rec, "CIUDAD2"),
			TELEFONO:   getS(rec, "TELEFONO"),
			TELFTRAB:   getS(rec, "TELFTRAB"),
			ATENCION:   getS(rec, "ATENCION"),
			REFERIDO:   getS(rec, "REFERIDO"),
			CEDFIAD:    getS(rec, "CEDFIAD"),
			RIM:        getS(rec, "RIM"),
			NOMFIAD:    getS(rec, "NOMFIAD"),
			DIRFIAD1:   getS(rec, "DIRFIAD1"),
			DIRFIAD2:   getS(rec, "DIRFIAD2"),
			DIRFIAD3:   getS(rec, "DIRFIAD3"),
			DIRFIAD4:   getS(rec, "DIRFIAD4"),
			CIUDADF1:   getS(rec, "CIUDADF1"),
			CIUDADF2:   getS(rec, "CIUDADF2"),
			TELFFIAD1:  getS(rec, "TELFFIAD1"),
			TELFFIAD2:  getS(rec, "TELFFIAD2"),
			SALCLI:     getF(rec, "SALCLI"),
			PLAZO:      getF(rec, "PLAZO"),
			LCREDCLI:   getF(rec, "LCREDCLI"),
			CODCUEN:    getS(rec, "CODCUEN"),
			CODEFEC:    getS(rec, "CODEFEC"),
			CODCAJ:     getS(rec, "CODCAJ"),
			SERIE:      getS(rec, "SERIE"),
			NATJU:      getS(rec, "NATJU"),
			DOLLAR:     getS(rec, "DOLLAR"),
			INIACT:     getS(rec, "INIACT"),
			REPLEG:     getS(rec, "REPLEG"),
			FECREG:     sqlDate(getD(rec, "FECREG")),
			ACTCOM:     getS(rec, "ACTCOM"),
			LOGMON:     getS(rec, "LOGMON"),
			ESTADO1:    getS(rec, "ESTADO1"),
			PAIS:       getS(rec, "PAIS"),
			CLOCK:      getB(rec, "CLOCK"),
			IM:         getB(rec, "IM"),
			SIN_RET:    getB(rec, "SIN_RET"),
			TBANCARIA:  getB(rec, "TBANCARIA"),
			PORCENT:    getF(rec, "PORCENT"),
			CODCXC:     getS(rec, "CODCXC"),
			CODND:      getS(rec, "CODND"),
			CONTACTO:   getS(rec, "CONTACTO"),
			TELEFONOC:  getS(rec, "TELEFONOC"),
			OBSERVA:    getS(rec, "OBSERVA"),
			TPLAZO:     getF(rec, "TPLAZO"),
			POR_IMP:    getF(rec, "POR_IMP"),
			POR_DST1:   getF(rec, "POR_DST1"),
			POR_DST2:   getF(rec, "POR_DST2"),
			POR_DST3:   getF(rec, "POR_DST3"),
			POR_DSTPP:  getF(rec, "POR_DSTPP"),
			COD_BAN1:   getS(rec, "COD_BAN1"),
			COD_BAN2:   getS(rec, "COD_BAN2"),
			COD_BAN3:   getS(rec, "COD_BAN3"),
			COD_BAN4:   getS(rec, "COD_BAN4"),
			COD_BAN5:   getS(rec, "COD_BAN5"),
			COD_BAN6:   getS(rec, "COD_BAN6"),
			CTABAN1:    getS(rec, "CTABAN1"),
			CTABAN2:    getS(rec, "CTABAN2"),
			CTABAN3:    getS(rec, "CTABAN3"),
			CTABAN4:    getS(rec, "CTABAN4"),
			CTABAN5:    getS(rec, "CTABAN5"),
			CTABAN6:    getS(rec, "CTABAN6"),
			DEPNOMBRE1: getS(rec, "DEPNOMBRE1"),
			DEPNOMBRE2: getS(rec, "DEPNOMBRE2"),
			DEPNOMBRE3: getS(rec, "DEPNOMBRE3"),
			DEPNOMBRE4: getS(rec, "DEPNOMBRE4"),
			DEPNOMBRE5: getS(rec, "DEPNOMBRE5"),
			DEPNOMBRE6: getS(rec, "DEPNOMBRE6"),
			PAG_WEB:    getS(rec, "PAG_WEB"),
			EMAILP:     getS(rec, "EMAILP"),
			EMAILC:     getS(rec, "EMAILC"),
			EMAILR:     getS(rec, "EMAILR"),
			REPRE:      getS(rec, "REPRE"),
			TELEFONOR:  getS(rec, "TELEFONOR"),
			POR_DSTP2:  getF(rec, "POR_DSTP2"),
			POR_DSTP3:  getF(rec, "POR_DSTP3"),
			LDEPOSITAR: getB(rec, "LDEPOSITAR"),
			DETAFIJO:   getS(rec, "DETAFIJO"),
			NSUJETO:    getF(rec, "NSUJETO"),
			ZONAP:      getS(rec, "ZONAP"),
			CODACTECO:  getS(rec, "CODACTECO"),
			DOLLARP:    getS(rec, "DOLLARP"),
			ID_PROVEE:  getI(rec, "ID_PROVEE"),
			CTIPOTASA:  getS(rec, "CTIPOTASA"),
			AREAAD:     getS(rec, "AREAAD"),
			TELEFONOAD: getS(rec, "TELEFONOAD"),
			CONTACTOAD: getS(rec, "CONTACTOAD"),
			EMAILAD:    getS(rec, "EMAILAD"),
			COD_BANT1:  getS(rec, "COD_BANT1"),
			NOMCLIT1:   getS(rec, "NOMCLIT1"),
			CEDCLIT1:   getS(rec, "CEDCLIT1"),
			CCUENTAT1:  getS(rec, "CCUENTAT1"),
			COD_BANT2:  getS(rec, "COD_BANT2"),
			NOMCLIT2:   getS(rec, "NOMCLIT2"),
			CEDCLIT2:   getS(rec, "CEDCLIT2"),
			CCUENTAT2:  getS(rec, "CCUENTAT2"),
			TIPO_CTA1:  getS(rec, "TIPO_CTA1"),
			TIPO_CTA2:  getS(rec, "TIPO_CTA2"),
			TIPO_CTA3:  getS(rec, "TIPO_CTA3"),
			TIPO_CTA4:  getS(rec, "TIPO_CTA4"),
			TIPO_CTA1T: getS(rec, "TIPO_CTA1T"),
			TIPO_CTA2T: getS(rec, "TIPO_CTA2T"),
			LREFCAMBI:  getB(rec, "LREFCAMBI"),
		})
	}
	return regs, nil
}

func InsertarMPRO(db *sql.DB, regs []MPRORegistro) (int, error) {
	if len(regs) == 0 {
		return 0, nil
	}
	const batchSize = 18
	total := 0
	cols := "CODCLI,ZONA,NOMCLI,CEDCLI,NITCLI,DIRCLI1,DIRCLI2,DIRTRAB1,DIRTRAB2,CIUDAD1,CIUDAD2,TELEFONO,TELFTRAB,ATENCION,REFERIDO,CEDFIAD,RIM,NOMFIAD,DIRFIAD1,DIRFIAD2,DIRFIAD3,DIRFIAD4,CIUDADF1,CIUDADF2,TELFFIAD1,TELFFIAD2,SALCLI,PLAZO,LCREDCLI,CODCUEN,CODEFEC,CODCAJ,SERIE,NATJU,DOLLAR,INIACT,REPLEG,FECREG,ACTCOM,LOGMON,ESTADO1,PAIS,CLOCK,IM,SIN_RET,TBANCARIA,PORCENT,CODCXC,CODND,CONTACTO,TELEFONOC,OBSERVA,TPLAZO,POR_IMP,POR_DST1,POR_DST2,POR_DST3,POR_DSTPP,COD_BAN1,COD_BAN2,COD_BAN3,COD_BAN4,COD_BAN5,COD_BAN6,CTABAN1,CTABAN2,CTABAN3,CTABAN4,CTABAN5,CTABAN6,DEPNOMBRE1,DEPNOMBRE2,DEPNOMBRE3,DEPNOMBRE4,DEPNOMBRE5,DEPNOMBRE6,PAG_WEB,EMAILP,EMAILC,EMAILR,REPRE,TELEFONOR,POR_DSTP2,POR_DSTP3,LDEPOSITAR,DETAFIJO,NSUJETO,ZONAP,CODACTECO,DOLLARP,ID_PROVEE,CTIPOTASA,AREAAD,TELEFONOAD,CONTACTOAD,EMAILAD,COD_BANT1,NOMCLIT1,CEDCLIT1,CCUENTAT1,COD_BANT2,NOMCLIT2,CEDCLIT2,CCUENTAT2,TIPO_CTA1,TIPO_CTA2,TIPO_CTA3,TIPO_CTA4,TIPO_CTA1T,TIPO_CTA2T,LREFCAMBI"
	singlePH := "(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)"

	for i := 0; i < len(regs); i += batchSize {
		end := i + batchSize
		if end > len(regs) {
			end = len(regs)
		}
		chunk := regs[i:end]

		placeholders := make([]string, len(chunk))
		args := make([]interface{}, 0, len(chunk)*111)
		for j, r := range chunk {
			placeholders[j] = singlePH
			args = append(args,
				r.CODCLI, r.ZONA, r.NOMCLI, r.CEDCLI, r.NITCLI, r.DIRCLI1, r.DIRCLI2, r.DIRTRAB1, r.DIRTRAB2, r.CIUDAD1,
				r.CIUDAD2, r.TELEFONO, r.TELFTRAB, r.ATENCION, r.REFERIDO, r.CEDFIAD, r.RIM, r.NOMFIAD, r.DIRFIAD1, r.DIRFIAD2,
				r.DIRFIAD3, r.DIRFIAD4, r.CIUDADF1, r.CIUDADF2, r.TELFFIAD1, r.TELFFIAD2, r.SALCLI, r.PLAZO, r.LCREDCLI, r.CODCUEN,
				r.CODEFEC, r.CODCAJ, r.SERIE, r.NATJU, r.DOLLAR, r.INIACT, r.REPLEG, r.FECREG, r.ACTCOM, r.LOGMON,
				r.ESTADO1, r.PAIS, r.CLOCK, r.IM, r.SIN_RET, r.TBANCARIA, r.PORCENT, r.CODCXC, r.CODND, r.CONTACTO,
				r.TELEFONOC, r.OBSERVA, r.TPLAZO, r.POR_IMP, r.POR_DST1, r.POR_DST2, r.POR_DST3, r.POR_DSTPP, r.COD_BAN1, r.COD_BAN2,
				r.COD_BAN3, r.COD_BAN4, r.COD_BAN5, r.COD_BAN6, r.CTABAN1, r.CTABAN2, r.CTABAN3, r.CTABAN4, r.CTABAN5, r.CTABAN6,
				r.DEPNOMBRE1, r.DEPNOMBRE2, r.DEPNOMBRE3, r.DEPNOMBRE4, r.DEPNOMBRE5, r.DEPNOMBRE6, r.PAG_WEB, r.EMAILP, r.EMAILC, r.EMAILR,
				r.REPRE, r.TELEFONOR, r.POR_DSTP2, r.POR_DSTP3, r.LDEPOSITAR, r.DETAFIJO, r.NSUJETO, r.ZONAP, r.CODACTECO, r.DOLLARP,
				r.ID_PROVEE, r.CTIPOTASA, r.AREAAD, r.TELEFONOAD, r.CONTACTOAD, r.EMAILAD, r.COD_BANT1, r.NOMCLIT1, r.CEDCLIT1, r.CCUENTAT1,
				r.COD_BANT2, r.NOMCLIT2, r.CEDCLIT2, r.CCUENTAT2, r.TIPO_CTA1, r.TIPO_CTA2, r.TIPO_CTA3, r.TIPO_CTA4, r.TIPO_CTA1T, r.TIPO_CTA2T,
				r.LREFCAMBI,
			)
		}

		query := fmt.Sprintf("INSERT INTO MPRO_Datos (%s) VALUES %s", cols, strings.Join(placeholders[:len(chunk)], ","))
		_, err := db.Exec(query, args...)
		if err != nil {
			for _, r := range chunk {
				singleArgs := []interface{}{
					r.CODCLI, r.ZONA, r.NOMCLI, r.CEDCLI, r.NITCLI, r.DIRCLI1, r.DIRCLI2, r.DIRTRAB1, r.DIRTRAB2, r.CIUDAD1,
					r.CIUDAD2, r.TELEFONO, r.TELFTRAB, r.ATENCION, r.REFERIDO, r.CEDFIAD, r.RIM, r.NOMFIAD, r.DIRFIAD1, r.DIRFIAD2,
					r.DIRFIAD3, r.DIRFIAD4, r.CIUDADF1, r.CIUDADF2, r.TELFFIAD1, r.TELFFIAD2, r.SALCLI, r.PLAZO, r.LCREDCLI, r.CODCUEN,
					r.CODEFEC, r.CODCAJ, r.SERIE, r.NATJU, r.DOLLAR, r.INIACT, r.REPLEG, r.FECREG, r.ACTCOM, r.LOGMON,
					r.ESTADO1, r.PAIS, r.CLOCK, r.IM, r.SIN_RET, r.TBANCARIA, r.PORCENT, r.CODCXC, r.CODND, r.CONTACTO,
					r.TELEFONOC, r.OBSERVA, r.TPLAZO, r.POR_IMP, r.POR_DST1, r.POR_DST2, r.POR_DST3, r.POR_DSTPP, r.COD_BAN1, r.COD_BAN2,
					r.COD_BAN3, r.COD_BAN4, r.COD_BAN5, r.COD_BAN6, r.CTABAN1, r.CTABAN2, r.CTABAN3, r.CTABAN4, r.CTABAN5, r.CTABAN6,
					r.DEPNOMBRE1, r.DEPNOMBRE2, r.DEPNOMBRE3, r.DEPNOMBRE4, r.DEPNOMBRE5, r.DEPNOMBRE6, r.PAG_WEB, r.EMAILP, r.EMAILC, r.EMAILR,
					r.REPRE, r.TELEFONOR, r.POR_DSTP2, r.POR_DSTP3, r.LDEPOSITAR, r.DETAFIJO, r.NSUJETO, r.ZONAP, r.CODACTECO, r.DOLLARP,
					r.ID_PROVEE, r.CTIPOTASA, r.AREAAD, r.TELEFONOAD, r.CONTACTOAD, r.EMAILAD, r.COD_BANT1, r.NOMCLIT1, r.CEDCLIT1, r.CCUENTAT1,
					r.COD_BANT2, r.NOMCLIT2, r.CEDCLIT2, r.CCUENTAT2, r.TIPO_CTA1, r.TIPO_CTA2, r.TIPO_CTA3, r.TIPO_CTA4, r.TIPO_CTA1T, r.TIPO_CTA2T,
					r.LREFCAMBI,
				}
				singleQuery := fmt.Sprintf("INSERT INTO MPRO_Datos (%s) VALUES %s", cols, singlePH)
				if _, errSingle := db.Exec(singleQuery, singleArgs...); errSingle == nil {
					total++
				}
			}
		} else {
			total += len(chunk)
		}
		if total%500 == 0 {
			fmt.Printf("\r  MPRO: %d/%d insertados...    ", total, len(regs))
		}
	}
	fmt.Printf("\n")
	return total, nil
}

// ---------- Procesamiento ----------

func ProcesarCxP(db *sql.DB, year int, modo string, tipoEjecucion string) {
	logInfo("CxP: Iniciando proceso año %d, modo=%s", year, modo)
	tStart := time.Now()

	// === PDOC ===
	fmt.Println("\n  === PDOC (Documentos CxP) ===")
	pdocPath := rutaPDOC()
	if _, errStat := os.Stat(pdocPath); errStat != nil {
		fmt.Printf("  PDOC no accesible: %s\n", pdocPath)
		logError("CxP: PDOC no accesible: %v", errStat)
	} else {
		fmt.Printf("  Leyendo PDOC001.DBF...\n")
		pdocRegs, errL := LeerPDOC(pdocPath)
		if errL != nil {
			fmt.Printf("  ERROR leyendo PDOC: %v\n", errL)
			logError("CxP: ERROR leyendo PDOC: %v", errL)
		} else {
			fmt.Printf("  PDOC: %d registros en DBF\n", len(pdocRegs))
			dbPDOC, errC := ConectarSQL_PDOC()
			if errC != nil {
				fmt.Printf("  ERROR conexion PDOC_Tiendas: %v\n", errC)
				logError("CxP: ERROR conexion PDOC_Tiendas: %v", errC)
			} else {
				defer dbPDOC.Close()
				if err := CrearTablaPDOC(dbPDOC); err != nil {
					fmt.Printf("  ERROR creando tabla PDOC: %v\n", errC)
					logError("CxP: ERROR creando tabla PDOC: %v", err)
				} else {
					dedupePDOC := make(map[string]bool)
					rows, _ := dbPDOC.Query("SELECT FNRODOC FROM PDOC_Datos")
					if rows != nil {
						for rows.Next() {
							var n string
							if rows.Scan(&n) == nil {
								dedupePDOC[strings.TrimSpace(n)] = true
							}
						}
						rows.Close()
					}
					fmt.Printf("  PDOC: %d existentes en SQL\n", len(dedupePDOC))

					var nuevos []PDOCRegistro
					for _, r := range pdocRegs {
						if !dedupePDOC[strings.TrimSpace(r.FNRODOC)] {
							nuevos = append(nuevos, r)
						}
					}
					fmt.Printf("  PDOC: %d nuevos a insertar\n", len(nuevos))

					if len(nuevos) > 0 {
						ins, err := InsertarPDOC(dbPDOC, nuevos)
						if err != nil {
							fmt.Printf("  ERROR insertando PDOC: %v\n", err)
							logError("CxP: ERROR insertando PDOC: %v", err)
						} else {
							fmt.Printf("  PDOC: %d insertados OK\n", ins)
							logInfo("CxP: PDOC %d registros DBF, %d existentes, %d insertados", len(pdocRegs), len(dedupePDOC), ins)
							InsertarH_Tiendas(db, "PDOC", year, 0, len(pdocRegs), len(dedupePDOC), ins, "OK", int(time.Since(tStart).Seconds()), modo, tipoEjecucion, "CxP")
						}
					} else {
						fmt.Println("  PDOC: Sin cambios (todo sincronizado)")
						logInfo("CxP: PDOC sin cambios, %d registros", len(pdocRegs))
						InsertarH_Tiendas(db, "PDOC", year, 0, len(pdocRegs), len(dedupePDOC), 0, "SALTADO", int(time.Since(tStart).Seconds()), modo, tipoEjecucion, "CxP")
					}
				}
			}
		}
	}

	// === PTRA ===
	fmt.Println("\n  === PTRA (Pagos/Traslados CxP) ===")
	ptraPath := rutaPTRA()
	if _, errStat := os.Stat(ptraPath); errStat != nil {
		fmt.Printf("  PTRA no accesible: %s\n", ptraPath)
		logError("CxP: PTRA no accesible: %v", errStat)
	} else {
		fmt.Printf("  Leyendo PTRA001.DBF...\n")
		ptraRegs, errL := LeerPTRA(ptraPath)
		if errL != nil {
			fmt.Printf("  ERROR leyendo PTRA: %v\n", errL)
			logError("CxP: ERROR leyendo PTRA: %v", errL)
		} else {
			fmt.Printf("  PTRA: %d registros en DBF\n", len(ptraRegs))
			dbPTRA, errC := ConectarSQL_PTRA()
			if errC != nil {
				fmt.Printf("  ERROR conexion PTRA_Tiendas: %v\n", errC)
				logError("CxP: ERROR conexion PTRA_Tiendas: %v", errC)
			} else {
				defer dbPTRA.Close()
				if err := CrearTablaPTRA(dbPTRA); err != nil {
					fmt.Printf("  ERROR creando tabla PTRA: %v\n", err)
					logError("CxP: ERROR creando tabla PTRA: %v", err)
				} else {
					dedupePTRA := make(map[string]bool)
					rows, _ := dbPTRA.Query("SELECT TNRODOC, TNROMOV FROM PTRA_Datos")
					if rows != nil {
						for rows.Next() {
							var n, m string
							if rows.Scan(&n, &m) == nil {
								dedupePTRA[strings.TrimSpace(n)+"|"+strings.TrimSpace(m)] = true
							}
						}
						rows.Close()
					}
					fmt.Printf("  PTRA: %d existentes en SQL\n", len(dedupePTRA))

					var nuevos []PTRARegistro
					for _, r := range ptraRegs {
						key := strings.TrimSpace(r.TNRODOC) + "|" + strings.TrimSpace(r.TNROMOV)
						if !dedupePTRA[key] {
							nuevos = append(nuevos, r)
						}
					}
					fmt.Printf("  PTRA: %d nuevos a insertar\n", len(nuevos))

					if len(nuevos) > 0 {
						ins, err := InsertarPTRA(dbPTRA, nuevos)
						if err != nil {
							fmt.Printf("  ERROR insertando PTRA: %v\n", err)
							logError("CxP: ERROR insertando PTRA: %v", err)
						} else {
							fmt.Printf("  PTRA: %d insertados OK\n", ins)
							logInfo("CxP: PTRA %d registros DBF, %d existentes, %d insertados", len(ptraRegs), len(dedupePTRA), ins)
							InsertarH_Tiendas(db, "PTRA", year, 0, len(ptraRegs), len(dedupePTRA), ins, "OK", int(time.Since(tStart).Seconds()), modo, tipoEjecucion, "CxP")
						}
					} else {
						fmt.Println("  PTRA: Sin cambios (todo sincronizado)")
						logInfo("CxP: PTRA sin cambios, %d registros", len(ptraRegs))
						InsertarH_Tiendas(db, "PTRA", year, 0, len(ptraRegs), len(dedupePTRA), 0, "SALTADO", int(time.Since(tStart).Seconds()), modo, tipoEjecucion, "CxP")
					}
				}
			}
		}
	}

	// === MPRO ===
	fmt.Println("\n  === MPRO (Maestro Proveedores) ===")
	mproPath := rutaMPRO()
	if _, errStat := os.Stat(mproPath); errStat != nil {
		fmt.Printf("  MPRO no accesible: %s\n", mproPath)
		logError("CxP: MPRO no accesible: %v", errStat)
	} else {
		fmt.Printf("  Leyendo MPRO001.DBF...\n")
		mproRegs, errL := LeerMPRO(mproPath)
		if errL != nil {
			fmt.Printf("  ERROR leyendo MPRO: %v\n", errL)
			logError("CxP: ERROR leyendo MPRO: %v", errL)
		} else {
			fmt.Printf("  MPRO: %d registros en DBF\n", len(mproRegs))
			dbMPRO, errC := ConectarSQL_MPRO()
			if errC != nil {
				fmt.Printf("  ERROR conexion MPRO_Tiendas: %v\n", errC)
				logError("CxP: ERROR conexion MPRO_Tiendas: %v", errC)
			} else {
				defer dbMPRO.Close()
				if err := CrearTablaMPRO(dbMPRO); err != nil {
					fmt.Printf("  ERROR creando tabla MPRO: %v\n", err)
					logError("CxP: ERROR creando tabla MPRO: %v", err)
				} else {
					dedupeMPRO := make(map[string]bool)
					rows, _ := dbMPRO.Query("SELECT CODCLI FROM MPRO_Datos")
					if rows != nil {
						for rows.Next() {
							var n string
							if rows.Scan(&n) == nil {
								dedupeMPRO[strings.TrimSpace(n)] = true
							}
						}
						rows.Close()
					}
					fmt.Printf("  MPRO: %d existentes en SQL\n", len(dedupeMPRO))

					var nuevos []MPRORegistro
					for _, r := range mproRegs {
						if !dedupeMPRO[strings.TrimSpace(r.CODCLI)] {
							nuevos = append(nuevos, r)
						}
					}
					fmt.Printf("  MPRO: %d nuevos a insertar\n", len(nuevos))

					if len(nuevos) > 0 {
						ins, err := InsertarMPRO(dbMPRO, nuevos)
						if err != nil {
							fmt.Printf("  ERROR insertando MPRO: %v\n", err)
							logError("CxP: ERROR insertando MPRO: %v", err)
						} else {
							fmt.Printf("  MPRO: %d insertados OK\n", ins)
							logInfo("CxP: MPRO %d registros DBF, %d existentes, %d insertados", len(mproRegs), len(dedupeMPRO), ins)
							InsertarH_Tiendas(db, "MPRO", year, 0, len(mproRegs), len(dedupeMPRO), ins, "OK", int(time.Since(tStart).Seconds()), modo, tipoEjecucion, "CxP")
						}
					} else {
						fmt.Println("  MPRO: Sin cambios (todo sincronizado)")
						logInfo("CxP: MPRO sin cambios, %d registros", len(mproRegs))
						InsertarH_Tiendas(db, "MPRO", year, 0, len(mproRegs), len(dedupeMPRO), 0, "SALTADO", int(time.Since(tStart).Seconds()), modo, tipoEjecucion, "CxP")
					}
				}
			}
		}
	}

	fmt.Printf("\n  CxP listo. Tiempo: %.1f min.\n", time.Since(tStart).Minutes())
	logInfo("CxP: Completado en %.1f min", time.Since(tStart).Minutes())
}
