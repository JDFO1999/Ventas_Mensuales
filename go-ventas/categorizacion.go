package main

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	mssql "github.com/denisenkom/go-mssqldb"
)

// ---------- Modelo MINV001 ----------

type MinvRegistro struct {
	NROITEM    string
	DESCRIP    string
	REFERENCIA string
	GRUPLIN    string
	ITEMREE    string
	UNIDAD     string
	RUBRO      string
	CANTEMP    float64
	PESO       float64
	UNIDADNUM  float64
	CLASE      string
	CNTPROD    float64
	STOCKMI    float64
	STOCKMA    float64
	ULTPREC    float64
	PRECPRO    float64
	C_REPOSI   float64
	COS_DOL    float64
	COS_DOLP   float64
	FACTOR     float64
	FACTOR_V   float64
	MONEDA     float64
	CODPROV    string
	FECHAING   sql.NullTime
	OPERADOR   string
	UTIL_DOL1  float64
	UTIL_DOL2  float64
	UTIL_DOL3  float64
	PVP1_DOL   float64
	PVP2_DOL   float64
	PVP3_DOL   float64
	PVP        float64
	POR_PVP    float64
	COMISIO    float64
	TANTOS     float64
	CUANTOS    float64
	PORGASTO_0 float64
	PORGASTO_1 float64
	PORGASTO_2 float64
	PORGASTO_3 float64
	COSTO_INDE float64
	PRODUCCION bool
	DESPOSTE   bool
	COMPUESTO  bool
	SERVICIO   bool
	PROMO      bool
	OFERTA     bool
	APLICOMI   bool
	CILINDRO   bool
	DESACT     bool
	REGULADO   bool
	EXENTO     bool
	SERIALVEN  bool
	LCONTROLAP bool
	LPESADO    bool
	RECIPE     bool
	LDESCARGA  bool
	LIMPORTADO bool
	LPASSW     bool
	LNOTCLAVE  bool
	LTASABC    bool
	LCONTEOPOS bool
	NLIMITECOM int
	NDIASCOM   int
	CANTEXI    float64
	MARCA      string
	OBSERVA    string
	DEP        string
	FAMILIA    string
	CODVEND    string
	COMIVEND   float64
	SUJETOCOMI bool
	IVA        float64
	KIT_PADRE  string
	KIT_PRODUC string
	ACTCOSTO   bool
	PROCEDEN   string
	MANEJO     float64
	KIT_FACTOR float64
	NODESC     float64
	EDITA_PCN  string
	COD_VENTA  string
	COD_COSTO  string
	COD_INVEN  string
	ACTBONIF   float64
	ACTFAC     float64
	ACTPAGO    float64
	BONMER     float64
	DESFAC     float64
	DESPP      float64
	UTILPVP    float64
	COSTUNIT   float64
	IMP18      float64
	COSTANT    float64
	COSTEMPQ   float64
	COSTBULTO  float64
	BANDEJA    float64
	ENVOPLAST  float64
	ETIQUETAS  float64
	COSTOEMP   float64
	FACCEMP    float64
	MAXDESC1   float64
	MAXDESC2   float64
	PROMO01    string
	PROMO02    string
	PROMO03    string
	NAMEPROV_1 string
	NAMEPROV_2 string
	NAMEPROV_3 string
	NAMEPROV_4 string
	NAMEPROV_5 string
	COSPROV_1  float64
	COSPROV_2  float64
	COSPROV_3  float64
	COSPROV_4  float64
	COSPROV_5  float64
	VTAMINIMA  float64
	APLIFLETE  float64
	NFERTILIZ  float64
	MODIFICADO bool
	IDMONEDA   float64
	RUTAFOTO   string
	NLOTESFECV float64
	NSERIALES  float64
	FLETE      float64
	FECHAVEN   sql.NullTime
	LPIEZA     bool
	NMAYOR     float64
	CFORMMAYOR string
	FACTOR_COP float64
	LWEB       bool
	LDESCONTI  bool
	NVACUNA    float64
	LPERCIBIDO float64
	SUBFAMILIA string
	LTRASTES   bool
	REF_PVP1   float64
	REF_PVP2   float64
	REF_PVP3   float64
	LREFCAMBI  bool
	LLOCKPVP   bool
}

