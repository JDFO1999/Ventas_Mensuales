package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	_ "github.com/denisenkom/go-mssqldb"
)

const (
	SQLServer   = "10.10.70.160"
	SQLDatabase = "Sistemas"
	SQLUser     = "Sa"
	SQLPassword = "Alkosto123"
)

func ConectarSQL() (*sql.DB, error) {
	connStr := fmt.Sprintf("server=%s;database=%s;user id=%s;password=%s;encrypt=disable;connection timeout=10",
		SQLServer, SQLDatabase, SQLUser, SQLPassword)
	db, err := sql.Open("mssql", connStr)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	return db, nil
}

func ConectarSQL_CA(_ string) (*sql.DB, error) {
	connStr := fmt.Sprintf("server=%s;database=BD_Tiendas;user id=%s;password=%s;encrypt=disable;connection timeout=10",
		SQLServer, SQLUser, SQLPassword)
	db, err := sql.Open("mssql", connStr)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(2)
	return db, nil
}

func ObtenerSucursales(db *sql.DB) ([]Sucursal, error) {
	rows, err := db.Query(`
		SELECT cCodigo, cNombre, cRutaIP, cRutaDBF, cCia
		FROM Sucursal
		WHERE lInactiva = 0 OR lInactiva IS NULL
		ORDER BY cCodigo
	`)
	if err != nil {
		fmt.Println("  Usando respaldo local de sucursales (SQL no disponible)")
		s, errJSON := CargarSucursalesJSON()
		if errJSON != nil {
			return nil, fmt.Errorf("sin SQL ni respaldo: %v / %v", err, errJSON)
		}
		return s, nil
	}
	defer rows.Close()

	var sucursales []Sucursal
	for rows.Next() {
		var s Sucursal
		var rutaIP, rutaDBF, cia sql.NullString
		if err := rows.Scan(&s.Codigo, &s.Nombre, &rutaIP, &rutaDBF, &cia); err != nil {
			return nil, err
		}
		parts := strings.Split(strings.TrimSpace(rutaIP.String), " ")
		if len(parts) > 0 {
			s.RutaIP = parts[0]
		}
		s.RutaDBF = strings.TrimSpace(rutaDBF.String)
		s.Cia = strings.TrimSpace(cia.String)
		sucursales = append(sucursales, s)
	}

	if len(sucursales) > 0 {
		GuardarSucursalesJSON(sucursales)
	}

	return sucursales, nil
}

func GuardarSucursalesJSON(sucursales []Sucursal) {
	respaldo := struct {
		UltimaSincronizacion string     `json:"ultima_sincronizacion"`
		Tiendas              []Sucursal `json:"tiendas"`
	}{
		UltimaSincronizacion: time.Now().Format("2006-01-02 15:04:05"),
		Tiendas:              sucursales,
	}
	data, err := json.MarshalIndent(respaldo, "", "  ")
	if err != nil {
		return
	}
	os.WriteFile("sucursales.json", data, 0644)
}

func CargarSucursalesJSON() ([]Sucursal, error) {
	data, err := os.ReadFile("sucursales.json")
	if err != nil {
		return nil, err
	}
	var respaldo struct {
		UltimaSincronizacion string     `json:"ultima_sincronizacion"`
		Tiendas              []Sucursal `json:"tiendas"`
	}
	if err := json.Unmarshal(data, &respaldo); err != nil {
		return nil, err
	}
	if len(respaldo.Tiendas) > 0 {
		fmt.Printf("  Respaldo del %s (%d tiendas)\n", respaldo.UltimaSincronizacion, len(respaldo.Tiendas))
		return respaldo.Tiendas, nil
	}
	// fallback: old format without wrapper
	var s []Sucursal
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	return s, nil
}

