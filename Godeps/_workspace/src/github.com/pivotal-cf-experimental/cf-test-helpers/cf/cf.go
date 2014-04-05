package cf

import (
	"github.com/pivotal-cf-experimental/cf-test-helpers/runner"
	"github.com/vito/cmdtest"
)

var Cf = func(args ...string) *cmdtest.Session {
	return runner.Run("gcf", args...)
}
