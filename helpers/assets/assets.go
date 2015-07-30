package assets

type Assets struct {
	AsyncServiceBroker       string
	Dora                     string
	DoraZip                  string
	Fuse                     string
	Golang                   string
	HelloWorld               string
	HelloRouting             string
	Java                     string
	LoggregatorLoadGenerator string
	Python                   string
	Node                     string
	NodeWithProcfile         string
	Php                      string
	RubySimple               string
	SecurityGroupBuildpack   string
	ServiceBroker            string
	Staticfile               string
	Binary                   string
	LoggingRouteServiceZip   string
}

func NewAssets() Assets {
	return Assets{
		AsyncServiceBroker: "../assets/service_broker",
		Dora:               "../assets/dora",
		DoraZip:            "../assets/dora.zip",
		Fuse:               "../assets/fuse-mount",
		Golang:             "../assets/golang",
		HelloRouting:       "../assets/hello-routing",
		HelloWorld:         "../assets/hello-world",
		Java:               "../assets/java",
		LoggregatorLoadGenerator: "../assets/loggregator-load-generator",
		Node:                   "../assets/node",
		NodeWithProcfile:       "../assets/node-with-procfile",
		Php:                    "../assets/php",
		Python:                 "../assets/python",
		RubySimple:             "../assets/ruby_simple",
		SecurityGroupBuildpack: "../assets/security_group_buildpack.zip",
		ServiceBroker:          "../assets/service_broker",
		Staticfile:             "../assets/staticfile",
		Binary:                 "../assets/binary",
		LoggingRouteServiceZip: "../assets/logging-route-service.zip",
	}
}
