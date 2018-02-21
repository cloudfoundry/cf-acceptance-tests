package handlers_test

import (
	"bytes"
	"example-apps/proxy/handlers"
	"math/rand"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("UploadHandler", func() {
	var (
		handler *handlers.UploadHandler
		resp    *httptest.ResponseRecorder
		req     *http.Request
	)

	BeforeEach(func() {
		handler = &handlers.UploadHandler{}
		resp = httptest.NewRecorder()
	})

	Describe("POST", func() {
		Context("when the request is for includes a 1000000 byte payload", func() {
			BeforeEach(func() {
				var err error
				reqBytes := make([]byte, 1000000)
				rand.Read(reqBytes)
				req, err = http.NewRequest("POST", "/upload", bytes.NewBuffer([]byte(reqBytes)))
				Expect(err).NotTo(HaveOccurred())
			})
			It("returns a body with the size of the bytes read in the request", func() {
				handler.ServeHTTP(resp, req)

				Expect(resp.Code).To(Equal(http.StatusOK))
				Expect(resp.Body.String()).To(Equal("1000000 bytes received and read"))
			})
		})

		Context("when the request body is nil", func() {
			BeforeEach(func() {
				var err error
				req, err = http.NewRequest("POST", "/upload", nil)
				Expect(err).NotTo(HaveOccurred())
			})
			It("returns an error", func() {
				handler.ServeHTTP(resp, req)

				Expect(resp.Code).To(Equal(http.StatusOK))
				Expect(resp.Body.String()).To(Equal("0 bytes received and read"))
			})
		})
	})
})
