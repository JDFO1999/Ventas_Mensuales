package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	mssql "github.com/denisenkom/go-mssqldb"
)

const (
	SQLServer       = "10.10.10.21"
	SQLDatabase     = "Sistemas"
	SQLUser         = "Jose_Sistemas"
	SQLPassword     = "Zuleidy14*$"
)

func ConectarSQL() (*sql.DB, error) {
	connStr := fmt.Sprintf("server=%s;database=%s;user id=%s;password=%s;encrypt=disable;connection timeout=30;timeout=600",
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
	connStr := fmt.Sprintf("server=%s;database=BD_Tiendas;user id=%s;password=%s;encrypt=disable;connection timeout=30;timeout=600",
		SQLServer, SQLUser, SQLPassword)
	db, err := sql.Open("mssql", connStr)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(0)
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
	if err != nil {
		return err
	}
	_, err = db.Exec(`
		IF OBJECT_ID('PK_PosVentas','PK') IS NULL
		BEGIN
			DELETE t FROM (
				SELECT *, ROW_NUMBER() OVER (
					PARTITION BY Tienda, Caja, Numero, Tipo ORDER BY FechaCarga
				) AS rn FROM PosVentas
			) t WHERE t.rn > 1
			ALTER TABLE PosVentas ADD CONSTRAINT PK_PosVentas PRIMARY KEY (Tienda, Caja, Numero, Tipo)
		END
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

func ContarFASQL(db *sql.DB, codigo string, year, month int) (int, error) {
	var count int
	startDate := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	endDate := startDate.AddDate(0, 1, 0)
	err := db.QueryRow(
		"SELECT COUNT(*) FROM PosVentas WHERE Tienda=? AND Fecha >= ? AND Fecha < ? AND Tipo='FA'",
		codigo, startDate, endDate,
	).Scan(&count)
	return count, err
}

func BorrarDatosTienda(ctx context.Context, db *sql.DB, codigo string, year, month int) error {
	startDate := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	endDate := startDate.AddDate(0, 1, 0)
	for {
		result, err := db.ExecContext(ctx,
			"DELETE TOP(10000) FROM PosVentas WHERE Tienda=? AND Fecha >= ? AND Fecha < ?",
			codigo, startDate, endDate,
		)
		if err != nil {
			return err
		}
		rows, _ := result.RowsAffected()
		if rows == 0 {
			break
		}
	}
	return nil
}

func ReconciliarPosVentas(db *sql.DB, codigo string, year, month int, dbfKeys map[string]bool) (int, error) {
	startDate := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	endDate := startDate.AddDate(0, 1, 0)

	rows, err := db.Query(
		`SELECT Caja, Numero, Tipo FROM PosVentas
		 WHERE Tienda=? AND Fecha >= ? AND Fecha < ?`,
		codigo, startDate, endDate)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	type key struct {
		caja   int
		numero string
		tipo   string
	}
	var huerfanos []key

	for rows.Next() {
		var caja int
		var numero, tipo string
		if err := rows.Scan(&caja, &numero, &tipo); err != nil {
			return 0, err
		}
		k := fmt.Sprintf("%d|%s|%s", caja, numero, tipo)
		if !dbfKeys[k] {
			huerfanos = append(huerfanos, key{caja, numero, tipo})
		}
	}

	eliminados := 0
	for _, h := range huerfanos {
		_, err := db.Exec(
			`DELETE FROM PosVentas WHERE Tienda=? AND Caja=? AND Numero=? AND Tipo=?`,
			codigo, h.caja, h.numero, h.tipo)
		if err != nil {
			return eliminados, err
		}
		eliminados++
	}
	return eliminados, nil
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
	pkName := "PK_" + tableName
	db.Exec(fmt.Sprintf("IF OBJECT_ID('%s','PK') IS NOT NULL ALTER TABLE %s DROP CONSTRAINT %s", pkName, tableName, pkName))
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
	if err != nil {
		return err
	}
	db.Exec(fmt.Sprintf(`
		IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE object_id = OBJECT_ID('%s') AND name = 'IX_POS_CA_Fecha')
		CREATE NONCLUSTERED INDEX IX_POS_CA_Fecha ON %s(Tienda, Fecha)
	`, tableName, tableName))
	return nil
}

func InsertarVentasCA(db *sql.DB, registros []VentaCARegistro, codigo string) error {
	const maxChunkSize = 5000
	tableName := "POS_CA_" + codigo
	query := fmt.Sprintf(`INSERT INTO %s (Fecha, Hora, Tienda, Caja, Tipo, STipo, Numero, Codigo, CodBar, Descrip,
		CodVen, Modelo, Serial, Cantidad, NCntd, NPvpDol, NPvp2Dol, NPvp3Dol, NPvpCop,
		Precio, NPrecio, IGV, NoDscto, CodCli, Anulada, Depto, Familia,
		Costo, NCosDol, Pvpt, Oferta, Devlto, Margen, PvpVen, LPesado, NroCie, FechaCie)
		VALUES (?,?,?,?,?,?,?,?,?,?,
			?,?,?,?,?,?,?,?,?,
			?,?,?,?,?,?,?,?,
			?,?,?,?,?,?,?,?,?,?)`, tableName)

	chunkSize := maxChunkSize
	for i := 0; i < len(registros); i += chunkSize {
		end := i + chunkSize
		if end > len(registros) {
			end = len(registros)
		}
		chunk := registros[i:end]

		var lastErr error
		for retry := 0; retry < 3; retry++ {
			if retry > 0 {
				time.Sleep(time.Duration(retry) * 2 * time.Second)
				if retry == 2 {
					chunkSize = maxChunkSize / 2
				}
				if retry == 1 {
					chunkSize = maxChunkSize
				}
				end = i + chunkSize
				if end > len(registros) {
					end = len(registros)
				}
				chunk = registros[i:end]
			}

			tx, err := db.Begin()
			if err != nil {
				lastErr = err
				continue
			}

			err = nil
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
					break
				}
			}

			if err != nil {
				tx.Rollback()
				lastErr = err
				continue
			}

			if err := tx.Commit(); err != nil {
				lastErr = err
				continue
			}

			lastErr = nil
			break
		}

		if lastErr != nil {
			return fmt.Errorf("chunk %d-%d fallo tras 3 intentos: %v", i, end, lastErr)
		}
	}
	return nil
}

func InsertarVentasCA_Incremental(db *sql.DB, registros []VentaCARegistro, codigo string, year, month int, forceReload ...bool) error {
	tableName := "POS_CA_" + codigo
	startDate := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	endDate := startDate.AddDate(0, 1, 0)

	force := len(forceReload) > 0 && forceReload[0]

	var nuevos []VentaCARegistro
	if !force {
		existing := make(map[string]bool)
		var loadErr error
		for retry := 0; retry < 3; retry++ {
			if retry > 0 {
				time.Sleep(time.Duration(retry) * time.Second)
			}
			existing = make(map[string]bool)
			rows, err := db.Query(
				fmt.Sprintf("SELECT Caja, Numero, Tipo, Codigo FROM %s WHERE Tienda=? AND Fecha >= ? AND Fecha < ?", tableName),
				codigo, startDate, endDate,
			)
			if err != nil {
				loadErr = err
				continue
			}
			for rows.Next() {
				var caja int
				var numero, tipo, cod string
				if err := rows.Scan(&caja, &numero, &tipo, &cod); err == nil {
					key := fmt.Sprintf("%d|%s|%s|%s", caja, numero, tipo, cod)
					existing[key] = true
				}
			}
			rows.Close()
			loadErr = nil
			break
		}
		if loadErr != nil {
			return fmt.Errorf("fallo leyendo claves existentes tras 3 intentos: %v", loadErr)
		}

		for _, r := range registros {
			key := fmt.Sprintf("%d|%s|%s|%s", r.Caja, r.Numero, r.Tipo, r.Codigo)
			if !existing[key] {
				nuevos = append(nuevos, r)
			}
		}
	} else {
		nuevos = registros
	}

	if len(nuevos) == 0 {
		return nil
	}

	return InsertarVentasCA(db, nuevos, codigo)
}

func ContarTiendaMes_SQL(ctx context.Context, db *sql.DB, codigo string, year, month int) (int, error) {
	var count int
	tableName := "POS_CA_" + codigo
	startDate := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	endDate := startDate.AddDate(0, 1, 0)
	err := db.QueryRowContext(ctx,
		fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE Tienda=? AND Fecha >= ? AND Fecha < ?", tableName),
		codigo, startDate, endDate,
	).Scan(&count)
	return count, err
}

func ContarTiendaRango_SQL(ctx context.Context, db *sql.DB, codigo string, inicio, fin time.Time) (int, error) {
	var count int
	tableName := "POS_CA_" + codigo
	err := db.QueryRowContext(ctx,
		fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE Tienda=? AND Fecha >= ? AND Fecha < ?", tableName),
		codigo, inicio, fin,
	).Scan(&count)
	return count, err
}

func BorrarDatosTiendaCA(ctx context.Context, db *sql.DB, codigo string, year, month int) error {
	tableName := "POS_CA_" + codigo
	startDate := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	endDate := startDate.AddDate(0, 1, 0)

	diasMes := endDate.Sub(startDate).Hours() / 24
	for d := 0; d < int(diasMes); d++ {
		dia := startDate.AddDate(0, 0, d)
		diaFin := dia.AddDate(0, 0, 1)
		if diaFin.After(endDate) {
			diaFin = endDate
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("DELETE timeout: %v", ctx.Err())
		default:
		}

		result, err := db.ExecContext(ctx,
			fmt.Sprintf("DELETE FROM %s WHERE Tienda=? AND Fecha >= ? AND Fecha < ?", tableName),
			codigo, dia, diaFin,
		)
		if err != nil {
			for retry := 0; retry < 2; retry++ {
				time.Sleep(time.Duration(retry+1) * time.Second)
				result, err = db.ExecContext(ctx,
					fmt.Sprintf("DELETE TOP(50000) FROM %s WHERE Tienda=? AND Fecha >= ? AND Fecha < ?", tableName),
					codigo, dia, diaFin,
				)
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

func BorrarRangoTiendaCA(ctx context.Context, db *sql.DB, codigo string, inicio, fin time.Time) error {
	tableName := "POS_CA_" + codigo

	dias := int(fin.Sub(inicio).Hours() / 24)
	for d := 0; d < dias; d++ {
		dia := inicio.AddDate(0, 0, d)
		diaFin := dia.AddDate(0, 0, 1)
		if diaFin.After(fin) {
			diaFin = fin
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("DELETE timeout: %v", ctx.Err())
		default:
		}

		result, err := db.ExecContext(ctx,
			fmt.Sprintf("DELETE FROM %s WHERE Tienda=? AND Fecha >= ? AND Fecha < ?", tableName),
			codigo, dia, diaFin,
		)
		if err != nil {
			for retry := 0; retry < 2; retry++ {
				time.Sleep(time.Duration(retry+1) * time.Second)
				result, err = db.ExecContext(ctx,
					fmt.Sprintf("DELETE TOP(50000) FROM %s WHERE Tienda=? AND Fecha >= ? AND Fecha < ?", tableName),
					codigo, dia, diaFin,
				)
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

func CrearTablaH_Tiendas(db *sql.DB) error {
	_, err := db.Exec(`
		IF OBJECT_ID('H_Tiendas','U') IS NULL
		CREATE TABLE H_Tiendas (
			Id              INT IDENTITY(1,1) PRIMARY KEY,
			FechaEjecucion  DATETIME NOT NULL DEFAULT GETDATE(),
			Tienda          VARCHAR(6) NOT NULL,
			Anio            INT NOT NULL,
			Mes             INT NOT NULL,
			RegistrosDBF    INT NOT NULL DEFAULT 0,
			RegistrosSQL    INT NOT NULL DEFAULT 0,
			Insertados      INT NOT NULL DEFAULT 0,
			Estado          VARCHAR(20) NOT NULL,
			DuracionSeg     INT NULL,
			Modo            VARCHAR(10) NOT NULL,
			TipoEjecucion   VARCHAR(20) NOT NULL,
			Sistema         VARCHAR(10) NULL
		)
	`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`
		IF (SELECT COLUMNPROPERTY(OBJECT_ID('H_Tiendas'),'Id','IsIdentity')) = 0
		BEGIN
			IF OBJECT_ID('dbo.H_Tiendas_New','U') IS NOT NULL
			DROP TABLE dbo.H_Tiendas_New;

			CREATE TABLE dbo.H_Tiendas_New (
				Id              INT IDENTITY(1,1) PRIMARY KEY,
				FechaEjecucion  DATETIME NOT NULL DEFAULT GETDATE(),
				Tienda          VARCHAR(6) NOT NULL,
				Anio            INT NOT NULL,
				Mes             INT NOT NULL,
				RegistrosDBF    INT NOT NULL DEFAULT 0,
				RegistrosSQL    INT NOT NULL DEFAULT 0,
				Insertados      INT NOT NULL DEFAULT 0,
				Estado          VARCHAR(20) NOT NULL,
				DuracionSeg     INT NULL,
				Modo            VARCHAR(10) NOT NULL,
				TipoEjecucion   VARCHAR(20) NOT NULL,
				Sistema         VARCHAR(10) NULL
			);

			SET IDENTITY_INSERT dbo.H_Tiendas_New ON;
			INSERT INTO dbo.H_Tiendas_New (Id, FechaEjecucion, Tienda, Anio, Mes, RegistrosDBF, RegistrosSQL, Insertados, Estado, DuracionSeg, Modo, TipoEjecucion, Sistema)
			SELECT Id, FechaEjecucion, Tienda, Anio, Mes, RegistrosDBF, RegistrosSQL, Insertados, Estado, DuracionSeg, Modo, TipoEjecucion, Sistema FROM dbo.H_Tiendas;
			SET IDENTITY_INSERT dbo.H_Tiendas_New OFF;

			DROP TABLE dbo.H_Tiendas;
			EXEC sp_rename N'dbo.H_Tiendas_New', N'H_Tiendas';

			IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE object_id = OBJECT_ID('H_Tiendas') AND name = 'IX_H_Tiendas_Busqueda')
			CREATE NONCLUSTERED INDEX IX_H_Tiendas_Busqueda ON H_Tiendas(Tienda, Anio, Mes)
		END
	`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`
		IF NOT EXISTS (SELECT 1 FROM sys.columns WHERE object_id = OBJECT_ID('H_Tiendas') AND name = 'Sistema')
		ALTER TABLE H_Tiendas ADD Sistema VARCHAR(10) NULL
	`)
	return err
}

func InsertarH_Tiendas(db *sql.DB, tienda string, anio, mes int, registrosDBF, registrosSQL, insertados int, estado string, duracionSeg int, modo string, tipoEjecucion string, sistema string) error {
	_, err := db.Exec(
		`INSERT INTO H_Tiendas (Tienda, Anio, Mes, RegistrosDBF, RegistrosSQL, Insertados, Estado, DuracionSeg, Modo, TipoEjecucion, Sistema)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		tienda, anio, mes, registrosDBF, registrosSQL, insertados, estado, duracionSeg, modo, tipoEjecucion, sistema,
	)
	if err != nil {
		logError("H_Tiendas insert fallo (%s %s %d): %v", sistema, tienda, anio, err)
	}
	return err
}

