package cats_test

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/cfcmdtrace"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	svchelpers "github.com/cloudfoundry/cf-acceptance-tests/helpers/services"
	"github.com/mholt/archiver/v3"

	_ "github.com/cloudfoundry/cf-acceptance-tests/app_syslog_tcp"
	_ "github.com/cloudfoundry/cf-acceptance-tests/apps"
	_ "github.com/cloudfoundry/cf-acceptance-tests/cnb"
	_ "github.com/cloudfoundry/cf-acceptance-tests/credhub"
	_ "github.com/cloudfoundry/cf-acceptance-tests/detect"
	_ "github.com/cloudfoundry/cf-acceptance-tests/docker"
	_ "github.com/cloudfoundry/cf-acceptance-tests/file_based_service_bindings"
	_ "github.com/cloudfoundry/cf-acceptance-tests/http2_routing"
	_ "github.com/cloudfoundry/cf-acceptance-tests/internet_dependent"
	_ "github.com/cloudfoundry/cf-acceptance-tests/ipv6"
	_ "github.com/cloudfoundry/cf-acceptance-tests/isolation_segments"
	_ "github.com/cloudfoundry/cf-acceptance-tests/route_services"
	_ "github.com/cloudfoundry/cf-acceptance-tests/routing"
	_ "github.com/cloudfoundry/cf-acceptance-tests/routing_isolation_segments"
	_ "github.com/cloudfoundry/cf-acceptance-tests/security_groups"
	_ "github.com/cloudfoundry/cf-acceptance-tests/service_credential_binding_rotation"
	_ "github.com/cloudfoundry/cf-acceptance-tests/service_discovery"
	_ "github.com/cloudfoundry/cf-acceptance-tests/services"
	_ "github.com/cloudfoundry/cf-acceptance-tests/ssh"
	_ "github.com/cloudfoundry/cf-acceptance-tests/tasks"
	_ "github.com/cloudfoundry/cf-acceptance-tests/tcp_routing"
	_ "github.com/cloudfoundry/cf-acceptance-tests/user_provided_services"
	_ "github.com/cloudfoundry/cf-acceptance-tests/v3"
	_ "github.com/cloudfoundry/cf-acceptance-tests/volume_services"
	_ "github.com/cloudfoundry/cf-acceptance-tests/windows"

	. "github.com/cloudfoundry/cf-acceptance-tests/helpers/cli_version_check"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/config"
	"github.com/cloudfoundry/cf-test-helpers/v2/cf"
	"github.com/cloudfoundry/cf-test-helpers/v2/helpers"
	"github.com/cloudfoundry/cf-test-helpers/v2/workflowhelpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

const minCliVersion = "8.5.0"

func TestCATS(t *testing.T) {
	var validationError error
	Config, validationError = config.NewCatsConfig(os.Getenv("CONFIG"))
	if validationError != nil {
		fmt.Println("Invalid configuration.  ")
		fmt.Println(validationError)
		fmt.Println("Please fix the contents of $CONFIG:\n  " + os.Getenv("CONFIG") + "\nbefore proceeding.")
		t.FailNow()
	}

	_, rc := GinkgoConfiguration()
	if Config.GetArtifactsDirectory() != "" {
		helpers.EnableCFTrace(Config, "CATS")
		rc.JUnitReport = filepath.Join(Config.GetArtifactsDirectory(), fmt.Sprintf("junit-%s-%d.xml", "CATS", GinkgoParallelProcess()))
	}

	if cfcmdtrace.Enabled() {
		cfcmdtrace.Enable(Config.GetNamePrefix())
	}

	RegisterFailHandler(Fail)
	RunSpecs(t, "CATS", rc)
}

var _ = SynchronizedBeforeSuite(func() []byte {
	installedVersion, err := GetInstalledCliVersionString()

	Expect(err).ToNot(HaveOccurred(), "Error trying to determine CF CLI version")

	PauseOutputInterception()
	fmt.Fprintf(GinkgoWriter, "Running CATs with CF CLI version %s\n", installedVersion)
	ResumeOutputInterception()

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
		"CGO_ENABLED=0",
		"GOOS=linux",
		"GOARCH=amd64",
	)
	buildCmd.Stdout = GinkgoWriter
	buildCmd.Stderr = GinkgoWriter

	err = buildCmd.Run()
	Expect(err).NotTo(HaveOccurred())

	if Config.GetIncludeWindows() {
		windowsBuildCmd := exec.Command("go", "build", "-o", "bin/catnip.exe")
		windowsBuildCmd.Dir = "assets/catnip"
		windowsBuildCmd.Env = append(os.Environ(),
			"CGO_ENABLED=0",
			"GOOS=windows",
			"GOARCH=amd64",
		)
		windowsBuildCmd.Stdout = GinkgoWriter
		windowsBuildCmd.Stderr = GinkgoWriter

		err = windowsBuildCmd.Run()
		Expect(err).NotTo(HaveOccurred())
	}

	doraFiles, err := os.ReadDir(assets.NewAssets().Dora)
	Expect(err).NotTo(HaveOccurred())

	var doraFileNames []string
	for _, doraFile := range doraFiles {
		doraFileNames = append(doraFileNames, assets.NewAssets().Dora+"/"+doraFile.Name())
	}
	zip := archiver.NewZip()
	err = zip.Archive(doraFileNames, assets.NewAssets().DoraZip)
	Expect(err).NotTo(HaveOccurred())

	return bootstrapSharedServiceBroker()
}, func(data []byte) {
	SetDefaultEventuallyTimeout(Config.DefaultTimeoutDuration())
	SetDefaultEventuallyPollingInterval(1 * time.Second)

	Expect(json.Unmarshal(data, &SharedServiceBroker)).To(Succeed(), "failed to decode shared service broker info broadcast from node 1")

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
		Expect(buildpacks).To(ContainSubstring(Config.GetPythonBuildpackName()), "Missing the python buildpack specified in the integration_config.json. There may be other missing buildpacks as well; please double-check your configuration against the buildpacks listed below.")
	})

	TestSetup.Setup()
})

