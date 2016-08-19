package cf

import (
	"github.com/cloudfoundry-incubator/cf-test-helpers/cf/internal"
	"github.com/cloudfoundry-incubator/cf-test-helpers/commandstarter"
	"github.com/onsi/gomega/gexec"
)

var Cf = func(args ...string) *gexec.Session {
	cmdStarter := runner.NewCommandStarter()
	return cfinternal.Cf(cmdStarter, args...)
}
