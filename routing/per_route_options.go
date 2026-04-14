package routing

import (
	"crypto/rand"
	"fmt"
	"path/filepath"
	"regexp"
	"slices"
	"sync"

	"github.com/cloudfoundry/cf-test-helpers/v2/cf"
	"github.com/cloudfoundry/cf-test-helpers/v2/helpers"
	"github.com/cloudfoundry/cf-test-helpers/v2/workflowhelpers"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

var (
	appInstanceRegex = regexp.MustCompile("^[[:alnum:]]{8}(-[[:alnum:]]{4}){4}$")
)

var _ = RoutingDescribe("Per-Route Options", func() {
	var (
		appName              string
		appId                string
		instanceIds          [2]string
		leastConnHost        string
		roundRobinHost       string
		hashBasedRoutingHost string
	)

	// Helper function to build URL for a given host
	buildUrl := func(host string) string {
		return fmt.Sprintf("%s%s.%s", Config.Protocol(), host, Config.GetAppsDomain())
	}

	Context("when an app sets the loadbalancing algorithm", func() {
		BeforeEach(func() {
			workflowhelpers.AsUser(TestSetup.AdminUserContext(), TestSetup.ShortTimeout(), func() {
				Expect(cf.Cf("enable-feature-flag", "hash_based_routing").Wait()).To(Exit(0))
			})
			appName = random_name.CATSRandomName("APP")
			asset := assets.NewAssets()
			leastConnHost = random_name.CATSRandomName("dora-lc")
			roundRobinHost = random_name.CATSRandomName("dora-rr")
			hashBasedRoutingHost = random_name.CATSRandomName("dora-hash")
			Expect(cf.Cf("push",
				appName,
				"-b", Config.GetRubyBuildpackName(),
				"-m", DEFAULT_MEMORY_LIMIT,
				"-p", asset.Dora,
				"--var", fmt.Sprintf("domain=%s", Config.GetAppsDomain()),
				"--var", fmt.Sprintf("hashbasedroutinghost=%s", hashBasedRoutingHost),
				"--var", fmt.Sprintf("leastconnhost=%s", leastConnHost),
				"--var", fmt.Sprintf("roundrobinhost=%s", roundRobinHost),
				"-f", filepath.Join(asset.Dora, "route_options_manifest.yml"),
			).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
			appId = app_helpers.GetAppGuid(appName)
			for i := range 2 {
				Eventually(func() bool {
					fmt.Fprintf(GinkgoWriter, "Waiting for app instance %d to start...\n", i)
					curl := helpers.Curl(Config, Config.Protocol()+leastConnHost+"."+Config.GetAppsDomain()+"/id", "-H", fmt.Sprintf("X-Cf-App-Instance: %s:%d", appId, i)).Wait()
					id := string(curl.Out.Contents())
					fmt.Fprintf(GinkgoWriter, "App instance %s\n", id)
					if appInstanceRegex.MatchString(id) {
						instanceIds[i] = id
						fmt.Fprintf(GinkgoWriter, "App instance %d has started. Instance ID: %s.\n", i, id)
						return true
					} else {
						fmt.Fprintf(GinkgoWriter, "App instance %d is not ready yet. Response: %s, curl error: %s.\n", i, id, string(curl.Err.Contents()))
						return false
					}
				}).Should(BeTrue())
			}
		})

		AfterEach(func() {
			app_helpers.AppReport(appName)
			Expect(cf.Cf("delete", appName, "-f", "-r").Wait()).To(Exit(0))
		})

		Context("when it's set to round-robin", func() {
			It("distributes requests evenly", func() {
				doraUrl := buildUrl(roundRobinHost)
				var wg sync.WaitGroup
				for i := 0; i < 10; i++ {
					wg.Add(1)
					go func() {
						defer wg.Done()
						defer GinkgoRecover()
						helpers.Curl(Config, fmt.Sprintf("%s/delay/10", doraUrl), "-H", fmt.Sprintf("X-Cf-App-Instance: %s:0", appId))
					}()
				}

				reqCount := [2]int{0, 0}
				for i := 0; i < 20; i++ {
					id := helpers.Curl(Config, fmt.Sprintf("%s/id", doraUrl)).Wait().Out.Contents()
					reqCount[slices.Index(instanceIds[:], string(id))] += 1
				}

				// allow for some wiggle-room
				Expect(reqCount[0]).To(BeNumerically(">=", 8))
				Expect(reqCount[1]).To(BeNumerically(">=", 8))
				wg.Wait()
			})
		})

		Context("when it's set to least-connection", func() {
			It("always sends the request to the instance with less active connections", func() {
				doraUrl := buildUrl(leastConnHost)
				var wg sync.WaitGroup
				for i := 0; i < 10; i++ {
					wg.Add(1)
					go func() {
						defer wg.Done()
						defer GinkgoRecover()
						helpers.Curl(Config, fmt.Sprintf("%s/delay/10", doraUrl), "-H", fmt.Sprintf("X-Cf-App-Instance: %s:0", appId))
					}()
				}

				reqCount := [2]int{0, 0}
				for i := 0; i < 20; i++ {
					id := helpers.Curl(Config, fmt.Sprintf("%s/id", doraUrl)).Wait().Out.Contents()
					reqCount[slices.Index(instanceIds[:], string(id))] += 1
				}

				// allow for some wiggle-room
				Expect(reqCount[0]).To(BeNumerically("<=", 8))
				Expect(reqCount[1]).To(BeNumerically(">=", 12))
				wg.Wait()
			})
		})
		Context("when it's set to hash", func() {
			Context("when the requests contain the same hash header", func() {
				It("routes requests to the same instance", func() {
					doraUrl := buildUrl(hashBasedRoutingHost)
					hashHeader := "X-Hash-Header: 1"

					reqCount := [2]int{0, 0}
					for i := 0; i < 20; i++ {
						id := helpers.Curl(Config, fmt.Sprintf("%s/id", doraUrl), "-H", hashHeader).Wait().Out.Contents()
						reqCount[slices.Index(instanceIds[:], string(id))] += 1
					}

					// All requests with the same hash should go to the same instance
					Expect(reqCount[0] == 20 || reqCount[1] == 20).To(BeTrue(), "All 20 requests should be routed to the same instance")
				})
			})
			Context("when the requests contain the different hash headers", func() {
				It("distributes requests evenly", func() {
					doraUrl := buildUrl(hashBasedRoutingHost)

					reqCount := [2]int{0, 0}
					requestsToSend := 100
					for i := 0; i < requestsToSend; i++ {
						// Generate random hash header
						uuid := make([]byte, 16)
						rand.Read(uuid)
						randomHashValue := fmt.Sprintf("%x", uuid)

						id := helpers.Curl(Config, fmt.Sprintf("%s/id", doraUrl), "-H", fmt.Sprintf("X-Hash-Header: %s", randomHashValue)).Wait().Out.Contents()
						reqCount[slices.Index(instanceIds[:], string(id))] += 1
					}

					// allow for some wiggle-room
					tolerance := 10
					Expect(reqCount[0]).To(BeNumerically(">=", (requestsToSend/2)-tolerance), "Approximately half of requests should be routed to the first instance")
					Expect(reqCount[1]).To(BeNumerically(">=", (requestsToSend/2)-tolerance), "Approximately half of requests should be routed to the second instance")
				})
			})
		})
	})
})