var _ = SynchronizedAfterSuite(func() {
	if TestSetup != nil {
		TestSetup.Teardown()
	}
	if cfcmdtrace.Enabled() {
		AddReportEntry("cfcmdtrace-suite", cfcmdtrace.DrainSuiteSetup())
	}
}, func() {
	teardownSharedServiceBroker()
	os.Remove(assets.NewAssets().DoraZip)
})

// bootstrapSharedServiceBroker runs ONCE, on node 1 only (the node-1 fn of
// SynchronizedBeforeSuite), before any spec runs. It creates a dedicated
// org+space that outlives every per-node TestSetup teardown, pushes the Go test
// broker app there, registers it globally as admin, and publicizes its plans so
// per-node regular users can create-service against it. The returned bytes are
// broadcast to every parallel node and decoded into the SharedServiceBroker
// global. Any failure here fails SynchronizedBeforeSuite, so all nodes fail and
// no specs run (all-or-nothing).
func bootstrapSharedServiceBroker() []byte {
	adminSetup := workflowhelpers.NewTestSuiteSetup(Config)

	suffix := random_name.CATSRandomName("")
	info := SharedBrokerInfo{
		BrokerName: fmt.Sprintf("%s-SHARED-BRKR-%s", Config.GetNamePrefix(), suffix),
		OrgName:    fmt.Sprintf("%s-SHARED-BRKR-ORG-%s", Config.GetNamePrefix(), suffix),
		SpaceName:  fmt.Sprintf("%s-SHARED-BRKR-SPACE-%s", Config.GetNamePrefix(), suffix),
	}

	broker := svchelpers.NewServiceBroker(info.BrokerName, assets.NewAssets().ServiceBroker, adminSetup)
	info.OfferingName = broker.Service.Name
	for _, p := range broker.SyncPlans {
		info.SyncPlans = append(info.SyncPlans, p.Name)
	}
	for _, p := range broker.AsyncPlans {
		info.AsyncPlans = append(info.AsyncPlans, p.Name)
	}

	workflowhelpers.AsUser(adminSetup.AdminUserContext(), Config.GetScaledTimeout(1*time.Minute), func() {
		Expect(cf.Cf("create-org", info.OrgName).Wait()).To(Exit(0), "failed to create shared broker org")
		Expect(cf.Cf("create-space", info.SpaceName, "-o", info.OrgName).Wait()).To(Exit(0), "failed to create shared broker space")
		Expect(cf.Cf("target", "-o", info.OrgName, "-s", info.SpaceName).Wait()).To(Exit(0), "failed to target shared broker space")

		broker.Push(Config)
		broker.Configure()
	})

	// Create (register) and PublicizePlans wrap their own AsUser(admin) blocks.
	broker.Create()
	broker.PublicizePlans()

	payload, err := json.Marshal(info)
	Expect(err).NotTo(HaveOccurred(), "failed to marshal shared service broker info")
	return payload
}

// teardownSharedServiceBroker runs ONCE, on node 1 only (the node-1 fn of
// SynchronizedAfterSuite), after every node's TestSetup.Teardown() has
// completed. It purges the shared offering, deletes the broker + its app, and
// removes the dedicated org/space. Every step is best-effort (logged, never
// fatal) so one failure cannot strand the remaining resources.
func teardownSharedServiceBroker() {
	info := SharedServiceBroker
	if info.BrokerName == "" {
		return
	}

	adminSetup := workflowhelpers.NewTestSuiteSetup(Config)
	workflowhelpers.AsUser(adminSetup.AdminUserContext(), Config.GetScaledTimeout(1*time.Minute), func() {
		bestEffort := func(args ...string) {
			session := cf.Cf(args...).Wait()
			if session.ExitCode() != 0 {
				fmt.Fprintf(GinkgoWriter, "shared broker teardown: `cf %s` exited %d (continuing)\n", strings.Join(args, " "), session.ExitCode())
			}
		}

		cf.Cf("target", "-o", info.OrgName, "-s", info.SpaceName).Wait()
		bestEffort("purge-service-offering", info.OfferingName, "-f")
		bestEffort("delete-service-broker", info.BrokerName, "-f")
		bestEffort("delete", info.BrokerName, "-f", "-r")
		bestEffort("delete-space", info.SpaceName, "-o", info.OrgName, "-f")
		bestEffort("delete-org", info.OrgName, "-f")
	})
}
