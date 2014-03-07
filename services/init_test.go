package services

import (
	"fmt"
	"testing"

	. "github.com/onsi/ginkgo"
	ginkgoconfig "github.com/onsi/ginkgo/config"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/cf-acceptance-tests/helpers"
	. "github.com/pivotal-cf-experimental/cf-test-helpers/cf"
)

var config = LoadConfig()
var testAssets = NewAssets()

func TestServices(t *testing.T) {
	RegisterFailHandler(Fail)

	AsUser(RegularUserContext, func () {
		RunSpecsWithDefaultAndCustomReporters(t, "Services", []Reporter{reporters.NewJUnitReporter(fmt.Sprintf("junit_%d.xml", ginkgoconfig.GinkgoConfig.ParallelNode))})
	})
}
