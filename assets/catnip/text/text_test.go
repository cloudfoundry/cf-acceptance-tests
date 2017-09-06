package text_test

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

var _ = Describe("Text", func() {
	var (
		server *httptest.Server
	)

	BeforeEach(func() {
		server = httptest.NewServer(router.New(os.Stdout, clock.NewClock()))
	})

	AfterEach(func() {
		server.Close()
	})
	Describe("LargeHandler", func() {
		It("returns a response of size :kbytes in kilobytes", func() {
			res, err := http.Get(fmt.Sprintf("%s/largetext/4", server.URL))
			Expect(err).NotTo(HaveOccurred())
			defer res.Body.Close()

			bodyBuf := bytes.NewBuffer([]byte{})
			_, err = bodyBuf.ReadFrom(res.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(bodyBuf.Len()).To(Equal(4096))
		})
	})
})
