package config

import (
	"fmt"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// AppConfig holds all runtime configuration loaded from environment variables.
type AppConfig struct {
	Port           string
	AppStage       string
	HttpProxy      string
	BackEndBaseUrl string

	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string

	// Connection pool
	MaxIdleConns string
	MaxOpenConns string

	AppCoria string

	// Ansible timeout
	AnsibleRequesttimeout int
	AAPURL                string
	AAPRESTVERSION        string
	AAPUsername           string
	AAPPassword           string
	AAPJobTemplateName    string

	// Solaris AAP instance (optional).
	AAPSolarisURL           string
	AAPRESTVERSIONSolaris   string
	AAPUsernameSolaris      string
	AAPPasswordSolaris      string
	AAPJobTemplateNameSolaris string

	SNOWBaseURL  string
	SNOWUsername string
	SNOWPassword string

	// JWT authentication
	JWTSecretKey           string
	JWTAccessTokenDuration int

	// LDAP authentication
	LDAPServer       string
	LDAPPort         int
	LDAPBaseDN       string
	LDAPBindDN       string
	LDAPBindUsername string
	LDAPBindPassword string
	LDAPUserFilter   string
	LDAPGroupFilter  string
	LDAPUseSSL       bool
	LDAPSkipTLS      bool

	// OS major versions allowed per OS type (loaded from os.yaml).
	OSVersions map[string][]int

	// StaleScanTimeout is the maximum time (in minutes) a scan job may stay in a
	// non-terminal state before the poller marks it as failed.
	StaleScanTimeout int
}

var (
	instance *AppConfig
	once     sync.Once
)

// Load initializes the singleton AppConfig from environment variables.
func Load() {
	once.Do(func() {
		instance = &AppConfig{
			Port:           getEnv("PORT", "8080"),
			AppStage:       getEnv("OPENSHIFT_STAGE", "DEV"),
			HttpProxy:      getEnv("HTTP_PROXY", ""),
			BackEndBaseUrl: getEnv("BACKEND_BASE_URL", ""),

			DBHost:     getEnv("DB_HOST", "localhost"),
			DBPort:     getEnv("DB_PORT", "5432"),
			DBUser:     getEnv("DB_USER", "ulas"),
			DBPassword: getEnv("DB_PASSWORD", "ulas"),
			DBName:     getEnv("DB_NAME", "ulas"),
			DBSSLMode:  getEnv("DB_SSLMODE", "disable"),

			// Connection pool
			MaxIdleConns: getEnv("MAXIDLECONNS", "10"),
			MaxOpenConns: getEnv("MAXOPENCONNS", "20"),

			AppCoria: getEnv("APP_CORIA", "test"),

			// Ansible timeout
			AnsibleRequesttimeout: getEnvAsInt("ANSIBLEREQUESTTIMEOUT", 7200),
			AAPURL:                getEnv("AAP_URL", "https://sandbox-aap-wolfsoldier47-dev.apps.rm1.0a51.p1.openshiftapps.com"),
			AAPRESTVERSION:        getEnv("AAP_REST_VERSION", "/api/controller/v2/"),
			AAPUsername:           getEnv("AAP_USERNAME", "admin"),
			AAPPassword:           getEnv("AAP_PASSWORD", "VunPlhzsILGqI96hmG9ET6L2OmfdhGo2"),
			AAPJobTemplateName:    getEnv("AAP_JOB_TEMPLATE_NAME", "ulas"),

			// Solaris AAP instance.
			AAPSolarisURL:             getEnv("AAPSOLARIS_URL", ""),
			AAPRESTVERSIONSolaris:     getEnv("AAPRESTVERSION_SOLARIS", "/api/controller/v2/"),
			AAPUsernameSolaris:        getEnv("AAPUSERNAME_SOLARIS", ""),
			AAPPasswordSolaris:        getEnv("AAPPASSWORD_SOLARIS", ""),
			AAPJobTemplateNameSolaris: getEnv("AAPJOBTEMPLATENAME_SOLARIS", ""),

			SNOWBaseURL:  getEnv("SNOW_BASE_URL", ""),
			SNOWUsername: getEnv("SNOW_USERNAME", ""),
			SNOWPassword: getEnv("SNOW_PASSWORD", ""),

			JWTSecretKey:           getEnv("JWT_SECRET_KEY", "heheeeeheeeeeeeehafskjdhsdafjkhsdfjkhjksdfajkhfsdahjksfdajhksdafhjkeeee"),
			JWTAccessTokenDuration: getEnvAsInt("JWT_ACCESS_TOKEN_DURATION", 480), // minutes

			// LDAPServer:       getEnv("LDAP_SERVER", ""),
			// LDAPPort:         getEnvAsInt("LDAP_PORT", 636),
			// LDAPBaseDN:       getEnv("LDAP_BASE_DN", ""),
			// LDAPBindDN:       getEnv("LDAP_BIND_DN", ""),
			// LDAPBindPassword: getEnv("LDAP_BIND_PASSWORD", ""),
			// LDAPUserFilter:   getEnv("LDAP_USER_FILTER", "(cn=%s)"),
			// LDAPGroupFilter:  getEnv("LDAP_GROUP_FILTER", "(memberUid=%s)"),
			// LDAPUseSSL:       getEnv("LDAP_USE_SSL", "true") == "true",
			// LDAPSkipTLS:      getEnv("LDAP_SKIP_TLS", "true") == "true",

			// LDAPServer:       getEnv("LDAP_SERVER", "zzzzzz"),
			// LDAPPort:         getEnvAsInt("LDAP_PORT", 636),
			// LDAPBaseDN:       getEnv("LDAP_BASE_DN", "OU=Users,OU=UserProvisioning,OU=Production,DC=ztb,DC=icb,DC=commerzbank,DC=com"),
			// LDAPBindUsername: getEnv("LDAP_BIND_USERNAME", "taaa"),
			// LDAPBindDN:       getEnv("LDAP_BIND_DN", "CN=%s,OU=ServiceAccounts,OU=UserProvisioning,OU=Production,DC=ztb,DC=icb,DC=commerzbank,DC=com"),
			// LDAPBindPassword: getEnv("LDAP_BIND_PASSWORD", ""),
			// LDAPUserFilter:   getEnv("LDAP_USER_FILTER", "(cn=%s)"),
			// LDAPGroupFilter:  getEnv("LDAP_GROUP_FILTER", "(memberUid=%s)"),
			// LDAPUseSSL:       getEnv("LDAP_USE_SSL", "true") == "true",
			// LDAPSkipTLS:      getEnv("LDAP_SKIP_TLS", "true") == "true",

			StaleScanTimeout: getEnvAsInt("STALE_SCAN_TIMEOUT", 1), //stalescan is in minutes
		}

		if err := instance.loadOSVersions(); err != nil {
			slog.Warn("failed to load os versions config", "error", err)
		}

		log.Println("Configuration loaded successfully")
	})
}

// Get returns the loaded AppConfig instance. Call Load() first.
func Get() *AppConfig {
	if instance == nil {
		slog.Warn("config accessed before Load(); loading defaults now")
		Load()
	}
	return instance
}

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	s := os.Getenv(key)
	if s == "" {
		return defaultValue
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return defaultValue
	}
	return v
}

