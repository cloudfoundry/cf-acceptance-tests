package helpers

import (
"os"
"fmt"

ginkgoconfig "github.com/onsi/ginkgo/config"
"github.com/pivotal-cf-experimental/cf-test-helpers/cf"
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
		fmt.Sprintf("CAT-user-%d", ginkgoconfig.GinkgoConfig.ParallelNode),
		fmt.Sprintf("CAT-user-pass-%d", ginkgoconfig.GinkgoConfig.ParallelNode),
		os.Getenv("CF_ORG"),
		fmt.Sprintf("CAT-space-%d", ginkgoconfig.GinkgoConfig.ParallelNode),
		os.Getenv("CF_LOGIN_FLAGS"),
	)
}
