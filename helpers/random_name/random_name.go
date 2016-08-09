package random_name

import "github.com/cloudfoundry-incubator/cf-test-helpers/generator"

func CATSRandomName(resource string) string {
	return generator.PrefixedRandomName("CATS", resource)
}
