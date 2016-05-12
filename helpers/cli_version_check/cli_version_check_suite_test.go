package cli_version_check_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestCliVersionCheck(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "CliVersionCheck Suite")
}
