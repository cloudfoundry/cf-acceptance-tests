package helpers

import (
	"fmt"
	"os"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/vito/cmdtest/matchers"

	ginkgoconfig "github.com/onsi/ginkgo/config"
	"github.com/onsi/ginkgo/reporters"

	. "github.com/pivotal-cf-experimental/cf-test-helpers/cf"
)

func GinkgoBootstrap(t *testing.T, suiteName string) {
	RegisterFailHandler(ExitFailHandler)
	CreateEnvironmentForUserContext(NewAdminUserContext(), NewRegularUserContext())
	RegisterFailHandler(Fail)

	outputFile := fmt.Sprintf("../results/%s-junit_%d.xml", suiteName, ginkgoconfig.GinkgoConfig.ParallelNode)
	RunSpecsWithDefaultAndCustomReporters(t, suiteName, []Reporter{reporters.NewJUnitReporter(outputFile)})

	RegisterFailHandler(ExitFailHandler)
	AsUser(NewAdminUserContext(), func() {
		Expect(Cf("delete-space", "-f", NewRegularUserContext().Space)).To(ExitWith(0))
	})
	RegisterFailHandler(Fail)

}

var originalCfHomeDir, currentCfHomeDir string

var ExitFailHandler = func(message string, callerSkip ...int) {
	fmt.Println("Initial User Context Setup Failed: " + message)
	os.Exit(1)
}

var _ = BeforeEach(func() {
	RegisterFailHandler(ExitFailHandler)

	originalCfHomeDir, currentCfHomeDir = InitiateUserContext(NewRegularUserContext())
	TargetSpace(NewRegularUserContext())
	RegisterFailHandler(Fail)
})

var _ = AfterEach(func() {
	RegisterFailHandler(ExitFailHandler)
	RestoreUserContext(NewRegularUserContext(), originalCfHomeDir, currentCfHomeDir)
	RegisterFailHandler(Fail)

})
