package v3_helpers

import (
	"encoding/json"
	"fmt"

	"github.com/cloudfoundry/cf-test-helpers/v2/cf"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

func GetRouteGuid(hostname string) string {
	routeQuery := fmt.Sprintf("/v3/routes?hosts=%s", hostname)
	getRoutesCurl := cf.Cf("curl", routeQuery)
	Expect(getRoutesCurl.Wait()).To(Exit(0))
	routesJSON := struct {
		Resources []struct {
			Guid string `json:"guid"`
		} `json:"resources"`
	}{}
	bytes := getRoutesCurl.Out.Contents()

	json.Unmarshal(bytes, &routesJSON)

	return routesJSON.Resources[0].Guid
}
