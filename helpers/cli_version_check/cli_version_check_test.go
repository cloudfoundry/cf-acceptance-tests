package cli_version_check_test

import (
	. "github.com/cloudfoundry/cf-acceptance-tests/helpers/cli_version_check"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("CliVersionCheck", func() {

	Describe("ParseRawCliVersionString", func() {
		It("returns a populated CliVersionCheck{} from a raw versioin string", func() {
			ver := ParseRawCliVersionString("cf version 6.8.0-b15c536-2014-12-10T23:34:29+00:00")
			Expect(ver.Revisions).To(Equal([]int{6, 8, 0}))
			Expect(ver.BuildFromSource).To(BeFalse())
		})

		It("returns a populated CliVersionCheck{} from a clean versioin string", func() {
			ver := ParseRawCliVersionString("5.10.3")
			Expect(ver.Revisions).To(Equal([]int{5, 10, 3}))
			Expect(ver.BuildFromSource).To(BeFalse())
		})

		It("returns a populated CliVersionCheck{} with BuildFromSource=true if the executable is built from source code", func() {
			ver := ParseRawCliVersionString("cf version BUILT_FROM_SOURCE-BUILT_AT_UNKNOWN_TIME")
			Expect(ver.Revisions).To(Equal([]int{}))
			Expect(ver.BuildFromSource).To(BeTrue())
		})

		It("handles semvers without patch version", func() {
			ver := ParseRawCliVersionString("6.15")
			Expect(ver.Revisions).To(Equal([]int{6, 15}))
			Expect(ver.BuildFromSource).To(BeFalse())
		})

		It("handles semvers without minor versions", func() {
			ver := ParseRawCliVersionString("6")
			Expect(ver.Revisions).To(Equal([]int{6}))
			Expect(ver.BuildFromSource).To(BeFalse())
		})
	})

	DescribeTable("AtLeast",
		func(x, y string, expected bool) {
			cliVersionCheck := ParseRawCliVersionString(x)
			Expect(cliVersionCheck.AtLeast(ParseRawCliVersionString(y))).To(Equal(expected))
		},

		// Major vs Major
		Entry("0 is at least 0", "0", "0", true),

		Entry("1 is at least 0", "1", "0", true),
		Entry("0 is not at least 1", "0", "1", false),

		// Major-minor vs Major-minor
		Entry("0.1 is at least 0.1", "0.1", "0.1", true),

		Entry("0.1 is at least 0.0", "0.1", "0.0", true),
		Entry("1.1 is at least 0.1", "1.1", "0.1", true),
		Entry("1.0 is at least 0.1", "1.0", "0.1", true),
		Entry("0.0 is not at least 0.1", "0.0", "0.1", false),
		Entry("0.1 is not at least 1.1", "0.1", "1.1", false),
		Entry("0.1 is not at least 1.0", "0.1", "1.0", false),

		// Major-minor vs Major only
		Entry("0.0 is at least 0", "0.0", "0", true),
		Entry("0 is at least 0.0", "0", "0.0", true),

		Entry("0.1 is at least 0", "0.1", "0", true),
		Entry("1 is at least 0.1", "1", "0.1", true),
		Entry("0 is not at least 0.1", "0", "0.1", false),
		Entry("0.1 is not at least 1", "0.1", "1", false),

		// Major-minor-patch vs Major-minor-patch
		Entry("0.1.1 is at least 0.1.1", "0.1.1", "0.1.1", true),

		Entry("0.1.1 is at least 0.1.0", "0.1.1", "0.1.0", true),
		Entry("0.1.1 is at least 0.0.1", "0.1.1", "0.0.1", true),
		Entry("1.1.1 is at least 0.1.1", "1.1.1", "0.1.1", true),

		Entry("0.1.0 is not at least 0.1.1", "0.1.0", "0.1.1", false),
		Entry("0.0.1 is not at least 0.1.1", "0.0.1", "0.1.1", false),
		Entry("0.1.1 is not at least 1.1.1", "0.1.1", "1.1.1", false),

		// Major-minor-patch vs major-minor
		Entry("0.1.0 is at least 0.1", "0.1.0", "0.1", true),
		Entry("0.1 is at least 0.1.0", "0.1", "0.1.0", true),

		Entry("0.1.1 is at least 0.1", "0.1.1", "0.1", true),
		Entry("0.1 is at least 0.0.1", "0.1", "0.0.1", true),
		Entry("0.1 is not at least 0.1.1", "0.1", "0.1.1", false),
		Entry("0.0.1 is not at least 0.1", "0.0.1", "0.1", false),

		// Major-minor-patch vs major only
		Entry("0.0.0 is at least 0", "0.0.0", "0", true),
		Entry("0 is at least 0.0.0", "0", "0.0.0", true),

		Entry("0.0.1 is at least 0", "0.0.1", "0", true),
		Entry("0.1.0 is at least 0", "0.1.0", "0", true),
		Entry("1.0.0 is at least 0", "1.0.0", "0", true),
		Entry("0 is not at least 0.0.1", "0", "0.0.1", false),
		Entry("0 is not at least 0.1.0", "0", "0.1.0", false),
		Entry("0 is not at least 1.0.0", "0", "1.0.0", false),

		Entry("0.0.1 is not at least 1", "0.0.1", "1", false),
		Entry("0.1.0 is not at least 1", "0.1.0", "1", false),
		Entry("1 is at least 0.0.1", "1", "0.0.1", true),
		Entry("1 is at least 0.1.0", "1", "0.1.0", true),
	)

	Describe("CliVersionCheck.AtLeast()", func() {
		var cliVersionCheck CliVersionCheck

		It("returns true if provided version is built from source'", func() {
			cliVersionCheck = ParseRawCliVersionString("cf version BUILT_FROM_SOURCE-BUILT_AT_UNKNOWN_TIME")
			Expect(cliVersionCheck.AtLeast(ParseRawCliVersionString("7.0.0"))).To(BeTrue())
		})
	})
})
