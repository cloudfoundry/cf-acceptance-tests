package helpersinternal

import (
	"github.com/cloudfoundry-incubator/cf-test-helpers/commandstarter"
	"github.com/onsi/gomega/gexec"
)

type starter interface {
	Start(commandstarter.Reporter, string, ...string) (*gexec.Session, error)
}

func Curl(cmdStarter starter, skipSsl bool, args ...string) *gexec.Session {
	curlArgs := append([]string{"-s"}, args...)
	if skipSsl {
		curlArgs = append([]string{"-k"}, curlArgs...)
	}

	reporter := commandstarter.NewDefaultReporter()
	request, err := cmdStarter.Start(reporter, "curl", curlArgs...)

	if err != nil {
		panic(err)
	}

	return request
}
