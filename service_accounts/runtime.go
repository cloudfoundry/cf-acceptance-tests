package service_accounts

import (
	archive_helpers "code.cloudfoundry.org/archiver/extractor/test_helper"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
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

var _ = Describe("[service_accounts] bearer and runtime identity", Label("service-accounts", "runtime"), func() {
	BeforeEach(func() {
		if !Config.GetWIFConfig().RuntimeTests {
			Skip("wif.runtime_tests is false")
		}
	})
	DescribeTable("preserves explicit authorization and launch identity", func(journey string) {
		settings := Config.GetWIFConfig()
		b := make([]byte, 8)
		_, err := rand.Read(b)
		Expect(err).NotTo(HaveOccurred())
		id := "cats-runtime-" + hex.EncodeToString(b)
		org := id
		apps := map[string][]string{}
		created := false
		command := func(args ...string) *Session {
			s := cf.Cf(args...).Wait(Config.CfPushTimeoutDuration())
			Expect(s).To(Exit(0), "CLI %s", args[0])
			return s
		}
		api := func(path string) map[string]any {
			s := command("curl", path)
			var result map[string]any
			Expect(json.Unmarshal(s.Out.Contents(), &result)).To(Succeed())
			Expect(result["errors"]).To(BeNil())
			return result
		}
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
				if created {
					command("delete-org", org, "-f")
					Expect(api("/v3/organizations?names=" + org)["resources"]).To(BeEmpty())
				}
			})
			AddReportEntry("retained runtime prefix", id)
		})
		workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.CfPushTimeoutDuration(), func() {
			command("create-org", org, "--service-account-prefix", id)
			created = true
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
			command("create-service-account", "replacement")
			command("entitle-service-account", "worker", "two")
			orgGUID := api("/v3/organizations?names=" + org)["resources"].([]any)[0].(map[string]any)["guid"].(string)
			subject := "cf:service-account:" + id + ":worker"
			one, two := id+"-one", id+"-two"
			dir, err := os.MkdirTemp("", "cats-runtime-")
			Expect(err).NotTo(HaveOccurred())
			DeferCleanup(os.RemoveAll, dir)
			binary, err := filepath.Abs("assets/wif/bin/wif")
			Expect(err).NotTo(HaveOccurred())
			// A custom buildpack runs the same redacted observer inside staging and
			// preserves its result in the droplet. No certificate/key is exported.
			buildpack := filepath.Join(dir, "buildpack.zip")
			files := []archive_helpers.ArchiveFile{}
			for name, body := range map[string]string{"detect": "#!/bin/sh\necho identity-observer\n", "compile": "#!/bin/sh\nset -eu\n\"$1/wif\" identity > \"$1/staging-identity.txt\"\n", "release": "#!/bin/sh\nprintf 'default_process_types:\\n  web: ./wif\\n'\n"} {
				files = append(files, archive_helpers.ArchiveFile{Name: "bin/" + name, Body: body})
			}
			archive_helpers.CreateZipArchive(buildpack, files)
			bpName := id + "-buildpack"
			command("create-buildpack", bpName, buildpack, "99")
			DeferCleanup(func() {
				workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.CfPushTimeoutDuration(), func() { command("delete-buildpack", bpName, "-f") })
			})
			guids := map[string]string{}
			for _, item := range []struct{ name, space string }{{one, "one"}, {two, "two"}} {
				command("target", "-o", org, "-s", item.space)
				appDir := filepath.Join(dir, item.name)
				Expect(os.Mkdir(appDir, 0700)).To(Succeed())
				data, err := os.ReadFile(binary)
				Expect(err).NotTo(HaveOccurred())
				Expect(os.WriteFile(filepath.Join(appDir, "wif"), data, 0700)).To(Succeed())
				manifest := map[string]any{"applications": []any{map[string]any{"name": item.name, "path": appDir, "memory": DEFAULT_MEMORY_LIMIT, "buildpacks": []string{bpName}, "command": "./wif", "env": map[string]string{"CAPI_URL": "https://" + Config.GetApiEndpoint(), "TRUSTED_CA_CERT": settings.CACert}}}}
				data, err = json.Marshal(manifest)
				Expect(err).NotTo(HaveOccurred())
				path := filepath.Join(dir, item.name+".json")
				Expect(os.WriteFile(path, data, 0600)).To(Succeed())
				command("push", "-f", path, "--no-start")
				apps[item.space] = append(apps[item.space], item.name)
				command("bind-service-account", item.name, "worker")
				command("start", item.name)
				guids[item.name] = strings.TrimSpace(string(command("app", item.name, "--guid").Out.Contents()))
			}
			roots, _ := x509.SystemCertPool()
			if roots == nil {
				roots = x509.NewCertPool()
			}
			if settings.CACert != "" {
				Expect(roots.AppendCertsFromPEM([]byte(settings.CACert))).To(BeTrue())
			}
			client := &http.Client{Timeout: 30 * time.Second, Transport: &http.Transport{TLSClientConfig: &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS12}}}
			probe := func(app, path string) map[string]any {
				resp, err := client.Get("https://" + app + "." + Config.GetAppsDomain() + path)
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()
				Expect(resp.StatusCode).To(Equal(200))
				var result map[string]any
				Expect(json.NewDecoder(resp.Body).Decode(&result)).To(Succeed())
				AddReportEntry(app+path, result)
				return result
			}
			if journey == "roles" {
				By("proving default bearer profile and roleless isolation for both apps")
				for _, app := range []string{one, two} {
					identity := probe(app, "/identity")
					var r map[string]any
					Eventually(func() any { r = probe(app, "/run?mode=bearer"); return r["token_status"] }, Config.DefaultTimeoutDuration(), time.Second).Should(BeEquivalentTo(200), "UAA issuance must become available")
					Expect(r["token_status"]).To(BeEquivalentTo(200))
					Expect(r["capi_status"]).To(BeEquivalentTo(200))
					Expect(r["visible_apps"]).To(BeEmpty())
					Expect(r["cnf_present"]).To(BeFalse())
					Expect(r["sub"]).To(Equal(subject))
					Expect(r["iss"]).To(Equal(settings.Issuer))
					Expect(r["app_guid"]).To(Equal(guids[app]))
					Expect(r["org_guid"]).To(Equal(orgGUID))
					Expect(r["space_guid"]).To(Equal(identity["space_guid"]))
					Expect(r["cf_instance_guid"]).To(Equal(identity["cf_instance_guid"]))
					ttl := r["exp"].(float64) - r["iat"].(float64)
					Expect(ttl).To(BeNumerically(">", 0))
					Expect(ttl).To(BeNumerically("<=", 300))
					expiry, err := time.Parse(time.RFC3339, identity["certificate_not_after"].(string))
					Expect(err).NotTo(HaveOccurred())
					Expect(r["exp"]).To(BeNumerically("<=", expiry.Unix()))
				}
				By("granting only one space and observing shared account authorization")
				command("set-space-role", subject, org, "one", "SpaceAuditor", "--client")
				for _, app := range []string{one, two} {
					Eventually(func() any { return probe(app, "/run?mode=bearer")["visible_apps"] }, Config.DefaultTimeoutDuration(), time.Second).Should(ConsistOf(guids[one]))
				}
				command("set-space-role", subject, org, "two", "SpaceAuditor", "--client")
				for _, app := range []string{one, two} {
					Expect(probe(app, "/run?mode=bearer")["visible_apps"]).To(ConsistOf(guids[one], guids[two]))
				}
			} else {
				By("inspecting actual staging output rather than inferring it from runtime")
				for _, app := range []string{one, two} {
					s := probe(app, "/staging")
					Expect(s["account_discovery_present"]).To(BeFalse())
					if sans, ok := s["dns_sans"].([]any); ok {
						for _, san := range sans {
							Expect(san).NotTo(HaveSuffix(".svc.identity"))
						}
					}
				}
				command("target", "-o", org, "-s", "one")
				before := probe(one, "/identity")
				Expect(before["client_id"]).To(Equal(subject))
				Expect(before["dns_sans"]).To(ContainElement("worker." + id + ".svc.identity"))
				Expect(probe(two, "/identity")["public_key_sha256"]).NotTo(Equal(before["public_key_sha256"]))
				By("keeping running launch identity through reassignment and capturing current desired identity in tasks")
				command("unbind-service-account", one)
				Expect(probe(one, "/identity")["client_id"]).To(Equal(subject))
				command("bind-service-account", one, "replacement")
				Expect(probe(one, "/identity")["client_id"]).To(Equal(subject))
				taskName := id + "-task"
				command("run-task", one, "--command", "./wif identity", "--name", taskName)
				Eventually(func() any {
					tasks := api("/v3/apps/" + guids[one] + "/tasks")["resources"].([]any)
					for _, t := range tasks {
						task := t.(map[string]any)
						if task["name"] == taskName {
							return task["state"]
						}
					}
					return "missing"
				}, Config.CfPushTimeoutDuration(), time.Second).Should(Equal("SUCCEEDED"))
				Eventually(func() string { return string(command("logs", one, "--recent").Out.Contents()) }, Config.DefaultTimeoutDuration(), time.Second).Should(ContainSubstring(`"client_id":"cf:service-account:` + id + `:replacement"`))
				logs := string(command("logs", one, "--recent").Out.Contents())
				var taskIdentity map[string]any
				for _, line := range strings.Split(logs, "\n") {
					if index := strings.Index(line, "CATS_IDENTITY "); index >= 0 {
						var candidate map[string]any
						if json.Unmarshal([]byte(strings.TrimSpace(line[index+len("CATS_IDENTITY "):])), &candidate) == nil && candidate["client_id"] == "cf:service-account:"+id+":replacement" {
							taskIdentity = candidate
						}
					}
				}
				Expect(taskIdentity).NotTo(BeNil())
				Expect(taskIdentity["dns_sans"]).To(ContainElement("replacement." + id + ".svc.identity"))
				Expect(taskIdentity["app_guid"]).To(Equal(guids[one]))
				Expect(taskIdentity["org_guid"]).To(Equal(orgGUID))
				AddReportEntry("task desired identity", taskIdentity)
				command("restart", one)
				Expect(probe(one, "/identity")["client_id"]).To(Equal("cf:service-account:" + id + ":replacement"))
				Expect(probe(one, "/identity")["dns_sans"]).To(ContainElement("replacement." + id + ".svc.identity"))
				Expect(probe(one, "/identity")["public_key_sha256"]).NotTo(Equal(before["public_key_sha256"]))
				By("retaining the running identity until unbind is applied by restart")
				command("unbind-service-account", one)
				Expect(probe(one, "/identity")["client_id"]).To(Equal("cf:service-account:" + id + ":replacement"))
				command("restart", one)
				after := probe(one, "/identity")
				Expect(after["account_discovery_present"]).To(BeFalse())
				for _, san := range after["dns_sans"].([]any) {
					Expect(san).NotTo(HaveSuffix(".svc.identity"))
				}
			}
		})
	}, Entry("HTTPS bearer profile and explicit shared-account roles", "roles"), Entry("assignment restart tasks and staging separation", "runtime"))
})
