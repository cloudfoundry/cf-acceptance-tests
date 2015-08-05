package cli_version_check_test

import (
	. "github.com/cloudfoundry/cf-acceptance-tests/helpers/cli_version_check"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("CliVersionCheck", func() {

	Describe("ParseRawCliVersionString", func() {
		It("returns a populated CliVersionCheck{} from a raw versioin string", func() {
			ver := ParseRawCliVersionString("cf version 6.8.0-b15c536-2014-12-10T23:34:29+00:00")
			Ω(ver.Revisions).To(Equal([]int{6, 8, 0}))
			Ω(ver.BuildFromSource).To(BeFalse())
		})

		It("returns a populated CliVersionCheck{} from a clean versioin string", func() {
			ver := ParseRawCliVersionString("5.10.3")
			Ω(ver.Revisions).To(Equal([]int{5, 10, 3}))
			Ω(ver.BuildFromSource).To(BeFalse())
		})

		It("returns a populated CliVersionCheck{} with BuildFromSource=true if the executable is built from source code", func() {
			ver := ParseRawCliVersionString("cf version BUILT_FROM_SOURCE-BUILT_AT_UNKNOWN_TIME")
			Ω(ver.Revisions).To(Equal([]int{}))
			Ω(ver.BuildFromSource).To(BeTrue())
		})
	})

	Describe("CliVersionCheck.AtLeast()", func() {
		var cliVersionCheck CliVersionCheck

		It("returns true if provided version is at least equal to the actual CF version", func() {
			cliVersionCheck = ParseRawCliVersionString("5.1.0")
			Ω(cliVersionCheck.AtLeast(ParseRawCliVersionString("5.0.0"))).To(BeTrue())

			cliVersionCheck = ParseRawCliVersionString("5.12.0")
			Ω(cliVersionCheck.AtLeast(ParseRawCliVersionString("5.9.0"))).To(BeTrue())

			cliVersionCheck = ParseRawCliVersionString("5.9.0")
			Ω(cliVersionCheck.AtLeast(ParseRawCliVersionString("5.9.0"))).To(BeTrue())
		})

		It("returns false if provided version is less than the actual CF version", func() {
			cliVersionCheck = ParseRawCliVersionString("6.1.1")
			Ω(cliVersionCheck.AtLeast(ParseRawCliVersionString("6.2.0"))).To(BeFalse())

			cliVersionCheck = ParseRawCliVersionString("6.1.1")
			Ω(cliVersionCheck.AtLeast(ParseRawCliVersionString("6.1.2"))).To(BeFalse())

			cliVersionCheck = ParseRawCliVersionString("6.9.1")
			Ω(cliVersionCheck.AtLeast(ParseRawCliVersionString("6.12.0"))).To(BeFalse())

			cliVersionCheck = ParseRawCliVersionString("5.12.0")
			Ω(cliVersionCheck.AtLeast(ParseRawCliVersionString("6.12.0"))).To(BeFalse())
		})

		It("returns true if provided version is built from source'", func() {
			cliVersionCheck = ParseRawCliVersionString("cf version BUILT_FROM_SOURCE-BUILT_AT_UNKNOWN_TIME")
			Ω(cliVersionCheck.AtLeast(ParseRawCliVersionString("7.0.0"))).To(BeTrue())
		})
	})
})
