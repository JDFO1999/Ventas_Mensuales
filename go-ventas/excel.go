package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

func GenerarExcel(resultados []ResultadoTienda, year, month int, outputDir string) error {
	nombreMes := MesesES[month]
	diasMes := time.Date(year, time.Month(month+1), 0, 0, 0, 0, 0, time.UTC).Day()

	f := excelize.NewFile()
	defer f.Close()

	ws := "Sheet1"
	f.SetSheetName(ws, nombreMes)
	ws = nombreMes

	// ---- Title ----
	titleStyle, _ := f.NewStyle(&excelize.Style{
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"4472C4"}, Pattern: 1},
		Font:      &excelize.Font{Bold: true, Size: 18, Color: "FFFFFF", Family: "Calibri"},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})
	f.SetCellValue(ws, "B2", fmt.Sprintf("%s (%d)", nombreMes, year))
	f.MergeCell(ws, "B2", "K3")
	f.SetCellStyle(ws, "B2", "K3", titleStyle)
	f.SetRowHeight(ws, 2, 12.75)
	f.SetRowHeight(ws, 3, 12.75)

	// ---- Headers ----
	headers := []string{
		"TIENDA", "Promedio por factura", "Clientes atendidos en el mes",
		"Cliente promedio por d\u00eda", "Hora con Mayor movimiento",
		"Clientes atendidos con mayor movimiento", "Hora con menor moviento",
		"Clientes atendidos con menor movimiento", "Total",
		"Proemdio venta por d\u00eda (30)",
	}
	hdStyle, _ := f.NewStyle(&excelize.Style{
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"4472C4"}, Pattern: 1},
		Font:      &excelize.Font{Bold: true, Size: 11, Color: "FFFFFF", Family: "Calibri"},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
	})
	f.SetRowHeight(ws, 4, 39.75)
	for i, h := range headers {
		col, _ := excelize.ColumnNumberToName(i + 2)
		cell := fmt.Sprintf("%s4", col)
		f.SetCellValue(ws, cell, h)
		f.SetCellStyle(ws, cell, cell, hdStyle)
	}

	// ---- Column widths ----
	widths := map[string]float64{
		"B": 9.43, "C": 10.57, "D": 17.71, "E": 16.43,
		"F": 38.14, "G": 20.38, "H": 15.0, "I": 21.43,
		"J": 13.29, "K": 15.14,
	}
	for col, w := range widths {
		f.SetColWidth(ws, col, col, w)
	}

	// ---- Data rows ----
	styleCache := make(map[string]int)

	bldStyle := func(fillColor string, numFmt int) int {
		key := fmt.Sprintf("%s_%d", fillColor, numFmt)
		if id, ok := styleCache[key]; ok {
			return id
		}
		id, _ := f.NewStyle(&excelize.Style{
			Fill: excelize.Fill{Type: "pattern", Color: []string{fillColor}, Pattern: 1},
			Border: []excelize.Border{
				{Type: "left", Color: "000000", Style: 1},
				{Type: "top", Color: "000000", Style: 1},
				{Type: "right", Color: "000000", Style: 1},
				{Type: "bottom", Color: "000000", Style: 1},
			},
			Alignment: &excelize.Alignment{Horizontal: "center"},
			Font:      &excelize.Font{Size: 11, Family: "Calibri"},
			NumFmt:    numFmt,
		})
		styleCache[key] = id
		return id
	}

	// Excel built-in format codes:
	// 0 = General, 2 = 0.00, 3 = #,##0, 4 = #,##0.00
	// https://xuri.me/excelize/en/style.html#number_format

	row := 5
	for _, res := range resultados {
		fc := SeccionColores[res.Tienda]
		if fc == "" {
			fc = "FFFFFF"
		}
		sc := strings.TrimPrefix(fc, "FF") // excelize no usa FF prefix

		f.SetCellValue(ws, fmt.Sprintf("B%d", row), res.Tienda)
		f.SetCellStyle(ws, fmt.Sprintf("B%d", row), fmt.Sprintf("B%d", row), bldStyle(sc, 0)) // General

		f.SetCellValue(ws, fmt.Sprintf("C%d", row), res.PromedioFactura)
		f.SetCellStyle(ws, fmt.Sprintf("C%d", row), fmt.Sprintf("C%d", row), bldStyle(sc, 2)) // 0.00

		f.SetCellValue(ws, fmt.Sprintf("D%d", row), res.Clientes)
		f.SetCellStyle(ws, fmt.Sprintf("D%d", row), fmt.Sprintf("D%d", row), bldStyle(sc, 3)) // #,##0

		f.SetCellFormula(ws, fmt.Sprintf("E%d", row), fmt.Sprintf("=D%d/%d", row, diasMes))
		f.SetCellStyle(ws, fmt.Sprintf("E%d", row), fmt.Sprintf("E%d", row), bldStyle(sc, 3)) // #,##0

		f.SetCellValue(ws, fmt.Sprintf("F%d", row), res.HoraMayor)
		f.SetCellStyle(ws, fmt.Sprintf("F%d", row), fmt.Sprintf("F%d", row), bldStyle(sc, 0)) // General

		f.SetCellValue(ws, fmt.Sprintf("G%d", row), res.ClientesMayor)
		f.SetCellStyle(ws, fmt.Sprintf("G%d", row), fmt.Sprintf("G%d", row), bldStyle(sc, 3)) // #,##0

		f.SetCellValue(ws, fmt.Sprintf("H%d", row), res.HoraMenor)
		f.SetCellStyle(ws, fmt.Sprintf("H%d", row), fmt.Sprintf("H%d", row), bldStyle(sc, 0)) // General

		f.SetCellValue(ws, fmt.Sprintf("I%d", row), res.ClientesMenor)
		f.SetCellStyle(ws, fmt.Sprintf("I%d", row), fmt.Sprintf("I%d", row), bldStyle(sc, 3)) // #,##0

		f.SetCellValue(ws, fmt.Sprintf("J%d", row), res.Total)
		f.SetCellStyle(ws, fmt.Sprintf("J%d", row), fmt.Sprintf("J%d", row), bldStyle(sc, 4)) // #,##0.00

		f.SetCellFormula(ws, fmt.Sprintf("K%d", row), fmt.Sprintf("=J%d/%d", row, diasMes))
		f.SetCellStyle(ws, fmt.Sprintf("K%d", row), fmt.Sprintf("K%d", row), bldStyle(sc, 4)) // #,##0.00

		f.SetRowHeight(ws, row, 15.0)
		row++
	}

	outputPath := fmt.Sprintf("%s\\Ventas_Mensuales_%s_%d.xlsx", outputDir, nombreMes, year)
	if err := f.SaveAs(outputPath); err != nil {
		return err
	}
	fmt.Printf("\nArchivo guardado: %s\n", outputPath)
	return nil
}

