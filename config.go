package main

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// AppConfig defines one app's KEDA scaling schedule resources.
type AppConfig struct {
	Name         string `yaml:"name"`
	Namespace    string `yaml:"namespace"`
	Deployment   string `yaml:"deployment"`
	ScaledObject string `yaml:"scaledObject"`
	PauseCronJob string `yaml:"pauseCronJob"`
	ResumeCronJob string `yaml:"resumeCronJob"`
	YAMLFile     string `yaml:"yamlFile"`
}

// Config is the top-level config file.
type Config struct {
	Apps []AppConfig `yaml:"apps"`
}

const configFile = "kedacadabra.yaml"

func LoadConfig() (Config, error) {
	data, err := os.ReadFile(configFile)
	if err != nil {
		return Config{}, fmt.Errorf("read %s: %w", configFile, err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse %s: %w", configFile, err)
	}
	if len(cfg.Apps) == 0 {
		return Config{}, fmt.Errorf("no apps defined in %s", configFile)
	}
	return cfg, nil
}
