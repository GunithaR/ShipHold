package config

import (
	"os"

	"github.com/GunithaR/ShipHold/internal/domain/policy"
	"gopkg.in/yaml.v3"
)

type File struct {
	Policy policy.Policy `yaml:"policy"`
}

func Load(path string) (File, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return File{}, err
	}

	var config File

	if err := yaml.Unmarshal(data, &config); err != nil {
		return File{}, err
	}

	return config, nil
}