func IniciarExcelCA(year, mesHasta int) (*excelize.File, string, string, int) {
	nombreMes := MesesES[mesHasta]
	tituloMes := nombreMes
	if mesHasta > 1 {
		tituloMes = "ENERO_" + nombreMes
	}

	f := excelize.NewFile()

	ws := "Sheet1"
	f.SetSheetName(ws, "CA_"+tituloMes)
	ws = "CA_" + tituloMes

	titleStyle, _ := f.NewStyle(&excelize.Style{
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"C00000"}, Pattern: 1},
		Font:      &excelize.Font{Bold: true, Size: 16, Color: "FFFFFF", Family: "Calibri"},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})
	f.SetCellValue(ws, "A1", fmt.Sprintf("DETALLE CA - %s a %s %d", MesesES[1], nombreMes, year))
	if mesHasta == 1 {
		f.SetCellValue(ws, "A1", fmt.Sprintf("DETALLE CA - %s %d", nombreMes, year))
	}
	f.MergeCell(ws, "A1", "AM1")
	f.SetCellStyle(ws, "A1", "AM1", titleStyle)
	f.SetRowHeight(ws, 1, 25)

	hdStyle, _ := f.NewStyle(&excelize.Style{
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"C00000"}, Pattern: 1},
		Font:      &excelize.Font{Bold: true, Size: 10, Color: "FFFFFF", Family: "Calibri"},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
	})

	headers := []string{
		"Tienda", "Caja", "Fecha", "Hora", "Tipo", "STipo", "Numero", "Codigo", "CodBar", "Descripcion",
		"CodVen", "Modelo", "Serial", "Cantidad", "NCntd", "NPvpDol", "NPvp2Dol", "NPvp3Dol", "NPvpCop",
		"Precio", "NPrecio", "IGV", "NoDscto", "CodCli", "Anulada", "Depto", "Familia",
		"Costo", "NCosDol", "Pvpt", "Oferta", "Devlto", "Margen", "PvpVen", "LPesado", "NroCie", "FechaCie",
	}
	for i, h := range headers {
		col, _ := excelize.ColumnNumberToName(i + 1)
		cell := fmt.Sprintf("%s2", col)
		f.SetCellValue(ws, cell, h)
		f.SetCellStyle(ws, cell, cell, hdStyle)
	}
	f.SetRowHeight(ws, 2, 30)

	for c := 1; c <= 37; c++ {
		colName, _ := excelize.ColumnNumberToName(c)
		f.SetColWidth(ws, colName, colName, 14)
	}
	f.SetColWidth(ws, "J", "J", 40)

	return f, ws, tituloMes, 3
}

