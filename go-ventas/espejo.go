package main

import (
	"database/sql"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	mssql "github.com/denisenkom/go-mssqldb"
)

// ---------- Configuracion ----------

type EspejoConfig struct {
	Sucursal       string   `json:"sucursal"`
	Base           string   `json:"base"`
	Datos          string   `json:"datos"`
	BaseDatos      string   `json:"base_datos"`
	Origenes       []string `json:"origenes"`
	ExcluirModulos []string `json:"excluir_modulos"`
	SoloAnio       int      `json:"solo_anio"`
	Batch          int      `json:"batch"`
}

func cargarConfigEspejo(path string) EspejoConfig {
	cfg := EspejoConfig{
		Sucursal:       "VC01",
		BaseDatos:      "VC01",
		Origenes:       []string{"data"},
		ExcluirModulos: []string{},
		Batch:          5000,
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			if out, e := json.MarshalIndent(cfg, "", "  "); e == nil {
				os.WriteFile(path, out, 0644)
			}
		}
		return cfg
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg
	}
	if strings.TrimSpace(cfg.Sucursal) == "" {
		cfg.Sucursal = "VC01"
	}
	if strings.TrimSpace(cfg.BaseDatos) == "" {
		cfg.BaseDatos = cfg.Sucursal
	}
	if len(cfg.Origenes) == 0 {
		cfg.Origenes = []string{"data"}
	}
	if cfg.Batch <= 0 {
		cfg.Batch = 5000
	}
	return cfg
}

// ---------- Conexion ----------

func conectarEspejoBD(nombre string) (*sql.DB, error) {
	connStr := fmt.Sprintf("server=%s;database=%s;user id=%s;password=%s;encrypt=disable;connection timeout=30;timeout=1800",
		SQLServer, nombre, SQLUser, SQLPassword)
	db, err := sql.Open("mssql", connStr)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(2)
	return db, nil
}

func crearBDEspejo(nombre string) error {
	db, err := conectarEspejoBD("master")
	if err != nil {
		return err
	}
	defer db.Close()
	_, err = db.Exec(fmt.Sprintf("IF DB_ID(%s) IS NULL BEGIN CREATE DATABASE %s END", qlEspejo(nombre), qiEspejo(nombre)))
	return err
}

func qiEspejo(s string) string {
	return "[" + strings.ReplaceAll(s, "]", "]]") + "]"
}

