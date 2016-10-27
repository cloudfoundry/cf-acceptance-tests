package assets

type Assets struct {
	AsyncServiceBroker       string
	Dora                     string
	DoraZip                  string
	DotnetCore               string
	Fuse                     string
	Golang                   string
	HelloWorld               string
	HelloRouting             string
	Java                     string
	JavaSpringZip            string
	JavaUnwriteableZip       string
	LoggregatorLoadGenerator string
	Python                   string
	Node                     string
	NodeWithProcfile         string
	Php                      string
	RubySimple               string
	SecurityGroupBuildpack   string
	ServiceBroker            string
	Staticfile               string
	SyslogDrainListener      string
	Binary                   string
	LoggingRouteService      string
	WorkerApp                string
	LatticeApp               string
	SpringSleuthZip          string
}

func NewAssets() Assets {
	return Assets{
		AsyncServiceBroker:       "assets/service_broker",
		Dora:                     "assets/dora",
		DoraZip:                  "assets/dora.zip",
		DotnetCore:               "assets/dotnet-core",
		Fuse:                     "assets/fuse-mount",
		Golang:                   "assets/golang",
		HelloRouting:             "assets/hello-routing",
		HelloWorld:               "assets/hello-world",
		Java:                     "assets/java",
		JavaSpringZip:            "assets/java-spring/java-spring.jar",
		JavaUnwriteableZip:       "assets/java-unwriteable-dir/java-unwriteable-dir.jar",
		LoggregatorLoadGenerator: "assets/loggregator-load-generator",
		Node:                   "assets/node",
		NodeWithProcfile:       "assets/node-with-procfile",
		Php:                    "assets/php",
		Python:                 "assets/python",
		RubySimple:             "assets/ruby_simple",
		SecurityGroupBuildpack: "assets/security_group_buildpack.zip",
		ServiceBroker:          "assets/service_broker",
		Staticfile:             "assets/staticfile",
		SyslogDrainListener:    "assets/syslog-drain-listener",
		Binary:                 "assets/binary",
		LoggingRouteService:    "assets/logging-route-service",
		WorkerApp:              "assets/worker-app",
		LatticeApp:             "assets/lattice-app",
		SpringSleuthZip:        "assets/spring-sleuth/spring-sleuth.jar",
	}
}
