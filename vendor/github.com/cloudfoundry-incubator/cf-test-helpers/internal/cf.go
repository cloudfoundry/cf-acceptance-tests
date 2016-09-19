package internal

import (
	"github.com/cloudfoundry-incubator/cf-test-helpers/commandreporter"
	"github.com/onsi/gomega/gexec"
)

func Cf(cmdStarter Starter, args ...string) *gexec.Session {
	reporter := commandreporter.NewCommandReporter()
	request, err := cmdStarter.Start(reporter, "cf", args...)
	if err != nil {
		panic(err)
	}
	return request
}