func qlEspejo(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

// ---------- Descubrimiento ----------

type espejoModulo struct {
	Empresa string
	Modulo  string
	Esquema string
	Ruta    string
	DBFs    []string
}

func ipSucursalEspejo(sucursal string) (string, error) {
	sucs, err := CargarSucursalesJSON()
	if err != nil {
		return "", err
	}
	for _, s := range sucs {
		if strings.EqualFold(strings.TrimSpace(s.Codigo), strings.TrimSpace(sucursal)) {
			ip := strings.TrimSpace(s.RutaIP)
			if ip == "" {
				return "", fmt.Errorf("sucursal %s sin RutaIP", sucursal)
			}
			return ip, nil
		}
	}
	return "", fmt.Errorf("sucursal %s no encontrada en sucursales.json", sucursal)
}

func resolverBaseEspejo(sucursal string) (string, error) {
	ip, err := ipSucursalEspejo(sucursal)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(`\\%s\Sistema\aBC-Soft\Data`, ip), nil
}

func resolverDatosEspejo(sucursal string) (string, error) {
	ip, err := ipSucursalEspejo(sucursal)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(`\\%s\Sistema\aBC-Soft\Datos`, ip), nil
}

func descubrirModulosEspejo(baseData string, excluir []string) ([]espejoModulo, error) {
	excl := map[string]bool{}
	for _, e := range excluir {
		excl[strings.ToUpper(strings.TrimSpace(e))] = true
	}

	entradas, err := os.ReadDir(baseData)
	if err != nil {
		return nil, err
	}

	var mods []espejoModulo
	for _, e := range entradas {
		if !e.IsDir() {
			continue
		}
		empresa := e.Name()
		rutaEmp := filepath.Join(baseData, empresa)

		if dbfs := listarDBFEspejo(rutaEmp); len(dbfs) > 0 {
			mods = append(mods, espejoModulo{
				Empresa: empresa,
				Modulo:  "root",
				Esquema: empresa + "_root",
				Ruta:    rutaEmp,
				DBFs:    dbfs,
			})
		}

		sub, err := os.ReadDir(rutaEmp)
		if err != nil {
			continue
		}
		for _, s := range sub {
			if !s.IsDir() {
				continue
			}
			mod := s.Name()
			if excl[strings.ToUpper(mod)] {
				continue
			}
			rutaMod := filepath.Join(rutaEmp, mod)
			dbfs := listarDBFEspejo(rutaMod)
			if len(dbfs) == 0 {
				continue
			}
			mods = append(mods, espejoModulo{
				Empresa: empresa,
				Modulo:  mod,
				Esquema: empresa + "_" + mod,
				Ruta:    rutaMod,
				DBFs:    dbfs,
			})
		}
	}

	sort.Slice(mods, func(i, j int) bool { return mods[i].Esquema < mods[j].Esquema })
	return mods, nil
}

func listarDBFEspejo(dir string) []string {
	return listarDBFEspejoFiltrado(dir, 0)
}

var anioEspejoRe = regexp.MustCompile(`(?i)(19|20)\d{2}$`)

func filtrarAnioEspejo(nombre string, soloAnio int) bool {
	if soloAnio <= 0 {
		return true
	}
	base := strings.TrimSuffix(filepath.Base(nombre), filepath.Ext(nombre))
	m := anioEspejoRe.FindString(base)
	if m == "" {
		return true
	}
	anio, err := strconv.Atoi(m)
	if err != nil {
		return true
	}
	return anio == soloAnio
}

func listarDBFEspejoFiltrado(dir string, soloAnio int) []string {
	entradas, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entradas {
		if e.IsDir() {
			continue
		}
		if !strings.EqualFold(filepath.Ext(e.Name()), ".dbf") {
			continue
		}
		if !filtrarAnioEspejo(e.Name(), soloAnio) {
			continue
		}
		out = append(out, filepath.Join(dir, e.Name()))
	}
	sort.Strings(out)
	return out
}

func descubrirRaizEspejo(dir, esquema string, soloAnio int) ([]espejoModulo, error) {
	info, err := os.Stat(dir)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s no es un directorio", dir)
	}
	dbfs := listarDBFEspejoFiltrado(dir, soloAnio)
	if len(dbfs) == 0 {
		return nil, nil
	}
	return []espejoModulo{{
		Empresa: "",
		Modulo:  esquema,
		Esquema: esquema,
		Ruta:    dir,
		DBFs:    dbfs,
	}}, nil
}

