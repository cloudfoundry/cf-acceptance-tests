package service_accounts

import (
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
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

// One independent journey owns its two orgs; no global UAA configuration is mutated.
var _ = Describe("[service_accounts] cross-org synthetic WIF", Label("service-accounts", "wif"), func() {
	BeforeEach(func() {
		if !Config.GetWIFConfig().Enabled {
			Skip("wif.enabled is false")
		}
	})
	It("exchanges only the trusted org-qualified identity for cross-org resource access", func() {
		settings := Config.GetWIFConfig()
		b := make([]byte, 8)
		_, err := rand.Read(b)
		Expect(err).NotTo(HaveOccurred())
		id := "cats-wif-" + hex.EncodeToString(b)
		workOrg, stsOrg := id+"-work", id+"-sts"
		prefix := id
		account, wrongAccount := "worker", "other-worker"
		stsApp, app, secondApp, wrongApp := id+"-sts", id+"-one", id+"-two", id+"-wrong"
		orgs := []string{}
		apps := map[string][]string{}
		command := func(args ...string) *Session {
			s := cf.Cf(args...).Wait(Config.CfPushTimeoutDuration())
			Expect(s).To(Exit(0), "CLI command %s", args[0])
			return s
		}
		api := func(path string) map[string]any {
			s := command("curl", path)
			var value map[string]any
			Expect(json.Unmarshal(s.Out.Contents(), &value)).To(Succeed())
			Expect(value["errors"]).To(BeNil())
			return value
		}
		DeferCleanup(func() {
			workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.CfPushTimeoutDuration(), func() {
				for i := len(orgs) - 1; i >= 0; i-- {
					org := orgs[i]
					command("target", "-o", org, "-s", "test")
					for _, name := range apps[org] {
						app_helpers.AppReport(name)
						command("delete", name, "-f", "-r")
					}
					command("delete-org", org, "-f")
					s := api("/v3/organizations?names=" + org)
					Expect(s["resources"]).To(BeEmpty())
				}
			})
			AddReportEntry("WIF reservations", map[string]any{"retained_prefix": prefix, "deleted_account_names": []string{account, wrongAccount}})
		})
		workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.CfPushTimeoutDuration(), func() {
			By("creating a prefixed workload org and an unprefixed federation org")
			command("create-org", workOrg, "--service-account-prefix", prefix)
			orgs = append(orgs, workOrg)
			command("create-space", "test", "-o", workOrg)
			command("create-org", stsOrg)
			orgs = append(orgs, stsOrg)
			command("create-space", "test", "-o", stsOrg)
			for _, org := range orgs {
				for _, group := range strings.Split(settings.RunningSecurityGroup, ",") {
					if group != "" {
						command("bind-security-group", group, org, "--space", "test", "--lifecycle", "running")
					}
				}
			}
			command("target", "-o", stsOrg, "-s", "test")
			orgResource := api("/v3/organizations?names=" + stsOrg)["resources"].([]any)[0].(map[string]any)
			Expect(orgResource["service_account_prefix"]).To(BeNil())
			command("target", "-o", workOrg, "-s", "test")
			command("create-service-account", account)
			command("create-service-account", wrongAccount)
			orgGUID := api("/v3/organizations?names=" + workOrg)["resources"].([]any)[0].(map[string]any)["guid"].(string)
			resource := api("/v3/service_accounts?organization_guids=" + orgGUID + "&names=" + account)["resources"].([]any)[0].(map[string]any)
			subject := resource["client_id"].(string)
			Expect(subject).To(Equal("cf:service-account:" + prefix + ":" + account))
			dir, err := os.MkdirTemp("", "cats-wif-")
			Expect(err).NotTo(HaveOccurred())
			DeferCleanup(os.RemoveAll, dir)
			binary, err := filepath.Abs("assets/wif/bin")
			Expect(err).NotTo(HaveOccurred())
			push := func(org, name string, env map[string]string) {
				command("target", "-o", org, "-s", "test")
				manifest := map[string]any{"applications": []any{map[string]any{"name": name, "path": binary, "memory": DEFAULT_MEMORY_LIMIT, "disk_quota": "128M", "buildpacks": []string{Config.GetBinaryBuildpackName()}, "command": "./wif", "env": env}}}
				body, err := json.Marshal(manifest)
				Expect(err).NotTo(HaveOccurred())
				path := filepath.Join(dir, name+".json")
				Expect(os.WriteFile(path, body, 0600)).To(Succeed())
				command("push", "-f", path, "--no-start")
				apps[org] = append(apps[org], name)
			}
			stsURL := "https://" + stsApp + "." + Config.GetAppsDomain()
			By("pushing the STS in the unprefixed org with the qualified account subject in its environment")
			push(stsOrg, stsApp, map[string]string{"WIF_MODE": "sts", "TRUSTED_SERVICE_ACCOUNT_SUBJECT": subject, "TRUSTED_ISSUER": settings.Issuer, "TRUSTED_JWKS_URI": settings.JWKSURI, "TRUSTED_FEDERATION_AUDIENCE": settings.Audience, "TRUSTED_CA_CERT": settings.CACert})
			command("start", stsApp)
			for _, item := range []struct{ name, account string }{{app, account}, {secondApp, account}, {wrongApp, wrongAccount}} {
				push(workOrg, item.name, map[string]string{"WIF_MODE": "workload", "STS_URL": stsURL, "CAPI_URL": "https://" + Config.GetApiEndpoint(), "TRUSTED_FEDERATION_AUDIENCE": settings.Audience, "TRUSTED_CA_CERT": settings.CACert})
				command("bind-service-account", item.name, item.account)
				command("start", item.name)
			}
			roots, _ := x509.SystemCertPool()
			if roots == nil {
				roots = x509.NewCertPool()
			}
			if settings.CACert != "" {
				Expect(roots.AppendCertsFromPEM([]byte(settings.CACert))).To(BeTrue())
			}
			client := &http.Client{Timeout: 90 * time.Second, Transport: &http.Transport{TLSClientConfig: &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS12}}}
			probe := func(name, mode string) map[string]any {
				resp, err := client.Get(fmt.Sprintf("https://%s.%s/run?mode=%s", name, Config.GetAppsDomain(), mode))
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()
				body, err := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(200))
				Expect(string(body)).NotTo(ContainSubstring("access_token"))
				var result map[string]any
				Expect(json.Unmarshal(body, &result)).To(Succeed())
				AddReportEntry(name+"/"+mode, result)
				return result
			}
			By("using real UAA assertions to exchange and use distinct STS credentials from two apps")
			for _, name := range []string{app, secondApp} {
				r := probe(name, "exchange")
				Expect(r["token_status"]).To(BeEquivalentTo(200))
				Expect(r["exchange_status"]).To(BeEquivalentTo(200))
				Expect(r["resource_status"]).To(BeEquivalentTo(200))
				Expect(r["direct_resource_status"]).To(BeEquivalentTo(401))
				Expect(r["exchanged_capi_status"]).To(BeEquivalentTo(401))
				Expect(r["client_id"]).To(Equal(subject))
			}
			By("denying a valid token from another account in the same org")
			r := probe(wrongApp, "exchange")
			Expect(r["token_status"]).To(BeEquivalentTo(200))
			Expect(r["exchange_status"]).To(BeEquivalentTo(401))
			Expect(r["resource_status"]).To(BeNil())
			By("denying the default CAPI audience and unlisted resource")
			r = probe(app, "default-audience")
			Expect(r["token_status"]).To(BeEquivalentTo(200))
			Expect(r["exchange_status"]).To(BeEquivalentTo(401))
			r = probe(app, "unlisted")
			Expect(r["token_status"]).To(BeEquivalentTo(400))
			Expect(r["token_error"]).To(Equal("invalid_target"))
			By("disabling fresh issuance and restoring it through native CLI lifecycle")
			command("disable-service-account", account)
			r = probe(app, "exchange")
			Expect(r["token_status"]).To(BeEquivalentTo(401))
			command("enable-service-account", account)
			r = probe(app, "exchange")
			Expect(r["resource_status"]).To(BeEquivalentTo(200))
			Expect(api("/v3/organizations?names=" + stsOrg)["resources"].([]any)[0].(map[string]any)["service_account_prefix"]).To(BeNil())
		})
	})
})
