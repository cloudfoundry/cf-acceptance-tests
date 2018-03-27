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
			"--no-start",
			"--no-route",
			"-p", filepath.Dir(workerPath),
			"-c", fmt.Sprintf(".\\%s", filepath.Base(workerPath)),
			"-u", "none",
			"-b", Config.GetBinaryBuildpackName(),
			"-s", Config.GetWindowsStack()).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
		app_helpers.SetBackend(appName)
		logs = logshelper.TailFollow(Config.GetUseLogCache(), appName)
		Expect(cf.Cf("start", appName).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
	})

	AfterEach(func() {
		app_helpers.AppReport(appName, Config.DefaultTimeoutDuration())
		Expect(cf.Cf("delete", appName, "-f", "-r").Wait(Config.DefaultTimeoutDuration())).Should(Exit(0))
		CleanupBuildArtifacts()
	})

	It("runs the app (and doesn't run healthcheck)", func() {
		Eventually(logs.Out, Config.DefaultTimeoutDuration()).Should(Say("Running Worker 1"))
		Eventually(logs.Out, Config.DefaultTimeoutDuration()).Should(Say("Running Worker 10"))
		Expect(logs.Out).ToNot(Say("healthcheck"))
	})
})