func AppendTiendaCA(f *excelize.File, ws string, registros []VentaCARegistro, row int) int {
	bodStyle, _ := f.NewStyle(&excelize.Style{
		Border: []excelize.Border{
			{Type: "left", Color: "D0D0D0", Style: 1},
			{Type: "top", Color: "D0D0D0", Style: 1},
			{Type: "right", Color: "D0D0D0", Style: 1},
			{Type: "bottom", Color: "D0D0D0", Style: 1},
		},
		Font:   &excelize.Font{Size: 9, Family: "Calibri"},
		NumFmt: 2,
	})

	intStyle, _ := f.NewStyle(&excelize.Style{
		Border: []excelize.Border{
			{Type: "left", Color: "D0D0D0", Style: 1},
			{Type: "top", Color: "D0D0D0", Style: 1},
			{Type: "right", Color: "D0D0D0", Style: 1},
			{Type: "bottom", Color: "D0D0D0", Style: 1},
		},
		Font:   &excelize.Font{Size: 9, Family: "Calibri"},
		NumFmt: 3,
	})

	for _, r := range registros {
		col := func(c int) string {
			name, _ := excelize.ColumnNumberToName(c)
			return fmt.Sprintf("%s%d", name, row)
		}

		f.SetCellValue(ws, col(1), r.Tienda)
		f.SetCellValue(ws, col(2), r.Caja)
		f.SetCellValue(ws, col(3), r.Fecha.Format("02/01/2006"))
		f.SetCellValue(ws, col(4), formatoHora(r.Hora))
		f.SetCellValue(ws, col(5), r.Tipo)
		f.SetCellValue(ws, col(6), r.STipo)
		f.SetCellValue(ws, col(7), r.Numero)
		f.SetCellValue(ws, col(8), r.Codigo)
		f.SetCellValue(ws, col(9), r.CodBar)
		f.SetCellValue(ws, col(10), r.Descrip)
		f.SetCellValue(ws, col(11), r.CodVen)
		f.SetCellValue(ws, col(12), r.Modelo)
		f.SetCellValue(ws, col(13), r.Serial)
		f.SetCellValue(ws, col(14), r.Cantidad)
		f.SetCellValue(ws, col(15), r.NCntd)
		f.SetCellValue(ws, col(16), r.NPvpDol)
		f.SetCellValue(ws, col(17), r.NPvp2Dol)
		f.SetCellValue(ws, col(18), r.NPvp3Dol)
		f.SetCellValue(ws, col(19), r.NPvpCop)
		f.SetCellValue(ws, col(20), r.Precio)
		f.SetCellValue(ws, col(21), r.NPrecio)
		f.SetCellValue(ws, col(22), r.IGV)
		f.SetCellValue(ws, col(23), r.NoDscto)
		f.SetCellValue(ws, col(24), r.CodCli)
		f.SetCellValue(ws, col(25), r.Anulada)
		f.SetCellValue(ws, col(26), r.Depto)
		f.SetCellValue(ws, col(27), r.Familia)
		f.SetCellValue(ws, col(28), r.Costo)
		f.SetCellValue(ws, col(29), r.NCosDol)
		f.SetCellValue(ws, col(30), r.Pvpt)
		f.SetCellValue(ws, col(31), r.Oferta)
		f.SetCellValue(ws, col(32), r.Devlto)
		f.SetCellValue(ws, col(33), r.Margen)
		f.SetCellValue(ws, col(34), r.PvpVen)
		f.SetCellValue(ws, col(35), r.LPesado)
		f.SetCellValue(ws, col(36), r.NroCie)
		if r.FechaCie != nil {
			f.SetCellValue(ws, col(37), r.FechaCie.Format("02/01/2006"))
		}

		for c := 1; c <= 37; c++ {
			cellRef := col(c)
			if c == 14 || c == 15 || c == 16 || c == 17 || c == 18 || c == 19 || c == 20 || c == 21 || c == 22 || c == 28 || c == 29 || c == 32 || c == 33 || c == 34 {
				f.SetCellStyle(ws, cellRef, cellRef, bodStyle)
			} else {
				f.SetCellStyle(ws, cellRef, cellRef, intStyle)
			}
		}
		row++
	}
	return row
}

func mustColumn(n int) string {
	name, _ := excelize.ColumnNumberToName(n)
	return name
}
