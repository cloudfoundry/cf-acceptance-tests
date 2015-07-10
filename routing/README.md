# Routing - CF Acceptance Tests

The following are routing-specific CF acceptance tests (CATS), for example:
- Routing API (register, unregister, list, SSE events)
- Route Services
- GoRouter (Context path, wildcard, SSL termination, sticky sessions)

All routing specific integration tests are located under **routing/** folder in
cf-acceptance-tests.

### Prerequisites
- Please ensure you have followed the Go environment and Test Setup instructions in
[cf-acceptance-tests](https://github.com/cloudfoundry/cf-acceptance-tests).

- Install the [Routing API CLI](https://github.com/cloudfoundry-incubator/routing-api-cli) v2.0

- Ensure you have configured the **client_secret** in integration_config.json

### Test Execution

To execute the routing-specific acceptance tests, please ensure you are in the
cf-acceptance-tests base directory and run the following:

```bash
bin/test_via_ginko routing
```
