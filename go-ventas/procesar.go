package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

func ListarPOS(codigo, rutaIP, rutaDBF, modo string) ([]string, error) {
	var path string
	if modo == "S" {
		if rutaDBF != "" {
			path = strings.TrimRight(rutaDBF, "\\") + "\\CIERRE_POS"
		} else {
			path = fmt.Sprintf("S:\\aBC-Soft\\Data\\%s\\CIERRE_POS", codigo)
		}
	} else {
		path = fmt.Sprintf("\\\\%s\\Sistema\\aBC-Soft\\Cierre_POS", rutaIP)
	}
	path = strings.TrimRight(path, " ")

	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("Ruta no accesible: %s", path)
	}

	var posDirs []string
	for _, e := range entries {
		if e.IsDir() && strings.HasPrefix(strings.ToUpper(e.Name()), "POS") {
			posDirs = append(posDirs, e.Name())
		}
	}

	if len(posDirs) == 0 {
		return nil, fmt.Errorf("No se encontraron carpetas POS en: %s", path)
	}

	sort.Strings(posDirs)
	return posDirs, nil
}

func LeerArchivoFC(filepath string, year, month int, tienda string, caja int) (map[int]float64, map[int]int, float64, int, []VentaRegistro, error) {
	ventasPorHora := make(map[int]float64)
	conteoPorHora := make(map[int]int)
	var registros []VentaRegistro

	dbf, err := OpenDBF(filepath)
	if err != nil {
		return nil, nil, 0, 0, nil, err
	}
	defer dbf.Close()

	for i := 0; i < dbf.NumRecords; i++ {
		rec, err := dbf.ReadRecord(i)
		if err != nil || len(rec) == 0 {
			continue
		}
		if rec[0] == 0x2A {
			continue
		}

		ti := dbf.FieldIndex("TIPO")
		ai := dbf.FieldIndex("ANULADA")
		if ti < 0 {
			continue
		}

		tipo := dbf.GetString(rec, dbf.Fields[ti])
		if tipo != "FA" && tipo != "DV" {
			continue
		}
		if ai >= 0 && dbf.GetString(rec, dbf.Fields[ai]) == "T" {
			continue
		}

		fi := dbf.FieldIndex("FECHA")
		if fi < 0 {
			continue
		}
		fecha, ok := dbf.GetDate(rec, dbf.Fields[fi])
		if !ok || fecha.Year() != year || int(fecha.Month()) != month {
			continue
		}

		hi := dbf.FieldIndex("HORA24")
		if hi < 0 {
			continue
		}
		hora, ok := dbf.GetTime(rec, dbf.Fields[hi])
		if !ok {
			continue
		}

		montoUSD := 0.0
		if mi := dbf.FieldIndex("MONTOFACDL"); mi >= 0 {
			montoUSD = dbf.GetFloat(rec, dbf.Fields[mi])
		}

		if tipo == "FA" {
			ventasPorHora[hora] += montoUSD
			conteoPorHora[hora]++
		} else {
			ventasPorHora[hora] -= montoUSD
		}

		r := VentaRegistro{Fecha: fecha, Hora: hora, Tipo: tipo, MontoUSD: montoUSD, Tienda: tienda, Caja: caja}
		getStr := func(name string, def string) string {
			if idx := dbf.FieldIndex(name); idx >= 0 {
				return strings.TrimSpace(dbf.GetString(rec, dbf.Fields[idx]))
			}
			return def
		}
		getFl := func(name string) float64 {
			if idx := dbf.FieldIndex(name); idx >= 0 {
				return dbf.GetFloat(rec, dbf.Fields[idx])
			}
			return 0
		}
		r.STipo = getStr("STIPO", "")
		r.Numero = getStr("NUMERO", "")
		r.Anulada = getStr("ANULADA", "")
		r.Operador = getStr("OPERADOR", "")
		r.SubTotal = getFl("SUBTOTAL")
		r.MontoBS = getFl("MONTOFAC")
		r.Impuesto = getFl("IMPUESTO")
		r.Descuento = getFl("DESCUENTO")
		r.IGTF = getFl("IGTF")
		r.Redondeo = getFl("REDONDEO")
		r.TasaDOL = getFl("TASADOL")
		r.CodCli = getStr("CODCLI", "")
		r.Nombres = getStr("NOMBRES", "")
		r.Apellidos = getStr("APELLIDOS", "")
		r.NIT = getStr("NIT", "")
		r.RIF = getStr("RIF", "")
		r.CodVen = getStr("CODVEN", "")
		r.NroZ = getStr("NRO_Z", "")
		r.Credito = dbf.GetBool(rec, dbf.Fields[dbf.FieldIndex("CREDITO")])
		r.Vuelto = getFl("VUELTO")
		r.LIMPRIMIO = dbf.GetBool(rec, dbf.Fields[dbf.FieldIndex("LIMPRIMIO")])
		r.NroCie = getStr("NRO_CIE", "")
		if dci := dbf.FieldIndex("DATE_CIE"); dci >= 0 {
			if dt, ok := dbf.GetDate(rec, dbf.Fields[dci]); ok {
				r.FechaCie = &dt
			}
		}
		registros = append(registros, r)
	}

	totalVentas := sumVentas(ventasPorHora)
	totalFacturas := 0
	for _, c := range conteoPorHora {
		totalFacturas += c
	}
	return ventasPorHora, conteoPorHora, totalVentas, totalFacturas, registros, nil
}

