package apps

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

const CFPushTimeout = 120.0
const DefaultTimeout = 30.0

var context helpers.SuiteContext

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
				fmt.Sprintf("CATS-TRACE-%s-%d.txt", "Applications", ginkgoconfig.GinkgoConfig.ParallelNode),
			),
		)

		rs = append(
			rs,
			reporters.NewJUnitReporter(
				filepath.Join(
					config.ArtifactsDirectory,
					fmt.Sprintf("junit-%s-%d.xml", "Applications", ginkgoconfig.GinkgoConfig.ParallelNode),
				),
			),
		)
	}

	RunSpecsWithDefaultAndCustomReporters(t, "Applications", rs)
}
