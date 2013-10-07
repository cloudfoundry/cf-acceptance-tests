package lifecycle

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/vito/cmdtest"


	"github.com/vito/runtime-integration/config"
	. "github.com/vito/runtime-integration/helpers"
)

func TestLifecycle(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Application Lifecycle")
}

var IntegrationConfig = config.Load()

var AppName = ""

var doraPath = "../assets/dora"
var helloPath = "../assets/hello-world"

func AppUri(endpoint string) string {
	return "http://" + AppName + "." + IntegrationConfig.AppsDomain + endpoint
}

func Curling(endpoint string) func() *cmdtest.Session {
	return func() *cmdtest.Session {
		return Curl(AppUri(endpoint))
	}
}
