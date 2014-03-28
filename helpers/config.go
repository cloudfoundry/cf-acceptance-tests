package helpers

import (
	"encoding/json"
	"os"
)

type Config struct {
	AppsDomain        string `json:"apps_domain"`
	PersistentAppHost string `json:"persistent_app_host"`
  ApiEndpoint       string
}

var loadedConfig *Config

func LoadConfig() Config {
	if loadedConfig == nil {
		loadedConfig = loadConfigJsonFromPath()
	}

	if loadedConfig.PersistentAppHost == "" {
		loadedConfig.PersistentAppHost = "persistent-app"
	}

	if loadedConfig.ApiEndpoint == "" {
		loadedConfig.ApiEndpoint = os.Getenv("API_ENDPOINT")
	}

	return *loadedConfig
}

func loadConfigJsonFromPath() *Config {
	var config *Config = &Config{}

	path := loadConfigPathFromEnv()

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

func loadConfigPathFromEnv() string {
	path := os.Getenv("CONFIG")
	if path == "" {
		panic("Must set $CONFIG to point to an integration config .json file.")
	}

	return path
}
