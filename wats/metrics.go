package wats

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry/noaa"
	"github.com/cloudfoundry/noaa/events"

	. "github.com/cloudfoundry-incubator/cf-test-helpers/workflowhelpers"
	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

var _ = WindowsDescribe("Metrics", func() {
	It("garden-windows emits metrics to the firehose", func() {
		duration, _ := time.ParseDuration("5s")
		AsUser(TestSetup.AdminUserContext(), duration, func() {
			authToken := getOauthToken()
			msgChan, errorChan, stopChan := createNoaaClient(dopplerUrl(), authToken)
			defer close(stopChan)

			Consistently(errorChan).ShouldNot(Receive())

			sipTheStream := func() string {
				if envelope, ok := <-msgChan; ok {
					return *envelope.Origin
				}
				return ""
			}
			Eventually(sipTheStream, "1m", "5ms").Should(Equal("garden-windows"))
		})
	})
})

func dopplerUrl() string {
	doppler := os.Getenv("DOPPLER_URL")
	if doppler == "" {
		curl := cf.Cf("curl", "/v2/info")
		Expect(curl.Wait(Config.DefaultTimeoutDuration())).To(Exit(0))

		var cfInfo struct {
			DopplerLoggingEndpoint string `json:"doppler_logging_endpoint"`
		}

		err := json.NewDecoder(bytes.NewReader(curl.Out.Contents())).Decode(&cfInfo)
		Expect(err).NotTo(HaveOccurred())

		doppler = cfInfo.DopplerLoggingEndpoint
	}
	return doppler
}

func getOauthToken() string {
	session := cf.Cf("oauth-token")
	session.Wait()
	out := string(session.Out.Contents())
	authToken := strings.Split(out, "\n")[0]
	Expect(authToken).To(HavePrefix("bearer"))
	return authToken
}

func createNoaaClient(dopplerUrl, authToken string) (<-chan *events.Envelope, <-chan error, chan struct{}) {
	connection := noaa.NewConsumer(dopplerUrl, &tls.Config{InsecureSkipVerify: true}, nil)

	msgChan := make(chan *events.Envelope, 100000)
	errorChan := make(chan error)
	stopChan := make(chan struct{})

	go connection.Firehose("firehose-a", authToken, msgChan, errorChan, stopChan)

	go func() {
		for err := range errorChan {
			fmt.Fprintf(os.Stderr, "%v\n", err.Error())
		}
	}()

	return msgChan, errorChan, stopChan
}
