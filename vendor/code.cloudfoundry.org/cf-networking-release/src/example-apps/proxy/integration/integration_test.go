package integration_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Integration", func() {
	var (
		session    *gexec.Session
		address    string
		listenPort int

		proxyDestinationServer *httptest.Server
		destinationAddress     string
	)

	var serverIsAvailable = func() error {
		return VerifyTCPConnection(address)
	}

	BeforeEach(func() {
		proxyDestinationServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintln(w, "Example Domain")
		}))
		destinationAddress = strings.Replace(proxyDestinationServer.URL, "http://", "", 1)

		listenPort = 44000 + GinkgoParallelNode()
		address = fmt.Sprintf("127.0.0.1:%d", listenPort)

		exampleAppCmd := exec.Command(exampleAppPath)
		exampleAppCmd.Env = []string{
			fmt.Sprintf("PORT=%d", listenPort),
		}
		var err error
		session, err = gexec.Start(exampleAppCmd, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())

		Eventually(serverIsAvailable, DEFAULT_TIMEOUT).Should(Succeed())
	})

	AfterEach(func() {
		session.Interrupt()
		Eventually(session, DEFAULT_TIMEOUT).Should(gexec.Exit())
		proxyDestinationServer.Close()
	})

	Describe("boring server behavior", func() {
		It("should boot and gracefully terminate", func() {
			Consistently(session).ShouldNot(gexec.Exit())

			session.Interrupt()
			Eventually(session, DEFAULT_TIMEOUT).Should(gexec.Exit())
		})
	})

	Describe("endpoints", func() {
		It("should respond to GET / with info", func() {
			response, err := http.DefaultClient.Get("http://" + address + "/")
			Expect(err).NotTo(HaveOccurred())
			defer response.Body.Close()
			Expect(response.StatusCode).To(Equal(200))

			responseBytes, err := ioutil.ReadAll(response.Body)
			Expect(err).NotTo(HaveOccurred())

			var responseData struct {
				ListenAddresses []string
				Port            int
			}

			Expect(json.Unmarshal(responseBytes, &responseData)).To(Succeed())

			Expect(responseData.ListenAddresses).To(ContainElement("127.0.0.1"))
			Expect(responseData.Port).To(Equal(listenPort))
		})

		It("should respond to /ping by pinging the provided address", func() {
			response, err := http.DefaultClient.Get("http://" + address + "/ping/" + destinationAddress)
			Expect(err).NotTo(HaveOccurred())
			defer response.Body.Close()
			Expect(response.StatusCode).To(Equal(200))

			responseBytes, err := ioutil.ReadAll(response.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(responseBytes).To(ContainSubstring("Ping succeeded"))
		})

		It("should respond to /proxy by proxying the request to the provided address", func() {
			response, err := http.DefaultClient.Get("http://" + address + "/proxy/" + destinationAddress)
			Expect(err).NotTo(HaveOccurred())
			defer response.Body.Close()
			Expect(response.StatusCode).To(Equal(200))

			responseBytes, err := ioutil.ReadAll(response.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(responseBytes).To(ContainSubstring("Example Domain"))
		})

		It("should report latency stats on /stats", func() {
			response, err := http.DefaultClient.Get("http://" + address + "/proxy/" + destinationAddress)
			Expect(err).NotTo(HaveOccurred())
			Expect(response.StatusCode).To(Equal(200))

			statsResponse, err := http.DefaultClient.Get("http://" + address + "/stats")
			Expect(err).NotTo(HaveOccurred())
			defer statsResponse.Body.Close()

			responseBytes, err := ioutil.ReadAll(statsResponse.Body)
			Expect(err).NotTo(HaveOccurred())
			var statsJSON struct {
				Latency []float64
			}
			Expect(json.Unmarshal(responseBytes, &statsJSON)).To(Succeed())
			Expect(len(statsJSON.Latency)).To(BeNumerically(">=", 1))
		})

		Context("when the proxy destination is invalid", func() {
			It("logs the error", func() {
				response, err := http.DefaultClient.Get("http://" + address + "/proxy/////!!")
				Expect(err).NotTo(HaveOccurred())
				defer response.Body.Close()
				Expect(response.StatusCode).To(Equal(500))

				Eventually(session.Err.Contents).Should(ContainSubstring("request failed: Get"))
			})
		})

		Context("when the ping destination is invalid", func() {
			It("logs the error", func() {
				response, err := http.DefaultClient.Get("http://" + address + "/ping/////!!")
				Expect(err).NotTo(HaveOccurred())
				defer response.Body.Close()
				Expect(response.StatusCode).To(Equal(500))

				Eventually(session.Err.Contents).Should(ContainSubstring("Ping failed"))
			})
		})
	})
})
