package main

import (
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
			} else if modo == "IP" {
				fmt.Printf("\n  %s: IP fallo, intentando S...", codigo)
				posDirs, err = ListarPOS(codigo, sucursal.RutaIP, sucursal.RutaDBF, "S")
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
			tt    tareaTienda
			res   ResultadoTienda
			fallo bool
		}
		resultChan := make(chan storeRes, len(pendientes))

		for _, t := range pendientes {
			wg.Add(1)
			go func(tt tareaTienda) {
				defer wg.Done()
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
				case <-time.After(30 * time.Minute):
					fmt.Printf("\n  [%d/%d] %s - TIMEOUT\n", tt.idx+1, total, tt.suc.Codigo)
					resultChan <- storeRes{tt: tt, fallo: true}
				}
			}(t)
		}

		wg.Wait()
		close(resultChan)

		for msg := range resultChan {
			if msg.fallo {
				nuevosPendientes = append(nuevosPendientes, msg.tt)
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
	dbf, err := OpenDBF(filepathCA)
	if err != nil {
		return nil, err
	}
	defer dbf.Close()

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

		diai := dbf.FieldIndex("DIA")
		mesi := dbf.FieldIndex("MES")
		anioi := dbf.FieldIndex("ANIO")
		if diai < 0 || mesi < 0 || anioi < 0 {
			continue
		}

		diaStr := dbf.GetString(rec, dbf.Fields[diai])
		mesStr := dbf.GetString(rec, dbf.Fields[mesi])
		anioStr := dbf.GetString(rec, dbf.Fields[anioi])

		var dd, mm, yy int
		fmt.Sscanf(diaStr, "%d", &dd)
		fmt.Sscanf(mesStr, "%d", &mm)
		fmt.Sscanf(anioStr, "%d", &yy)
		if month > 0 && (yy != year || mm != month) {
			continue
		}
		if yy != year {
			continue
		}

		fecha := time.Date(yy, time.Month(mm), dd, 0, 0, 0, 0, time.UTC)

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

func ProcesarTiendaCA(sucursal Sucursal, year, month int, modo string, idx, total int) error {
	codigo := sucursal.Codigo

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

		tStart := time.Now()
		sizeMB := float64(info.Size()) / (1024 * 1024)
		fmt.Printf("\r  %s", pfx("%s  [%s] %.1f MB leyendo...", codigo, filepath.Base(dbfPath), sizeMB))

		regs, err := LeerArchivoCA(dbfPath, year, month, codigo, posNum)
		tElapsed := time.Since(tStart)

		if err != nil || len(regs) == 0 {
			continue
		}

		fmt.Printf("\r  %s", pfx("%s  [%s] %d regs insertando...", codigo, filepath.Base(dbfPath), len(regs)))
		if err := InsertarVentasCA_Incremental(dbCA, regs, codigo, year, month); err != nil {
			fmt.Printf("\n  %s", pfx("%s  [%s] ERROR insert: %v", codigo, filepath.Base(dbfPath), err))
			continue
		}
		totalInsert += len(regs)

		fmt.Printf("\n  %s", pfx("%s  [%s] %d regs, %.1fs", codigo, filepath.Base(dbfPath), len(regs), tElapsed.Seconds()))
	}

	if totalInsert == 0 {
		return fmt.Errorf("sin datos")
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

	for _, s := range sucursales {
		wg.Add(1)
		go func(store Sucursal) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					logError("CA: PANIC en %s: %v", store.Codigo, r)
				}
			}()

			for m := mesInicio; m <= mesFin; m++ {
				mStart := time.Now()
				totalDBF := ContarRegistrosDBF_Mes(store, year, m, modo)
				if totalDBF == 0 {
					continue
				}

				dbCA, err := ConectarSQL_CA(store.Codigo)
				if err != nil {
					mu.Lock()
					logError("CA: %s %s FALLO conexion: %v", store.Codigo, MesesES[m], err)
					InsertarH_Tiendas(db, store.Codigo, year, m, totalDBF, 0, 0, "ERROR", int(time.Since(mStart).Seconds()), modo, tipoEjecucion)
					mu.Unlock()
					continue
				}
				totalSQL := ContarTiendaMes_SQL(dbCA, store.Codigo, year, m)
				dbCA.Close()

				if totalDBF == totalSQL {
					mu.Lock()
					logInfo("CA: %s %s OK (%d regs, saltado)", store.Codigo, MesesES[m], totalSQL)
					InsertarH_Tiendas(db, store.Codigo, year, m, totalDBF, totalSQL, 0, "SALTADO", int(time.Since(mStart).Seconds()), modo, tipoEjecucion)
					mu.Unlock()
					continue
				}

				var lastErr error
				ok := false
				for intento := 1; intento <= 3; intento++ {
					err := ProcesarTiendaCA(store, year, m, modo, 0, 0)
					if err == nil || strings.Contains(err.Error(), "sin datos") {
						ok = true
						break
					}
					lastErr = err
				}
				mu.Lock()
				if ok {
					insertados := totalDBF - totalSQL
					if insertados < 0 {
						insertados = 0
					}
					logInfo("CA: %s %s OK (+%d regs)", store.Codigo, MesesES[m], insertados)
					InsertarH_Tiendas(db, store.Codigo, year, m, totalDBF, totalSQL, insertados, "OK", int(time.Since(mStart).Seconds()), modo, tipoEjecucion)
				} else {
					logError("CA: %s %s FALLO: %v", store.Codigo, MesesES[m], lastErr)
					InsertarH_Tiendas(db, store.Codigo, year, m, totalDBF, totalSQL, 0, "ERROR", int(time.Since(mStart).Seconds()), modo, tipoEjecucion)
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


