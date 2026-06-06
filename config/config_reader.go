package config

import (
	"encoding/json"
	"fmt"
	"os"
)

const fileName = "application_%s.json"

func ReadConfig(env string) (*ApplicationConfig, error) {
	var cfg ApplicationConfig

	err := readJSON(fmt.Sprintf(fileName, env), &cfg)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}

func readJSON(fileName string, cfg any) error {
	data, err := os.ReadFile(fileName)
	if err != nil {
		return err
	}

	err = json.Unmarshal(data, &cfg)
	if err != nil {
		return err
	}

	return nil
}
