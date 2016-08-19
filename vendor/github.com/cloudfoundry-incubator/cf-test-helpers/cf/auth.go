package cf

import (
	"github.com/cloudfoundry-incubator/cf-test-helpers/cf/internal"
	"github.com/cloudfoundry-incubator/cf-test-helpers/commandstarter"
	"github.com/onsi/gomega/gexec"
)

var CfAuth = func(user, password string) *gexec.Session {
	cmdStarter := runner.NewCommandStarter()
	return cfinternal.CfAuth(user, password, cmdStarter)
}
