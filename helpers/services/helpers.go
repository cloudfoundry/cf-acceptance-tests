package helpers

import (
	"github.com/pivotal-cf-experimental/cf-acceptance-tests/config"
)

var integrationConfig = config.Load()

func AppUri(appName, endpoint string) string {
	return "http://" + appName + "." + integrationConfig.AppsDomain + endpoint
}