var minvCols = []string{
	"NROITEM", "DESCRIP", "REFERENCIA", "GRUPLIN", "ITEMREE", "UNIDAD", "RUBRO",
	"CANTEMP", "PESO", "UNIDADNUM", "CLASE", "CNTPROD", "STOCKMI", "STOCKMA",
	"ULTPREC", "PRECPRO", "C_REPOSI", "COS_DOL", "COS_DOLP", "FACTOR", "FACTOR_V",
	"MONEDA", "CODPROV", "FECHAING", "OPERADOR",
	"UTIL_DOL1", "UTIL_DOL2", "UTIL_DOL3", "PVP1_DOL", "PVP2_DOL", "PVP3_DOL",
	"PVP", "POR_PVP", "COMISIO", "TANTOS", "CUANTOS",
	"PORGASTO_0", "PORGASTO_1", "PORGASTO_2", "PORGASTO_3", "COSTO_INDE",
	"PRODUCCION", "DESPOSTE", "COMPUESTO", "SERVICIO", "PROMO", "OFERTA",
	"APLICOMI", "CILINDRO", "DESACT", "REGULADO", "EXENTO", "SERIALVEN",
	"LCONTROLAP", "LPESADO", "RECIPE", "LDESCARGA", "LIMPORTADO", "LPASSW",
	"LNOTCLAVE", "LTASABC", "LCONTEOPOS", "NLIMITECOM", "NDIASCOM", "CANTEXI",
	"MARCA", "OBSERVA", "DEP", "FAMILIA", "CODVEND", "COMIVEND", "SUJETOCOMI",
	"IVA", "KIT_PADRE", "KIT_PRODUC", "ACTCOSTO", "PROCEDEN", "MANEJO", "KIT_FACTOR",
	"NODESC", "EDITA_PCN", "COD_VENTA", "COD_COSTO", "COD_INVEN",
	"ACTBONIF", "ACTFAC", "ACTPAGO", "BONMER", "DESFAC", "DESPP", "UTILPVP",
	"COSTUNIT", "IMP18", "COSTANT", "COSTEMPQ", "COSTBULTO", "BANDEJA", "ENVOPLAST",
	"ETIQUETAS", "COSTOEMP", "FACCEMP", "MAXDESC1", "MAXDESC2",
	"PROMO01", "PROMO02", "PROMO03",
	"NAMEPROV_1", "NAMEPROV_2", "NAMEPROV_3", "NAMEPROV_4", "NAMEPROV_5",
	"COSPROV_1", "COSPROV_2", "COSPROV_3", "COSPROV_4", "COSPROV_5",
	"VTAMINIMA", "APLIFLETE", "NFERTILIZ", "MODIFICADO", "IDMONEDA", "RUTAFOTO",
	"NLOTESFECV", "NSERIALES", "FLETE", "FECHAVEN", "LPIEZA", "NMAYOR", "CFORMMAYOR",
	"FACTOR_COP", "LWEB", "LDESCONTI", "NVACUNA", "LPERCIBIDO", "SUBFAMILIA",
	"LTRASTES", "REF_PVP1", "REF_PVP2", "REF_PVP3", "LREFCAMBI", "LLOCKPVP",
}

// ---------- Ruta DBF ----------

func rutaMINVDBF() string {
	return `\\10.10.10.30\Sistema\aBC-Soft\Data\E001\Inv`
}

// ---------- Lectura DBF ----------

