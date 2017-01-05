package cats_test

import (
	"fmt"
	"os"
	"os/exec"
	"testing"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"

	_ "github.com/cloudfoundry/cf-acceptance-tests/apps"
	_ "github.com/cloudfoundry/cf-acceptance-tests/backend_compatibility"
	_ "github.com/cloudfoundry/cf-acceptance-tests/detect"
	_ "github.com/cloudfoundry/cf-acceptance-tests/docker"
	_ "github.com/cloudfoundry/cf-acceptance-tests/internet_dependent"
	_ "github.com/cloudfoundry/cf-acceptance-tests/route_services"
	_ "github.com/cloudfoundry/cf-acceptance-tests/routing"
	_ "github.com/cloudfoundry/cf-acceptance-tests/security_groups"
	_ "github.com/cloudfoundry/cf-acceptance-tests/services"
	_ "github.com/cloudfoundry/cf-acceptance-tests/ssh"
	_ "github.com/cloudfoundry/cf-acceptance-tests/v3"

	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry-incubator/cf-test-helpers/workflowhelpers"
	. "github.com/cloudfoundry/cf-acceptance-tests/helpers/cli_version_check"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/config"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const minCliVersion = "6.16.1"

func TestCATS(t *testing.T) {
	RegisterFailHandler(Fail)

	var validationError error
	Config, validationError = config.NewCatsConfig(os.Getenv("CONFIG"))

	var _ = BeforeSuite(func() {
		if validationError != nil {
			fmt.Println("Invalid configuration.  ")
			fmt.Println(validationError)
			Fail("Please fix the contents of $CONFIG:\n  " + os.Getenv("CONFIG") + "\nbefore proceeding.")
		}

		TestSetup = workflowhelpers.NewTestSuiteSetup(Config)

		installedVersion, err := GetInstalledCliVersionString()
		Expect(err).ToNot(HaveOccurred(), "Error trying to determine CF CLI version")

		Expect(ParseRawCliVersionString(installedVersion).AtLeast(ParseRawCliVersionString(minCliVersion))).To(BeTrue(), "CLI version "+minCliVersion+" is required")
		if Config.GetIncludeSsh() {
			ScpPath, err = exec.LookPath("scp")
			Expect(err).NotTo(HaveOccurred())

			SftpPath, err = exec.LookPath("sftp")
			Expect(err).NotTo(HaveOccurred())
		}
		TestSetup.Setup()
	})

	AfterSuite(func() {
		if TestSetup != nil {
			TestSetup.Teardown()
		}
	})

	rs := []Reporter{}

	if validationError == nil {
		if Config.GetArtifactsDirectory() != "" {
			helpers.EnableCFTrace(Config, "CATS")
			rs = append(rs, helpers.NewJUnitReporter(Config, "CATS"))
		}
	}

	RunSpecsWithDefaultAndCustomReporters(t, "CATS", rs)
}
