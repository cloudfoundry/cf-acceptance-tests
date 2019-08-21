package apps

import (
	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
)

var _ = AppsDescribe("Encoding", func() {
	var appName string

	BeforeEach(func() {
		appName = random_name.CATSRandomName("APP")
		Expect(cf.Cf("push",
			appName,
			"--no-start",
			"-b", Config.GetJavaBuildpackName(),
			"-p", assets.NewAssets().Java,
			"-m", "1024M",
		).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
		Expect(cf.Cf("set-env", appName, "JAVA_OPTS", "-Djava.security.egd=file:///dev/urandom").Wait()).To(Exit(0))
		Expect(cf.Cf("start", appName).Wait(CF_JAVA_TIMEOUT)).To(Exit(0))
	})

	AfterEach(func() {
		app_helpers.AppReport(appName)

		Expect(cf.Cf("delete", appName, "-f", "-r").Wait()).To(Exit(0))
	})

	PIt("Does not corrupt UTF-8 characters in filenames", func() {
		curlResponse := helpers.CurlApp(Config, appName, "/omega")
		Expect(curlResponse).Should(ContainSubstring("It's Ω!"))
		Expect(curlResponse).To(ContainSubstring("File encoding is UTF-8"))
	})
})
