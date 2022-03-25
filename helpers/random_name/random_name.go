package random_name

import "github.com/cloudfoundry/cf-test-helpers/generator"

func CATSRandomName(resource string) string {
	return generator.PrefixedRandomName("CATS", resource)
}