func BorrarDiaCA(ctx context.Context, db *sql.DB, codigo string, year, month, day int) (int, error) {
	tableName := "POS_CA_" + codigo
	dia := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	diaFin := dia.AddDate(0, 0, 1)

	totalBorrado := 0
	for {
		select {
		case <-ctx.Done():
			return totalBorrado, fmt.Errorf("DELETE timeout: %v", ctx.Err())
		default:
		}

		result, err := db.ExecContext(ctx,
			fmt.Sprintf("DELETE TOP(50000) FROM %s WHERE Tienda=? AND Fecha >= ? AND Fecha < ?", tableName),
			codigo, dia, diaFin,
		)
		if err != nil {
			return totalBorrado, err
		}

		rows, _ := result.RowsAffected()
		totalBorrado += int(rows)

		if rows == 0 {
			break
		}
	}

	return totalBorrado, nil
}

func ContarSQLPorDia(ctx context.Context, db *sql.DB, codigo string, year, month, day int) (int, error) {
	tableName := "POS_CA_" + codigo
	dia := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	diaFin := dia.AddDate(0, 0, 1)

	var count int
	err := db.QueryRowContext(ctx,
		fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE Tienda=? AND Fecha >= ? AND Fecha < ?", tableName),
		codigo, dia, diaFin,
	).Scan(&count)
	return count, err
}

