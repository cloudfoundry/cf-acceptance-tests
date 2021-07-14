package http2_routing

import (
	"context"
	"crypto/tls"
	"time"

	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	protobuff "github.com/cloudfoundry/cf-acceptance-tests/helpers/assets/test"
	. "github.com/cloudfoundry/cf-acceptance-tests/helpers/v3_helpers"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

var _ = HTTP2RoutingDescribe("HTTP/2 Routing", func() {
	SkipOnK8s("Not yet supported in CF-for-K8s")

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
				HTTPVersion: 2,
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

			Expect(cf.Cf(
				"push",
				appName,
				"-b", Config.GetGoBuildpackName(),
				"-c", "./grpc",
				"-p", assets.NewAssets().GRPC,
				"-m", DEFAULT_MEMORY_LIMIT,
				"--no-route",
			).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

			appGUID := app_helpers.GetAppGuid(appName)

			appsDomain := Config.GetAppsDomain()
			Expect(cf.Cf("create-route", appsDomain,
				"--hostname", appName,
			).Wait()).To(Exit(0))

			destination := Destination{
				App: App{
					GUID: appGUID,
				},
				HTTPVersion: 2,
			}
			InsertDestinations(GetRouteGuid(appName), []Destination{destination})

			appURI := appName + "." + appsDomain + ":443"

			tlsConfig := tls.Config{InsecureSkipVerify: true}
			creds := credentials.NewTLS(&tlsConfig)

			conn, err := grpc.Dial(
				appURI,
				grpc.WithTransportCredentials(creds),
				grpc.WithBlock(),
				grpc.FailOnNonTempDialError(true),
				grpc.WithTimeout(time.Duration(1)*time.Second),
			)
			Expect(err).ToNot(HaveOccurred())
			defer conn.Close()

			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			client := protobuff.NewTestClient(conn)
			response, err := client.Run(ctx, &protobuff.Request{})
			Expect(err).ToNot(HaveOccurred())
			Expect(response.GetBody()).To(Equal("Hello"))
		})
	})
})