func LeerMINV001() ([]MinvRegistro, error) {
	path := rutaMINVDBF() + `\MINV001.DBF`
	dbf, err := OpenDBF(path)
	if err != nil {
		return nil, fmt.Errorf("abriendo MINV001: %v", err)
	}
	defer dbf.Close()

	getS := func(rec []byte, name string) string {
		if i := dbf.FieldIndex(name); i >= 0 {
			return dbf.GetString(rec, dbf.Fields[i])
		}
		return ""
	}
	getN := func(rec []byte, name string) float64 {
		if i := dbf.FieldIndex(name); i >= 0 {
			return dbf.GetFloat(rec, dbf.Fields[i])
		}
		return 0
	}
	getI := func(rec []byte, name string) int {
		if i := dbf.FieldIndex(name); i >= 0 {
			return int(dbf.GetFloat(rec, dbf.Fields[i]))
		}
		return 0
	}
	getB := func(rec []byte, name string) bool {
		if i := dbf.FieldIndex(name); i >= 0 {
			return dbf.GetBool(rec, dbf.Fields[i])
		}
		return false
	}
	getD := func(rec []byte, name string) sql.NullTime {
		if i := dbf.FieldIndex(name); i >= 0 {
			t, ok := dbf.GetDate(rec, dbf.Fields[i])
			if ok && !t.IsZero() && t.Year() >= 1753 {
				return sql.NullTime{Time: t, Valid: true}
			}
		}
		return sql.NullTime{}
	}

	var regs []MinvRegistro
	for i := 0; i < dbf.NumRecords; i++ {
		rec, e := dbf.ReadRecord(i)
		if e != nil || len(rec) == 0 || rec[0] == 0x2A {
			continue
		}
		nro := strings.TrimSpace(getS(rec, "NROITEM"))
		if nro == "" {
			continue
		}
		regs = append(regs, MinvRegistro{
			NROITEM: nro, DESCRIP: strings.TrimSpace(getS(rec, "DESCRIP")),
			REFERENCIA: strings.TrimSpace(getS(rec, "REFERENCIA")),
			GRUPLIN: strings.TrimSpace(getS(rec, "GRUPLIN")),
			ITEMREE: strings.TrimSpace(getS(rec, "ITEMREE")),
			UNIDAD: strings.TrimSpace(getS(rec, "UNIDAD")),
			RUBRO: strings.TrimSpace(getS(rec, "RUBRO")),
			CANTEMP: getN(rec, "CANTEMP"), PESO: getN(rec, "PESO"),
			UNIDADNUM: getN(rec, "UNIDADNUM"), CLASE: strings.TrimSpace(getS(rec, "CLASE")),
			CNTPROD: getN(rec, "CNTPROD"), STOCKMI: getN(rec, "STOCKMI"),
			STOCKMA: getN(rec, "STOCKMA"), ULTPREC: getN(rec, "ULTPREC"),
			PRECPRO: getN(rec, "PRECPRO"), C_REPOSI: getN(rec, "C_REPOSI"),
			COS_DOL: getN(rec, "COS_DOL"), COS_DOLP: getN(rec, "COS_DOLP"),
			FACTOR: getN(rec, "FACTOR"), FACTOR_V: getN(rec, "FACTOR_V"),
			MONEDA: getN(rec, "MONEDA"), CODPROV: strings.TrimSpace(getS(rec, "CODPROV")),
			FECHAING: getD(rec, "FECHAING"), OPERADOR: strings.TrimSpace(getS(rec, "OPERADOR")),
			UTIL_DOL1: getN(rec, "UTIL_DOL1"), UTIL_DOL2: getN(rec, "UTIL_DOL2"),
			UTIL_DOL3: getN(rec, "UTIL_DOL3"),
			PVP1_DOL: getN(rec, "PVP1_DOL"), PVP2_DOL: getN(rec, "PVP2_DOL"),
			PVP3_DOL: getN(rec, "PVP3_DOL"),
			PVP: getN(rec, "PVP"), POR_PVP: getN(rec, "POR_PVP"),
			COMISIO: getN(rec, "COMISIO"), TANTOS: getN(rec, "TANTOS"),
			CUANTOS: getN(rec, "CUANTOS"),
			PORGASTO_0: getN(rec, "PORGASTO_0"), PORGASTO_1: getN(rec, "PORGASTO_1"),
			PORGASTO_2: getN(rec, "PORGASTO_2"), PORGASTO_3: getN(rec, "PORGASTO_3"),
			COSTO_INDE: getN(rec, "COSTO_INDE"),
			PRODUCCION: getB(rec, "PRODUCCION"), DESPOSTE: getB(rec, "DESPOSTE"),
			COMPUESTO: getB(rec, "COMPUESTO"), SERVICIO: getB(rec, "SERVICIO"),
			PROMO: getB(rec, "PROMO"), OFERTA: getB(rec, "OFERTA"),
			APLICOMI: getB(rec, "APLICOMI"), CILINDRO: getB(rec, "CILINDRO"),
			DESACT: getB(rec, "DESACT"), REGULADO: getB(rec, "REGULADO"),
			EXENTO: getB(rec, "EXENTO"), SERIALVEN: getB(rec, "SERIALVEN"),
			LCONTROLAP: getB(rec, "LCONTROLAP"), LPESADO: getB(rec, "LPESADO"),
			RECIPE: getB(rec, "RECIPE"), LDESCARGA: getB(rec, "LDESCARGA"),
			LIMPORTADO: getB(rec, "LIMPORTADO"), LPASSW: getB(rec, "LPASSW"),
			LNOTCLAVE: getB(rec, "LNOTCLAVE"), LTASABC: getB(rec, "LTASABC"),
			LCONTEOPOS: getB(rec, "LCONTEOPOS"),
			NLIMITECOM: getI(rec, "NLIMITECOM"), NDIASCOM: getI(rec, "NDIASCOM"),
			CANTEXI: getN(rec, "CANTEXI"),
			MARCA: strings.TrimSpace(getS(rec, "MARCA")),
			OBSERVA: strings.TrimSpace(getS(rec, "OBSERVA")),
			DEP: strings.TrimSpace(getS(rec, "DEP")),
			FAMILIA: strings.TrimSpace(getS(rec, "FAMILIA")),
			CODVEND: strings.TrimSpace(getS(rec, "CODVEND")),
			COMIVEND: getN(rec, "COMIVEND"), SUJETOCOMI: getB(rec, "SUJETOCOMI"),
			IVA: getN(rec, "IVA"),
			KIT_PADRE: strings.TrimSpace(getS(rec, "KIT_PADRE")),
			KIT_PRODUC: strings.TrimSpace(getS(rec, "KIT_PRODUC")),
			ACTCOSTO: getB(rec, "ACTCOSTO"),
			PROCEDEN: strings.TrimSpace(getS(rec, "PROCEDEN")),
			MANEJO: getN(rec, "MANEJO"), KIT_FACTOR: getN(rec, "KIT_FACTOR"),
			NODESC: getN(rec, "NODESC"),
			EDITA_PCN: strings.TrimSpace(getS(rec, "EDITA_PCN")),
			COD_VENTA: strings.TrimSpace(getS(rec, "COD_VENTA")),
			COD_COSTO: strings.TrimSpace(getS(rec, "COD_COSTO")),
			COD_INVEN: strings.TrimSpace(getS(rec, "COD_INVEN")),
			ACTBONIF: getN(rec, "ACTBONIF"), ACTFAC: getN(rec, "ACTFAC"),
			ACTPAGO: getN(rec, "ACTPAGO"), BONMER: getN(rec, "BONMER"),
			DESFAC: getN(rec, "DESFAC"), DESPP: getN(rec, "DESPP"),
			UTILPVP: getN(rec, "UTILPVP"), COSTUNIT: getN(rec, "COSTUNIT"),
			IMP18: getN(rec, "IMP18"), COSTANT: getN(rec, "COSTANT"),
			COSTEMPQ: getN(rec, "COSTEMPQ"), COSTBULTO: getN(rec, "COSTBULTO"),
			BANDEJA: getN(rec, "BANDEJA"), ENVOPLAST: getN(rec, "ENVOPLAST"),
			ETIQUETAS: getN(rec, "ETIQUETAS"), COSTOEMP: getN(rec, "COSTOEMP"),
			FACCEMP: getN(rec, "FACCEMP"),
			MAXDESC1: getN(rec, "MAXDESC1"), MAXDESC2: getN(rec, "MAXDESC2"),
			PROMO01: strings.TrimSpace(getS(rec, "PROMO01")),
			PROMO02: strings.TrimSpace(getS(rec, "PROMO02")),
			PROMO03: strings.TrimSpace(getS(rec, "PROMO03")),
			NAMEPROV_1: strings.TrimSpace(getS(rec, "NAMEPROV_1")),
			NAMEPROV_2: strings.TrimSpace(getS(rec, "NAMEPROV_2")),
			NAMEPROV_3: strings.TrimSpace(getS(rec, "NAMEPROV_3")),
			NAMEPROV_4: strings.TrimSpace(getS(rec, "NAMEPROV_4")),
			NAMEPROV_5: strings.TrimSpace(getS(rec, "NAMEPROV_5")),
			COSPROV_1: getN(rec, "COSPROV_1"), COSPROV_2: getN(rec, "COSPROV_2"),
			COSPROV_3: getN(rec, "COSPROV_3"), COSPROV_4: getN(rec, "COSPROV_4"),
			COSPROV_5: getN(rec, "COSPROV_5"),
			VTAMINIMA: getN(rec, "VTAMINIMA"),
			APLIFLETE: getN(rec, "APLIFLETE"), NFERTILIZ: getN(rec, "NFERTILIZ"),
			MODIFICADO: getB(rec, "MODIFICADO"), IDMONEDA: getN(rec, "IDMONEDA"),
			RUTAFOTO: strings.TrimSpace(getS(rec, "RUTAFOTO")),
			NLOTESFECV: getN(rec, "NLOTESFECV"), NSERIALES: getN(rec, "NSERIALES"),
			FLETE: getN(rec, "FLETE"), FECHAVEN: getD(rec, "FECHAVEN"),
			LPIEZA: getB(rec, "LPIEZA"), NMAYOR: getN(rec, "NMAYOR"),
			CFORMMAYOR: strings.TrimSpace(getS(rec, "CFORMMAYOR")),
			FACTOR_COP: getN(rec, "FACTOR_COP"),
			LWEB: getB(rec, "LWEB"), LDESCONTI: getB(rec, "LDESCONTI"),
			NVACUNA: getN(rec, "NVACUNA"), LPERCIBIDO: getN(rec, "LPERCIBIDO"),
			SUBFAMILIA: strings.TrimSpace(getS(rec, "SUBFAMILIA")),
			LTRASTES: getB(rec, "LTRASTES"),
			REF_PVP1: getN(rec, "REF_PVP1"), REF_PVP2: getN(rec, "REF_PVP2"),
			REF_PVP3: getN(rec, "REF_PVP3"),
			LREFCAMBI: getB(rec, "LREFCAMBI"), LLOCKPVP: getB(rec, "LLOCKPVP"),
		})
	}
	return regs, nil
}

