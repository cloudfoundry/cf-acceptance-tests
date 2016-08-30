package helpersinternal

import (
	"strings"

	"github.com/cloudfoundry-incubator/cf-test-helpers/config"
)

type AppUriCreator struct {
	Config config.Config
}

func (uriCreator *AppUriCreator) AppUri(appName string, path string) string {
	if path != "" && !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	var subdomain string
	if appName != "" {
		subdomain = appName + "."
	}

	return uriCreator.Config.Protocol() + subdomain + uriCreator.Config.AppsDomain + path
}
