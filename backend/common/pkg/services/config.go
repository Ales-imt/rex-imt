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
	Migration     MigrationConfig `yaml:"migration"`
	Programme     ProgrammeConfig `yaml:"programme"`
	SMTP          SMTPConfig      `yaml:"smtp"`
	Bullettin     BullettinConfig `yaml:"bullettin"`
}

// MigrationConfig choisit la source du référentiel scolaire synchronisé vers
// PostgreSQL par pkg/migration : promotions, profs, élèves, planning, groupes.
//
// Comme pour ProgrammeConfig, `source` n'a volontairement pas de valeur par
// défaut : une source absente ou inconnue doit faire échouer le démarrage.
// Synchroniser depuis une autre source que celle attendue ne se verrait qu'au
// moment où un cours manquerait à l'appel.
//
// Les deux sources décrivent le MÊME référentiel amont (cybema) et portent les
// mêmes identifiants : `webdfd` l'interroge en direct par HTTP, `hfsql` lit les
// exports JSON produits par le watcher à partir de ses sauvegardes.
type MigrationConfig struct {
	Source  string             `yaml:"source"`  // webdfd | hfsql
	Webdfd  MigrationWebdfdCfg `yaml:"webdfd"`  // service sync, source webdfd
	HFSQL   HFSQLConfig        `yaml:"hfsql"`   // service sync, source hfsql
	Watcher WatcherConfig      `yaml:"watcher"` // service export-webdfd

	// DumpPlanning : chemin d'un fichier JSON où déverser, à chaque cycle,
	// l'intégralité des créneaux collectés ET la liste de ceux qui ont été
	// écartés avec leur raison.
	//
	// Outil d'ENQUÊTE, vide par défaut. Il sert à répondre à « le planning
	// amont contient N créneaux, la base en montre moins, où sont les
	// autres ? » — une question que les logs ne peuvent pas trancher, les
	// créneaux étant écartés à cinq endroits différents du cycle. L'activer
	// mémorise chaque rejet, donc une empreinte mémoire proportionnelle au
	// planning, et réécrit le fichier à chaque cycle.
	DumpPlanning string `yaml:"dumpPlanning"`
}

// MigrationWebdfdCfg paramètre, côté service `sync`, le rythme d'interrogation
// de cybema. Les URL elles-mêmes vivent dans WebdfdConfig : ici on ne décide
// que du QUAND, comme HFSQLConfig.Interval pour l'autre source.
type MigrationWebdfdCfg struct {
	// Interval sépare deux cycles complets. cybema ne prévient pas de ses
	// changements, il faut donc l'interroger régulièrement — mais un cycle lit
	// le planning de chaque promotion, ce qui n'est pas gratuit pour l'amont.
	// Défaut 2h.
	Interval time.Duration `yaml:"interval"`
	// Retry remplace Interval après un cycle en échec (coupure réseau,
	// cgiempt indisponible), pour ne pas attendre le cycle nominal entier
	// avant de retenter. Défaut 5m.
	Retry time.Duration `yaml:"retry"`
}

// HFSQLConfig paramètre, côté service `sync`, la lecture des exports produits
// par `export-webdfd`. Le répertoire est monté en LECTURE SEULE : sync n'y
// écrit rien, la rétention appartient au producteur.
type HFSQLConfig struct {
	Data string `yaml:"data"` // répertoire des exports (chemin absolu)
	// Intervalle de sondage de `data`. Le sondage est préféré à inotify parce
	// qu'il est déclenché sur niveau et non sur front : un événement perdu
	// laisserait un export complet que plus personne ne viendrait lire. Une
	// sauvegarde arrive au plus une fois par jour, la latence est sans
	// conséquence. Défaut 1m.
	Interval time.Duration `yaml:"interval"`
}

// WatcherConfig paramètre le service `export-webdfd` : surveillance du
// répertoire de dépôt des sauvegardes cybema, dézippage, puis conversion en
// JSON par l'exécutable WinDev.
type WatcherConfig struct {
	Depot string `yaml:"depot"` // répertoire de dépôt surveillé (chemin absolu)
	Data  string `yaml:"data"`  // bases dézippées et exports (chemin absolu)
	// Runner choisit comment l'export est exécuté :
	//   - "docker" : `docker run <image>` — le service tourne sur l'hôte ;
	//   - "local"  : wine appelé directement — le service EST le conteneur
	//     wine32-hf55, et n'a donc besoin d'aucun accès au démon.
	// Défaut "docker", qui ne suppose rien du déploiement.
	Runner  string        `yaml:"runner"`
	Image   string        `yaml:"image"`   // runner docker : image de l'export ; défaut wine32-hf55
	Entree  string        `yaml:"entree"`  // runner local : script d'export ; défaut /entrypoint.sh
	Timeout time.Duration `yaml:"timeout"` // durée maximale d'un export ; défaut 15m
	Keep    int           `yaml:"keep"`    // exports conservés sous data ; 0 = tous
}

// ProgrammeConfig choisit la source du planning servi aux élèves et aux profs.
//
// `source` n'a volontairement pas de valeur par défaut : une source absente ou
// inconnue doit faire échouer le démarrage. Retomber silencieusement sur une
// source de repli servirait un planning venu d'ailleurs que celui attendu, sans
// que personne ne s'en aperçoive.
type ProgrammeConfig struct {
	Source string                `yaml:"source"` // bd | webdfd | aurega
	Webdfd ProgrammeWebdfdConfig `yaml:"webdfd"`
	Aurega ProgrammeAuregaConfig `yaml:"aurega"`
}

// ProgrammeWebdfdConfig : cgiempt.exe, interrogé en direct (TYPE=planning_txt).
// Distinct de WebdfdConfig, qui sert la synchronisation côté admin — les deux
// services peuvent viser des instances différentes.
type ProgrammeWebdfdConfig struct {
	BaseURL string `yaml:"baseURL"`
}

type ProgrammeAuregaConfig struct {
	BaseURL string `yaml:"baseURL"`
	APIKey  string `yaml:"apiKey"`
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
