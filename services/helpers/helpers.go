package helpers

import (
	"github.com/cloudfoundry/cf-acceptance-tests/helpers"
)

var integrationConfig = helpers.Load()

func AppUri(appName, endpoint string) string {
	return "http://" + appName + "." + integrationConfig.AppsDomain + endpoint
}
