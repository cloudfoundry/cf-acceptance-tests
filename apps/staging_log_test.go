package apps

import (
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/cloudfoundry/cf-acceptance-tests/helpers"
	"github.com/pivotal-cf-experimental/cf-test-helpers/cf"
	"github.com/pivotal-cf-experimental/cf-test-helpers/generator"
)

var _ = Describe("An application being staged", func() {
	var appName string

	BeforeEach(func() {
		appName = generator.RandomName()
	})

	AfterEach(func() {
		cf.Cf("delete", appName, "-f").Wait(DEFAULT_TIMEOUT)
	})

	It("has its staging log streamed during a push", func() {
		push := cf.Cf("push", appName, "-p", helpers.NewAssets().Dora).Wait(CF_PUSH_TIMEOUT)

		output := string(push.Buffer().Contents())
		expected := []string{"Installing dependencies", "Uploading droplet", "App started"}
		found := false
		for _, value := range expected {
			if strings.Contains(output, value) {
				found = true
				break
			}
		}
		Expect(found).To(BeTrue(), "Did not find one of the expected log lines: %s", expected)
	})
})
