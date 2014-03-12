package helpers

import (
	"fmt"
	"os"

	ginkgoconfig "github.com/onsi/ginkgo/config"

	"github.com/pivotal-cf-experimental/cf-test-helpers/cf"
)

var PersistentSpaceUserContext = cf.NewUserContext(os.Getenv("API_ENDPOINT"),
	os.Getenv("CF_USER"),
	os.Getenv("CF_USER_PASSWORD"),
	os.Getenv("CF_ORG"),
	"persistent-space")

func spaceNameForNode(basename string) string {
	return fmt.Sprintf("%s-%d", basename, ginkgoconfig.GinkgoConfig.ParallelNode)
}

func NewAdminUserContext() cf.UserContext {
	return cf.NewUserContext(os.Getenv("API_ENDPOINT"),
		os.Getenv("ADMIN_USER"),
		os.Getenv("ADMIN_PASSWORD"),
		os.Getenv("CF_ORG"),
		spaceNameForNode(os.Getenv("CF_SPACE")))
}

func NewRegularUserContext() cf.UserContext {
	return cf.NewUserContext(os.Getenv("API_ENDPOINT"),
		os.Getenv("CF_USER"),
		os.Getenv("CF_USER_PASSWORD"),
		os.Getenv("CF_ORG"),
		spaceNameForNode(os.Getenv("CF_SPACE")))
}

