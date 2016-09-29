package helpers

import (
	"github.com/cloudfoundry-incubator/cf-test-helpers/commandstarter"
	"github.com/cloudfoundry-incubator/cf-test-helpers/config"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers/internal"
	"github.com/onsi/gomega/gexec"
)

func Curl(cfg config.Config, args ...string) *gexec.Session {
	cmdStarter := commandstarter.NewCommandStarter()
	return helpersinternal.Curl(cmdStarter, cfg.SkipSSLValidation, args...)
}

func CurlSkipSSL(skip bool, args ...string) *gexec.Session {
	cmdStarter := commandstarter.NewCommandStarter()
	return helpersinternal.Curl(cmdStarter, skip, args...)
}
