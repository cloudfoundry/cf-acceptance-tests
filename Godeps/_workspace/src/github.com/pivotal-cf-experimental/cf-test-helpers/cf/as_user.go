package cf

import (
	"fmt"
	"io/ioutil"
	"os"

	ginkgoconfig "github.com/onsi/ginkgo/config"
	. "github.com/onsi/gomega"
	. "github.com/vito/cmdtest/matchers"
)

func AsUser(userContext UserContext, actions func()) {
	originalCfHomeDir, currentCfHomeDir := InitiateUserContext(userContext)
	defer func() {
		RestoreUserContext(userContext, originalCfHomeDir, currentCfHomeDir)
	}()

	TargetSpace(userContext)

	actions()
}

func InitiateUserContext(userContext UserContext) (originalCfHomeDir, currentCfHomeDir string) {
	originalCfHomeDir = os.Getenv("CF_HOME")
	currentCfHomeDir, err := ioutil.TempDir("", fmt.Sprintf("cf_home_%d", ginkgoconfig.GinkgoConfig.ParallelNode))

	if err != nil {
		panic("Error: could not create temporary home directory: " + err.Error())
	}

	os.Setenv("CF_HOME", currentCfHomeDir)

	if userContext.SkipSSLValidation {
		Expect(Cf("api", userContext.ApiUrl, "--skip-ssl-validation")).To(ExitWith(0))
	} else {
		Expect(Cf("api", userContext.ApiUrl)).To(ExitWith(0))
	}

	Expect(Cf("auth", userContext.Username, userContext.Password)).To(ExitWith(0))

	return
}

func TargetSpace(userContext UserContext) {
	if userContext.Org != "" {
		if userContext.Space != "" {
			Expect(Cf("target", "-o", userContext.Org, "-s", userContext.Space)).To(ExitWith(0))
		} else {
			Expect(Cf("target", "-o", userContext.Org)).To(ExitWith(0))
		}
	}
}

func RestoreUserContext(_ UserContext, originalCfHomeDir, currentCfHomeDir string) {
	Expect(Cf("logout")).To(ExitWith(0))
	os.Setenv("CF_HOME", originalCfHomeDir)
	os.RemoveAll(currentCfHomeDir)
}
