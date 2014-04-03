package helpers

import (
"os"
"fmt"

ginkgoconfig "github.com/onsi/ginkgo/config"
"github.com/pivotal-cf-experimental/cf-test-helpers/cf"
"github.com/pivotal-cf-experimental/cf-test-helpers/generator"
)

var AdminUserContext = cf.NewUserContext(
	os.Getenv("API_ENDPOINT"),
	os.Getenv("ADMIN_USER"),
	os.Getenv("ADMIN_PASSWORD"),
	"",
	"",
	os.Getenv("CF_LOGIN_FLAGS"),
)

func NewRegularUserContext() cf.UserContext {
	return cf.NewUserContext(
		os.Getenv("API_ENDPOINT"),
		fmt.Sprintf("CAT-user-%d-%s", ginkgoconfig.GinkgoConfig.ParallelNode, generator.RandomName()),
		"password",
		os.Getenv("CF_ORG"),
		fmt.Sprintf("CAT-space-%d-%s", ginkgoconfig.GinkgoConfig.ParallelNode, generator.RandomName()),
		os.Getenv("CF_LOGIN_FLAGS"),
	)
}
