package helpers

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
	ApiEndpoint string `json:"api"`

	AdminUser     string `json:"admin_user"`
	AdminPassword string `json:"admin_password"`

	AppsDomain        string `json:"apps_domain"`
	PersistentAppHost string `json:"persistent_app_host"`

	SkipSSLValidation bool `json:"skip_ssl_validation"`

	ArtifactsDirectory string `json:"artifacts_directory"`
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
		ApiEndpoint: os.Getenv("API_ENDPOINT"),

		AdminUser:     os.Getenv("ADMIN_USER"),
		AdminPassword: os.Getenv("ADMIN_PASSWORD"),

		PersistentAppHost: "persistent-app",

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