func CrearTablaPosVentas(db *sql.DB) error {
	_, err := db.Exec(`
		IF OBJECT_ID('PosVentas','U') IS NULL
		CREATE TABLE PosVentas (
			Fecha      DATE NOT NULL,
			Hora       TINYINT NOT NULL,
			Tienda     VARCHAR(6) NOT NULL,
			Caja       INT NOT NULL,
			Tipo       CHAR(2) NOT NULL,
			STipo      CHAR(2) NULL,
			Numero     VARCHAR(15) NOT NULL,
			Anulada    CHAR(1) NULL,
			Operador   VARCHAR(20) NULL,
			SubTotal   DECIMAL(18,2) NULL,
			MontoUSD   DECIMAL(18,2) NOT NULL,
			MontoBS    DECIMAL(18,2) NULL,
			Impuesto   DECIMAL(18,2) NULL,
			Descuento  DECIMAL(18,2) NULL,
			IGTF       DECIMAL(18,2) NULL,
			Redondeo   DECIMAL(18,2) NULL,
			TasaDOL    DECIMAL(20,6) NULL,
			CodCli     VARCHAR(10) NULL,
			Nombres    VARCHAR(100) NULL,
			Apellidos  VARCHAR(100) NULL,
			NIT        VARCHAR(14) NULL,
			RIF        VARCHAR(14) NULL,
			CodVen     VARCHAR(2) NULL,
			NroZ       VARCHAR(10) NULL,
			Credito    BIT NULL,
			Vuelto     DECIMAL(16,2) NULL,
			LIMPRIMIO  BIT NULL,
			NroCie     VARCHAR(8) NULL,
			FechaCie   DATE NULL,
			FechaCarga DATETIME DEFAULT GETDATE(),
			CONSTRAINT PK_PosVentas PRIMARY KEY (Tienda, Caja, Numero, Tipo)
		)
	`)
	return err
}

func TiendaTieneDatos(db *sql.DB, codigo string, year, month int) (bool, error) {
	var count int
	err := db.QueryRow(
		"SELECT COUNT(*) FROM PosVentas WHERE Tienda=? AND YEAR(Fecha)=? AND MONTH(Fecha)=?",
		codigo, year, month,
	).Scan(&count)
	return count > 0, err
}

func BorrarDatosTienda(db *sql.DB, codigo string, year, month int) error {
	_, err := db.Exec(
		"DELETE FROM PosVentas WHERE Tienda=? AND YEAR(Fecha)=? AND MONTH(Fecha)=?",
		codigo, year, month,
	)
	return err
}

func InsertarVentas(db *sql.DB, registros []VentaRegistro) error {
	const chunkSize = 5000
	query := `INSERT INTO PosVentas (Fecha, Hora, Tienda, Caja, Tipo, STipo, Numero, Anulada, Operador,
		SubTotal, MontoUSD, MontoBS, Impuesto, Descuento, IGTF, Redondeo, TasaDOL,
		CodCli, Nombres, Apellidos, NIT, RIF, CodVen, NroZ, Credito, Vuelto, LIMPRIMIO, NroCie, FechaCie)
		VALUES (?,?,?,?,?,?,?,?,?,
			?,?,?,?,?,?,?,?,
			?,?,?,?,?,?,?,?,?,?,?,?)`

	for i := 0; i < len(registros); i += chunkSize {
		end := i + chunkSize
		if end > len(registros) {
			end = len(registros)
		}
		chunk := registros[i:end]

		tx, err := db.Begin()
		if err != nil {
			return err
		}

		for _, r := range chunk {
			var fechaCie interface{}
			if r.FechaCie != nil {
				fechaCie = *r.FechaCie
			}
			_, err = tx.Exec(query,
				r.Fecha, r.Hora, r.Tienda, r.Caja, r.Tipo, r.STipo, r.Numero, r.Anulada, r.Operador,
				r.SubTotal, r.MontoUSD, r.MontoBS, r.Impuesto, r.Descuento, r.IGTF, r.Redondeo, r.TasaDOL,
				r.CodCli, r.Nombres, r.Apellidos, r.NIT, r.RIF, r.CodVen, r.NroZ,
				r.Credito, r.Vuelto, r.LIMPRIMIO, r.NroCie, fechaCie,
			)
			if err != nil {
				if strings.Contains(err.Error(), "PRIMARY") ||
					strings.Contains(err.Error(), "clave duplicada") ||
					strings.Contains(err.Error(), "duplicate") {
					continue
				}
				tx.Rollback()
				return err
			}
		}

		if err := tx.Commit(); err != nil {
			return err
		}
		fmt.Printf("\r  SQL Progreso: %d/%d (%d%%)", end, len(registros), 100*end/len(registros))
	}
	fmt.Println()
	return nil
}

