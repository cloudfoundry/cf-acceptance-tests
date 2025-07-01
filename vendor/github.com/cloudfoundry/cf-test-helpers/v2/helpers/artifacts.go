package helpers

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	ginkgoconfig "github.com/onsi/ginkgo/v2"
	"github.com/onsi/ginkgo/v2/reporters"
)

type artifactsDirectoryConfig interface {
	GetArtifactsDirectory() string
}

func EnableCFTrace(config artifactsDirectoryConfig, componentName string) {
	err := os.Setenv("CF_TRACE", traceLogFilePath(config, componentName))
	if err != nil {
		panic(err)
	}
}

func NewJUnitReporter(config artifactsDirectoryConfig, componentName string) *reporters.JUnitReporter {
	return reporters.NewJUnitReporter(jUnitReportFilePath(config, componentName))
}

func traceLogFilePath(config artifactsDirectoryConfig, componentName string) string {
	return filepath.Join(config.GetArtifactsDirectory(), fmt.Sprintf("CATS-TRACE-%s-%d.txt", sanitizeComponentName(componentName), ginkgoNode()))
}

func jUnitReportFilePath(config artifactsDirectoryConfig, componentName string) string {
	return filepath.Join(config.GetArtifactsDirectory(), fmt.Sprintf("junit-%s-%d.xml", sanitizeComponentName(componentName), ginkgoNode()))
}

func ginkgoNode() int {
	return ginkgoconfig.GinkgoParallelProcess()
}

func sanitizeComponentName(componentName string) string {
	return strings.ReplaceAll(componentName, " ", "_")
}
