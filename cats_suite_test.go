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

	_ "github.com/cloudfoundry/cf-acceptance-tests/app_syslog_tcp"
	_ "github.com/cloudfoundry/cf-acceptance-tests/apps"
	_ "github.com/cloudfoundry/cf-acceptance-tests/credhub"
	_ "github.com/cloudfoundry/cf-acceptance-tests/detect"
	_ "github.com/cloudfoundry/cf-acceptance-tests/docker"
	_ "github.com/cloudfoundry/cf-acceptance-tests/http2_routing"
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
	_ "github.com/cloudfoundry/cf-acceptance-tests/tcp_routing"
	_ "github.com/cloudfoundry/cf-acceptance-tests/v3"
	_ "github.com/cloudfoundry/cf-acceptance-tests/volume_services"
	_ "github.com/cloudfoundry/cf-acceptance-tests/windows"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry-incubator/cf-test-helpers/workflowhelpers"
	. "github.com/cloudfoundry/cf-acceptance-tests/helpers/cli_version_check"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/config"
	"github.com/cloudfoundry/custom-cats-reporters/honeycomb"
	"github.com/cloudfoundry/custom-cats-reporters/honeycomb/client"
	"github.com/honeycombio/libhoney-go"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

const minCliVersion = "6.33.1"

func TestCATS(t *testing.T) {
	RegisterFailHandler(Fail)

	var validationError error

	Config, validationError = config.NewCatsConfig(os.Getenv("CONFIG"))
	if validationError != nil {
		defer GinkgoRecover()
		fmt.Println("Invalid configuration.  ")
		fmt.Println(validationError)
		fmt.Println("Please fix the contents of $CONFIG:\n  " + os.Getenv("CONFIG") + "\nbefore proceeding.")
		t.Fail()
	}

	var _ = SynchronizedBeforeSuite(func() []byte {
		installedVersion, err := GetInstalledCliVersionString()

		Expect(err).ToNot(HaveOccurred(), "Error trying to determine CF CLI version")
		fmt.Println("Running CATs with CF CLI version ", installedVersion)

		Expect(ParseRawCliVersionString(installedVersion).AtLeast(ParseRawCliVersionString(minCliVersion))).To(BeTrue(), "CLI version "+minCliVersion+" is required")

		if Config.GetIncludeSsh() {
			ScpPath, err = exec.LookPath("scp")
			Expect(err).NotTo(HaveOccurred())

			SftpPath, err = exec.LookPath("sftp")
			Expect(err).NotTo(HaveOccurred())
		}

		buildCmd := exec.Command("go", "build", "-o", "bin/catnip")
		buildCmd.Dir = "assets/catnip"
		buildCmd.Env = append(os.Environ(),
			"GOOS=linux",
			"GOARCH=amd64",
		)
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
		SetDefaultEventuallyTimeout(Config.DefaultTimeoutDuration())
		SetDefaultEventuallyPollingInterval(1 * time.Second)

		TestSetup = workflowhelpers.NewTestSuiteSetup(Config)

		workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.GetScaledTimeout(1*time.Minute), func() {
			buildpacksSession := cf.Cf("buildpacks").Wait()
			Expect(buildpacksSession).To(Exit(0))
			buildpacks := string(buildpacksSession.Out.Contents())

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

	reporterConfig := Config.GetReporterConfig()

	if reporterConfig.HoneyCombDataset != "" && reporterConfig.HoneyCombWriteKey != "" {
		honeyCombClient := client.New(libhoney.Config{
			WriteKey: reporterConfig.HoneyCombWriteKey,
			Dataset:  reporterConfig.HoneyCombDataset,
		})

		globalTags := map[string]interface{}{
			"run_id":  os.Getenv("RUN_ID"),
			"env_api": Config.GetApiEndpoint(),
		}

		honeyCombReporter := honeycomb.New(honeyCombClient)
		honeyCombReporter.SetGlobalTags(globalTags)
		honeyCombReporter.SetCustomTags(reporterConfig.CustomTags)

		rs = append(rs, honeyCombReporter)
	}

	RunSpecsWithDefaultAndCustomReporters(t, "CATS", rs)
}
