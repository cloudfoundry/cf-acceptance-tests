package routing_api

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
)

var config helpers.Config

func TestApplications(t *testing.T) {
	RegisterFailHandler(Fail)

	config = helpers.LoadConfig()

	componentName := "routing_api"

	rs := []Reporter{}

	if config.ArtifactsDirectory != "" {
		helpers.EnableCFTrace(config, componentName)
		rs = append(rs, helpers.NewJUnitReporter(config, componentName))
	}

	RunSpecsWithDefaultAndCustomReporters(t, componentName, rs)
}

var _ = BeforeSuite(func() {
	Expect(config.SystemDomain).ToNot(Equal(""), "Must provide a system domain for the routing api suite")
	Expect(config.OauthPassword).ToNot(Equal(""), "Must provide an oauth password for the routing api suite")
})
