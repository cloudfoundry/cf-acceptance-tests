package helpers

import (
	"encoding/json"
	"os"
	"path"
)

type CfConfig struct {
	Target      string `json:"Target"`
	AccessToken string `json:"AccessToken"`
}

func LoadCfConfig() (config CfConfig) {
	configPath := path.Join(os.Getenv("HOME"), ".cf", "config.json")

	configFile, err := os.Open(configPath)
	if err != nil {
		panic(err)
	}

	decoder := json.NewDecoder(configFile)

	err = decoder.Decode(&config)
	if err != nil {
		panic(err)
	}

	return
}
