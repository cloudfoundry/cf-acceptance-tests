package cli_version_check_test

import (
	. "github.com/cloudfoundry/cf-acceptance-tests/helpers/cli_version_check"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/cli_version_check/cli_version/fakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("CliVersionCheck", func() {
	var (
		fakeCliVersion  *fakes.FakeCliVersion
		cliVersionCheck CliVersionCheck
	)

	BeforeEach(func() {
		fakeCliVersion = &fakes.FakeCliVersion{}
		cliVersionCheck = NewCliVersionCheck(fakeCliVersion)
	})

	Describe("GetCliVersion", func() {
		It("returns a version value that only contains numbers and dots", func() {
			fakeCliVersion.GetVersionReturns("abcdds1.2.3dksfj")

			Ω(cliVersionCheck.GetCliVersion()).To(Equal("1.2.3"))
		})

		It("returns a version value that only contains numbers and dots", func() {
			fakeCliVersion.GetVersionReturns("abcdds1.2.3dksfj")

			Ω(cliVersionCheck.GetCliVersion()).To(Equal("1.2.3"))
		})

		It("returns a version value that only contains numbers and dots", func() {
			fakeCliVersion.GetVersionReturns("abcdds1.2.3dksfj")

			Ω(cliVersionCheck.GetCliVersion()).To(Equal("1.2.3"))
		})

		It("returns an empty string when no version is found", func() {
			fakeCliVersion.GetVersionReturns("cf version BUILT_FROM_SOURCE-BUILT_AT_UNKNOWN_TIME")

			Ω(cliVersionCheck.GetCliVersion()).To(Equal(""))
		})
	})

	Describe("AtLeast", func() {
		It("returns true if provided version is at least equal to the actual CF version", func() {
			fakeCliVersion.GetVersionReturns("5.1.0")

			Ω(cliVersionCheck.AtLeast("5.2.0")).To(BeTrue())
		})

		It("returns false if provided version is less than the actual CF version", func() {
			fakeCliVersion.GetVersionReturns("6.1.1")

			Ω(cliVersionCheck.AtLeast("6.1.0")).To(BeFalse())
		})

		It("returns true if provided version is an empty string", func() {
			fakeCliVersion.GetVersionReturns("")

			Ω(cliVersionCheck.AtLeast("7.0.0")).To(BeTrue())
		})

	})
})
