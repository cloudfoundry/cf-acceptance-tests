package health_test

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"

	"code.cloudfoundry.org/clock"
	"github.com/cloudfoundry/cf-acceptance-tests/assets/catnip/router"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Health", func() {
	var (
		server *httptest.Server
	)

	BeforeEach(func() {
		server = httptest.NewServer(router.New(os.Stdout, clock.NewClock()))
	})

	AfterEach(func() {
		server.Close()
	})
	Describe("HealthHandler", func() {
		It("returns a 500 and prints the number of times it's been hit three times, then gets healthy", func() {
			callAndValidateHealth(server.URL, http.StatusInternalServerError, "Hit /health 1 times")
			callAndValidateHealth(server.URL, http.StatusInternalServerError, "Hit /health 2 times")
			callAndValidateHealth(server.URL, http.StatusInternalServerError, "Hit /health 3 times")

			callAndValidateHealth(server.URL, http.StatusOK, "I'm alive")
		})
	})
})

func callAndValidateHealth(serverUrl string, statusCode int, responseBody string) {
	res, err := http.Get(fmt.Sprintf("%s/health", serverUrl))
	Expect(err).NotTo(HaveOccurred())

	Expect(res.StatusCode).To(Equal(statusCode))
	bodyBuf := bytes.NewBuffer([]byte{})
	_, err = bodyBuf.ReadFrom(res.Body)
	defer res.Body.Close()
	Expect(err).NotTo(HaveOccurred())

	Expect(bodyBuf.String()).To(Equal(responseBody))
}
