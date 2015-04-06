package helpers

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type Config struct {
	ApiEndpoint string `json:"api"`
	AppsDomain  string `json:"apps_domain"`

	AdminUser     string `json:"admin_user"`
	AdminPassword string `json:"admin_password"`

	PersistentAppHost      string `json:"persistent_app_host"`
	PersistentAppSpace     string `json:"persistent_app_space"`
	PersistentAppOrg       string `json:"persistent_app_org"`
	PersistentAppQuotaName string `json:"persistent_app_quota_name"`

	SkipSSLValidation bool `json:"skip_ssl_validation"`

	ArtifactsDirectory string `json:"artifacts_directory"`

	DefaultTimeout     time.Duration `json:"default_timeout"`
	CfPushTimeout      time.Duration `json:"cf_push_timeout"`
	LongCurlTimeout    time.Duration `json:"long_curl_timeout"`
	BrokerStartTimeout time.Duration `json:"broker_start_timeout"`

	SyslogDrainPort int    `json:"syslog_drain_port"`
	SyslogIpAddress string `json:"syslog_ip_address"`
}

var loadedConfig *Config

func LoadConfig() Config {
	if loadedConfig == nil {
		loadedConfig = loadConfigJsonFromPath()
	}

	if loadedConfig.ApiEndpoint == "" {
		panic("missing configuration 'api'")
	}

	if loadedConfig.AdminUser == "" {
		panic("missing configuration 'admin_user'")
	}

	if loadedConfig.ApiEndpoint == "" {
		panic("missing configuration 'admin_password'")
	}

	return *loadedConfig
}

func loadConfigJsonFromPath() *Config {
	var config *Config = &Config{
		PersistentAppHost:      "CATS-persistent-app",
		PersistentAppSpace:     "CATS-persistent-space",
		PersistentAppOrg:       "CATS-persistent-org",
		PersistentAppQuotaName: "CATS-persistent-quota",

		ArtifactsDirectory: filepath.Join("..", "results"),
	}

	path := configPath()

	configFile, err := os.Open(path)
	if err != nil {
		panic(err)
	}

	decoder := json.NewDecoder(configFile)
	err = decoder.Decode(config)
	if err != nil {
		panic(err)
	}

	return config
}

func configPath() string {
	path := os.Getenv("CONFIG")
	if path == "" {
		panic("Must set $CONFIG to point to an integration config .json file.")
	}

	return path
}
