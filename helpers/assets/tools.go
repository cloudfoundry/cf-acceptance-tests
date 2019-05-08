// +build tools

package tools

import (
	_ "github.com/onsi/ginkgo/ginkgo"

	_ "code.cloudfoundry.org/cf-networking-release/src/example-apps/proxy"
	_ "code.cloudfoundry.org/cf-networking-release/src/example-apps/proxy/handlers"
)
