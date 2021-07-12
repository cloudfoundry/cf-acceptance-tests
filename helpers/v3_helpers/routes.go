package v3_helpers

import (
	"fmt"
	"regexp"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

func GetRouteGuid(hostname string) string {
	routeQuery := fmt.Sprintf("/v3/routes?hosts=%s", hostname)
	getRoutesCurl := cf.Cf("curl", routeQuery)
	Expect(getRoutesCurl.Wait()).To(Exit(0))

	routeGuidRegex := regexp.MustCompile(`\s+"guid": "(.+)"`)
	return routeGuidRegex.FindStringSubmatch(string(getRoutesCurl.Out.Contents()))[1]
}
