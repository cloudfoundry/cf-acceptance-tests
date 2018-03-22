package handlers_test

import (
	"example-apps/proxy/handlers"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("StatsHandler", func() {
	var (
		handler *handlers.StatsHandler
		resp    *httptest.ResponseRecorder
		req     *http.Request
		stats   *handlers.Stats
	)
	BeforeEach(func() {
		stats = &handlers.Stats{}
		stats.Latency = []float64{1, 2, 3}
		handler = &handlers.StatsHandler{
			Stats: stats,
		}

		resp = httptest.NewRecorder()
	})
	Describe("GET", func() {
		BeforeEach(func() {
			var err error
			req, err = http.NewRequest("GET", "/stats", nil)
			Expect(err).NotTo(HaveOccurred())
		})
		It("returns the latency of the requests", func() {
			handler.ServeHTTP(resp, req)

			Expect(resp.Code).To(Equal(http.StatusOK))
			Expect(resp.Body.String()).To(MatchJSON(`{"latency" : [1.0,2.0,3.0]}`))
		})
	})
	Describe("DELETE", func() {
		BeforeEach(func() {
			var err error
			req, err = http.NewRequest("DELETE", "/stats", nil)
			Expect(err).NotTo(HaveOccurred())
		})
		It("clears all statistics", func() {
			handler.ServeHTTP(resp, req)
			Expect(resp.Code).To(Equal(http.StatusOK))

			req, err := http.NewRequest("GET", "/stats", nil)
			Expect(err).NotTo(HaveOccurred())
			handler.ServeHTTP(resp, req)
			Expect(resp.Code).To(Equal(http.StatusOK))
			Expect(resp.Body.String()).To(MatchJSON(`{"latency" : []}`))
		})
	})
})