// ---------- SQL ----------

func CrearBaseAKOP(db *sql.DB) error {
	_, err := db.Exec(`IF DB_ID('AKOP') IS NULL BEGIN CREATE DATABASE AKOP END`)
	return err
}

func ConectarSQL_AKOP() (*sql.DB, error) {
	connStr := fmt.Sprintf("server=%s;database=AKOP;user id=%s;password=%s;encrypt=disable;connection timeout=30;timeout=900",
		SQLServer, SQLUser, SQLPassword)
	db, err := sql.Open("mssql", connStr)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(6)
	return db, nil
}

func CrearTablaAKOP_MINV(db *sql.DB) error {
	_, err := db.Exec(`
		IF OBJECT_ID('AKOP_MINV','U') IS NULL
		CREATE TABLE AKOP_MINV (
			NROITEM      VARCHAR(15) NOT NULL,
			DESCRIP      VARCHAR(120) NULL,
			REFERENCIA   VARCHAR(15) NULL,
			GRUPLIN      VARCHAR(6) NULL,
			ITEMREE      VARCHAR(15) NULL,
			UNIDAD       VARCHAR(8) NULL,
			RUBRO        VARCHAR(4) NULL,
			CANTEMP      DECIMAL(12,4) NULL,
			PESO         DECIMAL(12,4) NULL,
			UNIDADNUM    DECIMAL(12,4) NULL,
			CLASE        VARCHAR(1) NULL,
			CNTPROD      DECIMAL(14,2) NULL,
			STOCKMI      DECIMAL(13,4) NULL,
			STOCKMA      DECIMAL(13,4) NULL,
			ULTPREC      DECIMAL(18,4) NULL,
			PRECPRO      DECIMAL(18,4) NULL,
			C_REPOSI     DECIMAL(18,4) NULL,
			COS_DOL      DECIMAL(20,8) NULL,
			COS_DOLP     DECIMAL(20,8) NULL,
			FACTOR       DECIMAL(18,4) NULL,
			FACTOR_V     DECIMAL(18,4) NULL,
			MONEDA       DECIMAL(15,4) NULL,
			CODPROV      VARCHAR(10) NULL,
			FECHAING     DATE NULL,
			OPERADOR     VARCHAR(20) NULL,
			UTIL_DOL1    DECIMAL(10,2) NULL,
			UTIL_DOL2    DECIMAL(10,2) NULL,
			UTIL_DOL3    DECIMAL(10,2) NULL,
			PVP1_DOL     DECIMAL(16,3) NULL,
			PVP2_DOL     DECIMAL(16,3) NULL,
			PVP3_DOL     DECIMAL(16,3) NULL,
			PVP          DECIMAL(18,4) NULL,
			POR_PVP      DECIMAL(16,2) NULL,
			COMISIO      DECIMAL(8,3) NULL,
			TANTOS       DECIMAL(10,3) NULL,
			CUANTOS      DECIMAL(10,3) NULL,
			PORGASTO_0   DECIMAL(10,3) NULL,
			PORGASTO_1   DECIMAL(10,3) NULL,
			PORGASTO_2   DECIMAL(10,3) NULL,
			PORGASTO_3   DECIMAL(10,3) NULL,
			COSTO_INDE   DECIMAL(18,3) NULL,
			PRODUCCION   BIT NULL,
			DESPOSTE     BIT NULL,
			COMPUESTO    BIT NULL,
			SERVICIO     BIT NULL,
			PROMO        BIT NULL,
			OFERTA       BIT NULL,
			APLICOMI     BIT NULL,
			CILINDRO     BIT NULL,
			DESACT       BIT NULL,
			REGULADO     BIT NULL,
			EXENTO       BIT NULL,
			SERIALVEN    BIT NULL,
			LCONTROLAP   BIT NULL,
			LPESADO      BIT NULL,
			RECIPE       BIT NULL,
			LDESCARGA    BIT NULL,
			LIMPORTADO   BIT NULL,
			LPASSW       BIT NULL,
			LNOTCLAVE    BIT NULL,
			LTASABC      BIT NULL,
			LCONTEOPOS   BIT NULL,
			NLIMITECOM   INT NULL,
			NDIASCOM     INT NULL,
			CANTEXI      DECIMAL(15,4) NULL,
			MARCA        VARCHAR(30) NULL,
			OBSERVA      VARCHAR(60) NULL,
			DEP          VARCHAR(2) NULL,
			FAMILIA      VARCHAR(2) NULL,
			CODVEND      VARCHAR(2) NULL,
			COMIVEND     DECIMAL(8,2) NULL,
			SUJETOCOMI   BIT NULL,
			IVA          DECIMAL(13,4) NULL,
			KIT_PADRE    VARCHAR(15) NULL,
			KIT_PRODUC   VARCHAR(15) NULL,
			ACTCOSTO     BIT NULL,
			PROCEDEN     VARCHAR(1) NULL,
			MANEJO       DECIMAL(8,3) NULL,
			KIT_FACTOR   DECIMAL(15,4) NULL,
			NODESC       DECIMAL(1,0) NULL,
			EDITA_PCN    VARCHAR(1) NULL,
			COD_VENTA    VARCHAR(6) NULL,
			COD_COSTO    VARCHAR(6) NULL,
			COD_INVEN    VARCHAR(6) NULL,
			ACTBONIF     DECIMAL(1,0) NULL,
			ACTFAC       DECIMAL(1,0) NULL,
			ACTPAGO      DECIMAL(1,0) NULL,
			BONMER       DECIMAL(6,2) NULL,
			DESFAC       DECIMAL(6,2) NULL,
			DESPP        DECIMAL(6,2) NULL,
			UTILPVP      DECIMAL(6,2) NULL,
			COSTUNIT     DECIMAL(16,2) NULL,
			IMP18        DECIMAL(6,2) NULL,
			COSTANT      DECIMAL(16,2) NULL,
			COSTEMPQ     DECIMAL(16,2) NULL,
			COSTBULTO    DECIMAL(16,2) NULL,
			BANDEJA      DECIMAL(16,2) NULL,
			ENVOPLAST    DECIMAL(16,2) NULL,
			ETIQUETAS    DECIMAL(16,2) NULL,
			COSTOEMP     DECIMAL(16,2) NULL,
			FACCEMP      DECIMAL(16,2) NULL,
			MAXDESC1     DECIMAL(5,2) NULL,
			MAXDESC2     DECIMAL(5,2) NULL,
			PROMO01      VARCHAR(2) NULL,
			PROMO02      VARCHAR(2) NULL,
			PROMO03      VARCHAR(2) NULL,
			NAMEPROV_1   VARCHAR(200) NULL,
			NAMEPROV_2   VARCHAR(200) NULL,
			NAMEPROV_3   VARCHAR(200) NULL,
			NAMEPROV_4   VARCHAR(200) NULL,
			NAMEPROV_5   VARCHAR(200) NULL,
			COSPROV_1    DECIMAL(16,4) NULL,
			COSPROV_2    DECIMAL(16,4) NULL,
			COSPROV_3    DECIMAL(16,4) NULL,
			COSPROV_4    DECIMAL(16,4) NULL,
			COSPROV_5    DECIMAL(16,4) NULL,
			VTAMINIMA    DECIMAL(16,4) NULL,
			APLIFLETE    DECIMAL(1,0) NULL,
			NFERTILIZ    DECIMAL(1,0) NULL,
			MODIFICADO   BIT NULL,
			IDMONEDA     DECIMAL(10,0) NULL,
			RUTAFOTO     VARCHAR(200) NULL,
			NLOTESFECV   DECIMAL(1,0) NULL,
			NSERIALES    DECIMAL(1,0) NULL,
			FLETE        DECIMAL(18,2) NULL,
			FECHAVEN     DATE NULL,
			LPIEZA       BIT NULL,
			NMAYOR       DECIMAL(18,3) NULL,
			CFORMMAYOR   VARCHAR(60) NULL,
			FACTOR_COP   DECIMAL(18,4) NULL,
			LWEB         BIT NULL,
			LDESCONTI    BIT NULL,
			NVACUNA      DECIMAL(1,0) NULL,
			LPERCIBIDO   DECIMAL(1,0) NULL,
			SUBFAMILIA   VARCHAR(2) NULL,
			LTRASTES     BIT NULL,
			REF_PVP1     DECIMAL(10,2) NULL,
			REF_PVP2     DECIMAL(10,2) NULL,
			REF_PVP3     DECIMAL(10,2) NULL,
			LREFCAMBI    BIT NULL,
			LLOCKPVP     BIT NULL,
			FechaCarga   DATETIME DEFAULT GETDATE(),
			CONSTRAINT PK_AKOP_MINV PRIMARY KEY (NROITEM)
		)`)
	return err
}

