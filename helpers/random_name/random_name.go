package random_name

import "github.com/cloudfoundry/cf-test-helpers/v2/generator"

func CATSRandomName(resource string) string {
	return generator.PrefixedRandomName("CATS", resource)
}
