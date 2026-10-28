package main

import (
	"fmt"
	"strings"
)

func menuEspejo() {
	for {
		fmt.Println()
		fmt.Println(strings.Repeat("=", 52))
		fmt.Println("  ESPEJO DBF -> SQL SERVER")
		fmt.Println(strings.Repeat("=", 52))
		cfg := cargarConfigEspejo("espejo_config.json")
		fmt.Printf("  Sucursal : %s\n", cfg.Sucursal)
		fmt.Printf("  BD destino: [%s] en %s\n", cfg.BaseDatos, SQLServer)
		fmt.Printf("  Origenes : %s\n", strings.Join(cfg.Origenes, ", "))
		fmt.Println()
		fmt.Println("  [1] Cargar espejo (segun origenes configurados)")
		fmt.Println("  [2] Ver resumen de esquemas/tablas")
		fmt.Println("  [3] Ver configuracion (espejo_config.json)")
		fmt.Println()
		fmt.Println("  [0] Volver")
		fmt.Println()

		op := leerEntero("  Seleccione: ")
		switch op {
		case 1:
			confirmarYCargarEspejo()
		case 2:
			resumenEspejo(cfg.BaseDatos)
			fmt.Print("\n  Presione Enter para continuar...")
			leerLinea()
		case 3:
			mostrarConfigEspejo()
			fmt.Print("\n  Presione Enter para continuar...")
			leerLinea()
		case 0:
			return
		}
	}
}

func confirmarYCargarEspejo() {
	cfg := cargarConfigEspejo("espejo_config.json")
	fmt.Println()
	fmt.Printf("  Se creara/recargara la BD [%s] en %s.\n", cfg.BaseDatos, SQLServer)
	fmt.Printf("  Origenes: %s\n", strings.Join(cfg.Origenes, ", "))
	fmt.Println("  Cada tabla se DROP + CREATE (recarga completa).")
	fmt.Print("  ¿Continuar? (S/N): ")
	r := strings.TrimSpace(strings.ToUpper(leerLinea()))
	if r != "S" && r != "SI" && r != "SÍ" {
		fmt.Println("  Cancelado.")
		return
	}
	CargarEspejo(cfg)
	fmt.Print("\n  Presione Enter para continuar...")
	leerLinea()
}

func mostrarConfigEspejo() {
	cfg := cargarConfigEspejo("espejo_config.json")
	fmt.Println("\n  Configuracion espejo (espejo_config.json):")
	fmt.Printf("    sucursal        : %s\n", cfg.Sucursal)
	fmt.Printf("    base_datos      : %s\n", cfg.BaseDatos)
	fmt.Printf("    origenes        : %s\n", strings.Join(cfg.Origenes, ", "))
	if strings.TrimSpace(cfg.Base) != "" {
		fmt.Printf("    base (Data)     : %s\n", cfg.Base)
	} else {
		fmt.Println("    base (Data)     : (auto desde sucursales.json)")
	}
	if strings.TrimSpace(cfg.Datos) != "" {
		fmt.Printf("    datos (Datos)   : %s\n", cfg.Datos)
	} else {
		fmt.Println("    datos (Datos)   : (auto desde sucursales.json)")
	}
	if len(cfg.ExcluirModulos) == 0 {
		fmt.Println("    excluir_modulos : (ninguno)")
	} else {
		fmt.Printf("    excluir_modulos : %s\n", strings.Join(cfg.ExcluirModulos, ", "))
	}
	if cfg.SoloAnio > 0 {
		fmt.Printf("    solo_anio       : %d\n", cfg.SoloAnio)
	} else {
		fmt.Println("    solo_anio       : (todos)")
	}
	fmt.Printf("    batch           : %d\n", cfg.Batch)
}