// ---------- Insercion ----------

func minvArgs(r MinvRegistro) []interface{} {
	return []interface{}{
		r.NROITEM, r.DESCRIP, r.REFERENCIA, r.GRUPLIN, r.ITEMREE, r.UNIDAD, r.RUBRO,
		r.CANTEMP, r.PESO, r.UNIDADNUM, r.CLASE, r.CNTPROD, r.STOCKMI, r.STOCKMA,
		r.ULTPREC, r.PRECPRO, r.C_REPOSI, r.COS_DOL, r.COS_DOLP, r.FACTOR, r.FACTOR_V,
		r.MONEDA, r.CODPROV, r.FECHAING, r.OPERADOR,
		r.UTIL_DOL1, r.UTIL_DOL2, r.UTIL_DOL3, r.PVP1_DOL, r.PVP2_DOL, r.PVP3_DOL,
		r.PVP, r.POR_PVP, r.COMISIO, r.TANTOS, r.CUANTOS,
		r.PORGASTO_0, r.PORGASTO_1, r.PORGASTO_2, r.PORGASTO_3, r.COSTO_INDE,
		r.PRODUCCION, r.DESPOSTE, r.COMPUESTO, r.SERVICIO, r.PROMO, r.OFERTA,
		r.APLICOMI, r.CILINDRO, r.DESACT, r.REGULADO, r.EXENTO, r.SERIALVEN,
		r.LCONTROLAP, r.LPESADO, r.RECIPE, r.LDESCARGA, r.LIMPORTADO, r.LPASSW,
		r.LNOTCLAVE, r.LTASABC, r.LCONTEOPOS, r.NLIMITECOM, r.NDIASCOM, r.CANTEXI,
		r.MARCA, r.OBSERVA, r.DEP, r.FAMILIA, r.CODVEND, r.COMIVEND, r.SUJETOCOMI,
		r.IVA, r.KIT_PADRE, r.KIT_PRODUC, r.ACTCOSTO, r.PROCEDEN, r.MANEJO, r.KIT_FACTOR,
		r.NODESC, r.EDITA_PCN, r.COD_VENTA, r.COD_COSTO, r.COD_INVEN,
		r.ACTBONIF, r.ACTFAC, r.ACTPAGO, r.BONMER, r.DESFAC, r.DESPP, r.UTILPVP,
		r.COSTUNIT, r.IMP18, r.COSTANT, r.COSTEMPQ, r.COSTBULTO, r.BANDEJA, r.ENVOPLAST,
		r.ETIQUETAS, r.COSTOEMP, r.FACCEMP, r.MAXDESC1, r.MAXDESC2,
		r.PROMO01, r.PROMO02, r.PROMO03,
		r.NAMEPROV_1, r.NAMEPROV_2, r.NAMEPROV_3, r.NAMEPROV_4, r.NAMEPROV_5,
		r.COSPROV_1, r.COSPROV_2, r.COSPROV_3, r.COSPROV_4, r.COSPROV_5,
		r.VTAMINIMA, r.APLIFLETE, r.NFERTILIZ, r.MODIFICADO, r.IDMONEDA, r.RUTAFOTO,
		r.NLOTESFECV, r.NSERIALES, r.FLETE, r.FECHAVEN, r.LPIEZA, r.NMAYOR, r.CFORMMAYOR,
		r.FACTOR_COP, r.LWEB, r.LDESCONTI, r.NVACUNA, r.LPERCIBIDO, r.SUBFAMILIA,
		r.LTRASTES, r.REF_PVP1, r.REF_PVP2, r.REF_PVP3, r.LREFCAMBI, r.LLOCKPVP,
	}
}

