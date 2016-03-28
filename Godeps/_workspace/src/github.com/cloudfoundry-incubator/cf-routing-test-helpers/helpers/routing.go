package helpers

import (
	. "github.com/cloudfoundry/cf-acceptance-tests/Godeps/_workspace/src/github.com/onsi/gomega"

	routing_api "github.com/cloudfoundry-incubator/routing-api"
)

func GetRouterGroupGuid(routingApiClient routing_api.Client) string {
	routerGroups, err := routingApiClient.RouterGroups()
	Expect(err).ToNot(HaveOccurred())
	Expect(len(routerGroups)).ToNot(Equal(0))
	return routerGroups[0].Guid
}