func LeerDatosDesdeSQL(db *sql.DB, year, month int) (map[string]map[int]VentaPorHora, error) {
	rows, err := db.Query(`
		SELECT Tienda, Hora,
			SUM(CASE WHEN Tipo='FA' THEN MontoUSD ELSE -MontoUSD END) as TotalUSD,
			COUNT(CASE WHEN Tipo='FA' THEN 1 END) as Facturas
		FROM PosVentas
		WHERE YEAR(Fecha)=? AND MONTH(Fecha)=?
		GROUP BY Tienda, Hora
		ORDER BY Tienda, Hora
	`, year, month)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	data := make(map[string]map[int]VentaPorHora)
	for rows.Next() {
		var tienda string
		var hora, facturas int
		var totalUSD float64
		if err := rows.Scan(&tienda, &hora, &totalUSD, &facturas); err != nil {
			return nil, err
		}
		if data[tienda] == nil {
			data[tienda] = make(map[int]VentaPorHora)
		}
		v := data[tienda][hora]
		v.TotalUSD += totalUSD
		v.Facturas += facturas
		data[tienda][hora] = v
	}
	return data, rows.Err()
}

func CrearTablaPosVentasCA(db *sql.DB, codigo string) error {
	tableName := "POS_CA_" + codigo
	_, err := db.Exec(fmt.Sprintf(`
		IF OBJECT_ID('%s','U') IS NULL
		CREATE TABLE %s (
			Fecha     DATE NOT NULL,
			Hora      TINYINT NOT NULL,
			Tienda    VARCHAR(6) NOT NULL,
			Caja      INT NOT NULL,
			Tipo      CHAR(2) NOT NULL,
			STipo     CHAR(2) NULL,
			Numero    VARCHAR(15) NOT NULL,
			Codigo    VARCHAR(15) NOT NULL,
			CodBar    VARCHAR(15) NULL,
			Descrip   VARCHAR(120) NULL,
			CodVen    VARCHAR(2) NULL,
			Modelo    VARCHAR(20) NULL,
			Serial    VARCHAR(20) NULL,
			Cantidad  DECIMAL(10,3) NULL,
			NCntd     DECIMAL(10,3) NULL,
			NPvpDol   DECIMAL(18,4) NULL,
			NPvp2Dol  DECIMAL(18,4) NULL,
			NPvp3Dol  DECIMAL(18,4) NULL,
			NPvpCop   DECIMAL(18,2) NULL,
			Precio    DECIMAL(16,4) NULL,
			NPrecio   DECIMAL(18,4) NULL,
			IGV       DECIMAL(10,2) NULL,
			NoDscto   BIT NULL,
			CodCli    VARCHAR(10) NULL,
			Anulada   CHAR(1) NULL,
			Depto     VARCHAR(2) NULL,
			Familia   VARCHAR(2) NULL,
			Costo     DECIMAL(18,4) NULL,
			NCosDol   DECIMAL(20,8) NULL,
			Pvpt      CHAR(1) NULL,
			Oferta    BIT NULL,
			Devlto    DECIMAL(16,2) NULL,
			Margen    DECIMAL(10,2) NULL,
			PvpVen    DECIMAL(16,2) NULL,
			LPesado   BIT NULL,
			NroCie    VARCHAR(8) NULL,
			FechaCie  DATE NULL,
			FechaCarga DATETIME DEFAULT GETDATE()
		)
	`, tableName, tableName))
	return err
}

