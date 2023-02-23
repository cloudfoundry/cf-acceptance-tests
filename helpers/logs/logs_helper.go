package logs

import (
	"encoding/json"
	"fmt"

	logcache "code.cloudfoundry.org/go-log-cache/rpc/logcache_v1"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/config"
	"github.com/cloudfoundry/cf-test-helpers/v2/cf"
	"github.com/cloudfoundry/cf-test-helpers/v2/helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"google.golang.org/protobuf/encoding/protojson"
)

func RecentEnvelopes(appGuid, oauthToken string, config config.CatsConfig) *logcache.ReadResponse {
	GinkgoHelper()
	endpoint := getLogCacheEndpoint()
	reqURL := fmt.Sprintf("%s/api/v1/read/%s?envelope_type=LOG&limit=1000", endpoint, appGuid)
	session := helpers.CurlRedact(oauthToken, config, reqURL, "-H", fmt.Sprintf("Authorization: %s", oauthToken))
	Expect(session.Wait()).To(gexec.Exit(0))
	var resp logcache.ReadResponse
	err := protojson.Unmarshal(session.Buffer().Contents(), &resp)
	Expect(err).NotTo(HaveOccurred())
	return &resp
}

func Recent(appName string) *gexec.Session {
	return cf.Cf("logs", "--recent", appName)
}

func Follow(appName string) *gexec.Session {
	return cf.Cf("logs", appName)
}

func getLogCacheEndpoint() string {
	GinkgoHelper()
	infoCmd := cf.Cf("curl", "/")
	Expect(infoCmd.Wait()).To(gexec.Exit(0))

	var resp struct {
		Links struct {
			LogCache struct {
				HREF string `json:"href"`
			} `json:"log_cache"`
		} `json:"links"`
	}

	err := json.Unmarshal(infoCmd.Buffer().Contents(), &resp)
	Expect(err).NotTo(HaveOccurred())

	return resp.Links.LogCache.HREF
}
