package apps

import (
	"testing"

	"github.com/cloudfoundry/cf-acceptance-tests/helpers"
)

func TestApplications(t *testing.T) {
	helpers.GinkgoBootstrap(t, "Applications")
}
