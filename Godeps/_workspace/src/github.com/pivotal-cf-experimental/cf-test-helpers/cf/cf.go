package cf

import (
	. "github.com/onsi/gomega/gexec"
	"github.com/pivotal-cf-experimental/cf-test-helpers/runner"
)

var Cf = func(args ...string) *Session {
	return runner.Run("gcf", args...)
}
