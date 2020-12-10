# Example Route Service

An example route service for Cloud Foundry.

## Route Service Overview

The Route Service feature is currently in development, the proposal can be found in this [Google Doc](https://docs.google.com/document/d/1bGOQxiKkmaw6uaRWGd-sXpxL0Y28d3QihcluI15FiIA/edit#heading=h.8djffzes9pnb).

This example route service uses the new headers/features that have been added to the GoRouter. For example:

- `X-CF-Forwarded-Url`: A header that contains the original URL that the GoRouter received.
- `X-CF-Proxy-Signature`: A header that the GoRouter uses to determine if a request has gone through the route service.

## Getting Started

- Download this repository and `cf push` to your chosen CF deployment.
- Push your app which will be associated with the route service.
- Create a user-provided route service ([see docs](http://docs.cloudfoundry.org/services/route-services.html#user-provided))
- Bind the route service to the route (domain/hostname)
- Tail the logs of this route service in order to verify that requests to your app go through the route service. The example logging route service will log requests and responses to and from your app.

## Environment Variables

### ROUTE_SERVICE_SLEEP_MILLI

If you set this environment variable in the running app, the route service
will sleep for that many milliseconds before proxying the request. This can
be used to simulate route services that are slow to respond.

Example (10 seconds):

```sh
cf set-env logging-route-service ROUTE_SERVICE_SLEEP_MILLI 10000
cf restage logging-route-service
```

### SKIP_SSL_VALIDATION

Set this environment variable to true in order to skip the validation of SSL certificates.
By default the route service will attempt to validate certificates.

Example:

```sh
cf set-env logging-route-service SKIP_SSL_VALIDATION true
cf restart logging-route-service
```
