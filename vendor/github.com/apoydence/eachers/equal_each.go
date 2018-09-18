package eachers

import (
	"github.com/onsi/gomega"
	gomegaTypes "github.com/onsi/gomega/types"
)

func EqualEach(values ...interface{}) gomegaTypes.GomegaMatcher {
	return Each(gomega.Equal, values...)
}
