package helpers

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers/internal"
	ginkgoconfig "github.com/onsi/ginkgo/config"
	"github.com/onsi/ginkgo/reporters"
)

func EnableCFTrace(config helpersinternal.ArtifactsDirectoryConfig, componentName string) {
	os.Setenv("CF_TRACE", traceLogFilePath(config, componentName))
}

func NewJUnitReporter(config helpersinternal.ArtifactsDirectoryConfig, componentName string) *reporters.JUnitReporter {
	return reporters.NewJUnitReporter(jUnitReportFilePath(config, componentName))
}

func traceLogFilePath(config helpersinternal.ArtifactsDirectoryConfig, componentName string) string {
	return filepath.Join(config.GetArtifactsDirectory(), fmt.Sprintf("CATS-TRACE-%s-%d.txt", sanitizeComponentName(componentName), ginkgoNode()))
}

func jUnitReportFilePath(config helpersinternal.ArtifactsDirectoryConfig, componentName string) string {
	return filepath.Join(config.GetArtifactsDirectory(), fmt.Sprintf("junit-%s-%d.xml", sanitizeComponentName(componentName), ginkgoNode()))
}

func ginkgoNode() int {
	return ginkgoconfig.GinkgoConfig.ParallelNode
}

func sanitizeComponentName(componentName string) string {
	return strings.Replace(componentName, " ", "_", -1)
}