func InsertarVentasCA(db *sql.DB, registros []VentaCARegistro, codigo string) error {
	const chunkSize = 5000
	tableName := "POS_CA_" + codigo
	query := fmt.Sprintf(`INSERT INTO %s (Fecha, Hora, Tienda, Caja, Tipo, STipo, Numero, Codigo, CodBar, Descrip,
		CodVen, Modelo, Serial, Cantidad, NCntd, NPvpDol, NPvp2Dol, NPvp3Dol, NPvpCop,
		Precio, NPrecio, IGV, NoDscto, CodCli, Anulada, Depto, Familia,
		Costo, NCosDol, Pvpt, Oferta, Devlto, Margen, PvpVen, LPesado, NroCie, FechaCie)
		VALUES (?,?,?,?,?,?,?,?,?,?,
			?,?,?,?,?,?,?,?,?,
			?,?,?,?,?,?,?,?,
			?,?,?,?,?,?,?,?,?,?)`, tableName)

	for i := 0; i < len(registros); i += chunkSize {
		end := i + chunkSize
		if end > len(registros) {
			end = len(registros)
		}
		chunk := registros[i:end]

		tx, err := db.Begin()
		if err != nil {
			return err
		}

		for _, r := range chunk {
			var fechaCie interface{}
			if r.FechaCie != nil {
				fechaCie = *r.FechaCie
			}
			_, err = tx.Exec(query,
				r.Fecha, r.Hora, r.Tienda, r.Caja, r.Tipo, r.STipo, r.Numero, r.Codigo, r.CodBar, r.Descrip,
				r.CodVen, r.Modelo, r.Serial, r.Cantidad, r.NCntd, r.NPvpDol, r.NPvp2Dol, r.NPvp3Dol, r.NPvpCop,
				r.Precio, r.NPrecio, r.IGV, r.NoDscto, r.CodCli, r.Anulada, r.Depto, r.Familia,
				r.Costo, r.NCosDol, r.Pvpt, r.Oferta, r.Devlto, r.Margen, r.PvpVen, r.LPesado, r.NroCie, fechaCie,
			)
			if err != nil {
				tx.Rollback()
				return err
			}
		}

		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}

func BorrarDatosTiendaCA(db *sql.DB, codigo string, year, month int) error {
	tableName := "POS_CA_" + codigo
	_, err := db.Exec(
		fmt.Sprintf("DELETE FROM %s WHERE Tienda=? AND YEAR(Fecha)=? AND MONTH(Fecha)=?", tableName),
		codigo, year, month,
	)
	return err
}

func InsertarVentasCA_Incremental(db *sql.DB, registros []VentaCARegistro, codigo string, year, month int) error {
	tableName := "POS_CA_" + codigo

	existing := make(map[string]bool)
	rows, err := db.Query(
		fmt.Sprintf("SELECT Caja, Numero, Tipo, Codigo FROM %s WHERE Tienda=? AND YEAR(Fecha)=? AND MONTH(Fecha)=?", tableName),
		codigo, year, month,
	)
	if err == nil {
		for rows.Next() {
			var caja int
			var numero, tipo, cod string
			if err := rows.Scan(&caja, &numero, &tipo, &cod); err == nil {
				key := fmt.Sprintf("%d|%s|%s|%s", caja, numero, tipo, cod)
				existing[key] = true
			}
		}
		rows.Close()
	}

	var nuevos []VentaCARegistro
	for _, r := range registros {
		key := fmt.Sprintf("%d|%s|%s|%s", r.Caja, r.Numero, r.Tipo, r.Codigo)
		if !existing[key] {
			nuevos = append(nuevos, r)
		}
	}

	if len(nuevos) == 0 {
		return nil
	}

	return InsertarVentasCA(db, nuevos, codigo)
}

func ContarTiendaMes_SQL(db *sql.DB, codigo string, year, month int) int {
	var count int
	tableName := "POS_CA_" + codigo
	err := db.QueryRow(
		fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE Tienda=? AND YEAR(Fecha)=? AND MONTH(Fecha)=?", tableName),
		codigo, year, month,
	).Scan(&count)
	if err != nil {
		return 0
	}
	return count
}


