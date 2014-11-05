# Operator logging tests

## Purpose
This test exercises the syslog drain forwarding functionality. A TCP listener is
spun up on the running machine, an app is deployed to the target Cloud Foundry
and bound to that listener (as a syslog drain) and the drain is checked for log
messages.

## Execution

1. Create a configuration JSON file. For testing against [bosh-lite](https://github.com/cloudfoundry/bosh-lite), the following is recommended:
  ```json
  {
      "suite_name"         : "CF_OPERATOR_TESTS",
      "api"                : "api.10.244.0.34.xip.io",
      "apps_domain"        : "10.244.0.34.xip.io",
      "user"               : "admin",
      "password"           : "admin",
      "org"                : "CF-OPERATOR-ORG",
      "space"              : "CF-OPERATOR-SPACE",
      "use_existing_org"   : false,
      "use_existing_space" : false,
      "syslog_drain_port"  : AVAILABLE_PORT_ON_LOCAL_MACHINE,
      "syslog_ip_address"  : "IP_ADDRESS_OF_LOCAL_MACHINE",
      "skip_ssl_validation": true
  }
  ```
  Of course, you should replace `AVAILABLE_PORT_ON_LOCAL_MACHINE` with an integer
  for some available port, and `IP_ADDRESS_OF_LOCAL_MACHINE` with an IP address at
  which bosh-lite can access your computer.

1. Set the `CONFIG` environment variable to the path of the configuration file,
   e.g. `export CONFIG=/path/to/config.json`
1. Execute the tests by running `ginkgo`.