// ==================== STAGING TABLE FUNCTIONS ====================

func CrearTablaStagingCA(db *sql.DB, codigo string) error {
	stgTable := "POS_CA_" + codigo + "_STG"
	mainTable := "POS_CA_" + codigo
	db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", stgTable))
	_, err := db.Exec(fmt.Sprintf(`SELECT * INTO %s FROM %s WHERE 1=0`, stgTable, mainTable))
	if err != nil {
		return fmt.Errorf("error creando staging: %v", err)
	}
	db.Exec(fmt.Sprintf("ALTER TABLE %s ADD _RowID INT IDENTITY(1,1)", stgTable))
	db.Exec(fmt.Sprintf("CREATE CLUSTERED INDEX IX_STG_RowID ON %s(_RowID)", stgTable))
	return nil
}

func LimpiarStaging(db *sql.DB, codigo string) {
	stgTable := "POS_CA_" + codigo + "_STG"
	db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", stgTable))
}

func CargarDBFEnStaging(db *sql.DB, registros []VentaCARegistro, codigo string) error {
	if len(registros) == 0 {
		return nil
	}
	stgTable := "POS_CA_" + codigo + "_STG"

	cols := []string{
		"Fecha", "Hora", "Tienda", "Caja", "Tipo", "STipo", "Numero", "Codigo", "CodBar", "Descrip",
		"CodVen", "Modelo", "Serial", "Cantidad", "NCntd", "NPvpDol", "NPvp2Dol", "NPvp3Dol", "NPvpCop",
		"Precio", "NPrecio", "IGV", "NoDscto", "CodCli", "Anulada", "Depto", "Familia",
		"Costo", "NCosDol", "Pvpt", "Oferta", "Devlto", "Margen", "PvpVen", "LPesado", "NroCie", "FechaCie",
	}

	copyStmt := mssql.CopyIn(stgTable, mssql.BulkOptions{}, cols...)
	bulkStmt, err := db.Prepare(copyStmt)
	if err != nil {
		return fmt.Errorf("preparando bulk copy staging: %v", err)
	}

	for i, r := range registros {
		var fechaCie interface{}
		if r.FechaCie != nil {
			fechaCie = *r.FechaCie
		}
		_, err := bulkStmt.Exec(
			r.Fecha, r.Hora, r.Tienda, r.Caja, r.Tipo, r.STipo, r.Numero, r.Codigo, r.CodBar, r.Descrip,
			r.CodVen, r.Modelo, r.Serial, r.Cantidad, r.NCntd, r.NPvpDol, r.NPvp2Dol, r.NPvp3Dol, r.NPvpCop,
			r.Precio, r.NPrecio, r.IGV, r.NoDscto, r.CodCli, r.Anulada, r.Depto, r.Familia,
			r.Costo, r.NCosDol, r.Pvpt, r.Oferta, r.Devlto, r.Margen, r.PvpVen, r.LPesado, r.NroCie, fechaCie,
		)
		if err != nil {
			bulkStmt.Close()
			return fmt.Errorf("bulk copy row %d: %v", i, err)
		}
		if (i+1)%5000 == 0 || i+1 == len(registros) {
			fmt.Printf("\r  STG: %d/%d registros cargados...    ", i+1, len(registros))
		}
	}

	_, err = bulkStmt.Exec()
	bulkStmt.Close()
	if err != nil {
		return fmt.Errorf("finalizando bulk copy staging: %v", err)
	}
	fmt.Printf("\n")
	return nil
}

