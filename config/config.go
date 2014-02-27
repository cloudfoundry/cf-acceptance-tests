package config

import (
	"encoding/json"
	"os"
)

type IntegrationConfig struct {
	AppsDomain         string `json:"apps_domain"`
	PersistentAppHost  string `json:"persistent_app_host"`
	FirstBrokerServiceLabel string `json:"first_broker_service_label"`
	FirstBrokerPlanName     string `json:"first_broker_plan_name"`
	SecondBrokerServiceLabel string `json:"second_broker_service_label"`
	SecondBrokerPlanName     string `json:"second_broker_plan_name"`}

func Load() (config IntegrationConfig) {
	path := os.Getenv("CONFIG")
	if path == "" {
		panic("Must set $CONFIG to point to an integration config .json file.")
	}

	return LoadPath(path)
}

func LoadPath(path string) (config IntegrationConfig) {
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
