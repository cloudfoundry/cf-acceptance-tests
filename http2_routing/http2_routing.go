package http2_routing

import (
	"context"
	"crypto/tls"
	"time"

	protobuff "github.com/cloudfoundry/cf-acceptance-tests/helpers/assets/test"
	. "github.com/cloudfoundry/cf-acceptance-tests/helpers/v3_helpers"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	"github.com/cloudfoundry/cf-test-helpers/v2/cf"
	"github.com/cloudfoundry/cf-test-helpers/v2/helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

var _ = HTTP2RoutingDescribe("HTTP/2 Routing", func() {
	Context("when a destination only supports HTTP/2", func() {
		It("routes traffic to that destination over HTTP/2", func() {
			appName := random_name.CATSRandomName("APP")

			Expect(cf.Cf(app_helpers.HTTP2WithArgs(
				appName,
				"--no-route",
				"-m", DEFAULT_MEMORY_LIMIT)...).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

			appGUID := app_helpers.GetAppGuid(appName)

			Expect(cf.Cf("create-route", Config.GetAppsDomain(),
				"--hostname", appName,
			).Wait()).To(Exit(0))

			destination := Destination{
				App: App{
					GUID: appGUID,
				},
				Protocol: "http2",
			}
			InsertDestinations(GetRouteGuid(appName), []Destination{destination})

			Eventually(func() string {
				return helpers.CurlAppRoot(Config, appName)
			}).Should(ContainSubstring("Hello"))
		})
	})

	Context("when a destination serves gRPC", func() {
		It("successfully routes the gRPC traffic (requires HTTP/2 for all hops)", func() {
			appName := random_name.CATSRandomName("APP")

			Expect(cf.Cf(app_helpers.GRPCWithArgs(
				appName,
				"--no-route",
				"-m", DEFAULT_MEMORY_LIMIT)...).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

			appGUID := app_helpers.GetAppGuid(appName)

			appsDomain := Config.GetAppsDomain()
			Expect(cf.Cf("create-route", appsDomain,
				"--hostname", appName,
			).Wait()).To(Exit(0))

			destination := Destination{
				App: App{
					GUID: appGUID,
				},
				Protocol: "http2",
			}
			InsertDestinations(GetRouteGuid(appName), []Destination{destination})

			appURI := appName + "." + appsDomain + ":443"

			tlsConfig := tls.Config{InsecureSkipVerify: true}
			creds := credentials.NewTLS(&tlsConfig)

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			conn, err := grpc.DialContext(
				ctx,
				appURI,
				grpc.WithTransportCredentials(creds),
				grpc.WithBlock(),
				grpc.FailOnNonTempDialError(true),
			)
			Expect(err).ToNot(HaveOccurred())
			defer conn.Close()

			client := protobuff.NewTestClient(conn)
			response, err := client.Run(ctx, &protobuff.Request{})
			Expect(err).ToNot(HaveOccurred())
			Expect(response.GetBody()).To(Equal("Hello"))
		})
	})
})
