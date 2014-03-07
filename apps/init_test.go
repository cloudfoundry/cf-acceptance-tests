package apps

import (
	"fmt"
	"testing"

	. "github.com/onsi/ginkgo"
	ginkgoconfig "github.com/onsi/ginkgo/config"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/cf-acceptance-tests/helpers"
	. "github.com/pivotal-cf-experimental/cf-test-helpers/cf"
)

func TestLifecycle(t *testing.T) {
	RegisterFailHandler(Fail)

	AsUser(RegularUserContext, func () {
		suiteName := "Applications"
		outputFile := fmt.Sprintf("../results/%s-junit_%d.xml", suiteName, ginkgoconfig.GinkgoConfig.ParallelNode)

		RunSpecsWithDefaultAndCustomReporters(t, suiteName, []Reporter{reporters.NewJUnitReporter(outputFile)})
	})
}