func insertarAKOPMinv(db *sql.DB, regs []MinvRegistro) (int, error) {
	if len(regs) == 0 {
		return 0, nil
	}

	existing := make(map[string]bool)
	rows, err := db.Query("SELECT NROITEM FROM AKOP_MINV")
	if err != nil {
		return 0, fmt.Errorf("leyendo NROITEMs existentes: %v", err)
	}
	for rows.Next() {
		var k string
		if rows.Scan(&k) == nil {
			existing[strings.ToLower(strings.TrimSpace(k))] = true
		}
	}
	rows.Close()
	fmt.Printf("  %d existentes en AKOP_MINV\n", len(existing))

	seen := make(map[string]bool)
	var nuevos []MinvRegistro
	duplicados := 0
	for _, r := range regs {
		key := strings.ToLower(r.NROITEM)
		if existing[key] || seen[key] {
			duplicados++
		} else {
			seen[key] = true
			nuevos = append(nuevos, r)
		}
	}
	fmt.Printf("  %d en DBF, %d duplicados omitidos, %d nuevos\n", len(regs), duplicados, len(nuevos))
	if len(nuevos) == 0 {
		return 0, nil
	}

	copyStmt := mssql.CopyIn("AKOP_MINV", mssql.BulkOptions{}, minvCols...)
	bulkStmt, err := db.Prepare(copyStmt)
	if err != nil {
		return 0, fmt.Errorf("preparando bulk copy: %v", err)
	}

	total := 0
	for _, r := range nuevos {
		if _, err := bulkStmt.Exec(minvArgs(r)...); err != nil {
			continue
		}
		total++
		if total%1000 == 0 {
			fmt.Printf("\r  AKOP_MINV: %d/%d insertados...    ", total, len(nuevos))
		}
	}

	_, err = bulkStmt.Exec()
	bulkStmt.Close()
	if err != nil {
		fmt.Printf("\n  Bulk copy fallo, reintentando con INSERT individuales...\n")
		return insertarMinvIndividual(db, nuevos, total)
	}
	return total, nil
}