func formatoHora(hora int) string {
	if hora == 0 {
		return "12:00 AM"
	} else if hora < 12 {
		return fmt.Sprintf("%02d:00 AM", hora)
	} else if hora == 12 {
		return "12:00 PM"
	}
	return fmt.Sprintf("%02d:00 PM", hora-12)
}

func ProcesarTienda(sucursal Sucursal, db *sql.DB, year, month int, modo string, forceSQL bool) (chan struct {
	Resultado ResultadoTienda
	Insert    []VentaRegistro
	Err       error
}) {
	ch := make(chan struct {
		Resultado ResultadoTienda
		Insert    []VentaRegistro
		Err       error
	}, 1)

	go func() {
		defer close(ch)
		defer func() {
			if r := recover(); r != nil {
				errMsg := struct {
					Resultado ResultadoTienda
					Insert    []VentaRegistro
					Err       error
				}{Err: fmt.Errorf("PANIC: %v", r)}
				ch <- errMsg
			}
		}()
		codigo := sucursal.Codigo
		tiendaLetra := strings.ToUpper(string(codigo[0]))

		// Process from DBF
		posDirs, err := ListarPOS(codigo, sucursal.RutaIP, sucursal.RutaDBF, modo)
		if err != nil && modo == "S" && sucursal.RutaIP != "" {
			fmt.Printf("\n  %s: S fallo, intentando IP...", codigo)
			posDirs, err = ListarPOS(codigo, sucursal.RutaIP, sucursal.RutaDBF, "IP")
		}
		if err != nil {
			ch <- struct {
				Resultado ResultadoTienda
				Insert    []VentaRegistro
				Err       error
			}{Resultado: ResultadoTienda{Tienda: tiendaLetra}, Err: fmt.Errorf("FALLO: %v", err)}
			return
		}

		tStoreStart := time.Now()
		ventasCombinadas := make(map[int]float64)
		conteoCombinado := make(map[int]int)
		cajasProcesadas := 0

		for _, dirName := range posDirs {
			posNum, err := strconv.Atoi(dirName[3:])
			if err != nil {
				continue
			}

			// Build DBF path from the POS path
			var posPath string
			if modo == "S" {
				if sucursal.RutaDBF != "" {
					posPath = strings.TrimRight(sucursal.RutaDBF, "\\") + "\\CIERRE_POS\\" + dirName
				} else {
					posPath = fmt.Sprintf("S:\\aBC-Soft\\Data\\%s\\CIERRE_POS\\%s", codigo, dirName)
				}
			} else {
				posPath = fmt.Sprintf("\\\\%s\\Sistema\\aBC-Soft\\Cierre_POS\\%s", sucursal.RutaIP, dirName)
			}

			dbfPath := filepath.Join(posPath, fmt.Sprintf("FC%02d%d.DBF", posNum, year))

			if _, err := os.Stat(dbfPath); err != nil {
				continue
			}

			tFileStart := time.Now()
			vph, cph, _, totalFacturas, regs, err := LeerArchivoFC(dbfPath, year, month, codigo, posNum)
			tFileElapsed := time.Since(tFileStart)

			if err != nil || totalFacturas == 0 {
				if totalFacturas == 0 {
					fmt.Printf("\n  [%s] -> sin datos (%.1fs)", dbfPath, tFileElapsed.Seconds())
				}
				continue
			}

			for h, v := range vph {
				ventasCombinadas[h] += v
			}
			for h, c := range cph {
				conteoCombinado[h] += c
			}
			cajasProcesadas++

			// Guardar este POS inmediatamente (si falla, los demas ya estan salvados)
			if len(regs) > 0 {
				InsertarVentas(db, regs)
			}

			fileSize := 0.0
			if info, err := os.Stat(dbfPath); err == nil {
				fileSize = float64(info.Size()) / (1024 * 1024)
			}
			fmt.Printf("\n  [%s] (%d facts, $%.0f, %.1f MB, %.1fs)",
				filepath.Base(dbfPath), totalFacturas,
				sumVentas(vph), fileSize, tFileElapsed.Seconds())
		}

		_ = cajasProcesadas

		totalVentas := 0.0
		for _, v := range ventasCombinadas {
			totalVentas += v
		}
		totalFacturas := 0
		for _, c := range conteoCombinado {
			totalFacturas += c
		}

		if totalFacturas == 0 {
			ch <- struct {
				Resultado ResultadoTienda
				Insert    []VentaRegistro
				Err       error
			}{Resultado: ResultadoTienda{Tienda: tiendaLetra}}
			return
		}

		promedio := totalVentas / float64(totalFacturas)
		hmKey, cm := maxKey(conteoCombinado)
		hm := formatoHora(hmKey)
		hmiKey, cmi := minKey(conteoCombinado)
		hmi := formatoHora(hmiKey)

		_ = tStoreStart
		fmt.Printf("\n  => %d cajas | %d facts | Total: $%.2f | Prom: $%.2f | Pico: %s (%d)",
			cajasProcesadas, totalFacturas, totalVentas, promedio, hm, cm)

		ch <- struct {
			Resultado ResultadoTienda
			Insert    []VentaRegistro
			Err       error
		}{
			Resultado: ResultadoTienda{
				Tienda:          tiendaLetra,
				PromedioFactura: promedio,
				Clientes:        totalFacturas,
				Total:           totalVentas,
				HoraMayor:       hm,
				ClientesMayor:   cm,
				HoraMenor:       hmi,
				ClientesMenor:   cmi,
			},
			Insert: nil,
		}
	}()

	return ch
}

