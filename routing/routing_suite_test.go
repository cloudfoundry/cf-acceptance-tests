package routing

import (
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

var config helpers.Config

func TestRouting(t *testing.T) {
	RegisterFailHandler(Fail)

	config = helpers.LoadConfig()

	componentName := "Routing"

	rs := []Reporter{}

	context := helpers.NewContext(config)
	environment := helpers.NewEnvironment(context)

	BeforeSuite(func() {
		Expect(config.SystemDomain).ToNot(Equal(""), "Must provide a system domain for the routing suite")
		Expect(config.ClientSecret).ToNot(Equal(""), "Must provide a client secret for the routing suite")
		environment.Setup()
	})

	AfterSuite(func() {
		environment.Teardown()
	})

	if config.ArtifactsDirectory != "" {
		helpers.EnableCFTrace(config, componentName)
		rs = append(rs, helpers.NewJUnitReporter(config, componentName))
	}

	RunSpecsWithDefaultAndCustomReporters(t, componentName, rs)
}
