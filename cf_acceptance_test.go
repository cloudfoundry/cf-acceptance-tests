package cf_acceptance_tests_test

import (
	. "github.com/cloudfoundry/cf-acceptance-tests/helpers/cli_version_check"
	. "github.com/cloudfoundry/cf-acceptance-tests/helpers/cli_version_check/cli_version"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const minCliVersion = "6.11.3"

var _ = Describe("CATs", func() {

	It("fails if minimum cli version is not met", func() {
		installedVersion := NewInstalledCliVersion().GetVersion()
		cliUtils := NewCliVersionCheck(installedVersion)

		Expect(cliUtils.AtLeast(NewCliVersionCheck(minCliVersion))).To(BeTrue(), "CLI version "+minCliVersion+" is required")
	})
})
