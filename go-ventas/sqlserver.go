package main

import (
	"database/sql"
	"fmt"
	"strings"

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

func ObtenerSucursales(db *sql.DB) ([]Sucursal, error) {
	rows, err := db.Query(`
		SELECT cCodigo, cNombre, cRutaIP, cRutaDBF, cCia
		FROM Sucursal
		WHERE lInactiva = 0 OR lInactiva IS NULL
		ORDER BY cCodigo
	`)
	if err != nil {
		return nil, err
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
	return sucursales, nil
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
