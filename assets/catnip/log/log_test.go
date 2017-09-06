package log_test

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	"code.cloudfoundry.org/clock/fakeclock"
	"github.com/cloudfoundry/cf-acceptance-tests/assets/catnip/router"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Log", func() {
	var (
		fakeClock *fakeclock.FakeClock
		logBuf    *bytes.Buffer

		server *httptest.Server
	)

	BeforeEach(func() {
		fakeClock = fakeclock.NewFakeClock(time.Now())
		logBuf = bytes.NewBuffer([]byte{})

		server = httptest.NewServer(router.New(logBuf, fakeClock))
	})

	AfterEach(func() {
		server.Close()
	})

	Describe("SpewHandler", func() {
		It("spews the given amount of kb to the log", func() {
			res, err := http.Get(fmt.Sprintf("%s/logspew/4", server.URL))
			Expect(err).NotTo(HaveOccurred())
			defer res.Body.Close()

			Expect(logBuf.Len()).To(Equal(4100))
		})

		It("Returns how many kb is spewed", func() {
			res, err := http.Get(fmt.Sprintf("%s/logspew/4", server.URL))
			Expect(err).NotTo(HaveOccurred())
			defer res.Body.Close()

			bodyBuf := bytes.NewBuffer([]byte{})
			bodyBuf.ReadFrom(res.Body)

			Expect(bodyBuf.String()).To(Equal("Just wrote 4 kbytes to the log"))
		})
	})

	Describe("SleepHandler", func() {
		It("writes log output periodically", func() {
			res, err := http.Get(fmt.Sprintf("%s/log/sleep/4", server.URL))
			Expect(err).NotTo(HaveOccurred())
			defer res.Body.Close()

			Eventually(logBuf.String).Should(ContainSubstring("Muahaha... let's go. Waiting 0.000004 seconds between loglines. Logging 'Muahaha...' every time."))
			fakeClock.Increment(4 * time.Microsecond)
			Eventually(logBuf.String).Should(ContainSubstring("Muahaha...1"))
			fakeClock.Increment(4 * time.Microsecond)
			Eventually(logBuf.String).Should(ContainSubstring("Muahaha...2"))
		})
	})
})
