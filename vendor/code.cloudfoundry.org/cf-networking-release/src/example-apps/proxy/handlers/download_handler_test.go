package handlers_test

import (
	"example-apps/proxy/handlers"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("DownloadHandler", func() {
	var (
		handler *handlers.DownloadHandler
		resp    *httptest.ResponseRecorder
		req     *http.Request
	)
	BeforeEach(func() {
		handler = &handlers.DownloadHandler{}
		resp = httptest.NewRecorder()
	})
	Describe("GET", func() {
		Context("when the request is for 1000000 bytes", func() {
			BeforeEach(func() {
				var err error
				req, err = http.NewRequest("GET", "/download/1000000", nil)
				Expect(err).NotTo(HaveOccurred())

			})
			It("returns a body with the 1000000 bytes", func() {
				handler.ServeHTTP(resp, req)

				Expect(resp.Code).To(Equal(http.StatusOK))
				Expect(resp.Body.Len()).To(Equal(1000000))
			})
		})

		Context("when the request is for 2000000 bytes", func() {
			BeforeEach(func() {
				var err error
				req, err = http.NewRequest("GET", "/download/2000000", nil)
				Expect(err).NotTo(HaveOccurred())

			})
			It("returns a body with the 2000000 bytes", func() {
				handler.ServeHTTP(resp, req)

				Expect(resp.Code).To(Equal(http.StatusOK))
				Expect(resp.Body.Len()).To(Equal(2000000))
			})
		})

		Context("when the number of requested bytes is negative", func() {
			BeforeEach(func() {
				var err error
				req, err = http.NewRequest("GET", "/download/-42", nil)
				Expect(err).NotTo(HaveOccurred())

			})
			It("returns an error", func() {
				handler.ServeHTTP(resp, req)

				Expect(resp.Code).To(Equal(http.StatusInternalServerError))
				Expect(resp.Body.String()).To(Equal("requested number of bytes must be a positive integer, got: -42"))
			})
		})

		Context("when the request is for non-numeric bytes", func() {
			BeforeEach(func() {
				var err error
				req, err = http.NewRequest("GET", "/download/foo", nil)
				Expect(err).NotTo(HaveOccurred())

			})
			It("returns an error", func() {
				handler.ServeHTTP(resp, req)

				Expect(resp.Code).To(Equal(http.StatusInternalServerError))
				Expect(resp.Body.String()).To(Equal("requested number of bytes must be a positive integer, got: foo"))
			})
		})
	})
})
