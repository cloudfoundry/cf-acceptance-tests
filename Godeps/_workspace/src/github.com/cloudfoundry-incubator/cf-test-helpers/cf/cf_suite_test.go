package cf_test

import (
	"testing"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/runner"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
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