func CompararStagingConSQL(db *sql.DB, codigo string, inicio, fin time.Time) (iguales bool, countSQL int, countSTG int, err error) {
	stgTable := "POS_CA_" + codigo + "_STG"
	mainTable := "POS_CA_" + codigo

	err = db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", stgTable)).Scan(&countSTG)
	if err != nil {
		return false, 0, 0, err
	}

	err = db.QueryRow(fmt.Sprintf(
		"SELECT COUNT(*) FROM %s WHERE Tienda=? AND Fecha >= ? AND Fecha < ?", mainTable),
		codigo, inicio, fin,
	).Scan(&countSQL)
	if err != nil {
		return false, 0, 0, err
	}

	return countSTG == countSQL && countSTG > 0, countSQL, countSTG, nil
}

func BorrarRangoEnBatches(ctx context.Context, db *sql.DB, codigo string, inicio, fin time.Time) (int, error) {
	tableName := "POS_CA_" + codigo
	totalBorrado := 0
	batchSize := 10000
	maxReintentos := 3

	// Desactivar indice no agrupado temporalmente para acelerar el borrado masivo y evitar timeouts
	db.Exec(fmt.Sprintf("DROP INDEX IF EXISTS IX_POS_CA_Fecha ON %s", tableName))

	for {
		select {
		case <-ctx.Done():
			return totalBorrado, fmt.Errorf("DELETE timeout: %v", ctx.Err())
		default:
		}

		result, err := db.ExecContext(ctx,
			fmt.Sprintf("DELETE TOP(%d) FROM %s WHERE Tienda=? AND Fecha >= ? AND Fecha < ?", batchSize, tableName),
			codigo, inicio, fin,
		)
		if err != nil {
			reintentos := 0
			for reintentos < maxReintentos {
				select {
				case <-ctx.Done():
					return totalBorrado, fmt.Errorf("DELETE timeout: %v", ctx.Err())
				default:
				}
				// Reduccion dinamica del lote si hay saturacion de I/O o timeout
				if batchSize > 1000 {
					batchSize = batchSize / 2
				}
				logWarn("DELETE: batch fallo (%v), reduciendo lote a %d, reintento %d", err, batchSize, reintentos+1)
				time.Sleep(3 * time.Second)
				result, err = db.ExecContext(ctx,
					fmt.Sprintf("DELETE TOP(%d) FROM %s WHERE Tienda=? AND Fecha >= ? AND Fecha < ?", batchSize, tableName),
					codigo, inicio, fin,
				)
				if err == nil {
					break
				}
				reintentos++
			}
			if err != nil {
				return totalBorrado, err
			}
		}

		rows, _ := result.RowsAffected()
		if rows == 0 {
			break
		}
		totalBorrado += int(rows)
		// Breve descanso para permitir que SQL Server vacie el log de transacciones
		time.Sleep(50 * time.Millisecond)
	}

	return totalBorrado, nil
}

