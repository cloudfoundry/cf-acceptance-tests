package env_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"

	"code.cloudfoundry.org/clock"

	"github.com/cloudfoundry/cf-acceptance-tests/assets/catnip/router"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Env", func() {
	var (
		server *httptest.Server
	)

	BeforeEach(func() {
		server = httptest.NewServer(router.New(os.Stdout, clock.NewClock()))

		os.Setenv("CATNIP_ENVTEST", "Jellybean")
		os.Setenv("CF_INSTANCE_GUID", "FAKE_INSTANCE_ID")
	})

	AfterEach(func() {
		server.Close()

		os.Unsetenv("CATNIP_ENVTEST")
		os.Unsetenv("CF_INSTANCE_GUID")
	})

	Describe("NameHandler", func() {
		It("retrieves environment entry by name", func() {
			res, err := http.Get(fmt.Sprintf("%s/env/CATNIP_ENVTEST", server.URL))
			Expect(err).NotTo(HaveOccurred())
			defer res.Body.Close()

			bodyBuf := bytes.NewBuffer([]byte{})
			_, err = bodyBuf.ReadFrom(res.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(bodyBuf.String()).To(Equal("Jellybean"))
		})
	})

	Describe("JsonHandler", func() {
		It("returns JSON for the set environment", func() {
			res, err := http.Get(fmt.Sprintf("%s/env.json", server.URL))
			Expect(err).NotTo(HaveOccurred())
			defer res.Body.Close()

			bodyBuf := bytes.NewBuffer([]byte{})
			_, err = bodyBuf.ReadFrom(res.Body)
			Expect(err).NotTo(HaveOccurred())
			envMap := make(map[string]string)

			err = json.Unmarshal(bodyBuf.Bytes(), &envMap)
			Expect(err).NotTo(HaveOccurred())

			Expect(envMap["CATNIP_ENVTEST"]).To(Equal("Jellybean"))
		})
	})

	Describe("InstanceIdHandler", func() {
		It("returns the instance id", func() {
			res, err := http.Get(fmt.Sprintf("%s/id", server.URL))
			Expect(err).NotTo(HaveOccurred())
			defer res.Body.Close()

			bodyBuf := bytes.NewBuffer([]byte{})
			_, err = bodyBuf.ReadFrom(res.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(bodyBuf.String()).To(Equal("FAKE_INSTANCE_ID"))
		})
	})
})