func nombreTablaDBF(path string) string {
	base := filepath.Base(path)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

// ---------- DDL ----------

func crearEsquemaEspejo(db *sql.DB, esquema string) error {
	crear := "CREATE SCHEMA " + qiEspejo(esquema)
	_, err := db.Exec(fmt.Sprintf("IF SCHEMA_ID(%s) IS NULL EXEC(%s)", qlEspejo(esquema), qlEspejo(crear)))
	return err
}

func tipoSQLCampo(f DBFField) string {
	switch f.Type {
	case 'C', 'V':
		l := f.Length
		if l <= 0 {
			l = 1
		}
		if l > 4000 {
			return "NVARCHAR(MAX)"
		}
		return fmt.Sprintf("NVARCHAR(%d)", l)
	case 'N':
		l := f.Length
		d := f.DecimalCount
		if l <= 0 {
			l = 1
		}
		if l > 38 {
			l = 38
		}
		if d < 0 {
			d = 0
		}
		if d > l {
			d = l
		}
		return fmt.Sprintf("DECIMAL(%d,%d)", l, d)
	case 'F', 'O', 'B', 'b':
		return "FLOAT"
	case 'I', '+':
		return "INT"
	case 'L':
		return "BIT"
	case 'D':
		return "DATE"
	case 'T', '@':
		return "DATETIME2(3)"
	case 'Y':
		return "DECIMAL(19,4)"
	case 'M':
		return "NVARCHAR(MAX)"
	case 'G', 'W':
		return "VARBINARY(MAX)"
	default:
		l := f.Length
		if l <= 0 {
			return "NVARCHAR(255)"
		}
		if l > 4000 {
			return "NVARCHAR(MAX)"
		}
		return fmt.Sprintf("NVARCHAR(%d)", l)
	}
}

func crearTablaEspejo(db *sql.DB, esquema, tabla string, campos []DBFField) error {
	var cols []string
	for _, f := range campos {
		cols = append(cols, fmt.Sprintf("%s %s NULL", qiEspejo(f.Name), tipoSQLCampo(f)))
	}
	if len(cols) == 0 {
		return fmt.Errorf("sin columnas validas")
	}
	full := qiEspejo(esquema) + "." + qiEspejo(tabla)
	if _, err := db.Exec("DROP TABLE IF EXISTS " + full); err != nil {
		return err
	}
	ddl := fmt.Sprintf("CREATE TABLE %s (\n  %s\n)", full, strings.Join(cols, ",\n  "))
	_, err := db.Exec(ddl)
	return err
}

func limpiarModulosExcluidos(db *sql.DB, excluir []string) {
	mods := map[string]bool{}
	for _, e := range excluir {
		e = strings.ToUpper(strings.TrimSpace(e))
		if e != "" {
			mods[e] = true
		}
	}
	if len(mods) == 0 {
		return
	}
	rows, err := db.Query(`SELECT s.name, t.name FROM sys.tables t JOIN sys.schemas s ON s.schema_id = t.schema_id`)
	if err != nil {
		fmt.Printf("     aviso limpieza: %v\n", err)
		return
	}
	type tablaPar struct{ esquema, tabla string }
	var drops []tablaPar
	esquemas := map[string]bool{}
	for rows.Next() {
		var esq, tab string
		if rows.Scan(&esq, &tab) != nil {
			continue
		}
		partes := strings.Split(esq, "_")
		if len(partes) < 2 {
			continue
		}
		if mods[strings.ToUpper(partes[len(partes)-1])] {
			drops = append(drops, tablaPar{esq, tab})
			esquemas[esq] = true
		}
	}
	rows.Close()
	for _, d := range drops {
		if _, err := db.Exec("DROP TABLE " + qiEspejo(d.esquema) + "." + qiEspejo(d.tabla)); err != nil {
			fmt.Printf("     aviso: no se pudo borrar %s.%s: %v\n", d.esquema, d.tabla, err)
		}
	}
	for esq := range esquemas {
		if _, err := db.Exec("DROP SCHEMA " + qiEspejo(esq)); err != nil {
			fmt.Printf("     aviso: no se pudo borrar esquema %s: %v\n", esq, err)
		} else {
			fmt.Printf("     excluido: esquema [%s] eliminado\n", esq)
		}
	}
}

func limpiarTablasObsoletas(db *sql.DB, esquema string, objetivo map[string]bool) {
	rows, err := db.Query(`SELECT t.name FROM sys.tables t JOIN sys.schemas s ON s.schema_id = t.schema_id WHERE s.name = ?`, esquema)
	if err != nil {
		return
	}
	var sobrantes []string
	for rows.Next() {
		var tab string
		if rows.Scan(&tab) == nil && !objetivo[strings.ToLower(tab)] {
			sobrantes = append(sobrantes, tab)
		}
	}
	rows.Close()
	for _, tab := range sobrantes {
		db.Exec("DROP TABLE " + qiEspejo(esquema) + "." + qiEspejo(tab))
	}
}

// ---------- Lectura DBF / FPT ----------

func abrirDBFEspejo(path string) (*DBFFile, error) {
	var last error
	for i := 0; i < 3; i++ {
		d, err := OpenDBF(path)
		if err == nil {
			return d, nil
		}
		last = err
		time.Sleep(2 * time.Second)
	}
	return nil, last
}

type fptFile struct {
	f         *os.File
	blockSize int64
}

func abrirFPT(path string) (*fptFile, error) {
	var f *os.File
	var err error
	for i := 0; i < 3; i++ {
		f, err = os.Open(path)
		if err == nil {
			break
		}
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		return nil, err
	}
	h := make([]byte, 8)
	if _, err := io.ReadFull(f, h); err != nil {
		f.Close()
		return nil, err
	}
	bs := int64(binary.BigEndian.Uint16(h[6:8]))
	if bs <= 0 {
		bs = 64
	}
	return &fptFile{f: f, blockSize: bs}, nil
}

func (f *fptFile) Cerrar() {
	if f != nil && f.f != nil {
		f.f.Close()
	}
}

func (f *fptFile) Leer(block uint32) ([]byte, error) {
	if block == 0 {
		return nil, nil
	}
	off := int64(block) * f.blockSize
	hdr := make([]byte, 8)
	if _, err := f.f.ReadAt(hdr, off); err != nil {
		return nil, err
	}
	ln := int64(binary.BigEndian.Uint32(hdr[4:8]))
	if ln <= 0 {
		return nil, nil
	}
	if ln > 64*1024*1024 {
		return nil, fmt.Errorf("memo demasiado grande: %d", ln)
	}
	data := make([]byte, ln)
	if _, err := f.f.ReadAt(data, off+8); err != nil {
		return nil, err
	}
	return data, nil
}

func buscarFPT(pathDBF string) string {
	dir := filepath.Dir(pathDBF)
	base := strings.TrimSuffix(filepath.Base(pathDBF), filepath.Ext(pathDBF))
	entradas, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	for _, e := range entradas {
		if e.IsDir() {
			continue
		}
		n := e.Name()
		nbase := strings.TrimSuffix(n, filepath.Ext(n))
		if strings.EqualFold(nbase, base) && strings.EqualFold(filepath.Ext(n), ".fpt") {
			return filepath.Join(dir, n)
		}
	}
	return ""
}

func tieneMemo(campos []DBFField) bool {
	for _, f := range campos {
		if f.Type == 'M' || f.Type == 'G' || f.Type == 'W' {
			return true
		}
	}
	return false
}

func valorCampoEspejo(d *DBFFile, rec []byte, f DBFField, fpt *fptFile) interface{} {
	raw := d.GetRaw(rec, f)
	switch f.Type {
	case 'C', 'V':
		s := strings.TrimRight(d.decodeBytes(raw), " \x00")
		if s == "" {
			return nil
		}
		return s
	case 'N', 'F':
		s := strings.TrimSpace(string(raw))
		if s == "" || s == "." || s == "-" {
			return nil
		}
		v, err := strconv.ParseFloat(strings.ReplaceAll(s, ",", "."), 64)
		if err != nil {
			return nil
		}
		return v
	case 'I', '+':
		if len(raw) < 4 {
			return nil
		}
		return int32(binary.LittleEndian.Uint32(raw))
	case 'b', 'B', 'O':
		if len(raw) < 8 {
			return nil
		}
		return math.Float64frombits(binary.LittleEndian.Uint64(raw))
	case 'Y':
		if len(raw) < 8 {
			return nil
		}
		return float64(int64(binary.LittleEndian.Uint64(raw))) / 10000.0
	case 'L':
		s := strings.TrimSpace(string(raw))
		if s == "" {
			return nil
		}
		switch s[0] {
		case 'T', 't', 'Y', 'y':
			return true
		case 'F', 'f', 'N', 'n':
			return false
		}
		return nil
	case 'D':
		s := strings.TrimSpace(string(raw))
		if len(s) != 8 {
			return nil
		}
		t, err := time.Parse("20060102", s)
		if err != nil {
			return nil
		}
		return t
	case 'T', '@':
		if len(raw) < 8 {
			return nil
		}
		jd := int64(binary.LittleEndian.Uint32(raw[0:4]))
		ms := int64(binary.LittleEndian.Uint32(raw[4:8]))
		if jd == 0 {
			return nil
		}
		secs := (jd-2440588)*86400 + ms/1000
		nanos := (ms % 1000) * int64(time.Millisecond)
		return time.Unix(secs, nanos).UTC()
	case 'M':
		if fpt == nil || len(raw) < 4 {
			return nil
		}
		data, err := fpt.Leer(binary.LittleEndian.Uint32(raw[0:4]))
		if err != nil || len(data) == 0 {
			return nil
		}
		s := strings.TrimRight(d.decodeBytes(data), "\x00")
		if s == "" {
			return nil
		}
		return s
	case 'G', 'W':
		if fpt == nil || len(raw) < 4 {
			return nil
		}
		data, err := fpt.Leer(binary.LittleEndian.Uint32(raw[0:4]))
		if err != nil || len(data) == 0 {
			return nil
		}
		return data
	default:
		s := strings.TrimRight(d.decodeBytes(raw), " \x00")
		if s == "" {
			return nil
		}
		return s
	}
}

// ---------- Importacion ----------

func importarTablaEspejo(db *sql.DB, pathDBF, esquema, tabla string, batch int) (int, error) {
	d, err := abrirDBFEspejo(pathDBF)
	if err != nil {
		return 0, err
	}
	defer d.Close()

	var campos []DBFField
	for _, f := range d.Fields {
		if f.Type == 0 || f.Length == 0 || strings.TrimSpace(f.Name) == "" {
			continue
		}
		campos = append(campos, f)
	}
	if len(campos) == 0 {
		return 0, fmt.Errorf("sin campos validos")
	}

	if err := crearTablaEspejo(db, esquema, tabla, campos); err != nil {
		return 0, err
	}
	if d.NumRecords == 0 {
		return 0, nil
	}

	var fpt *fptFile
	if tieneMemo(campos) {
		if fp := buscarFPT(pathDBF); fp != "" {
			fpt, _ = abrirFPT(fp)
			if fpt != nil {
				defer fpt.Cerrar()
			}
		}
	}

	full := qiEspejo(esquema) + "." + qiEspejo(tabla)
	nombres := make([]string, len(campos))
	for i, f := range campos {
		nombres[i] = f.Name
	}

	stmt, err := db.Prepare(mssql.CopyIn(full, mssql.BulkOptions{
		CheckConstraints: false,
		RowsPerBatch:     batch,
		Tablock:          true,
	}, nombres...))
	if err != nil {
		return fallbackEspejo(db, d, esquema, tabla, campos, fpt, err)
	}

	leidos := 0
	var bulkErr error
	for idx := 0; idx < d.NumRecords; idx++ {
		rec, rerr := d.ReadRecord(idx)
		if rerr != nil {
			continue
		}
		if len(rec) > 0 && rec[0] == 0x2A {
			continue
		}
		vals := make([]interface{}, len(campos))
		for i, f := range campos {
			vals[i] = valorCampoEspejo(d, rec, f, fpt)
		}
		if _, err := stmt.Exec(vals...); err != nil {
			bulkErr = fmt.Errorf("fila %d: %w", idx, err)
			break
		}
		leidos++
		if leidos%5000 == 0 {
			fmt.Printf("\r     %-30s %9d filas...    ", tabla, leidos)
		}
	}
	if bulkErr == nil && leidos > 0 {
		if _, err := stmt.Exec(); err != nil {
			bulkErr = err
		}
	}
	stmt.Close()

	if bulkErr != nil {
		return fallbackEspejo(db, d, esquema, tabla, campos, fpt, bulkErr)
	}
	return leidos, nil
}

func fallbackEspejo(db *sql.DB, d *DBFFile, esquema, tabla string, campos []DBFField, fpt *fptFile, causa error) (int, error) {
	fmt.Printf("\r     %-30s bulk fallo (%v); insertando fila a fila...\n", tabla, causa)
	full := qiEspejo(esquema) + "." + qiEspejo(tabla)
	if _, err := db.Exec("TRUNCATE TABLE " + full); err != nil {
		return 0, err
	}
	cols := make([]string, len(campos))
	ph := make([]string, len(campos))
	for i, f := range campos {
		cols[i] = qiEspejo(f.Name)
		ph[i] = "?"
	}
	q := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", full, strings.Join(cols, ", "), strings.Join(ph, ", "))

	ins, om := 0, 0
	for idx := 0; idx < d.NumRecords; idx++ {
		rec, err := d.ReadRecord(idx)
		if err != nil {
			continue
		}
		if len(rec) > 0 && rec[0] == 0x2A {
			continue
		}
		vals := make([]interface{}, len(campos))
		for i, f := range campos {
			vals[i] = valorCampoEspejo(d, rec, f, fpt)
		}
		if _, err := db.Exec(q, vals...); err != nil {
			om++
			continue
		}
		ins++
		if ins%2000 == 0 {
			fmt.Printf("\r     %-30s %9d filas...    ", tabla, ins)
		}
	}
	if om > 0 {
		fmt.Printf("\r     %-30s %9d filas (%d omitidas)        \n", tabla, ins, om)
	}
	return ins, nil
}

// ---------- Orquestador ----------

func CargarEspejo(cfg EspejoConfig) {
	origenes := cfg.Origenes
	if len(origenes) == 0 {
		origenes = []string{"data"}
	}

	var mods []espejoModulo
	for _, o := range origenes {
		switch strings.ToLower(strings.TrimSpace(o)) {
		case "data":
			base := strings.TrimSpace(cfg.Base)
			if base == "" {
				var err error
				base, err = resolverBaseEspejo(cfg.Sucursal)
				if err != nil {
					fmt.Printf("  ERROR resolviendo Data: %v\n", err)
					return
				}
			}
			fmt.Printf("\n  Origen Data : %s\n", base)
			m, err := descubrirModulosEspejo(base, cfg.ExcluirModulos)
			if err != nil {
				fmt.Printf("  ERROR leyendo Data: %v\n", err)
				return
			}
			mods = append(mods, m...)
		case "datos":
			datos := strings.TrimSpace(cfg.Datos)
			if datos == "" {
				var err error
				datos, err = resolverDatosEspejo(cfg.Sucursal)
				if err != nil {
					fmt.Printf("  ERROR resolviendo Datos: %v\n", err)
					return
				}
			}
			fmt.Printf("\n  Origen Datos: %s\n", datos)
			m, err := descubrirRaizEspejo(datos, "Datos", cfg.SoloAnio)
			if err != nil {
				fmt.Printf("  ERROR leyendo Datos: %v\n", err)
				return
			}
			mods = append(mods, m...)
		default:
			fmt.Printf("  ERROR: origen desconocido %q (use data o datos)\n", o)
			return
		}
	}
	if cfg.SoloAnio > 0 {
		fmt.Printf("  Solo anio   : %d\n", cfg.SoloAnio)
	}

	totalDBF := 0
	for _, m := range mods {
		totalDBF += len(m.DBFs)
	}
	fmt.Printf("\n  Esquemas    : %d\n", len(mods))
	fmt.Printf("  Tablas DBF  : %d\n", totalDBF)
	if totalDBF == 0 {
		fmt.Println("  No se encontraron archivos .DBF.")
		return
	}

	fmt.Printf("\n  Creando/verificando BD [%s] en %s...\n", cfg.BaseDatos, SQLServer)
	if err := crearBDEspejo(cfg.BaseDatos); err != nil {
		fmt.Printf("  ERROR creando BD: %v\n", err)
		return
	}
	db, err := conectarEspejoBD(cfg.BaseDatos)
	if err != nil {
		fmt.Printf("  ERROR conectando BD: %v\n", err)
		return
	}
	defer db.Close()

	if len(cfg.ExcluirModulos) > 0 {
		fmt.Printf("\n  Aplicando exclusiones: %s\n", strings.Join(cfg.ExcluirModulos, ", "))
		limpiarModulosExcluidos(db, cfg.ExcluirModulos)
	}

	t0 := time.Now()
	ok, fail, filasTot := 0, 0, 0
	for _, m := range mods {
		fmt.Printf("\n  >> Esquema [%s]  (%d tablas)\n", m.Esquema, len(m.DBFs))
		if err := crearEsquemaEspejo(db, m.Esquema); err != nil {
			fmt.Printf("     ERROR creando esquema: %v\n", err)
			fail += len(m.DBFs)
			continue
		}
		objetivo := map[string]bool{}
		for _, p := range m.DBFs {
			objetivo[strings.ToLower(nombreTablaDBF(p))] = true
		}
		limpiarTablasObsoletas(db, m.Esquema, objetivo)
		for _, p := range m.DBFs {
			tabla := nombreTablaDBF(p)
			ini := time.Now()
			n, err := importarTablaEspejo(db, p, m.Esquema, tabla, cfg.Batch)
			if err != nil {
				fmt.Printf("\r     %-30s ERROR: %v        \n", tabla, err)
				fail++
				continue
			}
			fmt.Printf("\r     %-30s %9d filas  %6.1fs        \n", tabla, n, time.Since(ini).Seconds())
			ok++
			filasTot += n
		}
	}

	fmt.Println()
	fmt.Println(strings.Repeat("-", 62))
	fmt.Printf("  RESUMEN: %d tablas OK, %d con error, %d filas, %.1fs\n", ok, fail, filasTot, time.Since(t0).Seconds())
	fmt.Printf("  BD [%s] en %s\n", cfg.BaseDatos, SQLServer)
}

func resumenEspejo(baseDatos string) {
	db, err := conectarEspejoBD(baseDatos)
	if err != nil {
		fmt.Printf("  ERROR conexion [%s]: %v\n", baseDatos, err)
		return
	}
	defer db.Close()

	rows, err := db.Query(`
		SELECT s.name AS esquema, t.name AS tabla, SUM(p.rows) AS filas
		FROM sys.tables t
		JOIN sys.schemas s ON s.schema_id = t.schema_id
		JOIN sys.partitions p ON p.object_id = t.object_id AND p.index_id IN (0,1)
		GROUP BY s.name, t.name
		ORDER BY s.name, t.name`)
	if err != nil {
		fmt.Printf("  ERROR consulta: %v\n", err)
		return
	}
	defer rows.Close()

	esqActual := ""
	totalTablas := 0
	var totalFilas int64
	for rows.Next() {
		var esq, tab string
		var filas sql.NullInt64
		if err := rows.Scan(&esq, &tab, &filas); err != nil {
			continue
		}
		if esq != esqActual {
			fmt.Printf("\n  [%s]\n", esq)
			esqActual = esq
		}
		fmt.Printf("    %-30s %10d\n", tab, filas.Int64)
		totalTablas++
		totalFilas += filas.Int64
	}
	fmt.Printf("\n  Total: %d tablas, %d filas\n", totalTablas, totalFilas)
}
