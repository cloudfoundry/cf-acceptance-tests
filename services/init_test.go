package services

import (
	"fmt"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo"
	ginkgoconfig "github.com/onsi/ginkgo/config"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"

	"github.com/cloudfoundry/cf-acceptance-tests/helpers"
)

func TestServices(t *testing.T) {
	RegisterFailHandler(Fail)

	config := helpers.LoadConfig()

	helpers.SetupEnvironment(t, helpers.NewContext(config))

	rs := []Reporter{}

	if config.ArtifactsDirectory != "" {
		rs = append(
			rs,
			reporters.NewJUnitReporter(
				filepath.Join(
					config.ArtifactsDirectory,
					fmt.Sprintf("%s-junit_%d.xml", "Services", ginkgoconfig.GinkgoConfig.ParallelNode),
				),
			),
		)
	}

	RunSpecsWithDefaultAndCustomReporters(t, "Services", rs)
}
