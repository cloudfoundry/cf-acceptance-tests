package helpersinternal

import (
	"time"

	"github.com/cloudfoundry-incubator/cf-test-helpers/config"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

type uriCreator interface {
	AppUri(appName, path string) string
}

type AppCurler struct {
	CurlFunc   func(config.Config, ...string) *gexec.Session
	UriCreator uriCreator
}

func NewAppCurler(curlFunc func(config.Config, ...string) *gexec.Session, cfg config.Config) *AppCurler {
	uriCreator := &AppUriCreator{Config: cfg}
	return &AppCurler{
		UriCreator: uriCreator,
		CurlFunc:   curlFunc,
	}
}

func (appCurler *AppCurler) CurlAndWait(cfg config.Config, appName string, path string, timeout time.Duration, args ...string) string {
	appUri := appCurler.UriCreator.AppUri(appName, path)
	curlArgs := append([]string{appUri}, args...)

	curlCmd := appCurler.CurlFunc(cfg, curlArgs...).Wait(timeout)

	ExpectWithOffset(3, curlCmd).To(gexec.Exit(0))
	ExpectWithOffset(3, string(curlCmd.Err.Contents())).To(HaveLen(0))
	return string(curlCmd.Out.Contents())
}
