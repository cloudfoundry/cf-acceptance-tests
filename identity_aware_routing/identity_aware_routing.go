package identity_aware_routing

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry/cf-test-helpers/v2/cf"
	"github.com/cloudfoundry/cf-test-helpers/v2/helpers"
	"github.com/cloudfoundry/cf-test-helpers/v2/workflowhelpers"
)

type mtlsProxyResponse struct {
	Status     string            `json:"status"`
	StatusCode int               `json:"status_code"`
	Body       string            `json:"body"`
	Headers    map[string]string `json:"headers"`
	Error      string            `json:"error"`
}

var _ = IdentityAwareRoutingDescribe("Identity-Aware Routing", func() {
	var appNameFrontend string
	var appNameBackend string
	var appNameUnauthorized string
	var backendHostName string
	var identityAwareDomain string

	BeforeEach(func() {
		identityAwareDomain = Config.GetIdentityAwareDomain()

		backendHostName = random_name.CATSRandomName("HOST")
		appNameFrontend = random_name.CATSRandomName("APP-FRONT")
		appNameBackend = random_name.CATSRandomName("APP-BACK")
		appNameUnauthorized = random_name.CATSRandomName("APP-UNAUTH")

		// push backend app (proxy app so it has /headers endpoint)
		Expect(cf.Cf(
			"push", appNameBackend,
			"-b", Config.GetGoBuildpackName(),
			"-m", DEFAULT_MEMORY_LIMIT,
			"-p", assets.NewAssets().Proxy,
			"-f", assets.NewAssets().Proxy+"/manifest.yml",
		).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

		// map identity-aware route to backend app
		Expect(cf.Cf("map-route", appNameBackend, identityAwareDomain, "--hostname", backendHostName).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

		// push frontend app (proxy app with /mtls_proxy endpoint)
		Expect(cf.Cf(
			"push", appNameFrontend,
			"-b", Config.GetGoBuildpackName(),
			"-m", DEFAULT_MEMORY_LIMIT,
			"-p", assets.NewAssets().Proxy,
			"-f", assets.NewAssets().Proxy+"/manifest.yml",
		).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

		// push unauthorized app (same proxy app, different identity)
		Expect(cf.Cf(
			"push", appNameUnauthorized,
			"-b", Config.GetGoBuildpackName(),
			"-m", DEFAULT_MEMORY_LIMIT,
			"-p", assets.NewAssets().Proxy,
			"-f", assets.NewAssets().Proxy+"/manifest.yml",
		).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
	})

	AfterEach(func() {
		app_helpers.AppReport(appNameFrontend)
		app_helpers.AppReport(appNameBackend)
		app_helpers.AppReport(appNameUnauthorized)

		Expect(cf.Cf("delete", appNameFrontend, "-f", "-r").Wait()).To(Exit(0))
		Expect(cf.Cf("delete", appNameBackend, "-f", "-r").Wait()).To(Exit(0))
		Expect(cf.Cf("delete", appNameUnauthorized, "-f", "-r").Wait()).To(Exit(0))
	})

	mtlsProxyURL := func(appName, backendHost, domain, path string) string {
		return fmt.Sprintf("%s%s.%s/mtls_proxy/%s.%s/%s",
			Config.Protocol(), appName, Config.GetAppsDomain(),
			backendHost, domain, path)
	}

	curlMtlsProxy := func(appName, backendHost, domain, path string) mtlsProxyResponse {
		curlArgs := mtlsProxyURL(appName, backendHost, domain, path)
		curl := helpers.Curl(Config, curlArgs).Wait()
		var resp mtlsProxyResponse
		err := json.Unmarshal(curl.Out.Contents(), &resp)
		ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Failed to parse mtls_proxy response: %s", string(curl.Out.Contents()))
		return resp
	}

	Describe("mTLS authorization with route policies", func() {
		It("denies access by default and allows after adding a route policy", func() {
			By("verifying the frontend is denied without route policies (default deny)")
			Eventually(func() int {
				resp := curlMtlsProxy(appNameFrontend, backendHostName, identityAwareDomain, "headers")
				return resp.StatusCode
			}, 2*time.Minute).Should(Equal(403))

			By("creating a route policy for the frontend app")
			Expect(cf.Cf(
				"add-route-policy", identityAwareDomain,
				"--source-app", appNameFrontend,
				"--hostname", backendHostName,
			).Wait(Config.DefaultTimeoutDuration())).To(Exit(0))

			By("verifying the route policy is listed")
			routePoliciesOutput := cf.Cf("route-policies", "--domain", identityAwareDomain).Wait(Config.DefaultTimeoutDuration())
			Expect(routePoliciesOutput).To(Exit(0))
			Expect(string(routePoliciesOutput.Out.Contents())).To(ContainSubstring(appNameFrontend))

			By("verifying the frontend can now reach the backend")
			Eventually(func() int {
				resp := curlMtlsProxy(appNameFrontend, backendHostName, identityAwareDomain, "headers")
				return resp.StatusCode
			}, 2*time.Minute).Should(Equal(200))
		})

		It("denies access from an unauthorized app even with a valid certificate", func() {
			By("creating a route policy only for the frontend app")
			Expect(cf.Cf(
				"add-route-policy", identityAwareDomain,
				"--source-app", appNameFrontend,
				"--hostname", backendHostName,
			).Wait(Config.DefaultTimeoutDuration())).To(Exit(0))

			By("verifying the authorized frontend can reach the backend")
			Eventually(func() int {
				resp := curlMtlsProxy(appNameFrontend, backendHostName, identityAwareDomain, "headers")
				return resp.StatusCode
			}, 2*time.Minute).Should(Equal(200))

			By("verifying the unauthorized app is denied")
			Consistently(func() int {
				resp := curlMtlsProxy(appNameUnauthorized, backendHostName, identityAwareDomain, "headers")
				return resp.StatusCode
			}, 30*time.Second).Should(Equal(403))
		})

		It("forwards X-Forwarded-Client-Cert header with caller identity in Envoy format", func() {
			frontendGuid := GuidForAppName(appNameFrontend)

			By("creating a route policy for the frontend app")
			Expect(cf.Cf(
				"add-route-policy", identityAwareDomain,
				"--source-app", appNameFrontend,
				"--hostname", backendHostName,
			).Wait(Config.DefaultTimeoutDuration())).To(Exit(0))

			By("calling the backend and examining the XFCC header")
			var xfcc string
			Eventually(func() string {
				resp := curlMtlsProxy(appNameFrontend, backendHostName, identityAwareDomain, "headers")
				if resp.StatusCode != 200 {
					return ""
				}
				// The backend returns its request headers as JSON via /headers
				var headers map[string]string
				err := json.Unmarshal([]byte(resp.Body), &headers)
				if err != nil {
					return ""
				}
				xfcc = headers["X-Forwarded-Client-Cert"]
				return xfcc
			}, 2*time.Minute).ShouldNot(BeEmpty())

			By("verifying the XFCC header is in Envoy format")
			Expect(xfcc).To(ContainSubstring("Hash="))
			Expect(xfcc).To(ContainSubstring("Subject="))

			By("verifying the XFCC header contains the frontend app GUID")
			Expect(strings.ToLower(xfcc)).To(ContainSubstring("ou=app:" + strings.ToLower(frontendGuid)))
		})
	})
})

var _ = IdentityAwareRoutingDescribe("Route Policy Domain Management", func() {
	var domainName string

	BeforeEach(func() {
		domainName = random_name.CATSRandomName("DOMAIN") + "." + Config.GetAppsDomain()
	})

	AfterEach(func() {
		workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
			cf.Cf("target", "-o", TestSetup.GetOrganizationName()).Wait(Config.DefaultTimeoutDuration())
			cf.Cf("delete-shared-domain", domainName, "-f").Wait()
		})
	})

	It("creates a shared domain with route policy enforcement and verifies it in cf domains", func() {
		By("creating the domain with --enforce-route-policies and --scope space")
		workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
			createSession := cf.Cf(
				"create-shared-domain", domainName,
				"--enforce-route-policies",
				"--scope", "space",
			)
			Expect(createSession.Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
			Expect(string(createSession.Out.Contents())).To(ContainSubstring("identity-aware"))
		})

		By("verifying the domain appears in cf domains")
		domainsOutput := cf.Cf("domains").Wait(Config.DefaultTimeoutDuration())
		Expect(domainsOutput).To(Exit(0))

		var domainLine string
		for _, line := range strings.Split(string(domainsOutput.Out.Contents()), "\n") {
			if strings.Contains(line, domainName) {
				domainLine = line
				break
			}
		}
		Expect(domainLine).NotTo(BeEmpty(), "domain %s not found in cf domains output:\n%s", domainName, string(domainsOutput.Out.Contents()))
	})
})