func sumVentas(m map[int]float64) float64 {
	var s float64
	for _, v := range m {
		s += v
	}
	return s
}

func maxKey(m map[int]int) (int, int) {
	var bestK int
	var bestV int
	first := true
	for k, v := range m {
		if first || v > bestV {
			bestK = k
			bestV = v
			first = false
		}
	}
	return bestK, bestV
}

func minKey(m map[int]int) (int, int) {
	var bestK int
	var bestV int
	first := true
	for k, v := range m {
		if first || v < bestV {
			bestK = k
			bestV = v
			first = false
		}
	}
	return bestK, bestV
}

func calcularResultado(tiendaLetra string, data map[int]VentaPorHora) ResultadoTienda {
	conteo := make(map[int]int)
	total := 0.0
	clientes := 0
	for h, v := range data {
		conteo[h] = v.Facturas
		total += v.TotalUSD
		clientes += v.Facturas
	}

	promedio := 0.0
	if clientes > 0 {
		promedio = total / float64(clientes)
	}

	hmKey, cm := maxKey(conteo)
	hm := formatoHora(hmKey)
	hmiKey, cmi := minKey(conteo)
	hmi := formatoHora(hmiKey)

	return ResultadoTienda{
		Tienda:          tiendaLetra,
		PromedioFactura: promedio,
		Clientes:        clientes,
		Total:           total,
		HoraMayor:       hm,
		ClientesMayor:   cm,
		HoraMenor:       hmi,
		ClientesMenor:   cmi,
	}
}
