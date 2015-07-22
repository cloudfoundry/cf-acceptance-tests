package cli_version_check_test

import (
	. "github.com/cloudfoundry/cf-acceptance-tests/helpers/cli_version_check"
	// "github.com/cloudfoundry/cf-acceptance-tests/helpers/cli_version_check/cli_version/fakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("CliVersionCheck", func() {
	var (
		cliVersionCheck CliVersionCheck
	)

	Describe("AtLeast", func() {
		It("returns true if provided version is at least equal to the actual CF version", func() {
			cliVersionCheck = NewCliVersionCheck("5.1.0")
			Ω(cliVersionCheck.AtLeast(NewCliVersionCheck("5.0.0"))).To(BeTrue())

			cliVersionCheck = NewCliVersionCheck("5.12.0")
			Ω(cliVersionCheck.AtLeast(NewCliVersionCheck("5.9.0"))).To(BeTrue())

			cliVersionCheck = NewCliVersionCheck("5.9.0")
			Ω(cliVersionCheck.AtLeast(NewCliVersionCheck("5.9.0"))).To(BeTrue())
		})

		It("returns false if provided version is less than the actual CF version", func() {
			cliVersionCheck = NewCliVersionCheck("6.1.1")
			Ω(cliVersionCheck.AtLeast(NewCliVersionCheck("6.2.0"))).To(BeFalse())

			cliVersionCheck = NewCliVersionCheck("6.1.1")
			Ω(cliVersionCheck.AtLeast(NewCliVersionCheck("6.1.2"))).To(BeFalse())

			cliVersionCheck = NewCliVersionCheck("6.9.1")
			Ω(cliVersionCheck.AtLeast(NewCliVersionCheck("6.12.0"))).To(BeFalse())

			cliVersionCheck = NewCliVersionCheck("5.12.0")
			Ω(cliVersionCheck.AtLeast(NewCliVersionCheck("6.12.0"))).To(BeFalse())
		})

		It("returns true if provided version contains 'BUILT_FROM_SOURCE'", func() {
			cliVersionCheck = NewCliVersionCheck("cf version BUILT_FROM_SOURCE-BUILT_AT_UNKNOWN_TIME")
			Ω(cliVersionCheck.AtLeast(NewCliVersionCheck("7.0.0"))).To(BeTrue())
		})
	})
})
