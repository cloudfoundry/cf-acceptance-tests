package cats_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"testing"
	"time"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/mholt/archiver"

	_ "github.com/cloudfoundry/cf-acceptance-tests/apps"
	_ "github.com/cloudfoundry/cf-acceptance-tests/backend_compatibility"
	_ "github.com/cloudfoundry/cf-acceptance-tests/capi_experimental"
	_ "github.com/cloudfoundry/cf-acceptance-tests/credhub"
	_ "github.com/cloudfoundry/cf-acceptance-tests/detect"
	_ "github.com/cloudfoundry/cf-acceptance-tests/docker"
	_ "github.com/cloudfoundry/cf-acceptance-tests/internet_dependent"
	_ "github.com/cloudfoundry/cf-acceptance-tests/isolation_segments"
	_ "github.com/cloudfoundry/cf-acceptance-tests/route_services"
	_ "github.com/cloudfoundry/cf-acceptance-tests/routing"
	_ "github.com/cloudfoundry/cf-acceptance-tests/routing_isolation_segments"
	_ "github.com/cloudfoundry/cf-acceptance-tests/security_groups"
	_ "github.com/cloudfoundry/cf-acceptance-tests/service_discovery"
	_ "github.com/cloudfoundry/cf-acceptance-tests/services"
	_ "github.com/cloudfoundry/cf-acceptance-tests/ssh"
	_ "github.com/cloudfoundry/cf-acceptance-tests/tasks"
	_ "github.com/cloudfoundry/cf-acceptance-tests/v3"
	_ "github.com/cloudfoundry/cf-acceptance-tests/windows"

	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry-incubator/cf-test-helpers/workflowhelpers"
	. "github.com/cloudfoundry/cf-acceptance-tests/helpers/buildpacks"
	. "github.com/cloudfoundry/cf-acceptance-tests/helpers/cli_version_check"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/config"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const minCliVersion = "6.33.1"

func TestCATS(t *testing.T) {
	RegisterFailHandler(Fail)

	var validationError error
	Config, validationError = config.NewCatsConfig(os.Getenv("CONFIG"))

	var _ = SynchronizedBeforeSuite(func() []byte {
		installedVersion, err := GetInstalledCliVersionString()

		Expect(err).ToNot(HaveOccurred(), "Error trying to determine CF CLI version")
		fmt.Println("Running CATs with CF CLI version ", installedVersion)

		Expect(ParseRawCliVersionString(installedVersion).AtLeast(ParseRawCliVersionString(minCliVersion))).To(BeTrue(), "CLI version "+minCliVersion+" is required")

		if validationError != nil {
			fmt.Println("Invalid configuration.  ")
			fmt.Println(validationError)
			Fail("Please fix the contents of $CONFIG:\n  " + os.Getenv("CONFIG") + "\nbefore proceeding.")
		}

		if Config.GetIncludeSsh() {
			ScpPath, err = exec.LookPath("scp")
			Expect(err).NotTo(HaveOccurred())

			SftpPath, err = exec.LookPath("sftp")
			Expect(err).NotTo(HaveOccurred())
		}

		buildCmd := exec.Command("go", "build", "-o", "bin/catnip")
		buildCmd.Dir = "assets/catnip"
		buildCmd.Env = []string{
			fmt.Sprintf("GOPATH=%s", os.Getenv("GOPATH")),
			fmt.Sprintf("GOROOT=%s", os.Getenv("GOROOT")),
			"GOOS=linux",
			"GOARCH=amd64",
		}
		buildCmd.Stdout = GinkgoWriter
		buildCmd.Stderr = GinkgoWriter

		err = buildCmd.Run()
		Expect(err).NotTo(HaveOccurred())

		doraFiles, err := ioutil.ReadDir(assets.NewAssets().Dora)
		Expect(err).NotTo(HaveOccurred())

		var doraFileNames []string
		for _, doraFile := range doraFiles {
			doraFileNames = append(doraFileNames, assets.NewAssets().Dora+"/"+doraFile.Name())
		}

		err = archiver.Zip.Make(assets.NewAssets().DoraZip, doraFileNames)
		Expect(err).NotTo(HaveOccurred())

		return []byte{}
	}, func([]byte) {
		TestSetup = workflowhelpers.NewTestSuiteSetup(Config)

		workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.GetScaledTimeout(1*time.Minute), func() {
			buildpacks, err := GetBuildpacks()
			Expect(err).ToNot(HaveOccurred(), "Error getting buildpacks")

			Expect(buildpacks).To(ContainSubstring(Config.GetBinaryBuildpackName()), "Missing the binary buildpack specified in the integration_config.json. There may be other missing buildpacks as well; please double-check your configuration against the buildpacks listed below.")
			Expect(buildpacks).To(ContainSubstring(Config.GetGoBuildpackName()), "Missing the go buildpack specified in the integration_config.json. There may be other missing buildpacks as well; please double-check your configuration against the buildpacks listed below.")
			Expect(buildpacks).To(ContainSubstring(Config.GetJavaBuildpackName()), "Missing the java buildpack specified in the integration_config.json. There may be other missing buildpacks as well; please double-check your configuration against the buildpacks listed below.")
			Expect(buildpacks).To(ContainSubstring(Config.GetNodejsBuildpackName()), "Missing the NodeJS buildpack specified in the integration_config.json. There may be other missing buildpacks as well; please double-check your configuration against the buildpacks listed below.")
			Expect(buildpacks).To(ContainSubstring(Config.GetRubyBuildpackName()), "Missing the ruby buildpack specified in the integration_config.json. There may be other missing buildpacks as well; please double-check your configuration against the buildpacks listed below.")
		})

		TestSetup.Setup()
	})

	SynchronizedAfterSuite(func() {
		if TestSetup != nil {
			TestSetup.Teardown()
		}
	}, func() {
		os.Remove(assets.NewAssets().DoraZip)
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
