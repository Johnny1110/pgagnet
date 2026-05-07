package config

import (
	"fmt"
	"os"
	"sort"

	"gopkg.in/yaml.v3"
)

type Database struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DBName   string `yaml:"dbname"`
	SSLMode  string `yaml:"sslmode"`
}

type Config struct {
	Databases map[string]Database `yaml:"databases"`
}

func Load(path string) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var c Config
	if err := yaml.Unmarshal(raw, &c); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if len(c.Databases) == 0 {
		return nil, fmt.Errorf("no databases defined in %s", path)
	}
	for name, db := range c.Databases {
		if db.Host == "" || db.DBName == "" || db.User == "" {
			return nil, fmt.Errorf("database %q is missing host/dbname/user", name)
		}
	}
	return &c, nil
}

func (c *Config) Names() []string {
	names := make([]string, 0, len(c.Databases))
	for k := range c.Databases {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}
