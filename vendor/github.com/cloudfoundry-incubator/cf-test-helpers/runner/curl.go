package runner

import "github.com/onsi/gomega/gexec"

func Curl(args ...string) *gexec.Session {
	return CurlSkipSSL(SkipSSLValidation, args...)
}

func CurlSkipSSL(skip bool, args ...string) *gexec.Session {
	curlArgs := append([]string{"-s"}, args...)
	if skip {
		curlArgs = append([]string{"-k"}, curlArgs...)
	}
	return Run("curl", curlArgs...)
}
