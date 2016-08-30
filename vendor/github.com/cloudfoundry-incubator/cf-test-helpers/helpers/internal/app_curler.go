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
	CurlFunc   func(...string) *gexec.Session
	UriCreator uriCreator
}

func NewAppCurler(curlFunc func(...string) *gexec.Session) *AppCurler {
	config := config.LoadConfig()
	uriCreator := &AppUriCreator{Config: config}
	return &AppCurler{
		UriCreator: uriCreator,
		CurlFunc:   curlFunc,
	}
}

func (appCurler *AppCurler) CurlAndWait(appName string, path string, timeout time.Duration, args ...string) string {
	appUri := appCurler.UriCreator.AppUri(appName, path)
	curlArgs := append([]string{appUri}, args...)

	curlCmd := appCurler.CurlFunc(curlArgs...).Wait(timeout)

	ExpectWithOffset(3, curlCmd).To(gexec.Exit(0))
	ExpectWithOffset(3, string(curlCmd.Err.Contents())).To(HaveLen(0))
	return string(curlCmd.Out.Contents())
}
