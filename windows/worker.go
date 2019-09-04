package windows

import (
	"fmt"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	logshelper "github.com/cloudfoundry/cf-acceptance-tests/helpers/logs"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
)

var _ = WindowsDescribe("apps without a port", func() {
	var (
		appName string
		logs    *Session
	)

	BeforeEach(func() {
		workerPath, err := BuildWithEnvironment(filepath.Join(assets.NewAssets().WindowsWorker, "worker.go"),
			[]string{"GOARCH=amd64", "GOOS=windows"})
		Expect(err).NotTo(HaveOccurred())

		appName = random_name.CATSRandomName("APP")

		Expect(cf.Cf("push", appName,
			"--no-route",
			"--no-start",
			"-p", filepath.Dir(workerPath),
			"-c", fmt.Sprintf(".\\%s", filepath.Base(workerPath)),
			"-u", "none",
			"-b", Config.GetBinaryBuildpackName(),
			"-s", Config.GetWindowsStack()).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
		logs = logshelper.TailFollow(Config.GetUseLogCache(), appName)
		Expect(cf.Cf("start", appName).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
	})

	AfterEach(func() {
		app_helpers.AppReport(appName)
		Expect(cf.Cf("delete", appName, "-f", "-r").Wait()).Should(Exit(0))
		CleanupBuildArtifacts()
	})

	// Relies on start to stage, so pending until that exists
	PIt("runs the app (and doesn't run healthcheck)", func() {
		// check that the app keeps running
		// by checking we see incrementing integers in the logs
		Eventually(logs.Out).Should(Say(`Running Worker \d`))
		Eventually(logs.Out).Should(Say(`Running Worker \d{2}`))

		Expect(logs.Out).ToNot(Say("healthcheck"))
	})
})
