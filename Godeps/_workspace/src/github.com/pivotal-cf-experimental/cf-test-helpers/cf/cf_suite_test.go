package cf_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf-experimental/cf-test-helpers/cf"
	"github.com/pivotal-cf-experimental/cf-test-helpers/runner"
)

var originalCf = cf.Cf
var originalCommandInterceptor = runner.CommandInterceptor

var _ = AfterEach(func() {
	cf.Cf = originalCf
	runner.CommandInterceptor = originalCommandInterceptor
})

func TestCf(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Cf Suite")
}
