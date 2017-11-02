package nimbus

import (

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
)


var _ = NimbusDescribe("nocache=true request param", func() {

	var appName string

	BeforeEach(func() {
		appName = random_name.CATSRandomName("APP")
		Expect(cf.Cf("push", appName, "-p", assets.NewAssets().NimbusServices, "--no-start", "-i", "2").Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

		app_helpers.SetBackend(appName)
		Expect(cf.Cf("start", appName).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
	})

	AfterEach(func() {
		Expect(cf.Cf("delete", appName, "-f").Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
	})

	It("prevents responses being served from cache", func() {
		responses := make(map[string]uint8)

		// send 25 requests and make sure each response is different
		// in prod there are 12 routers in each DC (24 overall)
		// hence running 25 request and making sure every one is unique (not from cache)
		for i := 0; i <= 24; i++ {
			resp := helpers.CurlApp(Config, appName, "/currtime?nocache=true")
			responses[resp]++
		}

		Expect(len(responses)).To(Equal(25))
	})

})
