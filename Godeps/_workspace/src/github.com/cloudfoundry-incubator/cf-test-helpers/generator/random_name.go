package generator

import (
	uuid "github.com/cloudfoundry/cf-acceptance-tests/Godeps/_workspace/src/github.com/nu7hatch/gouuid"
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
