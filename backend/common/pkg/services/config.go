package services

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Database DatabaseConfig `yaml:"database"`
	Server   ServerConfig   `yaml:"server"`
	JWT      JWTConfig      `yaml:"jwt"`
	LDAP     LDAPConfig     `yaml:"ldap"`
}

type DatabaseConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Name     string `yaml:"name"`
}

type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type JWTConfig struct {
	Secret                string        `yaml:"secret"`
	AccessTokenExpiresIn  time.Duration `yaml:"accessTokenExpiresIn"`
	RefreshTokenExpiresIn time.Duration `yaml:"refreshTokenExpiresIn"`
}

type LDAPConfig struct {
	URL    string `yaml:"url"`
	BaseDN string `yaml:"baseDN"`
}

// LoadConfig loads environment variables from a .env file and returns a Config struct
func LoadConfigYaml(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var cfg Config
	decoder := yaml.NewDecoder(f)
	if err := decoder.Decode(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
