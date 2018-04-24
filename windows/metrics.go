package windows

import (
	"crypto/tls"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry-incubator/cf-test-helpers/workflowhelpers"
	"github.com/cloudfoundry/noaa"
	"github.com/cloudfoundry/noaa/events"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	. "github.com/cloudfoundry/cf-acceptance-tests/helpers/matchers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = WindowsDescribe("Metrics", func() {
	var appName string

	BeforeEach(func() {
		appName = random_name.CATSRandomName("APP")

		Expect(cf.Cf("push",
			appName,
			"--no-start",
			"-s", Config.GetWindowsStack(),
			"-b", Config.GetHwcBuildpackName(),
			"-m", DEFAULT_MEMORY_LIMIT,
			"-p", assets.NewAssets().Nora,
			"-i", "2",
			"-d", Config.GetAppsDomain()).Wait(Config.DefaultTimeoutDuration())).To(gexec.Exit(0))
		app_helpers.SetBackend(appName)
		Expect(cf.Cf("start", appName).Wait(Config.CfPushTimeoutDuration())).To(gexec.Exit(0))
	})

	AfterEach(func() {
		app_helpers.AppReport(appName, Config.DefaultTimeoutDuration())

		Expect(cf.Cf("delete", appName, "-f", "-r").Wait(Config.DefaultTimeoutDuration())).To(gexec.Exit(0))
	})

	It("shows logs and metrics", func() {
		noaaConnection := noaa.NewConsumer(getDopplerEndpoint(), &tls.Config{InsecureSkipVerify: Config.GetSkipSSLValidation()}, nil)
		msgChan := make(chan *events.Envelope, 100000)
		errorChan := make(chan error)
		stopchan := make(chan struct{})

		go noaaConnection.Firehose(random_name.CATSRandomName("SUBSCRIPTION-ID"), getAdminUserAccessToken(), msgChan, errorChan, stopchan)
		defer close(stopchan)

		helpers.CurlApp(Config, appName, "/print/Muahaha")
		Eventually(msgChan, Config.DefaultTimeoutDuration()).Should(Receive(EnvelopeContainingMessageLike("Muahaha")), "To enable the logging & metrics firehose feature, please ask your CF administrator to add the 'doppler.firehose' scope to your CF admin user.")
	})

	It("shows container metrics", func() {
		appGuid := strings.TrimSpace(string(cf.Cf("app", appName, "--guid").Wait(Config.DefaultTimeoutDuration()).Out.Contents()))

		noaaConnection := noaa.NewConsumer(getDopplerEndpoint(), &tls.Config{InsecureSkipVerify: Config.GetSkipSSLValidation()}, nil)
		msgChan := make(chan *events.Envelope, 100000)
		errorChan := make(chan error)
		stopchan := make(chan struct{})
		go noaaConnection.Firehose(random_name.CATSRandomName("SUBSCRIPTION-ID"), getAdminUserAccessToken(), msgChan, errorChan, stopchan)
		defer close(stopchan)

		containerMetrics := make([]*events.ContainerMetric, 2)
		Eventually(func() bool {
			for {
				select {
				case msg := <-msgChan:
					if cm := msg.GetContainerMetric(); cm != nil {
						if cm.GetApplicationId() == appGuid {
							containerMetrics[cm.GetInstanceIndex()] = cm

							if containerMetrics[0] != nil && containerMetrics[1] != nil {
								return true
							}
						}
					}
				case e := <-errorChan:
					Expect(e).ToNot(HaveOccurred())
				default:
					return false
				}
			}
		}, 2*Config.DefaultTimeoutDuration()).Should(BeTrue())

		for _, cm := range containerMetrics {
			Expect(cm.GetMemoryBytes()).ToNot(BeZero())
			Expect(cm.GetDiskBytes()).ToNot(BeZero())
		}
	})
})

type cfHomeConfig struct {
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
