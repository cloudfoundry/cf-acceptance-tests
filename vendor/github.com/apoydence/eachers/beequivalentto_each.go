package eachers

import (
	"github.com/onsi/gomega"
	gomegaTypes "github.com/onsi/gomega/types"
)

func BeEquivalentToEach(values ...interface{}) gomegaTypes.GomegaMatcher {
	return Each(gomega.BeEquivalentTo, values...)
}