// DatabaseDSN builds the PostgreSQL DSN from config fields.
func (c *AppConfig) DatabaseDSN() string {
	return "host=" + c.DBHost +
		" port=" + c.DBPort +
		" user=" + c.DBUser +
		" password=" + c.DBPassword +
		" dbname=" + c.DBName +
		" sslmode=" + c.DBSSLMode
}

// MaxIdleConnsInt returns MaxIdleConns as an int.
func (c *AppConfig) MaxIdleConnsInt() int {
	v, _ := strconv.Atoi(c.MaxIdleConns)
	if v <= 0 {
		return 10
	}
	return v
}

// JWTAccessTokenDurationDuration returns the JWT access token duration.
func (c *AppConfig) JWTAccessTokenDurationDuration() time.Duration {
	return time.Duration(c.JWTAccessTokenDuration) * time.Minute
}

// StaleScanTimeoutDuration returns the stale scan timeout as a time.Duration.
func (c *AppConfig) StaleScanTimeoutDuration() time.Duration {
	if c.StaleScanTimeout <= 0 {
		return 2 * time.Hour
	}
	return time.Duration(c.StaleScanTimeout) * time.Minute
}

// MaxOpenConnsInt returns MaxOpenConns as an int.
func (c *AppConfig) MaxOpenConnsInt() int {
	v, _ := strconv.Atoi(c.MaxOpenConns)
	if v <= 0 {
		return 20
	}
	return v
}

// loadOSVersions reads the os.yaml file configured by OS_VERSIONS_FILE.
func (c *AppConfig) loadOSVersions() error {
	path := getEnv("OS_VERSIONS_FILE", "os.yaml")
	if !filepath.IsAbs(path) {
		abs, err := filepath.Abs(path)
		if err == nil {
			path = abs
		}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var versions map[string][]int
	if err := yaml.Unmarshal(data, &versions); err != nil {
		return err
	}

	c.OSVersions = versions
	return nil
}

// AllowedOSVersions returns the configured major versions for an OS type.
func (c *AppConfig) AllowedOSVersions(osType string) []int {
	if c.OSVersions == nil {
		return nil
	}
	versions := make([]int, len(c.OSVersions[osType]))
	copy(versions, c.OSVersions[osType])
	return versions
}

// reports whether a major version is allowed for an OS type.
func (c *AppConfig) IsAllowedOSVersion(osType string, version int) bool {
	for _, v := range c.AllowedOSVersions(osType) {
		if v == version {
			return true
		}
	}
	return false
}

// ResolvedLDAPBindDN returns the configured LDAP bind DN with the
// `LDAPBindUsername` interpolated into `LDAPBindDN` when a %s placeholder
// is present. If no formatting is needed the raw `LDAPBindDN` is returned.
func (c *AppConfig) ResolvedLDAPBindDN() string {
	if c == nil {
		return ""
	}
	if c.LDAPBindDN == "" {
		return ""
	}
	if c.LDAPBindUsername != "" && strings.Contains(c.LDAPBindDN, "%s") {
		return fmt.Sprintf(c.LDAPBindDN, c.LDAPBindUsername)
	}
	return c.LDAPBindDN
}
