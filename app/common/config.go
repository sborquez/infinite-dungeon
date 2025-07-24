package common

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Log struct {
		Level string `yaml:"level"`
	} `yaml:"log"`

	// Engine struct {
	// 	Database struct {
	// 		Type          string `yaml:"type"`
	// 		ConnectionURI string `yaml:"uri"`
	// 		Name          string `yaml:"name"`
	// 	} `yaml:"database"`
	// } `yaml:"engine"`

	Render struct {
		Window struct {
			Width      int  `yaml:"width"`
			Height     int  `yaml:"height"`
			Fullscreen bool `yaml:"fullscreen"`
		} `yaml:"window"`
	} `yaml:"render"`
}

func LoadConfig(path string) (*Config, error) {
	file, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(file, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
