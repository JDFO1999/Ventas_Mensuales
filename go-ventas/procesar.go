package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
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
		if err != nil {
			if modo == "S" && sucursal.RutaIP != "" {
				fmt.Printf("\n  %s: S fallo, intentando IP...", codigo)
				posDirs, err = ListarPOS(codigo, sucursal.RutaIP, sucursal.RutaDBF, "IP")
				if err == nil {
					modo = "IP"
				}
			} else if modo == "IP" {
				fmt.Printf("\n  %s: IP fallo, intentando S...", codigo)
				posDirs, err = ListarPOS(codigo, sucursal.RutaIP, sucursal.RutaDBF, "S")
				if err == nil {
					modo = "S"
				}
			}
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
		dbfKeys := make(map[string]bool)

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
				if err != nil {
					logError("FA: error leyendo %s: %v", dbfPath, err)
				} else {
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
				for _, r := range regs {
					key := fmt.Sprintf("%d|%s|%s", r.Caja, r.Numero, r.Tipo)
					dbfKeys[key] = true
				}
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

		// Reconciliar: eliminar registros en SQL que no estan en DBF
		if len(dbfKeys) > 0 {
			eliminados, err := ReconciliarPosVentas(db, codigo, year, month, dbfKeys)
			if err != nil {
				logError("FA: error reconciliando %s: %v", codigo, err)
			} else if eliminados > 0 {
				fmt.Printf("\n  %s: %d registros huerfanos eliminados (no estan en DBF)", codigo, eliminados)
			}
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

func procesarConReintentos(tasks []tareaTienda, total int, db *sql.DB, year, month int, modo string, conProgreso bool) []ResultadoTienda {
	maxRetries := 3
	resultados := make([]ResultadoTienda, total)
	pendientes := make([]tareaTienda, len(tasks))
	copy(pendientes, tasks)

	for attempt := 0; attempt < maxRetries && len(pendientes) > 0; attempt++ {
		if attempt > 0 {
			fmt.Printf("\n  >>> REINTENTO %d: %d tiendas pendientes...\n", attempt, len(pendientes))
		}

		var nuevosPendientes []tareaTienda
		var wg sync.WaitGroup

		type storeRes struct {
			tt      tareaTienda
			res     ResultadoTienda
			fallo   bool
			timeout bool
		}
		resultChan := make(chan storeRes, len(pendientes))

		sem := make(chan struct{}, 6)
		for _, t := range pendientes {
			sem <- struct{}{}
			wg.Add(1)
			go func(tt tareaTienda) {
				defer wg.Done()
				defer func() { <-sem }()
				defer func() {
					if r := recover(); r != nil {
						fmt.Printf("\n  %s - PANIC: %v\n", tt.suc.Codigo, r)
						resultChan <- storeRes{tt: tt, fallo: true}
					}
				}()

				ch := ProcesarTienda(tt.suc, db, year, month, modo, true)
				select {
				case msg := <-ch:
					if msg.Err != nil {
						fmt.Printf("\n  [%d/%d] %s - ERROR: %v\n", tt.idx+1, total, tt.suc.Codigo, msg.Err)
						resultChan <- storeRes{tt: tt, fallo: true}
					} else {
						resultChan <- storeRes{tt: tt, res: msg.Resultado}
					}
				case <-time.After(5 * time.Minute):
					fmt.Printf("\n  [%d/%d] %s - TIMEOUT (5 min)\n", tt.idx+1, total, tt.suc.Codigo)
					resultChan <- storeRes{tt: tt, fallo: true, timeout: true}
				}
			}(t)
		}

		wg.Wait()
		close(resultChan)

		for msg := range resultChan {
			if msg.fallo {
				if !msg.timeout {
					nuevosPendientes = append(nuevosPendientes, msg.tt)
				} else {
					fmt.Printf("  %s - TIMEOUT, no se reintentara (problema de red)\n", msg.tt.suc.Codigo)
				}
			} else {
				resultados[msg.tt.idx] = msg.res
				if conProgreso && msg.res.Clientes > 0 {
					fmt.Printf("\n  [%d/%d] %s - %d facts | Total: $%.2f | Pico: %s (%d)\n",
						msg.tt.idx+1, total, msg.tt.suc.Codigo,
						msg.res.Clientes, msg.res.Total,
						msg.res.HoraMayor, msg.res.ClientesMayor)
				}
			}
		}

		pendientes = nuevosPendientes
	}

	if len(pendientes) > 0 {
		fmt.Printf("\n  *** ADVERTENCIA: %d tiendas sin procesar tras %d intentos ***\n", len(pendientes), maxRetries)
		for _, t := range pendientes {
			fmt.Printf("    - %s\n", t.suc.Codigo)
		}
	}

	return resultados
}

func LeerArchivoCA(filepathCA string, year, month int, tienda string, caja int) ([]VentaCARegistro, error) {
	if month > 0 {
		inicio := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
		fin := inicio.AddDate(0, 1, 0)
		return leerArchivoCARango(filepathCA, year, inicio, fin, tienda, caja)
	}
	inicio := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
	fin := inicio.AddDate(1, 0, 0)
	return leerArchivoCARango(filepathCA, year, inicio, fin, tienda, caja)
}

func leerArchivoCARango(filepathCA string, year int, inicio, fin time.Time, tienda string, caja int) ([]VentaCARegistro, error) {
	dbf, err := OpenDBF(filepathCA)
	if err != nil {
		return nil, err
	}
	defer dbf.Close()

	if dbf.NumRecords == 0 {
		return nil, nil
	}

	diai := dbf.FieldIndex("DIA")
	mesi := dbf.FieldIndex("MES")
	anioi := dbf.FieldIndex("ANIO")
	if diai < 0 || mesi < 0 || anioi < 0 {
		return nil, fmt.Errorf("campos DIA/MES/ANIO no encontrados (DIA=%d MES=%d ANIO=%d)", diai, mesi, anioi)
	}

	var registros []VentaCARegistro

	for i := 0; i < dbf.NumRecords; i++ {
		rec, err := dbf.ReadRecord(i)
		if err != nil || len(rec) == 0 {
			continue
		}
		if rec[0] == 0x2A {
			continue
		}

		ti := dbf.FieldIndex("TIPO")
		if ti < 0 {
			continue
		}
		tipo := dbf.GetString(rec, dbf.Fields[ti])

		ai := dbf.FieldIndex("ANULADA")
		if ai >= 0 && dbf.GetString(rec, dbf.Fields[ai]) == "T" {
			continue
		}

		diaStr := dbf.GetString(rec, dbf.Fields[diai])
		mesStr := dbf.GetString(rec, dbf.Fields[mesi])
		anioStr := dbf.GetString(rec, dbf.Fields[anioi])

		var dd, mm, yy int
		fmt.Sscanf(diaStr, "%d", &dd)
		fmt.Sscanf(mesStr, "%d", &mm)
		fmt.Sscanf(anioStr, "%d", &yy)
		if yy != year {
			continue
		}

		fecha := time.Date(yy, time.Month(mm), dd, 0, 0, 0, 0, time.UTC)
		if fecha.Before(inicio) || !fecha.Before(fin) {
			continue
		}

		hi := dbf.FieldIndex("HORA24")
		hora := 0
		if hi >= 0 {
			hora, _ = dbf.GetTime(rec, dbf.Fields[hi])
		}

		getStr := func(name string, def string) string {
			if idx := dbf.FieldIndex(name); idx >= 0 {
				return strings.TrimSpace(dbf.GetString(rec, dbf.Fields[idx]))
			}
			return def
		}
		getVc := func(name string, def string) string {
			if idx := dbf.FieldIndex(name); idx >= 0 {
				return strings.TrimSpace(dbf.GetVarChar(rec, dbf.Fields[idx]))
			}
			return def
		}
		getFl := func(name string) float64 {
			if idx := dbf.FieldIndex(name); idx >= 0 {
				return dbf.GetFloat(rec, dbf.Fields[idx])
			}
			return 0
		}
		getBo := func(name string) bool {
			if idx := dbf.FieldIndex(name); idx >= 0 {
				return dbf.GetBool(rec, dbf.Fields[idx])
			}
			return false
		}

		r := VentaCARegistro{
			Fecha:    fecha,
			Hora:     hora,
			Tienda:   tienda,
			Caja:     caja,
			Tipo:     tipo,
			STipo:    getStr("STIPO", ""),
			Numero:   getStr("NUMERO", ""),
			Codigo:   getStr("CODIGO", ""),
			CodBar:   getStr("CODBAR", ""),
			Descrip:  getVc("DESCRIP", ""),
			CodVen:   getStr("CODVEN", ""),
			Modelo:   getVc("MODELO", ""),
			Serial:   getVc("SERIAL", ""),
			Cantidad: getFl("CANTIDAD"),
			NCntd:    getFl("NCNTD"),
			NPvpDol:  getFl("NPVP_DOL"),
			NPvp2Dol: getFl("NPVP2_DOL"),
			NPvp3Dol: getFl("NPVP3_DOL"),
			NPvpCop:  getFl("NPVP_COP"),
			Precio:   getFl("PRECIO"),
			NPrecio:  getFl("NPRECIO"),
			IGV:      getFl("IGV"),
			NoDscto:  getBo("NODSCTO"),
			CodCli:   getStr("CODCLI", ""),
			Anulada:  getStr("ANULADA", ""),
			Depto:    getStr("DEPTO", ""),
			Familia:  getStr("FAMILIA", ""),
			Costo:    getFl("COSTO"),
			NCosDol:  getFl("NCOS_DOL"),
			Pvpt:     getStr("PVPT", ""),
			Oferta:   getBo("OFERTA"),
			Devlto:   getFl("DEVLTO"),
			Margen:   getFl("MARGEN"),
			PvpVen:   getFl("PVPVEN"),
			LPesado:  getBo("LPESADO"),
			NroCie:   getStr("NRO_CIE", ""),
		}
		if dci := dbf.FieldIndex("DATE_CIE"); dci >= 0 {
			if dt, ok := dbf.GetDate(rec, dbf.Fields[dci]); ok {
				r.FechaCie = &dt
			}
		}
		registros = append(registros, r)
	}
	return registros, nil
}

func rangoUltimosDias(year, month, n int) (time.Time, time.Time) {
	primerDia := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	finMes := primerDia.AddDate(0, 1, 0)
	hoy := time.Now()
	if hoy.Year() != year || int(hoy.Month()) != month {
		return primerDia, finMes
	}
	diaInicio := hoy.Day() - n + 1
	if diaInicio < 1 {
		diaInicio = 1
	}
	inicio := time.Date(year, time.Month(month), diaInicio, 0, 0, 0, 0, time.UTC)
	fin := time.Date(year, time.Month(month), hoy.Day()+1, 0, 0, 0, 0, time.UTC)
	if fin.After(finMes) {
		fin = finMes
	}
	return inicio, fin
}

func ContarRegistrosDBF_Mes(sucursal Sucursal, year, month int, modo string) int {
	codigo := sucursal.Codigo

	posDirs, err := ListarPOS(codigo, sucursal.RutaIP, sucursal.RutaDBF, modo)
	if err != nil {
		if modo == "S" && sucursal.RutaIP != "" {
			posDirs, err = ListarPOS(codigo, sucursal.RutaIP, sucursal.RutaDBF, "IP")
		} else if modo == "IP" {
			posDirs, err = ListarPOS(codigo, sucursal.RutaIP, sucursal.RutaDBF, "S")
		}
	}
	if err != nil {
		return 0
	}

	total := 0
	for _, dirName := range posDirs {
		posNum, err := strconv.Atoi(dirName[3:])
		if err != nil {
			continue
		}

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

		dbfPath := filepath.Join(posPath, fmt.Sprintf("CA%02d%d.DBF", posNum, year))
		if _, err := os.Stat(dbfPath); err != nil {
			continue
		}

		regs, err := LeerArchivoCA(dbfPath, year, month, codigo, posNum)
		if err != nil {
			continue
		}
		total += len(regs)
	}
	return total
}

func contarRegistrosCARango(dbfPath string, year int, inicio, fin time.Time) (int, error) {
	dbf, err := OpenDBF(dbfPath)
	if err != nil {
		return 0, err
	}
	defer dbf.Close()

	ai := dbf.FieldIndex("ANULADA")
	diai := dbf.FieldIndex("DIA")
	mesi := dbf.FieldIndex("MES")
	anioi := dbf.FieldIndex("ANIO")
	if diai < 0 || mesi < 0 || anioi < 0 {
		return 0, fmt.Errorf("campos DIA/MES/ANIO no encontrados")
	}
	if dbf.NumRecords == 0 {
		return 0, nil
	}

	count := 0
	for i := 0; i < dbf.NumRecords; i++ {
		rec, err := dbf.ReadRecord(i)
		if err != nil || len(rec) == 0 {
			continue
		}
		if rec[0] == 0x2A {
			continue
		}
		if ai >= 0 && dbf.GetString(rec, dbf.Fields[ai]) == "T" {
			continue
		}

		diaStr := dbf.GetString(rec, dbf.Fields[diai])
		mesStr := dbf.GetString(rec, dbf.Fields[mesi])
		anioStr := dbf.GetString(rec, dbf.Fields[anioi])
		var dd, mm, yy int
		fmt.Sscanf(diaStr, "%d", &dd)
		fmt.Sscanf(mesStr, "%d", &mm)
		fmt.Sscanf(anioStr, "%d", &yy)
		if yy != year {
			continue
		}
		fecha := time.Date(yy, time.Month(mm), dd, 0, 0, 0, 0, time.UTC)
		if fecha.Before(inicio) || !fecha.Before(fin) {
			continue
		}
		count++
	}
	return count, nil
}

func ContarRegistrosDBF_Rango(sucursal Sucursal, year int, inicio, fin time.Time, modo string) int {
	codigo := sucursal.Codigo

	posDirs, err := ListarPOS(codigo, sucursal.RutaIP, sucursal.RutaDBF, modo)
	if err != nil {
		if modo == "S" && sucursal.RutaIP != "" {
			posDirs, err = ListarPOS(codigo, sucursal.RutaIP, sucursal.RutaDBF, "IP")
		} else if modo == "IP" {
			posDirs, err = ListarPOS(codigo, sucursal.RutaIP, sucursal.RutaDBF, "S")
		}
	}
	if err != nil {
		return 0
	}

	total := 0
	for _, dirName := range posDirs {
		posNum, err := strconv.Atoi(dirName[3:])
		if err != nil {
			continue
		}

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

		dbfPath := filepath.Join(posPath, fmt.Sprintf("CA%02d%d.DBF", posNum, year))
		if _, err := os.Stat(dbfPath); err != nil {
			continue
		}

		n, err := contarRegistrosCARango(dbfPath, year, inicio, fin)
		if err != nil {
			continue
		}
		total += n
	}
	return total
}

func tieneDatosFA(sucursal Sucursal, year int, modo string) bool {
	posDirs, err := ListarPOS(sucursal.Codigo, sucursal.RutaIP, sucursal.RutaDBF, modo)
	if err != nil {
		if modo == "S" && sucursal.RutaIP != "" {
			posDirs, err = ListarPOS(sucursal.Codigo, sucursal.RutaIP, sucursal.RutaDBF, "IP")
		} else if modo == "IP" {
			posDirs, err = ListarPOS(sucursal.Codigo, sucursal.RutaIP, sucursal.RutaDBF, "S")
		}
	}
	if err != nil {
		return false
	}

	for _, dirName := range posDirs {
		posNum, err := strconv.Atoi(dirName[3:])
		if err != nil {
			continue
		}
		var posPath string
		if modo == "S" {
			if sucursal.RutaDBF != "" {
				posPath = strings.TrimRight(sucursal.RutaDBF, "\\") + "\\CIERRE_POS\\" + dirName
			} else {
				posPath = fmt.Sprintf("S:\\aBC-Soft\\Data\\%s\\CIERRE_POS\\%s", sucursal.Codigo, dirName)
			}
		} else {
			posPath = fmt.Sprintf("\\\\%s\\Sistema\\aBC-Soft\\Cierre_POS\\%s", sucursal.RutaIP, dirName)
		}
		dbfPath := filepath.Join(posPath, fmt.Sprintf("FC%02d%d.DBF", posNum, year))
		if _, err := os.Stat(dbfPath); err == nil {
			return true
		}
	}
	return false
}

func ContarRegistrosDBF_FA(sucursal Sucursal, year, month int, modo string) int {
	codigo := sucursal.Codigo

	posDirs, err := ListarPOS(codigo, sucursal.RutaIP, sucursal.RutaDBF, modo)
	if err != nil {
		if modo == "S" && sucursal.RutaIP != "" {
			posDirs, err = ListarPOS(codigo, sucursal.RutaIP, sucursal.RutaDBF, "IP")
		} else if modo == "IP" {
			posDirs, err = ListarPOS(codigo, sucursal.RutaIP, sucursal.RutaDBF, "S")
		}
	}
	if err != nil {
		return 0
	}

	total := 0
	for _, dirName := range posDirs {
		posNum, err := strconv.Atoi(dirName[3:])
		if err != nil {
			continue
		}

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

		_, _, _, totalFacturas, _, err := LeerArchivoFC(dbfPath, year, month, codigo, posNum)
		if err != nil {
			continue
		}
		total += totalFacturas
	}
	return total
}

func ProcesarTiendaCA(sucursal Sucursal, year, month int, inicio, fin time.Time, modo string, idx, total int, forceReload ...bool) error {
	codigo := sucursal.Codigo
	force := len(forceReload) > 0 && forceReload[0]

	dbCA, err := ConectarSQL_CA(codigo)
	if err != nil {
		return fmt.Errorf("FALLO BD CA: %v", err)
	}
	defer dbCA.Close()

	CrearTablaPosVentasCA(dbCA, codigo)

	pfx := func(format string, args ...interface{}) string {
		if idx > 0 {
			return fmt.Sprintf("[%d/%d] "+format, append([]interface{}{idx, total}, args...)...)
		}
		return fmt.Sprintf(format, args...)
	}

	posDirs, err := ListarPOS(codigo, sucursal.RutaIP, sucursal.RutaDBF, modo)
	if err != nil {
		if modo == "S" && sucursal.RutaIP != "" {
			fmt.Printf("\r  %s", pfx("%s  S fallo, intentando IP...", codigo))
			posDirs, err = ListarPOS(codigo, sucursal.RutaIP, sucursal.RutaDBF, "IP")
		} else if modo == "IP" {
			fmt.Printf("\r  %s", pfx("%s  IP fallo, intentando S...", codigo))
			posDirs, err = ListarPOS(codigo, sucursal.RutaIP, sucursal.RutaDBF, "S")
		}
	}
	if err != nil {
		return fmt.Errorf("FALLO ruta: %v", err)
	}

	totalInsert := 0
	var fallidos []string
	for _, dirName := range posDirs {
		posNum, err := strconv.Atoi(dirName[3:])
		if err != nil {
			continue
		}

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

		dbfPath := filepath.Join(posPath, fmt.Sprintf("CA%02d%d.DBF", posNum, year))

		info, errStat := os.Stat(dbfPath)
		if errStat != nil {
			continue
		}

		posOK := false
		for intentoPOS := 1; intentoPOS <= 3; intentoPOS++ {
			if intentoPOS > 1 {
				time.Sleep(time.Duration(intentoPOS) * time.Second)
			}

			tStart := time.Now()
			sizeMB := float64(info.Size()) / (1024 * 1024)
			fmt.Printf("\r  %s", pfx("%s  [%s] %.1f MB leyendo... (intento %d/3)", codigo, filepath.Base(dbfPath), sizeMB, intentoPOS))

			regs, err := leerArchivoCARango(dbfPath, year, inicio, fin, codigo, posNum)
			tElapsed := time.Since(tStart)

			if err != nil {
				if intentoPOS == 3 {
					fmt.Printf("\n  %s", pfx("%s  [%s] ERROR: %v", codigo, filepath.Base(dbfPath), err))
				}
				continue
			}

			if len(regs) == 0 {
				if intentoPOS == 3 {
					if info.Size() == 0 {
						fmt.Printf("\n  %s", pfx("%s  [%s] archivo vacio, saltando", codigo, filepath.Base(dbfPath)))
					} else {
						fmt.Printf("\n  %s", pfx("%s  [%s] 0 regs en rango, saltando", codigo, filepath.Base(dbfPath)))
					}
				}
				posOK = true
				break
			}

			fmt.Printf("\r  %s", pfx("%s  [%s] %d regs insertando... (intento %d/3)", codigo, filepath.Base(dbfPath), len(regs), intentoPOS))
			if err := InsertarVentasCA_Incremental(dbCA, regs, codigo, year, month, force); err != nil {
				fmt.Printf("\n  %s", pfx("%s  [%s] ERROR insert intento %d: %v", codigo, filepath.Base(dbfPath), intentoPOS, err))
				continue
			}
			totalInsert += len(regs)
			fmt.Printf("\n  %s", pfx("%s  [%s] %d regs, %.1fs", codigo, filepath.Base(dbfPath), len(regs), tElapsed.Seconds()))
			posOK = true
			break
		}

		if !posOK {
			fallidos = append(fallidos, filepath.Base(dbfPath))
		}
	}

	if totalInsert == 0 {
		return fmt.Errorf("sin datos")
	}
	if len(fallidos) > 0 {
		return fmt.Errorf("POS fallidos: %s", strings.Join(fallidos, ", "))
	}
	return nil
}

func ProcesarCA(db *sql.DB, sucursales []Sucursal, year, mesInicio, mesFin int, modo string, tipoEjecucion string) error {
	defer func() {
		if r := recover(); r != nil {
			logError("PANIC en ProcesarCA: %v", r)
		}
	}()
	if mesFin <= 0 {
		mesFin = int(time.Now().Month())
	}

	CrearTablaH_Tiendas(db)

	logInfo("CA: Iniciando %d tiendas, meses %d a %d, modo=%s", len(sucursales), mesInicio, mesFin, modo)
	tStart := time.Now()

	var wg sync.WaitGroup
	erroresGlobal := 0
	var mu sync.Mutex
	sem := make(chan struct{}, 2)

	for _, s := range sucursales {
		sem <- struct{}{}
		wg.Add(1)
		go func(store Sucursal) {
			defer wg.Done()
			defer func() { <-sem }()
			defer func() {
				if r := recover(); r != nil {
					logError("CA: PANIC en %s: %v", store.Codigo, r)
				}
			}()

			for m := mesInicio; m <= mesFin; m++ {
				mStart := time.Now()

				primerDia := time.Date(year, time.Month(m), 1, 0, 0, 0, 0, time.UTC)
				finMes := primerDia.AddDate(0, 1, 0)
				inicio, fin := primerDia, finMes

				// PASO 1: Leer todos los DBFs del mes en memoria
				fmt.Printf("\r  %s  %s Leyendo DBFs...", store.Codigo, MesesES[m])
				todosRegs, errDBF := LeerDBFRangoCompleto(store, year, inicio, fin, modo)
				if errDBF != nil {
					mu.Lock()
					logError("CA: %s %s error leyendo DBFs: %v", store.Codigo, MesesES[m], errDBF)
					InsertarH_Tiendas(db, store.Codigo, year, m, 0, 0, 0, "ERROR", int(time.Since(mStart).Seconds()), modo, tipoEjecucion, "CA")
					erroresGlobal++
					mu.Unlock()
					continue
				}

				totalDBF := len(todosRegs)
				if totalDBF == 0 {
					mu.Lock()
					logInfo("CA: %s %s sin datos DBF", store.Codigo, MesesES[m])
					InsertarH_Tiendas(db, store.Codigo, year, m, 0, 0, 0, "SIN_DATOS", int(time.Since(mStart).Seconds()), modo, tipoEjecucion, "CA")
					mu.Unlock()
					continue
				}

				fmt.Printf("\r  %s  %s DBF: %d registros            \n", store.Codigo, MesesES[m], totalDBF)

				// PASO 2: Conectar, crear tabla y staging
				dbCA, err := ConectarSQL_CA(store.Codigo)
				if err != nil {
					mu.Lock()
					logError("CA: %s %s FALLO conexion: %v", store.Codigo, MesesES[m], err)
					InsertarH_Tiendas(db, store.Codigo, year, m, totalDBF, 0, 0, "ERROR", int(time.Since(mStart).Seconds()), modo, tipoEjecucion, "CA")
					erroresGlobal++
					mu.Unlock()
					continue
				}
				CrearTablaPosVentasCA(dbCA, store.Codigo)
				CrearTablaStagingCA(dbCA, store.Codigo)

				// PASO 3: Cargar datos del DBF en staging
				fmt.Printf("  %s  %s Cargando %d registros en staging...\n", store.Codigo, MesesES[m], totalDBF)
				if err := CargarDBFEnStaging(dbCA, todosRegs, store.Codigo); err != nil {
					LimpiarStaging(dbCA, store.Codigo)
					dbCA.Close()
					mu.Lock()
					logError("CA: %s %s error staging: %v", store.Codigo, MesesES[m], err)
					InsertarH_Tiendas(db, store.Codigo, year, m, totalDBF, 0, 0, "ERROR", int(time.Since(mStart).Seconds()), modo, tipoEjecucion, "CA")
					erroresGlobal++
					mu.Unlock()
					continue
				}

				// PASO 4: Comparar staging vs SQL
				iguales, countSQL, _, errComp := CompararStagingConSQL(dbCA, store.Codigo, inicio, fin)
				if errComp != nil {
					LimpiarStaging(dbCA, store.Codigo)
					dbCA.Close()
					mu.Lock()
					logError("CA: %s %s error comparando: %v", store.Codigo, MesesES[m], errComp)
					InsertarH_Tiendas(db, store.Codigo, year, m, totalDBF, 0, 0, "ERROR", int(time.Since(mStart).Seconds()), modo, tipoEjecucion, "CA")
					erroresGlobal++
					mu.Unlock()
					continue
				}

				if iguales {
					LimpiarStaging(dbCA, store.Codigo)
					dbCA.Close()
					mu.Lock()
					logInfo("CA: %s %s OK (%d regs, sin cambios)", store.Codigo, MesesES[m], countSQL)
					InsertarH_Tiendas(db, store.Codigo, year, m, totalDBF, countSQL, 0, "SALTADO", int(time.Since(mStart).Seconds()), modo, tipoEjecucion, "CA")
					mu.Unlock()
					continue
				}

				// PASO 5: Borrar datos actuales en batches de 5000
				fmt.Printf("  %s  %s Borrando %d registros actuales...\n", store.Codigo, MesesES[m], countSQL)
				ctxDel, cancelDel := context.WithTimeout(context.Background(), 60*time.Minute)
				borrados, delErr := BorrarRangoEnBatches(ctxDel, dbCA, store.Codigo, inicio, fin)
				cancelDel()

				if delErr != nil {
					LimpiarStaging(dbCA, store.Codigo)
					dbCA.Close()
					mu.Lock()
					logError("CA: %s %s DELETE fallo: %v", store.Codigo, MesesES[m], delErr)
					InsertarH_Tiendas(db, store.Codigo, year, m, totalDBF, countSQL, 0, "ERROR", int(time.Since(mStart).Seconds()), modo, tipoEjecucion, "CA")
					erroresGlobal++
					mu.Unlock()
					continue
				}
				fmt.Printf("  %s  %s Borrados %d registros\n", store.Codigo, MesesES[m], borrados)

				// PASO 6: Copiar staging a tabla principal
				ctxCopy, cancelCopy := context.WithTimeout(context.Background(), 30*time.Minute)
				copiados, errCopy := CopiarStagingATable(ctxCopy, dbCA, store.Codigo)
				cancelCopy()
				LimpiarStaging(dbCA, store.Codigo)

				if errCopy != nil {
					dbCA.Close()
					mu.Lock()
					logError("CA: %s %s COPY fallo: %v", store.Codigo, MesesES[m], errCopy)
					InsertarH_Tiendas(db, store.Codigo, year, m, totalDBF, countSQL, 0, "ERROR", int(time.Since(mStart).Seconds()), modo, tipoEjecucion, "CA")
					erroresGlobal++
					mu.Unlock()
					continue
				}

				// PASO 7: Verificar resultado final
				ctxVer, cancelVer := context.WithTimeout(context.Background(), 10*time.Minute)
				finalCount, _ := ContarTiendaRango_SQL(ctxVer, dbCA, store.Codigo, inicio, fin)
				cancelVer()
				dbCA.Close()

				mu.Lock()
				if finalCount == totalDBF {
					logInfo("CA: %s %s OK (regs=%d, insertados=%d, %ds)", store.Codigo, MesesES[m], finalCount, int(copiados), int(time.Since(mStart).Seconds()))
					InsertarH_Tiendas(db, store.Codigo, year, m, totalDBF, finalCount, int(copiados), "OK", int(time.Since(mStart).Seconds()), modo, tipoEjecucion, "CA")
				} else {
					logWarn("CA: %s %s REVISAR (DBF=%d SQL=%d diferencia=%d)", store.Codigo, MesesES[m], totalDBF, finalCount, totalDBF-finalCount)
					InsertarH_Tiendas(db, store.Codigo, year, m, totalDBF, finalCount, int(copiados), "REVISAR", int(time.Since(mStart).Seconds()), modo, tipoEjecucion, "CA")
					erroresGlobal++
				}
				mu.Unlock()
			}
		}(s)
	}

	wg.Wait()
	fmt.Printf("\n  CA listo. Tiempo: %.1f min. (%d errores)\n", time.Since(tStart).Minutes(), erroresGlobal)
	logInfo("CA: Completado en %.1f min, %d errores", time.Since(tStart).Minutes(), erroresGlobal)
	return nil
}

func ProcesarCAFecha(db *sql.DB, sucursales []Sucursal, year int, inicio, fin time.Time, modo string) {
	CrearTablaH_Tiendas(db)
	logInfo("CA_FECHA: Iniciando %d tiendas, rango %s a %s, modo=%s", len(sucursales),
		inicio.Format("02/01/2006"), fin.Format("02/01/2006"), modo)
	tStart := time.Now()

	var wg sync.WaitGroup
	erroresGlobal := 0
	var mu sync.Mutex
	sem := make(chan struct{}, 2)

	for _, s := range sucursales {
		sem <- struct{}{}
		wg.Add(1)
		go func(store Sucursal) {
			defer wg.Done()
			defer func() { <-sem }()
			defer func() {
				if r := recover(); r != nil {
					logError("CA_FECHA: PANIC en %s: %v", store.Codigo, r)
				}
			}()

			mStart := time.Now()

			// PASO 1: Leer todos los DBFs del rango en memoria
			fmt.Printf("\r  %s  Leyendo DBFs...", store.Codigo)
			todosRegs, errDBF := LeerDBFRangoCompleto(store, year, inicio, fin, modo)
			if errDBF != nil {
				mu.Lock()
				logError("CA_FECHA: %s error leyendo DBFs: %v", store.Codigo, errDBF)
				InsertarH_Tiendas(db, store.Codigo, year, int(inicio.Month()), 0, 0, 0, "ERROR", int(time.Since(mStart).Seconds()), modo, "MANUAL", "CA")
				erroresGlobal++
				mu.Unlock()
				return
			}

			totalDBF := len(todosRegs)
			if totalDBF == 0 {
				mu.Lock()
				logInfo("CA_FECHA: %s sin datos DBF en rango", store.Codigo)
				InsertarH_Tiendas(db, store.Codigo, year, int(inicio.Month()), 0, 0, 0, "SIN_DATOS", int(time.Since(mStart).Seconds()), modo, "MANUAL", "CA")
				mu.Unlock()
				return
			}

			fmt.Printf("\r  %s  DBF: %d registros en rango     \n", store.Codigo, totalDBF)

			// PASO 2: Conectar, crear tabla y staging
			dbCA, err := ConectarSQL_CA(store.Codigo)
			if err != nil {
				mu.Lock()
				logError("CA_FECHA: %s FALLO conexion: %v", store.Codigo, err)
				InsertarH_Tiendas(db, store.Codigo, year, int(inicio.Month()), totalDBF, 0, 0, "ERROR", int(time.Since(mStart).Seconds()), modo, "MANUAL", "CA")
				erroresGlobal++
				mu.Unlock()
				return
			}
			CrearTablaPosVentasCA(dbCA, store.Codigo)
			CrearTablaStagingCA(dbCA, store.Codigo)

			// PASO 3: Cargar datos del DBF en staging
			fmt.Printf("  %s  Cargando %d registros en staging...\n", store.Codigo, totalDBF)
			if err := CargarDBFEnStaging(dbCA, todosRegs, store.Codigo); err != nil {
				dbCA.Close()
				LimpiarStaging(dbCA, store.Codigo)
				mu.Lock()
				logError("CA_FECHA: %s error cargando staging: %v", store.Codigo, err)
				InsertarH_Tiendas(db, store.Codigo, year, int(inicio.Month()), totalDBF, 0, 0, "ERROR", int(time.Since(mStart).Seconds()), modo, "MANUAL", "CA")
				erroresGlobal++
				mu.Unlock()
				return
			}

			// PASO 4: Comparar staging vs SQL
			iguales, countSQL, countSTG, errComp := CompararStagingConSQL(dbCA, store.Codigo, inicio, fin)
			if errComp != nil {
				dbCA.Close()
				LimpiarStaging(dbCA, store.Codigo)
				mu.Lock()
				logError("CA_FECHA: %s error comparando: %v", store.Codigo, errComp)
				InsertarH_Tiendas(db, store.Codigo, year, int(inicio.Month()), totalDBF, 0, 0, "ERROR", int(time.Since(mStart).Seconds()), modo, "MANUAL", "CA")
				erroresGlobal++
				mu.Unlock()
				return
			}

			if iguales {
				dbCA.Close()
				LimpiarStaging(dbCA, store.Codigo)
				mu.Lock()
				logInfo("CA_FECHA: %s OK (DBF=%d SQL=%d, sin cambios)", store.Codigo, totalDBF, countSQL)
				InsertarH_Tiendas(db, store.Codigo, year, int(inicio.Month()), totalDBF, countSQL, 0, "SALTADO", int(time.Since(mStart).Seconds()), modo, "MANUAL", "CA")
				mu.Unlock()
				return
			}

			// PASO 5: Borrar datos actuales en batches de 5000
			fmt.Printf("  %s  Borrando %d registros actuales...\n", store.Codigo, countSQL)
			ctxDel, cancelDel := context.WithTimeout(context.Background(), 60*time.Minute)
			borrados, delErr := BorrarRangoEnBatches(ctxDel, dbCA, store.Codigo, inicio, fin)
			cancelDel()

			if delErr != nil {
				dbCA.Close()
				LimpiarStaging(dbCA, store.Codigo)
				mu.Lock()
				logError("CA_FECHA: %s error borrando: %v", store.Codigo, delErr)
				InsertarH_Tiendas(db, store.Codigo, year, int(inicio.Month()), totalDBF, countSQL, 0, "ERROR", int(time.Since(mStart).Seconds()), modo, "MANUAL", "CA")
				erroresGlobal++
				mu.Unlock()
				return
			}
			fmt.Printf("  %s  Borrados %d registros\n", store.Codigo, borrados)

			// PASO 6: Copiar staging a tabla principal
			fmt.Printf("  %s  Copiando %d registros desde staging...\n", store.Codigo, countSTG)
			ctxCopy, cancelCopy := context.WithTimeout(context.Background(), 30*time.Minute)
			copiados, errCopy := CopiarStagingATable(ctxCopy, dbCA, store.Codigo)
			cancelCopy()

			// PASO 7: Limpiar staging siempre
			LimpiarStaging(dbCA, store.Codigo)

			if errCopy != nil {
				dbCA.Close()
				mu.Lock()
				logError("CA_FECHA: %s error copiando staging: %v", store.Codigo, errCopy)
				InsertarH_Tiendas(db, store.Codigo, year, int(inicio.Month()), totalDBF, countSQL, 0, "ERROR", int(time.Since(mStart).Seconds()), modo, "MANUAL", "CA")
				erroresGlobal++
				mu.Unlock()
				return
			}

			// PASO 8: Verificar resultado final
			ctxVer, cancelVer := context.WithTimeout(context.Background(), 10*time.Minute)
			finalSQL, _ := ContarTiendaRango_SQL(ctxVer, dbCA, store.Codigo, inicio, fin)
			cancelVer()
			dbCA.Close()

			estado := "OK"
			if totalDBF != finalSQL {
				estado = "REVISAR"
				logWarn("CA_FECHA: %s REVISAR (DBF=%d SQL=%d diferencia=%d)", store.Codigo, totalDBF, finalSQL, totalDBF-finalSQL)
			} else {
				logInfo("CA_FECHA: %s OK (regs=%d, duracion=%ds)", store.Codigo, finalSQL, int(time.Since(mStart).Seconds()))
			}

			mu.Lock()
			InsertarH_Tiendas(db, store.Codigo, year, int(inicio.Month()), totalDBF, finalSQL, int(copiados), estado, int(time.Since(mStart).Seconds()), modo, "MANUAL", "CA")
			mu.Unlock()
		}(s)
	}

	wg.Wait()
	fmt.Printf("\n  CA Fecha listo. Tiempo: %.1f min. (%d errores)\n", time.Since(tStart).Minutes(), erroresGlobal)
	logInfo("CA_FECHA: Completado en %.1f min, %d errores", time.Since(tStart).Minutes(), erroresGlobal)
}

func LeerDBFRangoCompleto(store Sucursal, year int, inicio, fin time.Time, modo string) ([]VentaCARegistro, error) {
	posDirs, err := ListarPOS(store.Codigo, store.RutaIP, store.RutaDBF, modo)
	if err != nil {
		if modo == "S" && store.RutaIP != "" {
			posDirs, err = ListarPOS(store.Codigo, store.RutaIP, store.RutaDBF, "IP")
		} else if modo == "IP" {
			posDirs, err = ListarPOS(store.Codigo, store.RutaIP, store.RutaDBF, "S")
		}
	}
	if err != nil {
		return nil, fmt.Errorf("FALLO ruta: %v", err)
	}

	var todosRegs []VentaCARegistro
	for _, dirName := range posDirs {
		posNum, errPos := strconv.Atoi(dirName[3:])
		if errPos != nil {
			continue
		}

		var posPath string
		if modo == "S" {
			if store.RutaDBF != "" {
				posPath = strings.TrimRight(store.RutaDBF, "\\") + "\\CIERRE_POS\\" + dirName
			} else {
				posPath = fmt.Sprintf("S:\\aBC-Soft\\Data\\%s\\CIERRE_POS\\%s", store.Codigo, dirName)
			}
		} else {
			posPath = fmt.Sprintf("\\\\%s\\Sistema\\aBC-Soft\\Cierre_POS\\%s", store.RutaIP, dirName)
		}

		dbfPath := filepath.Join(posPath, fmt.Sprintf("CA%02d%d.DBF", posNum, year))

		if _, errStat := os.Stat(dbfPath); errStat != nil {
			continue
		}

		for intento := 1; intento <= 3; intento++ {
			if intento > 1 {
				time.Sleep(time.Duration(intento) * time.Second)
			}
			regs, errLectura := leerArchivoCARango(dbfPath, year, inicio, fin, store.Codigo, posNum)
			if errLectura != nil {
				continue
			}
			if len(regs) > 0 {
				todosRegs = append(todosRegs, regs...)
			}
			break
		}
	}
	return todosRegs, nil
}

func CorregirDeptoFamilia(dbSistemas *sql.DB, tienda string) (int, int, error) {
	tableName := "POS_CA_" + tienda

	dbCA, err := ConectarSQL_CA(tienda)
	if err != nil {
		return 0, 0, fmt.Errorf("FALLO conexion: %v", err)
	}
	defer dbCA.Close()

	var totalSinDepto int
	err = dbCA.QueryRow(fmt.Sprintf(`
		SELECT COUNT(*) FROM %s
		WHERE (Depto IS NULL OR Depto = '') AND (Familia IS NULL OR Familia = '')
	`, tableName)).Scan(&totalSinDepto)
	if err != nil {
		return 0, 0, fmt.Errorf("error contando sin Depto: %v", err)
	}
	fmt.Printf("    Registros sin Depto/Familia: %d\n", totalSinDepto)

	if totalSinDepto == 0 {
		return 0, 0, nil
	}

	fmt.Println("    Paso 1: Buscando en misma tienda...")
	var actualizados1 int
	_, err = dbCA.Exec(fmt.Sprintf(`
		UPDATE A SET
			A.Depto = (
				SELECT TOP 1 B.Depto FROM %s B
				WHERE B.Codigo = A.Codigo
					AND B.Depto IS NOT NULL AND B.Depto <> ''
				ORDER BY B.Fecha DESC
			),
			A.Familia = (
				SELECT TOP 1 B.Familia FROM %s B
				WHERE B.Codigo = A.Codigo
					AND B.Familia IS NOT NULL AND B.Familia <> ''
				ORDER BY B.Fecha DESC
			)
		FROM %s A
		WHERE (A.Depto IS NULL OR A.Depto = '')
			AND EXISTS (
				SELECT 1 FROM %s B
				WHERE B.Codigo = A.Codigo
					AND (B.Depto IS NOT NULL AND B.Depto <> '')
			)
	`, tableName, tableName, tableName, tableName))
	if err != nil {
		return 0, 0, fmt.Errorf("error paso 1: %v", err)
	}
	dbCA.QueryRow(fmt.Sprintf(`
		SELECT COUNT(*) FROM %s
		WHERE (Depto IS NULL OR Depto = '') AND (Familia IS NULL OR Familia = '')
	`, tableName)).Scan(&actualizados1)
	actualizados1 = totalSinDepto - actualizados1
	fmt.Printf("    Paso 1: %d actualizados\n", actualizados1)

	var pendientes int
	dbCA.QueryRow(fmt.Sprintf(`
		SELECT COUNT(DISTINCT Codigo) FROM %s
		WHERE (Depto IS NULL OR Depto = '') AND (Familia IS NULL OR Familia = '')
	`, tableName)).Scan(&pendientes)

	if pendientes == 0 {
		fmt.Println("    Todos los productos tienen Depto/Familia.")
		return actualizados1, 0, nil
	}

	fmt.Printf("    Paso 2: Buscando en otras tiendas (%d productos pendientes)...\n", pendientes)

	rows, err := dbCA.Query(fmt.Sprintf(`
		SELECT DISTINCT Codigo FROM %s
		WHERE (Depto IS NULL OR Depto = '') AND (Familia IS NULL OR Familia = '')
	`, tableName))
	if err != nil {
		return actualizados1, pendientes, nil
	}
	defer rows.Close()

	type mapEntry struct {
		Depto   string
		Familia string
	}
	mappings := make(map[string]mapEntry)

	tablesQuery := `SELECT TABLE_NAME FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_NAME LIKE 'POS_CA_%' AND TABLE_NAME <> @p1 ORDER BY TABLE_NAME`
	allTables, err := dbCA.Query(tablesQuery, tableName)
	if err == nil {
		defer allTables.Close()
		var otherTables []string
		for allTables.Next() {
			var t string
			if allTables.Scan(&t) == nil {
				otherTables = append(otherTables, t)
			}
		}

		for rows.Next() {
			var codigo string
			if rows.Scan(&codigo) != nil {
				continue
			}
			for _, otherTable := range otherTables {
				var d, f string
				errQ := dbCA.QueryRow(fmt.Sprintf(`
					SELECT TOP 1 Depto, Familia FROM %s
					WHERE Codigo = @p1
						AND Depto IS NOT NULL AND Depto <> ''
						AND Familia IS NOT NULL AND Familia <> ''
					ORDER BY Fecha DESC
				`, otherTable), codigo).Scan(&d, &f)
				if errQ == nil && d != "" && f != "" {
					mappings[codigo] = mapEntry{Depto: d, Familia: f}
					break
				}
			}
		}
	}

	actualizados2 := 0
	for codigo, m := range mappings {
		res, err := dbCA.Exec(fmt.Sprintf(`
			UPDATE %s SET Depto = @p1, Familia = @p2
			WHERE Codigo = @p3
				AND (Depto IS NULL OR Depto = '')
		`, tableName), m.Depto, m.Familia, codigo)
		if err == nil {
			n, _ := res.RowsAffected()
			actualizados2 += int(n)
		}
	}
	fmt.Printf("    Paso 2: %d actualizados desde otras tiendas\n", actualizados2)

	var pendientesFinal int
	dbCA.QueryRow(fmt.Sprintf(`
		SELECT COUNT(DISTINCT Codigo) FROM %s
		WHERE (Depto IS NULL OR Depto = '') AND (Familia IS NULL OR Familia = '')
	`, tableName)).Scan(&pendientesFinal)

	if pendientesFinal > 0 {
		fmt.Printf("    Productos sin mapeo (%d):\n", pendientesFinal)
		sinMapeo, err := dbCA.Query(fmt.Sprintf(`
			SELECT TOP 20 Codigo, Descrip FROM %s
			WHERE (Depto IS NULL OR Depto = '') AND (Familia IS NULL OR Familia = '')
			GROUP BY Codigo, Descrip
		`, tableName))
		if err == nil {
			defer sinMapeo.Close()
			for sinMapeo.Next() {
				var cod, desc string
				if sinMapeo.Scan(&cod, &desc) == nil {
					fmt.Printf("      %s - %s\n", cod, desc)
				}
			}
		}
	}

	return actualizados1 + actualizados2, pendientesFinal, nil
}

func ContarDBFPorDia(sucursal Sucursal, year, month, day int, modo string) int {
	inicio := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	fin := inicio.AddDate(0, 0, 1)
	return ContarRegistrosDBF_Rango(sucursal, year, inicio, fin, modo)
}

func CargarDBFDia(dbCA *sql.DB, sucursal Sucursal, year, month, day int, modo string, forceReload bool) (int, error) {
	codigo := sucursal.Codigo
	inicio := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	fin := inicio.AddDate(0, 0, 1)

	posDirs, err := ListarPOS(codigo, sucursal.RutaIP, sucursal.RutaDBF, modo)
	if err != nil {
		if modo == "S" && sucursal.RutaIP != "" {
			posDirs, err = ListarPOS(codigo, sucursal.RutaIP, sucursal.RutaDBF, "IP")
		} else if modo == "IP" {
			posDirs, err = ListarPOS(codigo, sucursal.RutaIP, sucursal.RutaDBF, "S")
		}
	}
	if err != nil {
		return 0, fmt.Errorf("FALLO ruta: %v", err)
	}

	totalInsert := 0
	for _, dirName := range posDirs {
		posNum, err := strconv.Atoi(dirName[3:])
		if err != nil {
			continue
		}

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

		dbfPath := filepath.Join(posPath, fmt.Sprintf("CA%02d%d.DBF", posNum, year))
		if _, err := os.Stat(dbfPath); err != nil {
			continue
		}

		regs, err := leerArchivoCARango(dbfPath, year, inicio, fin, codigo, posNum)
		if err != nil || len(regs) == 0 {
			continue
		}

		if err := InsertarVentasCA_Incremental(dbCA, regs, codigo, year, month, forceReload); err != nil {
			continue
		}
		totalInsert += len(regs)
	}

	return totalInsert, nil
}

func ProcesarAutoCA(db *sql.DB, sucursales []Sucursal, year, month int, modo string) {
	CrearTablaH_Tiendas(db)

	hoy := time.Now()
	dias := []int{hoy.Day() - 2, hoy.Day() - 1, hoy.Day()}

	var diasValidos []int
	for _, d := range dias {
		if d >= 1 && d <= 31 {
			fechaTest := time.Date(year, time.Month(month), d, 0, 0, 0, 0, time.UTC)
			if fechaTest.Month() == time.Month(month) {
				diasValidos = append(diasValidos, d)
			}
		}
	}

	fmt.Printf("CA_AUTO: %d tiendas, %d dias a procesar, modo=%s\n", len(sucursales), len(diasValidos), modo)
	logInfo("CA_AUTO: Iniciando %d tiendas, dias %v, modo=%s", len(sucursales), diasValidos, modo)
	tStart := time.Now()

	var wg sync.WaitGroup
	erroresGlobal := 0
	var mu sync.Mutex
	sem := make(chan struct{}, 5)

	for _, s := range sucursales {
		sem <- struct{}{}
		wg.Add(1)
		go func(store Sucursal) {
			defer wg.Done()
			defer func() { <-sem }()
			defer func() {
				if r := recover(); r != nil {
					logError("CA_AUTO: PANIC en %s: %v", store.Codigo, r)
				}
			}()

			diasOK := 0
			diasTotal := 0

			for _, dia := range diasValidos {
				diasTotal++
				diaStart := time.Now()

				diaInicio := time.Date(year, time.Month(month), dia, 0, 0, 0, 0, time.UTC)
				diaFin := diaInicio.AddDate(0, 0, 1)

				// PASO 1: Leer DBF del día en memoria
				fmt.Printf("  %s  %02d/%02d Leyendo DBFs...\n", store.Codigo, dia, month)
				todosRegs, errDBF := LeerDBFRangoCompleto(store, year, diaInicio, diaFin, modo)
				if errDBF != nil {
					mu.Lock()
					logError("CA_AUTO: %s %02d/%02d error leyendo DBF: %v", store.Codigo, dia, month, errDBF)
					InsertarH_Tiendas(db, store.Codigo, year, month, 0, 0, 0, "ERROR", int(time.Since(diaStart).Seconds()), modo, "AUTOMATICO", "CA")
					erroresGlobal++
					mu.Unlock()
					continue
				}

				dbfCount := len(todosRegs)
				if dbfCount == 0 {
					mu.Lock()
					logInfo("CA_AUTO: %s %02d/%02d SIN_DATOS", store.Codigo, dia, month)
					InsertarH_Tiendas(db, store.Codigo, year, month, 0, 0, 0, "SIN_DATOS", int(time.Since(diaStart).Seconds()), modo, "AUTOMATICO", "CA")
					mu.Unlock()
					continue
				}

				fmt.Printf("  %s  %02d/%02d DBF: %d registros\n", store.Codigo, dia, month, dbfCount)

				// PASO 2: Conectar, crear tabla y staging
				dbCA, err := ConectarSQL_CA(store.Codigo)
				if err != nil {
					mu.Lock()
					logError("CA_AUTO: %s %02d/%02d FALLO conexion: %v", store.Codigo, dia, month, err)
					InsertarH_Tiendas(db, store.Codigo, year, month, dbfCount, 0, 0, "ERROR", int(time.Since(diaStart).Seconds()), modo, "AUTOMATICO", "CA")
					erroresGlobal++
					mu.Unlock()
					continue
				}
				CrearTablaPosVentasCA(dbCA, store.Codigo)
				CrearTablaStagingCA(dbCA, store.Codigo)

				// PASO 3: Cargar datos del DBF en staging
				fmt.Printf("  %s  %02d/%02d Cargando %d registros en staging...\n", store.Codigo, dia, month, dbfCount)
				if err := CargarDBFEnStaging(dbCA, todosRegs, store.Codigo); err != nil {
					LimpiarStaging(dbCA, store.Codigo)
					dbCA.Close()
					mu.Lock()
					logError("CA_AUTO: %s %02d/%02d error staging: %v", store.Codigo, dia, month, err)
					InsertarH_Tiendas(db, store.Codigo, year, month, dbfCount, 0, 0, "ERROR", int(time.Since(diaStart).Seconds()), modo, "AUTOMATICO", "CA")
					erroresGlobal++
					mu.Unlock()
					continue
				}

				// PASO 4: Comparar staging vs SQL
				iguales, countSQL, _, errComp := CompararStagingConSQL(dbCA, store.Codigo, diaInicio, diaFin)
				if errComp != nil {
					LimpiarStaging(dbCA, store.Codigo)
					dbCA.Close()
					mu.Lock()
					logError("CA_AUTO: %s %02d/%02d error comparando: %v", store.Codigo, dia, month, errComp)
					InsertarH_Tiendas(db, store.Codigo, year, month, dbfCount, 0, 0, "ERROR", int(time.Since(diaStart).Seconds()), modo, "AUTOMATICO", "CA")
					erroresGlobal++
					mu.Unlock()
					continue
				}

				if iguales {
					LimpiarStaging(dbCA, store.Codigo)
					dbCA.Close()
					mu.Lock()
					logInfo("CA_AUTO: %s %02d/%02d SALTADO (%d regs)", store.Codigo, dia, month, countSQL)
					InsertarH_Tiendas(db, store.Codigo, year, month, dbfCount, countSQL, 0, "SALTADO", int(time.Since(diaStart).Seconds()), modo, "AUTOMATICO", "CA")
					diasOK++
					mu.Unlock()
					continue
				}

				// PASO 5: Borrar datos del día en batches de 1000
				ctxDel, cancelDel := context.WithTimeout(context.Background(), 60*time.Minute)
				borrados, delErr := BorrarRangoEnBatches(ctxDel, dbCA, store.Codigo, diaInicio, diaFin)
				cancelDel()

				if delErr != nil {
					LimpiarStaging(dbCA, store.Codigo)
					dbCA.Close()
					mu.Lock()
					logError("CA_AUTO: %s %02d/%02d DELETE fallo: %v", store.Codigo, dia, month, delErr)
					InsertarH_Tiendas(db, store.Codigo, year, month, dbfCount, countSQL, 0, "ERROR", int(time.Since(diaStart).Seconds()), modo, "AUTOMATICO", "CA")
					erroresGlobal++
					mu.Unlock()
					continue
				}
				logInfo("CA_AUTO: %s %02d/%02d Borrado %d registros", store.Codigo, dia, month, borrados)

				// PASO 6: Copiar staging a tabla principal
				ctxCopy, cancelCopy := context.WithTimeout(context.Background(), 60*time.Minute)
				copiados, errCopy := CopiarStagingATable(ctxCopy, dbCA, store.Codigo)
				cancelCopy()
				LimpiarStaging(dbCA, store.Codigo)

				if errCopy != nil {
					dbCA.Close()
					mu.Lock()
					logError("CA_AUTO: %s %02d/%02d COPY fallo: %v", store.Codigo, dia, month, errCopy)
					InsertarH_Tiendas(db, store.Codigo, year, month, dbfCount, countSQL, 0, "ERROR", int(time.Since(diaStart).Seconds()), modo, "AUTOMATICO", "CA")
					erroresGlobal++
					mu.Unlock()
					continue
				}

				// PASO 7: Verificar resultado final
				ctxVer, cancelVer := context.WithTimeout(context.Background(), 5*time.Minute)
				finalCount, _ := ContarSQLPorDia(ctxVer, dbCA, store.Codigo, year, month, dia)
				cancelVer()
				dbCA.Close()

				mu.Lock()
				if finalCount == dbfCount {
					logInfo("CA_AUTO: %s %02d/%02d OK (regs=%d, insertados=%d, %ds)", store.Codigo, dia, month, finalCount, int(copiados), int(time.Since(diaStart).Seconds()))
					InsertarH_Tiendas(db, store.Codigo, year, month, dbfCount, finalCount, int(copiados), "OK", int(time.Since(diaStart).Seconds()), modo, "AUTOMATICO", "CA")
					diasOK++
				} else {
					logWarn("CA_AUTO: %s %02d/%02d REVISAR (DBF=%d SQL=%d dif=%d)", store.Codigo, dia, month, dbfCount, finalCount, dbfCount-finalCount)
					InsertarH_Tiendas(db, store.Codigo, year, month, dbfCount, finalCount, int(copiados), "REVISAR", int(time.Since(diaStart).Seconds()), modo, "AUTOMATICO", "CA")
					erroresGlobal++
				}
				mu.Unlock()
			}

			mu.Lock()
			if diasOK == diasTotal {
				fmt.Printf("  %s OK (%d/%d dias)\n", store.Codigo, diasOK, diasTotal)
			} else {
				fmt.Printf("  %s PARCIAL (%d/%d dias)\n", store.Codigo, diasOK, diasTotal)
			}
			mu.Unlock()
		}(s)
	}

	wg.Wait()
	fmt.Printf("\nCA_AUTO listo. Tiempo: %.1f min. (%d errores)\n", time.Since(tStart).Minutes(), erroresGlobal)
	logInfo("CA_AUTO: Completado en %.1f min, %d errores", time.Since(tStart).Minutes(), erroresGlobal)
}