func insertarMinvIndividual(db *sql.DB, regs []MinvRegistro, _ int) (int, error) {
	cols := strings.Join(minvCols, ", ")
	placeholders := make([]string, len(minvCols))
	for i := range placeholders {
		placeholders[i] = "?"
	}
	ph := strings.Join(placeholders, ", ")

	insertados := 0
	skipped := 0
	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}
	for _, r := range regs {
		exists := 0
		err := tx.QueryRow("SELECT COUNT(*) FROM AKOP_MINV WHERE NROITEM = ?", r.NROITEM).Scan(&exists)
		if err == nil && exists > 0 {
			skipped++
			continue
		}
		query := fmt.Sprintf("INSERT INTO AKOP_MINV (%s) VALUES (%s)", cols, ph)
		_, err = tx.Exec(query, minvArgs(r)...)
		if err != nil {
			if strings.Contains(err.Error(), "PRIMARY KEY") ||
				strings.Contains(err.Error(), "duplicate") ||
				strings.Contains(err.Error(), "duplicada") {
				skipped++
				continue
			}
			tx.Rollback()
			return insertados, err
		}
		insertados++
		if insertados%1000 == 0 {
			fmt.Printf("\r  AKOP_MINV: %d insertados, %d omitidos...    ", insertados, skipped)
		}
	}
	if err := tx.Commit(); err != nil {
		return insertados, err
	}
	if skipped > 0 {
		fmt.Printf("\r  AKOP_MINV: %d insertados, %d duplicados omitidos.        \n", insertados, skipped)
	}
	return insertados, nil
}

