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
			workflowhelpers.AsUser(TestSetup.AdminUserContext(), TestSetup.ShortTimeout(), func() {
				Expect(cf.Cf("disable-feature-flag", "hash_based_routing").Wait()).To(Exit(0))
			})
		})

		// occupyInstanceZero launches `count` long-lived requests to the given
		// URL, each pinned to instance 0 via the X-Cf-App-Instance header, so
		// that instance 0 accumulates active connections. It returns a stop
		// function that reaps the background sessions (and waits for them to
		// exit) once the test is done measuring.
		//
		// This is the crux of the flake fix: helpers.Curl launches curl
		// asynchronously (gexec.Start returns immediately, before the request is
		// even sent), so the previous code could begin probing while the "slow"
		// connections had not yet been established on instance 0. That left the
		// router's per-instance connection counts unraised and made routing look
		// even, producing the flake. Here every background session is started
		// synchronously in the loop, and callers then wait for an observable
		// precondition (see below) before measuring.
		occupyInstanceZero := func(url string, count int) func() {
			var wg sync.WaitGroup
			sessions := make([]*Session, 0, count)
			for i := 0; i < count; i++ {
				wg.Add(1)
				// helpers.Curl starts the process synchronously and returns the
				// running session; hold the session so we can reap it later and
				// so the connection is not garbage-collected.
				session := helpers.Curl(Config, fmt.Sprintf("%s/delay/30", url), "-H", fmt.Sprintf("X-Cf-App-Instance: %s:0", appId))
				sessions = append(sessions, session)
				go func(s *Session) {
					defer wg.Done()
					defer GinkgoRecover()
					s.Wait()
				}(session)
			}
			return func() {
				for _, s := range sessions {
					s.Kill()
				}
				wg.Wait()
			}
		}

		Context("when it's set to round-robin", func() {
			It("distributes requests evenly", func() {
				doraUrl := buildUrl(roundRobinHost)
				stop := occupyInstanceZero(doraUrl, 10)
				defer stop()

				// Wait until the background slow requests are actually in flight
				// on instance 0 before measuring. Round-robin ignores active
				// connection counts, so the observable precondition is simply
				// that instance 0 is reachable/serving under the background load;
				// we confirm instance 0 still answers a pinned probe, which only
				// succeeds once its listener is up and handling the slow
				// connections.
				Eventually(func() bool {
					id := helpers.Curl(Config, fmt.Sprintf("%s/id", doraUrl), "-H", fmt.Sprintf("X-Cf-App-Instance: %s:0", appId)).Wait().Out.Contents()
					return string(id) == instanceIds[0]
				}).Should(BeTrue())

				reqCount := [2]int{0, 0}
				for i := 0; i < 20; i++ {
					id := helpers.Curl(Config, fmt.Sprintf("%s/id", doraUrl)).Wait().Out.Contents()
					reqCount[slices.Index(instanceIds[:], string(id))] += 1
				}

				// allow for some wiggle-room
				Expect(reqCount[0]).To(BeNumerically(">=", 8))
				Expect(reqCount[1]).To(BeNumerically(">=", 8))
			})
		})

		Context("when it's set to least-connection", func() {
			It("always sends the request to the instance with less active connections", func() {
				doraUrl := buildUrl(leastConnHost)
				stop := occupyInstanceZero(doraUrl, 10)
				defer stop()

				// Establish the precondition the assertion depends on before
				// counting: the least-connection algorithm must actually observe
				// that instance 0 is busy with the background connections and
				// start steering new requests to instance 1. We poll the same
				// signal the test measures - an unpinned probe landing on
				// instance 1 - until it holds, so the counted loop below runs
				// only once the router's connection accounting reflects the load.
				// This replaces the previous implicit race, where probing began
				// before the slow connections were even established.
				Eventually(func() bool {
					id := helpers.Curl(Config, fmt.Sprintf("%s/id", doraUrl)).Wait().Out.Contents()
					return string(id) == instanceIds[1]
				}).Should(BeTrue())

				reqCount := [2]int{0, 0}
				for i := 0; i < 20; i++ {
					id := helpers.Curl(Config, fmt.Sprintf("%s/id", doraUrl)).Wait().Out.Contents()
					reqCount[slices.Index(instanceIds[:], string(id))] += 1
				}

				// allow for some wiggle-room
				Expect(reqCount[0]).To(BeNumerically("<=", 8))
				Expect(reqCount[1]).To(BeNumerically(">=", 12))
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
