package generator

import (
	"strconv"

	uuid "github.com/nu7hatch/gouuid"
	"github.com/onsi/ginkgo/config"
)

func RandomName() string {
	guid, err := uuid.NewV4()
	if err != nil {
		panic(err)
	}

	return guid.String()
}

func PrefixedRandomName(namePrefix string) string {
	return namePrefix + RandomName()
}

func RandomNameForResource(resourceName string) string {
	return "CATS-" + strconv.Itoa(config.GinkgoConfig.ParallelNode) + "-" + resourceName + "-" + RandomName()
}
