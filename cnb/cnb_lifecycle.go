//go:build !noInternet && !noCNB
// +build !noInternet,!noCNB

package cnb

import (
	"path/filepath"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	"github.com/cloudfoundry/cf-test-helpers/v2/cf"
	"github.com/cloudfoundry/cf-test-helpers/v2/helpers"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

var _ = CNBDescribe("CloudNativeBuildpacks lifecycle", func() {
	var appName string

	BeforeEach(func() {
		appName = random_name.CATSRandomName("CNB-APP")
	})

	AfterEach(func() {
		app_helpers.AppReport(appName)
		Eventually(cf.Cf("delete", appName, "-f")).Should(Exit(0))
	})

	Describe("pushing Node.js application with Cloud Native Buildpacks", func() {
		It("makes the app reachable via its bound route", func() {
			Expect(cf.Cf("push",
				appName,
				"-p", assets.NewAssets().CNBNode,
				"-b", Config.GetCNBNodejsBuildpackName(),
				"-f", filepath.Join(assets.NewAssets().CNBNode, "manifest.yml"),
			).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

			Eventually(func() string {
				return helpers.CurlAppRoot(Config, appName)
			}).Should(ContainSubstring("Hello from a node app!"))
		})
	})

	Describe("pushing Node.js application with CNB lifecycle and no buildpacks", func() {
		It("fails", func() {
			push := cf.Cf("push",
				appName,
				"-p", assets.NewAssets().CNBNode,
				"-f", filepath.Join(assets.NewAssets().CNBNode, "manifest.yml"),
			).Wait(Config.CfPushTimeoutDuration())

			Expect(push).To(Exit(1))
			Expect(combineOutput(push.Out, push.Err)).To(Say("Buildpack\\(s\\) must be specified when using Cloud Native Buildpacks"))
		})
	})
})

func combineOutput(outBuffer *Buffer, errBuffer *Buffer) *Buffer {
	combinedOutput := BufferWithBytes(outBuffer.Contents())
	combinedOutput.Write(errBuffer.Contents())
	return combinedOutput
}
