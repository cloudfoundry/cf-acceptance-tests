package session_test

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

var _ = Describe("Session", func() {
	var (
		server *httptest.Server
	)

	BeforeEach(func() {
		server = httptest.NewServer(router.New(os.Stdout, clock.NewClock()))
		os.Setenv("CF_INSTANCE_GUID", "FAKE_INSTANCE_ID")
	})

	AfterEach(func() {
		server.Close()
		os.Unsetenv("CF_INSTANCE_GUID")
	})

	Describe("StickyHandler", func() {
		It("Sets a JSESSIONID cookie", func() {
			res, err := http.Post(fmt.Sprintf("%s/session", server.URL), "text/plain", bytes.NewBuffer([]byte{}))
			Expect(err).NotTo(HaveOccurred())
			defer res.Body.Close()

			cookies := res.Cookies()
			var sessionCookie string
			for _, c := range cookies {
				if c.Name == "JSESSIONID" {
					sessionCookie = c.Value
				}
			}
			Expect(sessionCookie).To(Equal("FAKE_INSTANCE_ID"))
		})

		It("Prints a message about sticky sessions", func() {
			res, err := http.Post(fmt.Sprintf("%s/session", server.URL), "text/plain", bytes.NewBuffer([]byte{}))
			Expect(err).NotTo(HaveOccurred())
			defer res.Body.Close()

			bodyBuf := bytes.NewBuffer([]byte{})
			bodyBuf.ReadFrom(res.Body)

			Expect(bodyBuf.String()).To(Equal("Please read the README.md for help on how to use sticky sessions."))
		})
	})
})
