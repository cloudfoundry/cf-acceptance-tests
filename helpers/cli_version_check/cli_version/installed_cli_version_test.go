package cli_version_test

import (
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/cli_version_check/cli_version"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("NewInstalledCliVersion", func() {
	var ver cli_version.InstalledCliVersion

	BeforeEach(func() {
		ver = cli_version.NewInstalledCliVersion()
	})

	Context("GetVersion", func() {
		It("returns a version value that only contains numbers and dots", func() {
			ver.SetFullVersionString("abcdds1.2.3dksfj")
			Ω(ver.GetVersion()).To(Equal("1.2.3"))

			ver.SetFullVersionString("5.2.3  dksfj")
			Ω(ver.GetVersion()).To(Equal("5.2.3"))

			ver.SetFullVersionString("____ 3.4.3 -dksfj")
			Ω(ver.GetVersion()).To(Equal("3.4.3"))
		})

		It("returns the entire string when no version is found", func() {
			ver.SetFullVersionString("cf version BUILT_FROM_SOURCE-BUILT_AT_UNKNOWN_TIME")

			Ω(ver.GetVersion()).To(Equal("cf version BUILT_FROM_SOURCE-BUILT_AT_UNKNOWN_TIME"))
		})
	})
})
