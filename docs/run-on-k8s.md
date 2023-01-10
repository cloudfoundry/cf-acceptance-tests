# How to run Cloud Foundry Acceptance Tests on CF for Kubernetes
Running CATS on CF for Kubernetes is largely the same, but with a restricted test set. Currently, only the app suite should be run.

Some of the tests are skipped for various reasons, including:
* Incompatible with Kubernetes components
* Interface changes due to replatform
* Additional issues that need attention from component teams

The sample configuration will run CATS in an apps suite only mode, with additional tests skipped from the infrastructure property.

## Sample Configuration
```json
{
  "api": "<your-api-endpoint>",
  "admin_user": "admin",
  "admin_password": "<your-cf-admin-password>",
  "apps_domain": "<your-apps-domain>",
  "skip_ssl_validation": true,
  "timeout_scale": 1,
  "include_apps": true,
  "include_backend_compatibility": false,
  "include_deployments": false,
  "include_detect": false,
  "include_docker": false,
  "include_internet_dependent": false,
  "include_private_docker_registry": false,
  "include_route_services": false,
  "include_routing": false,
  "include_service_discovery": false,
  "include_service_instance_sharing": false,
  "include_services": false,
  "include_tasks": false,
  "include_v3": false,
  "infrastructure": "kubernetes",
  "ruby_buildpack_name": "paketo-community/ruby",
  "go_buildpack_name": "paketo-buildpacks/go",
  "java_buildpack_name": "paketo-buildpacks/java",
  "nodejs_buildpack_name": "paketo-buildpacks/nodejs",
  "binary_buildpack_name": "paketo-buildpacks/procfile"
}
```

```bash
export CONFIG=<location-of-configuration>
<cf-acceptance-tests-dir>/bin/test
```

## Additional Requirements
CF CLI version 7+ is required.
