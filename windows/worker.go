package windows

import (
	"fmt"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	logshelper "github.com/cloudfoundry/cf-acceptance-tests/helpers/logs"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	"github.com/cloudfoundry/cf-test-helpers/cf"
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
			"-u", "process",
			"-b", Config.GetBinaryBuildpackName(),
			"-s", Config.GetWindowsStack(),
			"-m", DEFAULT_WINDOWS_MEMORY_LIMIT,
		).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
		logs = logshelper.Follow(appName)
		Expect(cf.Cf("start", appName).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
	})

	AfterEach(func() {
		app_helpers.AppReport(appName)
		Expect(cf.Cf("delete", appName, "-f", "-r").Wait()).Should(Exit(0))
		CleanupBuildArtifacts()
	})

	It("runs the app (and doesn't run healthcheck)", func() {
		// check that the app keeps running
		// by checking we see incrementing integers in the logs
		Eventually(logs.Out).Should(Say(`Running Worker \d`))
		Eventually(logs.Out).Should(Say(`Running Worker \d{2}`))

		Expect(logs.Out).ToNot(Say("healthcheck"))
	})
})
