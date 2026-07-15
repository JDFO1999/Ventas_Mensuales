package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
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
		fmt.Println("  SQL: tabla Sucursal no accesible, cargando respaldo local...")
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
	data, err := json.MarshalIndent(sucursales, "", "  ")
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

func CrearTablaPosVentasCA(db *sql.DB) error {
	_, err := db.Exec(`
		IF OBJECT_ID('Pos_Ventas_CA','U') IS NULL
		CREATE TABLE Pos_Ventas_CA (
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
			FechaCarga DATETIME DEFAULT GETDATE(),
			CONSTRAINT PK_Pos_Ventas_CA PRIMARY KEY (Tienda, Caja, Numero, Tipo, Codigo)
		)
	`)
	return err
}

func InsertarVentasCA(db *sql.DB, registros []VentaCARegistro) error {
	const chunkSize = 5000
	query := `INSERT INTO Pos_Ventas_CA (Fecha, Hora, Tienda, Caja, Tipo, STipo, Numero, Codigo, CodBar, Descrip,
		CodVen, Modelo, Serial, Cantidad, NCntd, NPvpDol, NPvp2Dol, NPvp3Dol, NPvpCop,
		Precio, NPrecio, IGV, NoDscto, CodCli, Anulada, Depto, Familia,
		Costo, NCosDol, Pvpt, Oferta, Devlto, Margen, PvpVen, LPesado, NroCie, FechaCie)
		VALUES (?,?,?,?,?,?,?,?,?,?,
			?,?,?,?,?,?,?,?,?,
			?,?,?,?,?,?,?,?,
			?,?,?,?,?,?,?,?,?,?)`

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
				if strings.Contains(err.Error(), "PRIMARY") ||
					strings.Contains(err.Error(), "clave duplicada") ||
					strings.Contains(err.Error(), "duplicada") ||
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
		fmt.Printf("\r  SQL CA: %d/%d (%d%%)", end, len(registros), 100*end/len(registros))
	}
	fmt.Print("\r")
	return nil
}

func LeerDatosDesdeSQL_CA(db *sql.DB, year, month int) ([]VentaCARegistro, error) {
	rows, err := db.Query(`
		SELECT Fecha, Hora, Tienda, Caja, Tipo, STipo, Numero, Codigo, CodBar, Descrip,
			CodVen, Modelo, Serial, Cantidad, NCntd, NPvpDol, NPvp2Dol, NPvp3Dol, NPvpCop,
			Precio, NPrecio, IGV, NoDscto, CodCli, Anulada, Depto, Familia,
			Costo, NCosDol, Pvpt, Oferta, Devlto, Margen, PvpVen, LPesado, NroCie, FechaCie
		FROM Pos_Ventas_CA
		WHERE YEAR(Fecha)=? AND MONTH(Fecha)=?
		ORDER BY Tienda, Fecha, Caja, Numero
	`, year, month)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var registros []VentaCARegistro
	for rows.Next() {
		var r VentaCARegistro
		var fechaCie sql.NullTime
		var codBar, modelo, serial, codCli, anulada, pvpt, nroCie sql.NullString
		if err := rows.Scan(&r.Fecha, &r.Hora, &r.Tienda, &r.Caja, &r.Tipo, &r.STipo, &r.Numero,
			&r.Codigo, &codBar, &r.Descrip, &r.CodVen, &modelo, &serial,
			&r.Cantidad, &r.NCntd, &r.NPvpDol, &r.NPvp2Dol, &r.NPvp3Dol, &r.NPvpCop,
			&r.Precio, &r.NPrecio, &r.IGV, &r.NoDscto, &codCli, &anulada, &r.Depto, &r.Familia,
			&r.Costo, &r.NCosDol, &pvpt, &r.Oferta, &r.Devlto, &r.Margen, &r.PvpVen,
			&r.LPesado, &nroCie, &fechaCie); err != nil {
			return nil, err
		}
		r.CodBar = codBar.String
		r.Modelo = modelo.String
		r.Serial = serial.String
		r.CodCli = codCli.String
		r.Anulada = anulada.String
		r.Pvpt = pvpt.String
		r.NroCie = nroCie.String
		if fechaCie.Valid {
			t := fechaCie.Time
			r.FechaCie = &t
		}
		registros = append(registros, r)
	}
	return registros, rows.Err()
}

func TiendaTieneDatosCA(db *sql.DB, codigo string, year int) bool {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM Pos_Ventas_CA WHERE Tienda=? AND YEAR(Fecha)=?", codigo, year).Scan(&count)
	if err != nil {
		return false
	}
	return count > 0
}
