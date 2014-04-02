package helpers

import (
	"fmt"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/vito/cmdtest/matchers"

	ginkgoconfig "github.com/onsi/ginkgo/config"
	"github.com/onsi/ginkgo/reporters"

	"github.com/pivotal-cf-experimental/cf-test-helpers/cf"
)

func GinkgoBootstrap(t *testing.T, suiteName string) {
	RegisterFailHandler(Fail)

	cf.AsUser(RegularUserContext, func () {
		outputFile := fmt.Sprintf("../results/%s-junit_%d.xml", suiteName, ginkgoconfig.GinkgoConfig.ParallelNode)
		RunSpecsWithDefaultAndCustomReporters(t, suiteName, []Reporter{reporters.NewJUnitReporter(outputFile)})
	})
}

var _ = BeforeEach(func() {
	Expect(cf.Cf("target", "-s", RegularUserContext.Space, "-o", RegularUserContext.Org)).To(ExitWith(0))
})
