package diego

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo"
	ginkgoconfig "github.com/onsi/ginkgo/config"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"

	"github.com/cloudfoundry/cf-acceptance-tests/helpers"
)

var context helpers.SuiteContext

const CFPushTimeout = 120.0
const DefaultTimeout = 30.0

func TestApplications(t *testing.T) {
	RegisterFailHandler(Fail)

	config := helpers.LoadConfig()
	context = helpers.NewContext(config)
	environment := helpers.NewEnvironment(context)

	BeforeSuite(func() {
		environment.Setup()
	})

	AfterSuite(func() {
		environment.Teardown()
	})

	rs := []Reporter{}

	if config.ArtifactsDirectory != "" {
		os.Setenv(
			"CF_TRACE",
			filepath.Join(
				config.ArtifactsDirectory,
				fmt.Sprintf("CATS-TRACE-%s-%d.txt", "DiegoApplications", ginkgoconfig.GinkgoConfig.ParallelNode),
			),
		)

		rs = append(
			rs,
			reporters.NewJUnitReporter(
				filepath.Join(
					config.ArtifactsDirectory,
					fmt.Sprintf("junit-%s-%d.xml", "DiegoApplications", ginkgoconfig.GinkgoConfig.ParallelNode),
				),
			),
		)
	}

	RunSpecsWithDefaultAndCustomReporters(t, "Applications with Diego", rs)
}
