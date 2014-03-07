package helpers

import (
	"encoding/json"
	"os"
)

type Config struct {
	AppsDomain        string `json:"apps_domain"`
	PersistentAppHost string `json:"persistent_app_host"`
}

func LoadConfig() (config Config) {
	path := os.Getenv("CONFIG")
	if path == "" {
		panic("Must set $CONFIG to point to an integration config .json file.")
	}

	return loadConfigJsonFromPath(path)
}

func loadConfigJsonFromPath(path string) (config Config) {
	configFile, err := os.Open(path)
	if err != nil {
		panic(err)
	}

	decoder := json.NewDecoder(configFile)
	err = decoder.Decode(&config)
	if err != nil {
		panic(err)
	}

	if config.PersistentAppHost == "" {
		config.PersistentAppHost = "persistent-app"
	}

	return
}