// ---------- Orquestador ----------

func CargarMaestroFamilias() {
	tStart := time.Now()
	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("  MAESTRO DE INVENTARIO (AKOP - MINV)")
	fmt.Println(strings.Repeat("=", 50))

	db, err := ConectarSQL()
	if err != nil {
		fmt.Printf("  ERROR conexion SQL: %v\n", err)
		return
	}
	defer db.Close()

	if err := CrearBaseAKOP(db); err != nil {
		fmt.Printf("  ERROR creando BD AKOP: %v\n", err)
		return
	}

	dbAKOP, err := ConectarSQL_AKOP()
	if err != nil {
		fmt.Printf("  ERROR conexion AKOP: %v\n", err)
		return
	}
	defer dbAKOP.Close()

	if err := CrearTablaAKOP_MINV(dbAKOP); err != nil {
		fmt.Printf("  ERROR creando tabla AKOP_MINV: %v\n", err)
		return
	}

	fmt.Println("\n  Leyendo MINV001.DBF...")
	minvs, err := LeerMINV001()
	if err != nil {
		fmt.Printf("  ERROR leyendo MINV001: %v\n", err)
		return
	}
	fmt.Printf("  MINV001: %d productos en DBF\n", len(minvs))

	fmt.Println("  Insertando en AKOP_MINV...")
	ins, err := insertarAKOPMinv(dbAKOP, minvs)
	if err != nil {
		fmt.Printf("  ERROR insertando MINV: %v\n", err)
		return
	}

	var totalTabla int
	dbAKOP.QueryRow("SELECT COUNT(*) FROM AKOP_MINV").Scan(&totalTabla)
	fmt.Printf("\n  AKOP_MINV: %d insertados de %d productos | Total en tabla: %d | Tiempo: %.1f seg.\n",
		ins, len(minvs), totalTabla, time.Since(tStart).Seconds())
	logHistorial(fmt.Sprintf("AKOP: MINV cargado. %d insertados de %d (%.1f seg)",
		ins, len(minvs), time.Since(tStart).Seconds()))
}
