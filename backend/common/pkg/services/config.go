package services

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Database      DatabaseConfig  `yaml:"database"`
	MariaDBConfig MariaDBConfig   `yaml:"mariaDB"`
	Server        ServerConfig    `yaml:"server"`
	JWT           JWTConfig       `yaml:"jwt"`
	LDAP          LDAPConfig      `yaml:"ldap"`
	IA            IAConfig        `yaml:"ia"`
	Ollama        OllamaConfig    `yaml:"ollama"`
	RAGaRenn      RAGaRennConfig  `yaml:"ragarenn"`
	Age           AgeConfig       `yaml:"age"`
	Rack          RackConfig      `yaml:"rack"`
	Presence      PresenceConfig  `yaml:"presence"`
	Webdfd        WebdfdConfig    `yaml:"webdfd"`
	SMTP          SMTPConfig      `yaml:"smtp"`
	Bullettin     BullettinConfig `yaml:"bullettin"`
}

// BullettinConfig configure la génération des bulletins. La conversion
// docx -> PDF est déléguée à un service Gotenberg (sidecar).
type BullettinConfig struct {
	GotenbergURL string `yaml:"gotenbergURL"` // ex: http://gotenberg:3000
}

type WebdfdConfig struct {
	BaseURL        string `yaml:"baseURL"`        // cgiempt.exe (promos, cours, eleves, planning)
	ListeGroupeURL string `yaml:"listeGroupeURL"` // cgihtml.exe (listegroupe MODE=10)
}

type PresenceConfig struct {
	TokenSecret string          `yaml:"tokenSecret"`
	PlanningURL string          `yaml:"planningURL"`
	Timestamp   TimestampConfig `yaml:"timestamp"`
	Witness     WitnessConfig   `yaml:"witness"`
}

// TimestampConfig configures RFC 3161 external anchoring.
// All fields have safe defaults; only tokenSecret is mandatory.
type TimestampConfig struct {
	Enabled       bool          `yaml:"enabled"`
	URLs          []string      `yaml:"urls"`          // TSA endpoints; defaults to FreeTSA if empty
	HashAlgorithm string        `yaml:"hashAlgorithm"` // "sha256" (only value currently supported)
	Timeout       time.Duration `yaml:"timeout"`       // per-TSA HTTP timeout, e.g. 10s
	CaCertPath    string        `yaml:"caCertPath"`    // path to TSA root CA PEM for offline verification
}

// WitnessConfig configures the external witness email sent after each new
// RFC 3161 anchor. The recipient mailbox MUST be controlled by a role distinct
// from the infrastructure administrators, and the application must only have
// send rights on it (see docs/temoin-externe.md) — the code cannot enforce
// this; it is a deployment requirement.
type WitnessConfig struct {
	Enabled    bool       `yaml:"enabled"`
	Recipients []string   `yaml:"recipients"` // external mailboxes receiving the witness
	SMTP       SMTPConfig `yaml:"smtp"`
}

// SMTPConfig holds send-only SMTP parameters. Secrets come from env vars via
// the ${VAR} substitution done by LoadConfigYaml.
type SMTPConfig struct {
	Host     string        `yaml:"host"`
	Port     int           `yaml:"port"`
	Username string        `yaml:"username"`
	Password string        `yaml:"password"`
	From     string        `yaml:"from"`
	StartTLS bool          `yaml:"startTLS"` // upgrade the connection with STARTTLS before auth
	Timeout  time.Duration `yaml:"timeout"`  // network timeout, e.g. 10s
}

// DefaultTSAURL is used when TimestampConfig.URLs is empty.
const DefaultTSAURL = "https://freetsa.org/tsr"

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
	BaseURL string `yaml:"baseURL"`
	Model   string `yaml:"model"`
	APIKey  string `yaml:"apiKey"`
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
