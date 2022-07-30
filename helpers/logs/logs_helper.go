package logs

import (
	"github.com/cloudfoundry/cf-test-helpers/v2/cf"
	"github.com/onsi/gomega/gexec"
)

func Recent(appName string) *gexec.Session {
	return cf.Cf("logs", "--recent", appName)
}

func Follow(appName string) *gexec.Session {
	return cf.Cf("logs", appName)
}
