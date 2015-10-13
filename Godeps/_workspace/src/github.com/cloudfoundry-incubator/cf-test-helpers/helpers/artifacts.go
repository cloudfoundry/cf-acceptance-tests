package helpers

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	ginkgoconfig "github.com/onsi/ginkgo/config"
	"github.com/onsi/ginkgo/reporters"
)

func EnableCFTrace(config Config, componentName string) {
	os.Setenv("CF_TRACE", traceLogFilePath(config, componentName))
}

func traceLogFilePath(config Config, componentName string) string {
	return filepath.Join(config.ArtifactsDirectory, fmt.Sprintf("CATS-TRACE-%s-%d.txt", sanitizeComponentName(componentName), ginkgoNode()))
}

func NewJUnitReporter(config Config, componentName string) *reporters.JUnitReporter {
	return reporters.NewJUnitReporter(jUnitReportFilePath(config, componentName))
}

func jUnitReportFilePath(config Config, componentName string) string {
	return filepath.Join(config.ArtifactsDirectory, fmt.Sprintf("junit-%s-%d.xml", sanitizeComponentName(componentName), ginkgoNode()))
}

func ginkgoNode() int {
	return ginkgoconfig.GinkgoConfig.ParallelNode
}

func sanitizeComponentName(componentName string) string {
	return strings.Replace(componentName, " ", "_", -1)
}
