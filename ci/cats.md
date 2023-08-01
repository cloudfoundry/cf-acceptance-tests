# CATs

## Purpose

This pipeline validates changes to `cf-acceptance-tests` for compatibility with the latest Cloud Foundry release (cf-deployment, cf-for-k8s) and provides manually triggered jobs for the publication of its own releases.

## Validation Strategy

### Unit Testing

As a blocking step before running acceptance test suites, we run a separate suite of unit tests to validate the configuration interface and cf cli integration. Currently, the suites are located in the `helpers/config`, `helpers/download` and `helpers/cli_version_check` directories.

### Acceptance Testing

We validate every test suite but windows, using the following config to enable those test suites:

```
  "include_apps": true,
  "include_backend_compatibility": true,
  "include_container_networking": true,
  "include_detect": true,
  "include_docker": true,
  "include_internet_dependent": true,
  "include_private_docker_registry": true,
  "include_route_services": true,
  "include_routing": true,
  "include_security_groups": true,
  "include_service_instance_sharing": true,
  "include_services": true,
  "include_ssh": true,
  "include_sso": true,
  "include_tasks": true,
  "include_tcp_routing": true,
  "include_v3": true,
  "include_windows": false,
  "include_zipkin": true
```

### Kubernetes-specific Testing

With the development of cf-for-k8s, we discovered different behavior in Kubernetes component implementations like the istio service mesh and kpack for the integration of cloud native buildpacks. As such, we include separate tests to validate that different behavior using the following non-default configuration:

```
"infrastructure": "kubernetes",
"ruby_buildpack_name": "paketo-buildpacks/ruby",
"go_buildpack_name": "paketo-buildpacks/go",
"java_buildpack_name": "paketo-buildpacks/java",
"nodejs_buildpack_name": "paketo-buildpacks/nodejs",
"binary_buildpack_name": "paketo-buildpacks/procfile"
```

## Infrastructure

This pipeline claims infrastructure provisioned by other pipelines through concourse resource-pool resources. For the management of these resources, `acquire-pool` and `release-pool` jobs are present for each type of infrastructure used. If a job fails and you are done investigating the environment, remember to clean up the environment manually release the pool lock.

### CF on VMs

For CF on VMs, we deploy the latest [cf-deployment](https://github.com/cloudfoundry/cf-deployment) release with an isolation segment enabled. The Bosh director is provisioned by a separate infrastructure pipeline that uses the bosh-bootloader utility.

The deployment is managed by the `deploy-cf` and `delete-cf` jobs which deploy and clean up cf-deployment respectively.

### Kubernetes

For CF on Kubernetes, we deploy the latest [cf-for-k8s](https://github.com/cloudfoundry/cf-for-k8s) release on a Terraform-provisioned [GKE](https://cloud.google.com/kubernetes-engine) cluster on the [rapid release channel](https://cloud.google.com/kubernetes-engine/docs/concepts/release-channels).

## Release Management

See the `CATS-release` team wiki page for the full release management documentation: https://github.com/cloudfoundry/cf-acceptance-tests/wiki/Releasing

## Pipeline management

This pipeline is managed by the `ci/pipeline.yml` file. To make changes to the pipeline, update the file directly and either run the `ci/configure` script to apply the changes (if you've already set up your fly cli to use the `relint-ci` target) or manually run the fly cli `set-pipeline` command.
