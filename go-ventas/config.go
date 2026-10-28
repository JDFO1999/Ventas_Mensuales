package main

import (
	"encoding/json"
	"os"
)

type Config struct {
	Modo             string   `json:"modo"`
	OutputDir        string   `json:"output_dir"`
	HoraInicio       int      `json:"hora_inicio"`
	HoraFin          int      `json:"hora_fin"`
	IntervaloMinutos int      `json:"intervalo_minutos"`
	HorasFijas       []string `json:"horas_fijas,omitempty"`
}

func CargarConfig(path string) (Config, error) {
	var c Config
	data, err := os.ReadFile(path)
	if err != nil {
		return c, err
	}
	err = json.Unmarshal(data, &c)
	return c, err
}

func GuardarConfig(c Config, path string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
