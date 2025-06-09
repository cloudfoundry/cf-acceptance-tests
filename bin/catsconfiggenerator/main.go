package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	cfg "github.com/cloudfoundry/cf-acceptance-tests/helpers/config"
)

var requiredConfig = `{
  "api": "api.bosh-lite.env.wg-ard.ci.cloudfoundry.org",
  "admin_user": "admin",
  "admin_password": "password",
  "skip_ssl_validation": false,
  "apps_domain": "cf-app.bosh-lite.env.wg-ard.ci.cloudfoundry.org",
  "use_http": false
}`

// This Golang utility generates a complete config.json on demand from the existing defaults in the code.
// Users could make use of it and modify the resulting JSON file as desired for their environments.
func main() {

	tmpFile, err := os.CreateTemp("", "cats-config-with-required-fields-*.json")
	if err != nil {
		fmt.Println("Error creating temporary file:", err)
		return
	}
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.Write([]byte(requiredConfig))
	if err != nil {
		fmt.Println("Error writing to temporary file:", err)
		return
	}

	err = tmpFile.Close()
	if err != nil {
		fmt.Println("Error closing temporary file:", err)
		return
	}

	defaultsConfig, err := cfg.NewCatsConfig(tmpFile.Name())
	if err != nil {
		fmt.Println("Error getting default config:", err)
		return
	}

	configJSON, err := json.MarshalIndent(defaultsConfig, "", "  ")
	if err != nil {
		fmt.Println("Error marshalling config to JSON:", err)
		return
	}

	catsConfigName := "complete-cats-config.json"
	err = os.WriteFile(filepath.Join(".", catsConfigName), configJSON, 0644)
	if err != nil {
		fmt.Println("Error writing JSON to file:", err)
		return
	}

	fmt.Println("Config JSON saved to", catsConfigName)
}
