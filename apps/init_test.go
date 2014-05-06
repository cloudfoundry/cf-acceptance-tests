package apps

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	ginkgoconfig "github.com/onsi/ginkgo/config"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"

	"github.com/cloudfoundry/cf-acceptance-tests/helpers"
)

const (
	DEFAULT_TIMEOUT   = 30 * time.Second
	CF_PUSH_TIMEOUT   = 2 * time.Minute
	LONG_CURL_TIMEOUT = 2 * time.Minute
)

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
		os.Setenv("CF_TRACE", traceLogFilePath(config))
		rs = append(rs, reporters.NewJUnitReporter(jUnitReportFilePath(config)))
	}

	RunSpecsWithDefaultAndCustomReporters(t, "Applications", rs)
}

func traceLogFilePath(config helpers.Config) string {
	return filepath.Join(config.ArtifactsDirectory, fmt.Sprintf("CATS-TRACE-%s-%d.txt", "Applications", ginkgoNode()))
}

func jUnitReportFilePath(config helpers.Config) string {
	return filepath.Join(config.ArtifactsDirectory, fmt.Sprintf("junit-%s-%d.xml", "Applications", ginkgoNode()))
}

func ginkgoNode() int {
	return ginkgoconfig.GinkgoConfig.ParallelNode
}
