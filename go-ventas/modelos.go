package main

import "time"

type Sucursal struct {
	Codigo  string
	Nombre  string
	RutaIP  string
	RutaDBF string
	Cia     string
}

type VentaRegistro struct {
	Fecha      time.Time
	Hora       int
	Tienda     string
	Caja       int
	Tipo       string
	STipo      string
	Numero     string
	Anulada    string
	Operador   string
	SubTotal   float64
	MontoUSD   float64
	MontoBS    float64
	Impuesto   float64
	Descuento  float64
	IGTF       float64
	Redondeo   float64
	TasaDOL    float64
	CodCli     string
	Nombres    string
	Apellidos  string
	NIT        string
	RIF        string
	CodVen     string
	NroZ       string
	Credito    bool
	Vuelto     float64
	LIMPRIMIO  bool
	NroCie     string
	FechaCie   *time.Time
}

type ResultadoTienda struct {
	Tienda          string
	PromedioFactura float64
	Clientes        int
	Total           float64
	HoraMayor       string
	ClientesMayor   int
	HoraMenor       string
	ClientesMenor   int
}

type VentaPorHora struct {
	TotalUSD float64
	Facturas int
}

type VentaCARegistro struct {
	Fecha    time.Time
	Hora     int
	Tienda   string
	Caja     int
	Tipo     string
	STipo    string
	Numero   string
	Codigo   string
	CodBar   string
	Descrip  string
	CodVen   string
	Modelo   string
	Serial   string
	Cantidad float64
	NCntd    float64
	NPvpDol  float64
	NPvp2Dol float64
	NPvp3Dol float64
	NPvpCop  float64
	Precio   float64
	NPrecio  float64
	IGV      float64
	NoDscto  bool
	CodCli   string
	Anulada  string
	Depto    string
	Familia  string
	Costo    float64
	NCosDol  float64
	Pvpt     string
	Oferta   bool
	Devlto   float64
	Margen   float64
	PvpVen   float64
	LPesado  bool
	NroCie   string
	FechaCie *time.Time
}

type tareaTienda struct {
	idx int
	suc Sucursal
}

var MesesES = map[int]string{
	1: "ENERO", 2: "FEBRERO", 3: "MARZO", 4: "ABRIL",
	5: "MAYO", 6: "JUNIO", 7: "JULIO", 8: "AGOSTO",
	9: "SEPTIEMBRE", 10: "OCTUBRE", 11: "NOVIEMBRE", 12: "DICIEMBRE",
}

var SeccionColores = map[string]string{
	"A": "FFD9E1F2",
	"F": "FFFFE699",
	"H": "FFC6E0B4",
	"V": "FFF8CBAD",
}
