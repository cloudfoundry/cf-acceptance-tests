package services

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	. "github.com/onsi/ginkgo"
	ginkgoconfig "github.com/onsi/ginkgo/config"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"
	. "github.com/vito/cmdtest/matchers"

	"github.com/cloudfoundry/cf-acceptance-tests/config"
	. "github.com/pivotal-cf-experimental/cf-test-helpers/cf"
)

var IntegrationConfig = config.Load()
var homePath string

func TestServices(t *testing.T) {
	RegisterFailHandler(Fail)

	CreateHomeConfig()
	RunSpecsWithDefaultAndCustomReporters(t, "Services", []Reporter{reporters.NewJUnitReporter(fmt.Sprintf("junit_%d.xml", ginkgoconfig.GinkgoConfig.ParallelNode))})
	RemoveHomeConfig()
}

func CreateHomeConfig() {
	homePath = fmt.Sprintf("%s/cf_config_%s", os.Getenv("HOME"), strconv.Itoa(ginkgoconfig.GinkgoConfig.ParallelNode))
	os.MkdirAll(homePath, os.ModePerm)
	os.Setenv("CF_HOME", homePath)

	Expect(Cf("api", os.Getenv("API_ENDPOINT"))).To(ExitWith(0))
	Expect(Cf("login", "-u", os.Getenv("CF_USER"), "-p", os.Getenv("CF_USER_PASSWORD"), "-o", os.Getenv("CF_ORG"), "-s", os.Getenv("CF_SPACE"))).To(ExitWith(0))
}

func RemoveHomeConfig() {
	os.RemoveAll(homePath)
}
