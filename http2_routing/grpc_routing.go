package http2_routing

import (
	"context"
	"fmt"
	"time"

	protobuff "github.com/cloudfoundry/cf-acceptance-tests/helpers/assets/test"
	. "github.com/cloudfoundry/cf-acceptance-tests/helpers/v3_helpers"
	"google.golang.org/grpc"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	"github.com/cloudfoundry/cf-test-helpers/v2/cf"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

var _ = HTTP2RoutingDescribe("gRPC apps", func() {
	var appName string

	BeforeEach(func() {
		appName = random_name.CATSRandomName("APP")

		pushArgs := app_helpers.GRPCWithArgs(appName, "--no-route", "-m", DEFAULT_MEMORY_LIMIT)
		Expect(cf.Cf(pushArgs...).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

		domain := Config.GetAppsDomain()
		Expect(cf.Cf("map-route", domain, "--hostname", appName, "--app-protocol", "http2").Wait()).To(Exit(0))
	})

	It("can serve gRPC traffic (requires HTTP/2 for all hops)", func() {
		domain := Config.GetAppsDomain()
		target := fmt.Sprintf("%s.%s:443", appName, domain)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		conn, err := grpc.DialContext(ctx, target, grpc.WithBlock(), grpc.FailOnNonTempDialError(true))
		Expect(err).ToNot(HaveOccurred())
		defer conn.Close()

		tc := protobuff.NewTestClient(conn)

		resp, err := tc.Run(ctx, &protobuff.Request{})
		Expect(err).ToNot(HaveOccurred())
		Expect(resp.GetBody()).To(Equal("Hello"))
	})
})
