package helpers

import (
	"fmt"
	"os"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	ginkgoconfig "github.com/onsi/ginkgo/config"
	"github.com/onsi/ginkgo/reporters"

	. "github.com/pivotal-cf-experimental/cf-test-helpers/cf"
)

func GinkgoBootstrap(t *testing.T, suiteName string) {
	RegisterFailHandler(ExitFailHandler)
	CreateEnvironmentForUserContext(AdminUserContext, RegularUserContext)
	RegisterFailHandler(Fail)

	outputFile := fmt.Sprintf("../results/%s-junit_%d.xml", suiteName, ginkgoconfig.GinkgoConfig.ParallelNode)
	RunSpecsWithDefaultAndCustomReporters(t, suiteName, []Reporter{reporters.NewJUnitReporter(outputFile)})
}

var originalCfHomeDir, currentCfHomeDir string

var ExitFailHandler = func(message string, callerSkip ...int) {
	fmt.Println("Initial User Context Setup Failed: " + message)
	os.Exit(1)
}

var _ = BeforeEach(func() {
	RegisterFailHandler(ExitFailHandler)

	originalCfHomeDir, currentCfHomeDir = InitiateUserContext(RegularUserContext)
	TargetSpace(RegularUserContext)
	RegisterFailHandler(Fail)
})

var _ = AfterEach(func() {
	RegisterFailHandler(ExitFailHandler)
	RestoreUserContext(RegularUserContext, originalCfHomeDir, currentCfHomeDir)
	RegisterFailHandler(Fail)

})
