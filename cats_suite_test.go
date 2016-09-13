package cats_test

import (
	"os/exec"
	"testing"
	"time"

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

	"github.com/cloudfoundry-incubator/cf-test-helpers/config"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry-incubator/cf-test-helpers/workflowhelpers"
	// . "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const minCliVersion = "6.16.1"

func TestCATS(t *testing.T) {
	RegisterFailHandler(Fail)

	Config = config.LoadConfig()

	if Config.DefaultTimeout > 0 {
		DEFAULT_TIMEOUT = Config.DefaultTimeout * time.Second
	}

	if Config.SleepTimeout > 0 {
		SLEEP_TIMEOUT = Config.SleepTimeout * time.Second
	}

	if Config.CfPushTimeout > 0 {
		CF_PUSH_TIMEOUT = Config.CfPushTimeout * time.Second
	}

	if Config.LongCurlTimeout > 0 {
		LONG_CURL_TIMEOUT = Config.LongCurlTimeout * time.Second
	}

	if Config.DetectTimeout > 0 {
		DETECT_TIMEOUT = Config.DetectTimeout * time.Second
	}

	UserContext = workflowhelpers.NewContext(Config)
	environment := workflowhelpers.NewEnvironment(UserContext)

	var _ = SynchronizedBeforeSuite(setup, func(encodedSSHPaths []byte) {
		installedVersion, err := GetInstalledCliVersionString()
		Expect(err).ToNot(HaveOccurred(), "Error trying to determine CF CLI version")

		Expect(ParseRawCliVersionString(installedVersion).AtLeast(ParseRawCliVersionString(minCliVersion))).To(BeTrue(), "CLI version "+minCliVersion+" is required")
		if Config.IncludeSsh {
			ScpPath, err := exec.LookPath("scp")
			Expect(err).NotTo(HaveOccurred())

			SftpPath, err := exec.LookPath("sftp")
			Expect(err).NotTo(HaveOccurred())
		}
		environment.Setup()
	})

	AfterSuite(func() {
		environment.Teardown()
	})

	rs := []Reporter{}

	if Config.ArtifactsDirectory != "" {
		helpers.EnableCFTrace(Config, "CATS")
		rs = append(rs, helpers.NewJUnitReporter(Config, "CATS"))
	}

	RunSpecsWithDefaultAndCustomReporters(t, "CATS", rs)
}