func CopiarStagingATable(ctx context.Context, db *sql.DB, codigo string) (int64, error) {
	stgTable := "POS_CA_" + codigo + "_STG"
	mainTable := "POS_CA_" + codigo
	cols := "Fecha, Hora, Tienda, Caja, Tipo, STipo, Numero, Codigo, CodBar, Descrip, CodVen, Modelo, Serial, Cantidad, NCntd, NPvpDol, NPvp2Dol, NPvp3Dol, NPvpCop, Precio, NPrecio, IGV, NoDscto, CodCli, Anulada, Depto, Familia, Costo, NCosDol, Pvpt, Oferta, Devlto, Margen, PvpVen, LPesado, NroCie, FechaCie"

	var countSTG int
	db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", stgTable)).Scan(&countSTG)
	logInfo("COPY: staging=%s tiene %d registros", stgTable, countSTG)
	if countSTG == 0 {
		return 0, fmt.Errorf("staging vacio")
	}

	var maxRowID int
	err := db.QueryRow(fmt.Sprintf("SELECT ISNULL(MAX(_RowID), 0) FROM %s", stgTable)).Scan(&maxRowID)
	if err != nil {
		return 0, fmt.Errorf("error obteniendo max rowid de staging: %v", err)
	}

	db.Exec(fmt.Sprintf("DROP INDEX IF EXISTS IX_POS_CA_Fecha ON %s", mainTable))

	batchSize := 10000
	maxReintentos := 3
	var totalCopied int64
	offset := 0

	for offset < maxRowID {
		select {
		case <-ctx.Done():
			return totalCopied, fmt.Errorf("COPY timeout: %v", ctx.Err())
		default:
		}

		limit := offset + batchSize
		result, err := db.ExecContext(ctx, fmt.Sprintf(
			"INSERT INTO %s (%s) SELECT %s FROM %s WHERE _RowID > %d AND _RowID <= %d",
			mainTable, cols, cols, stgTable, offset, limit))
		if err != nil {
			reintentos := 0
			for reintentos < maxReintentos {
				select {
				case <-ctx.Done():
					return totalCopied, fmt.Errorf("COPY timeout: %v", ctx.Err())
				default:
				}
				logWarn("COPY: batch %d-%d fallo (%v), reintento %d", offset, limit, err, reintentos+1)
				time.Sleep(3 * time.Second)
				result, err = db.ExecContext(ctx, fmt.Sprintf(
					"INSERT INTO %s (%s) SELECT %s FROM %s WHERE _RowID > %d AND _RowID <= %d",
					mainTable, cols, cols, stgTable, offset, limit))
				if err == nil {
					break
				}
				reintentos++
			}
			if err != nil {
				return totalCopied, fmt.Errorf("batch %d-%d error: %v", offset, limit, err)
			}
		}

		rows, _ := result.RowsAffected()
		totalCopied += rows
		if rows > 0 {
			logInfo("COPY: batch %d-%d copiados %d (progreso: %d/%d)", offset, limit, rows, totalCopied, countSTG)
		}
		offset += batchSize
		time.Sleep(30 * time.Millisecond)
	}

	if totalCopied != int64(countSTG) {
		logWarn("COPY: registros copiados (%d) difiere del total en staging (%d)", totalCopied, countSTG)
	}

	logInfo("COPY: %d registros copiados a %s", totalCopied, mainTable)

	db.Exec(fmt.Sprintf(`
		IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE object_id = OBJECT_ID('%s') AND name = 'IX_POS_CA_Fecha')
		CREATE NONCLUSTERED INDEX IX_POS_CA_Fecha ON %s(Tienda, Fecha)`, mainTable, mainTable))

	return totalCopied, nil
}


