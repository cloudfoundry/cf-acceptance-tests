package apps

import (
	"context"
	"fmt"

	"code.cloudfoundry.org/go-loggregator/v10"
	"code.cloudfoundry.org/go-loggregator/v10/rpc/loggregator_v2"
	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"

	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	logshelper "github.com/cloudfoundry/cf-acceptance-tests/helpers/logs"
	. "github.com/cloudfoundry/cf-acceptance-tests/helpers/matchers"
	. "github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	"github.com/cloudfoundry/cf-test-helpers/v2/cf"
	"github.com/cloudfoundry/cf-test-helpers/v2/helpers"
	"github.com/cloudfoundry/cf-test-helpers/v2/workflowhelpers"
	"github.com/cloudfoundry/noaa/v2/consumer"

	"crypto/tls"
	"strings"

	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

var _ = AppsDescribe("loggregator", func() {
	var appName string
	const oneSecond = 1000000 // this app uses millionth of seconds

	BeforeEach(func() {
		appName = CATSRandomName("APP")

		Expect(cf.Cf("push",
			appName,
			"-b", Config.GetRubyBuildpackName(),
			"-m", DEFAULT_MEMORY_LIMIT,
			"-p", assets.NewAssets().LoggregatorLoadGenerator,
			"-i", "2",
		).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
	})

	AfterEach(func() {
		app_helpers.AppReport(appName)

		Expect(cf.Cf("delete", appName, "-f", "-r").Wait()).To(Exit(0))
	})

	Context("cf logs", func() {
		var logs *Session

		BeforeEach(func() {
			logs = logshelper.Follow(appName)
		})

		AfterEach(func() {
			// logs might be nil if the BeforeEach panics
			if logs != nil {
				logs.Interrupt()
			}
		})

		It("exercises basic loggregator behavior", func() {
			Eventually(func() string {
				return helpers.CurlApp(Config, appName, fmt.Sprintf("/log/sleep/%d", oneSecond))
			}).Should(ContainSubstring("Muahaha"))

			Eventually(logs, Config.DefaultTimeoutDuration()*2).Should(Say("Muahaha"))
		})
	})

	Context("cf logs --recent", func() {
		It("makes loggregator buffer and dump log messages", func() {
			Eventually(func() string {
				return helpers.CurlApp(Config, appName, fmt.Sprintf("/log/sleep/%d", oneSecond))
			}).Should(ContainSubstring("Muahaha"))

			Eventually(func() *Session {
				appLogsSession := logshelper.Recent(appName)
				Expect(appLogsSession.Wait()).To(Exit(0))
				return appLogsSession
			}, Config.DefaultTimeoutDuration()*2).Should(Say("Muahaha"))
		})
	})

	Context("firehose data", func() {
		It("shows logs and metrics", func() {
			c := consumer.New(getDopplerEndpoint(), &tls.Config{InsecureSkipVerify: Config.GetSkipSSLValidation()}, nil)
			msgChan, errChan := c.Firehose(CATSRandomName("SUBSCRIPTION-ID"), getAdminUserAccessToken())
			defer c.Close()

			Eventually(func() string {
				return helpers.CurlApp(Config, appName, fmt.Sprintf("/log/sleep/%d", oneSecond))
			}).Should(ContainSubstring("Muahaha"))

			if len(errChan) == 1 {
				Expect(<-errChan).ToNot(HaveOccurred(), "Failed to establish firehose websocket connection. On AWS you may need to reconfigure the doppler ports.")
			}
			Eventually(msgChan, Config.DefaultTimeoutDuration(), time.Millisecond).Should(Receive(EnvelopeContainingMessageLike("Muahaha")), "To enable the logging & metrics firehose feature, please ask your CF administrator to add the 'doppler.firehose' scope to your CF admin user.")
		})
	})

	Context("reverse log proxy", func() {
		It("streams logs", func() {
			rlpClient := loggregator.NewRLPGatewayClient(
				getLogStreamEndpoint(),
				loggregator.WithRLPGatewayHTTPClient(newAuthClient()),
			)

			ebr := &loggregator_v2.EgressBatchRequest{
				ShardId: CATSRandomName("SUBSCRIPTION-ID"),
				Selectors: []*loggregator_v2.Selector{
					{
						SourceId: app_helpers.GetAppGuid(appName),
						Message:  &loggregator_v2.Selector_Log{Log: &loggregator_v2.LogSelector{}},
					},
				},
			}

			ctx, cancelFunc := context.WithTimeout(context.Background(), Config.DefaultTimeoutDuration())
			defer cancelFunc()

			es := rlpClient.Stream(ctx, ebr)

			Eventually(func() string {
				return helpers.CurlApp(Config, appName, fmt.Sprintf("/log/sleep/%d", oneSecond))
			}).Should(ContainSubstring("Muahaha"))

			Eventually(func() string {
				envelopes := es()
				var messages []string
				for _, e := range envelopes {
					log, ok := e.Message.(*loggregator_v2.Envelope_Log)
					Expect(ok).To(BeTrue())
					messages = append(messages, string(log.Log.Payload))
				}
				return strings.Join(messages, "")
			}, Config.DefaultTimeoutDuration(), time.Millisecond).Should(ContainSubstring("Muahaha"), "To enable the log-stream feature, please ask your CF administrator to enable the RLP Gateway and to add the 'doppler.firehose' scope to your CF admin user.")
		})
	})
})

type cfHomeConfig struct {
	Target          string
	AccessToken     string
	DopplerEndPoint string
}

func getCfHomeConfig() *cfHomeConfig {
	myCfHomeConfig := &cfHomeConfig{}

	workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
		path := filepath.Join(os.Getenv("CF_HOME"), ".cf", "config.json")

		configFile, err := os.Open(path)
		if err != nil {
			panic(err)
		}

		decoder := json.NewDecoder(configFile)
		err = decoder.Decode(myCfHomeConfig)
		if err != nil {
			panic(err)
		}
	})

	return myCfHomeConfig
}

func getAdminUserAccessToken() string {
	return getCfHomeConfig().AccessToken
}

func getDopplerEndpoint() string {
	return getCfHomeConfig().DopplerEndPoint
}

func getLogStreamEndpoint() string {
	return strings.Replace(getCfHomeConfig().Target, "api", "log-stream", 1)
}

type authClient struct {
	c *http.Client
}

func newAuthClient() *authClient {
	return &authClient{
		c: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: Config.GetSkipSSLValidation()},
			},
		},
	}
}

func (a *authClient) Do(r *http.Request) (*http.Response, error) {
	r.Header.Set("Authorization", getAdminUserAccessToken())
	return a.c.Do(r)
}
