package services

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Database      DatabaseConfig `yaml:"database"`
	MariaDBConfig MariaDBConfig  `yaml:"mariaDB"`
	Server        ServerConfig   `yaml:"server"`
	JWT           JWTConfig      `yaml:"jwt"`
	LDAP          LDAPConfig     `yaml:"ldap"`
	IA            IAConfig       `yaml:"ia"`
	Ollama        OllamaConfig   `yaml:"ollama"`
	RAGaRenn      RAGaRennConfig `yaml:"ragarenn"`
	Age           AgeConfig      `yaml:"age"`
	Rack          RackConfig     `yaml:"rack"`
	Presence      PresenceConfig `yaml:"presence"`
	Webdfd        WebdfdConfig   `yaml:"webdfd"`
}

type WebdfdConfig struct {
	BaseURL        string `yaml:"baseURL"`        // cgiempt.exe (promos, cours, eleves, planning)
	ListeGroupeURL string `yaml:"listeGroupeURL"` // cgihtml.exe (listegroupe MODE=10)
}

type PresenceConfig struct {
	TokenSecret string `yaml:"tokenSecret"`
	PlanningURL string `yaml:"planningURL"`
}

type AgeConfig struct {
	PublicKey string `yaml:"publicKey"`
}

type IAConfig struct {
	Provider string `yaml:"provider"`
}

type OllamaConfig struct {
	BaseURL string `yaml:"baseURL"`
	Model   string `yaml:"model"`
}

type RAGaRennConfig struct {
	BaseURL string `yaml:"baseURL"`
	APIKey  string `yaml:"apiKey"`
	Model   string `yaml:"model"`
}

type RackConfig struct {
	BaseURL    string `yaml:"baseURL"`
	Model      string `yaml:"model"`
	APIKey     string `yaml:"apiKey"`
	CaCertPath string `yaml:"caCertPath"`
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

// MariaDBConfig regroupe les paramètres de connexion à la base de données.
type MariaDBConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Database string `yaml:"database"`
}

// LoadConfig loads environment variables from a .env file and returns a Config struct
func LoadConfigYaml(path string) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	expanded := os.ExpandEnv(string(raw))

	var cfg Config
	if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
