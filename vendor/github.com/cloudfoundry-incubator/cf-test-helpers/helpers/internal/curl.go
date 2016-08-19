package helpersinternal

import (
	"github.com/cloudfoundry-incubator/cf-test-helpers/commandstarter"
	"github.com/onsi/gomega/gexec"
)

type starter interface {
	Start(runner.Reporter, string, ...string) (*gexec.Session, error)
}

var SkipSSLValidation bool

func Curl(cmdStarter starter, args ...string) *gexec.Session {
	return CurlSkipSSL(cmdStarter, SkipSSLValidation, args...)
}

func CurlSkipSSL(cmdStarter starter, skip bool, args ...string) *gexec.Session {
	curlArgs := append([]string{"-s"}, args...)
	if skip {
		curlArgs = append([]string{"-k"}, curlArgs...)
	}

	reporter := runner.NewDefaultReporter()
	request, err := cmdStarter.Start(reporter, "curl", curlArgs...)

	if err != nil {
		panic(err)
	}

	return request
}
