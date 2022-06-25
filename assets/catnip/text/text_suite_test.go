package text_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"testing"
)

func TestText(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Text Suite")
}
