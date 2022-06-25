package download_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/cloudfoundry/cf-acceptance-tests/helpers/download"
)

var _ = Describe("CheckRedirect", func() {
	Context("Response is a redirect", func() {
		const moreHeaderInfoIncludingLocation = "< Content-Type: text/html; charset=utf-8\r\n" +
			"< Date: Fri, 25 Jan 2019 14:51:35 GMT\r\n" +
			"< Location: https://example.com\r\n" +
			"< Referrer-Policy: strict-origin-when-cross-origin\r\n" +
			"< Server: nginx"

		Context("HTTP version 1.1", func() {
			It("returns true and the redirect location ", func() {
				isRedirect, location, err := download.CheckRedirect("< HTTP/1.1 302 Found\r\n" + moreHeaderInfoIncludingLocation)

				Expect(err).NotTo(HaveOccurred())
				Expect(isRedirect).To(BeTrue())
				Expect(location).To(Equal("https://example.com"))
			})

		})

		Context("HTTP version 2", func() {
			It("returns true and the redirect location ", func() {
				isRedirect, location, err := download.CheckRedirect("< HTTP/2 302 Found\r\n" + moreHeaderInfoIncludingLocation)

				Expect(err).NotTo(HaveOccurred())
				Expect(isRedirect).To(BeTrue())
				Expect(location).To(Equal("https://example.com"))
			})
		})

		Context("Location is missing", func() {
			It("returns an error", func() {
				_, _, err := download.CheckRedirect("< HTTP/1.1 302 Found\r\n" +
					"< Content-Type: text/html; charset=utf-8\r\n" +
					"< Date: Fri, 25 Jan 2019 14:51:35 GMT\r\n" +
					"< Referrer-Policy: strict-origin-when-cross-origin\r\n" +
					"< Server: nginx")

				Expect(err).To(HaveOccurred())
			})
		})
	})

	Context("Response is not a redirect", func() {
		const moreHeaderInfo = "< Content-Type: text/html; charset=utf-8\r\n" +
			"< Date: Fri, 25 Jan 2019 14:51:35 GMT\r\n" +
			"< Server: nginx"

		Context("HTTP version 1.1", func() {
			It("returns false and empty redirect location ", func() {
				isRedirect, location, err := download.CheckRedirect("< HTTP/1.1 200 OK\r\n" + moreHeaderInfo)

				Expect(err).NotTo(HaveOccurred())
				Expect(isRedirect).To(BeFalse())
				Expect(location).To(BeEmpty())
			})
		})

		Context("HTTP version 2", func() {
			It("returns false and empty redirect location ", func() {
				isRedirect, location, err := download.CheckRedirect("< HTTP/2 200 OK\r\n" + moreHeaderInfo)

				Expect(err).NotTo(HaveOccurred())
				Expect(isRedirect).To(BeFalse())
				Expect(location).To(BeEmpty())
			})

			Context("HTTP line does not contain the textual form of the status code", func() {
				It("returns false and empty redirect location ", func() {
					isRedirect, location, err := download.CheckRedirect("< HTTP/2 502\r\n" + moreHeaderInfo)

					Expect(err).NotTo(HaveOccurred())
					Expect(isRedirect).To(BeFalse())
					Expect(location).To(BeEmpty())
				})
			})
		})
	})
})
