package cf

import (
	"fmt"
	"io/ioutil"
	"os"

	ginkgoconfig "github.com/onsi/ginkgo/config"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
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

	cfSetApiArgs := []string{"api", userContext.ApiUrl}
	if userContext.SkipSSLValidation {
		cfSetApiArgs = append(cfSetApiArgs, "--skip-ssl-validation")
	}

	Expect(Cf(cfSetApiArgs...).Wait(CF_API_TIMEOUT)).To(Exit(0))

	Expect(Cf("auth", userContext.Username, userContext.Password).Wait(CF_API_TIMEOUT)).To(Exit(0))

	return
}

func TargetSpace(userContext UserContext) {
	if userContext.Org != "" {
		if userContext.Space != "" {
			Expect(Cf("target", "-o", userContext.Org, "-s", userContext.Space).Wait(CF_API_TIMEOUT)).To(Exit(0))
		} else {
			Expect(Cf("target", "-o", userContext.Org).Wait(CF_API_TIMEOUT)).To(Exit(0))
		}
	}
}

func RestoreUserContext(_ UserContext, originalCfHomeDir, currentCfHomeDir string) {
	Expect(Cf("logout").Wait(CF_API_TIMEOUT)).To(Exit(0))
	os.Setenv("CF_HOME", originalCfHomeDir)
	os.RemoveAll(currentCfHomeDir)
}
