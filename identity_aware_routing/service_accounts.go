package identity_aware_routing

import (
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-test-helpers/v2/cf"
	"github.com/cloudfoundry/cf-test-helpers/v2/workflowhelpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

var _ = IdentityAwareRoutingDescribe("Service-account route authorization", func() {
	BeforeEach(func() {
		if !Config.GetWIFConfig().RouteTests {
			Skip("wif.route_tests is false")
		}
	})
	DescribeTable("enforces certificate account identity across entitled spaces", func(boundary string) {
		settings := Config.GetWIFConfig()
		command := func(args ...string) *Session {
			s := cf.Cf(args...).Wait(Config.CfPushTimeoutDuration())
			Expect(s).To(Exit(0), "CLI %s", args[0])
			return s
		}
		api := func(method, path string, body any) map[string]any {
			args := []string{"curl", path, "-X", method}
			if body != nil {
				b, err := json.Marshal(body)
				Expect(err).NotTo(HaveOccurred())
				args = append(args, "-d", string(b))
			}
			s := command(args...)
			var result map[string]any
			Expect(json.Unmarshal(s.Out.Contents(), &result)).To(Succeed())
			Expect(result["errors"]).To(BeNil())
			return result
		}
		b := make([]byte, 8)
		_, err := rand.Read(b)
		Expect(err).NotTo(HaveOccurred())
		id := "cats-route-" + hex.EncodeToString(b)
		org := id
		domain := Config.GetIdentityAwareDomain()
		scopedDomain := id + "-scope." + domain
		one, two, wrong, unbound := id+"-one", id+"-two", id+"-wrong", id+"-unbound"
		createdOrg, createdDomain := false, false
		apps := map[string][]string{}
		DeferCleanup(func() {
			workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.CfPushTimeoutDuration(), func() {
				for _, space := range []string{"one", "two"} {
					if len(apps[space]) > 0 {
						command("target", "-o", org, "-s", space)
					}
					for _, app := range apps[space] {
						app_helpers.AppReport(app)
						command("delete", app, "-f", "-r")
					}
				}
				if createdDomain {
					command("delete-private-domain", scopedDomain, "-f")
				}
				if createdOrg {
					command("delete-org", org, "-f")
					Expect(api("GET", "/v3/organizations?names="+org, nil)["resources"]).To(BeEmpty())
				}
			})
			AddReportEntry("route reservation", id)
		})
		workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.CfPushTimeoutDuration(), func() {
			By("creating one account shared by two entitled spaces")
			command("create-org", org, "--service-account-prefix", id)
			createdOrg = true
			for _, space := range []string{"one", "two"} {
				command("create-space", space, "-o", org)
				for _, group := range strings.Split(settings.RunningSecurityGroup, ",") {
					if group != "" {
						command("bind-security-group", group, org, "--space", space, "--lifecycle", "running")
					}
				}
			}
			command("target", "-o", org, "-s", "one")
			command("create-service-account", "worker")
			command("create-service-account", "other-worker")
			command("entitle-service-account", "worker", "two")
			baseDomain := api("GET", "/v3/domains?names="+domain, nil)["resources"].([]any)
			Expect(baseDomain).To(HaveLen(1))
			Expect(baseDomain[0].(map[string]any)["enforce_route_policies"]).To(BeTrue())
			Expect(baseDomain[0].(map[string]any)["route_policies_scope"]).To(Equal("any"), "base domain must allow cross-space callers")
			command("create-private-domain", org, scopedDomain, "--enforce-route-policies", "--scope", "space")
			createdDomain = true
			dir, err := os.MkdirTemp("", "cats-route-")
			Expect(err).NotTo(HaveOccurred())
			DeferCleanup(os.RemoveAll, dir)
			binary, err := filepath.Abs("assets/wif/bin")
			Expect(err).NotTo(HaveOccurred())
			for _, item := range []struct{ name, space, account string }{{one, "one", "worker"}, {two, "two", "worker"}, {wrong, "one", "other-worker"}, {unbound, "one", ""}} {
				command("target", "-o", org, "-s", item.space)
				env := map[string]string{"WIF_MODE": "route", "TRUSTED_CA_CERT": settings.CACert, "ACCOUNT_ROUTE_URL": "https://" + id + "." + domain + "/health", "SPACE_ROUTE_URL": "https://" + scopedDomain + "/health", "SPOOFED_ACCOUNT_SAN": "worker." + id + ".svc.identity"}
				body, err := json.Marshal(map[string]any{"applications": []any{map[string]any{"name": item.name, "path": binary, "memory": DEFAULT_MEMORY_LIMIT, "buildpacks": []string{Config.GetBinaryBuildpackName()}, "command": "./wif", "env": env}}})
				Expect(err).NotTo(HaveOccurred())
				path := filepath.Join(dir, item.name+".json")
				Expect(os.WriteFile(path, body, 0600)).To(Succeed())
				command("push", "-f", path, "--no-start")
				apps[item.space] = append(apps[item.space], item.name)
				if item.account != "" {
					command("bind-service-account", item.name, item.account)
				}
				command("start", item.name)
			}
			command("target", "-o", org, "-s", "one")
			command("map-route", one, domain, "--hostname", id)
			command("map-route", one, scopedDomain)
			roots, _ := x509.SystemCertPool()
			if roots == nil {
				roots = x509.NewCertPool()
			}
			if settings.CACert != "" {
				Expect(roots.AppendCertsFromPEM([]byte(settings.CACert))).To(BeTrue())
			}
			client := &http.Client{Timeout: 30 * time.Second, Transport: &http.Transport{TLSClientConfig: &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS12}}}
			probe := func(app, target, mode string) map[string]any {
				resp, err := client.Get(fmt.Sprintf("https://%s.%s/route?target=%s&mode=%s", app, Config.GetAppsDomain(), target, mode))
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()
				Expect(resp.StatusCode).To(Equal(200))
				var result map[string]any
				Expect(json.NewDecoder(resp.Body).Decode(&result)).To(Succeed())
				return result
			}
			status := func(app, target, mode string, expected int) {
				Eventually(func() any { return probe(app, target, mode)["status"] }, Config.DefaultTimeoutDuration(), time.Second).Should(BeEquivalentTo(expected))
				AddReportEntry(app+"/"+target+"/"+mode, probe(app, target, mode))
			}
			By("denying all callers before an account-only route grant")
			status(one, "account", "normal", 403)
			command("add-route-policy", domain, "--hostname", id, "--source-service-account", "worker")
			By("allowing both entitled-space apps with no JWT")
			status(one, "account", "normal", 200)
			status(two, "account", "normal", 200)
			if boundary == "account" {
				By("rejecting wrong and unbound identities even with forged XFCC")
				for _, app := range []string{wrong, unbound} {
					status(app, "account", "normal", 403)
					status(app, "account", "spoof", 403)
				}
				noCert := probe(unbound, "account", "no-cert")
				Expect(noCert["tls_error"]).To(BeTrue())
				AddReportEntry("no-certificate", noCert)
				By("preserving certificate-only route access after disabling token issuance")
				command("disable-service-account", "worker")
				status(one, "account", "normal", 200)
				status(two, "account", "normal", 200)
				command("enable-service-account", "worker")
			} else {
				By("enforcing a space-scoped domain even for the same shared account")
				routes := api("GET", "/v3/routes?hosts=&domain_guids="+api("GET", "/v3/domains?names="+scopedDomain, nil)["resources"].([]any)[0].(map[string]any)["guid"].(string), nil)["resources"].([]any)
				Expect(routes).To(HaveLen(1))
				api("POST", "/v3/route_policies", map[string]any{"source": "cf:svc:" + id + ":worker", "relationships": map[string]any{"route": map[string]any{"data": map[string]any{"guid": routes[0].(map[string]any)["guid"]}}}})
				status(one, "space", "normal", 200)
				status(two, "space", "normal", 403)
			}
			By("removing the qualified account source and observing default denial")
			command("remove-route-policy", domain, "--hostname", id, "--source-service-account", "worker", "-f")
			status(one, "account", "normal", 403)
			status(two, "account", "normal", 403)
		})
	}, Entry("account identity and disable semantics", "account"), Entry("same-account space domain restriction", "space"))
})
