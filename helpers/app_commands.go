package helpers

func AppUri(appName, endpoint string, appsDomain string) string {
	return "http://" + appName + "." + appsDomain + endpoint
}
