package cf_acceptance_tests_test

import (
	. "github.com/cloudfoundry/cf-acceptance-tests/helpers/cli_version_check"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const minCliVersion = "6.11.3"

var _ = Describe("cf CLI version", func() {
	It("meets the minimum required CLI version for the CATs", func() {
		installedVersion, err := GetInstalledCliVersionString()
		Ω(err).ToNot(HaveOccurred(), "Error trying to determine CF CLI version")

		Expect(ParseRawCliVersionString(installedVersion).AtLeast(ParseRawCliVersionString(minCliVersion))).To(BeTrue(), "CLI version "+minCliVersion+" is required")
	})
})
