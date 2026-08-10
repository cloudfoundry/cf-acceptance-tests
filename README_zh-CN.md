# CF验收测试（CATs）

<!-- hy-mt2-i18n:start -->
[English](./README.md) | **中文** | [日本語](./README_ja.md) | [Español](./README_es.md)
<!-- hy-mt2-i18n:end -->

该测试套件通过`cf` CLI与`curl`工具来对[Cloud Foundry](https://github.com/cloudfoundry/cf-deployment)部署功能进行测试。
其测试范围仅限于面向用户的端到端功能。

例如，某项测试会使用 `cf push` 推送一个应用，再通过 `curl` 调用该应用中的某个接口使其崩溃，最后验证是否能在 `cf events` 中看到对应的崩溃事件。

此处不会包含的测试包括在 Cloud Controller 中对对象进行基本 CRUD 操作之类的内容。这类测试应归入与其相关的组件中。

这些测试并非用于生产环境。它们适用于开发 Cloud Foundry 版本时所使用的验收环境。尽管这些测试会尽力自行清理残留，但无法保证不会以不可取的方式改变系统的状态。如需可在生产环境中安全运行的轻量级系统测试，请使用[CF 烟雾测试](https://github.com/cloudfoundry/cf-smoke-tests)。

**注意：** 由于我们需要并行执行测试，因此应编写能够独立运行的测试用例。测试不得依赖其他测试中的状态，也不得以影响其他测试的方式修改 Cloud Foundry 的状态。

# 测试环境准备
### 运行 CATS 的前提条件
- 安装版本大于或等于 `1.24.0` 的 golang。请参考 [golang.org](http://golang.org/doc/install) 的说明来搭建您的 golang 开发环境。
- 安装版本大于或等于 `8.5.0` 的 [`cf CLI`](https://github.com/cloudfoundry/cli)，并确保该工具已添加到您的 `$PATH` 环境变量中。
- 安装 [curl](http://curl.haxx.se/)

## 测试环境准备
### 运行 CATS 的前置条件
- 安装版本大于等于 `1.24.0` 的 golang。请参考 [golang.org](http://golang.org/doc/install) 的说明来搭建你的 golang 开发环境。
- 安装版本大于等于 `8.5.0` 的 [`cf CLI`](https://github.com/cloudfoundry/cli)，并确保该工具可在你的 `$PATH` 路径下被访问。
- 安装 [curl](http://curl.haxx.se/)。
- 克隆一份 `cf-acceptance-tests` 代码。由于该项目使用了 Go 模块，因此无需将其放入 `$GOPATH` 目录中。
- 部署一个正在运行的 Cloud Foundry 应用，以便针对其运行这些验收测试。例如 bosh-lite。

### 更新 `go` 依赖项
CATs 所需的所有 `go` 依赖项均被打包在 `vendor` 目录中。

请确保已安装 Golang 1.24 及更高版本。

要将当前依赖项更新为特定版本，请执行以下操作：

```bash
cd cf-acceptance-tests
source.envrc
go get <import_path>@<revision_number>
go mod vendor
```

如果想要添加新的依赖项，只需执行以下命令：

```bash
go mod tidy
go mod vendor
```

## 测试配置
您必须设置环境变量 `$CONFIG`，该变量需指向一个 JSON 文件，文件中包含用于配置验收测试的多项数据，例如告知测试如何定位您正在运行的 Cloud Foundry 部署以及应运行哪些测试。

可将以下内容粘贴到终端中，
它将配置好所需的 `$CONFIG`，
从而针对 CF 的 [BOSH-Lite](https://github.com/cloudfoundry/bosh-lite)
部署运行核心测试套件。

可参考 [`example-cats-config.json`](example-cats-config.json) 文件。

默认情况下仅会运行以下测试组：
```
include_apps
include_detect
include_routing
include_v3
include_app_syslog_tcp
```

# 完整的配置参数说明如下：
##### 必需参数：
* `api`：Cloud Controller API 接口地址，无需指定协议类型（HTTP/S）。
* `admin_user`：您的 CF 实例中拥有管理权限的用户名称。该管理员用户必须具备 `doppler.firehose` 权限范围。
* `admin_password`：上述管理员用户的密码。
* `apps_domain`：测试可用于创建子域名的共享域名，这些子域名将指向测试中创建的应用程序，无需指定协议类型（HTTP/S）。
* `skip_ssl_validation`：如果用于通往您 CF 实例的流量使用了无效证书（例如自签名证书），则将该值设置为 true；对于 CF 的 BOSH-Lite 部署，此值通常始终为 true。
* `skip_dns_validation`：跳过 CF API 和应用域名的 DNS 验证。在代理环境中可使用 true 值。默认值为 false。

##### 可选参数：
`include_*` 参数用于根据部署配置来指定是否跳过相关测试。
* `include_app_syslog_tcp`：用于包含通过 TCP 进行应用系统日志传输测试的标志。
* `include_apps`：用于包含应用测试组的标志。
* `readiness_health_checks_enabled`：默认值为 `true`。若使用的环境不支持就绪性健康检查，则将其设置为 `false`。
* `include_cnb`：用于包含与使用云原生构建包构建应用相关的测试的标志。这些测试需在已部署 Diego 的前提下，并且需启用 CC API 的 diego_cnb 特性标志才能通过，同时 CF CLI 版本必须至少为 v8.14.0。
* `include_container_networking`：用于包含与容器网络相关的测试的标志。
* `credhub_mode`：有效值为 `assisted` 或 `non-assisted`。[详见下方](#credhub-modes)。
* `credhub_location`：CredHub 实例的位置；默认值为 `https://credhub.service.cf.internal:8844`。
* `credhub_client`：服务代理访问 CredHub 所需的 UAA 客户端凭证（进行 CredHub 测试时必需）；默认值为 `credhub_admin_client`。
* `credhub_secret`：服务代理访问 CredHub 所需的 UAA 客户端密钥（进行 CredHub 测试时必需）。
* `include_deployments`：用于包含针对云控制器滚动部署的测试的标志。同时还需启用 V3 功能。
* `include_detect`：用于包含 detect 组中测试的标志。
* `include_docker`：用于包含与在 Diego 上运行 Docker 应用相关的测试的标志。这些测试需在已部署 Diego 的前提下，并且需启用 CC API 的 diego_docker 特性标志才能通过。
* `include_file_based_service_bindings`：用于包含基于文件的服务绑定测试的标志。详情请参阅 [RFC0030](https://github.com/cloudfoundry/community/blob/main/toc/rfc/rfc-0030-add-support-for-file-based-service-binding.md)。
* `include_http2_routing`：用于包含 HTTP/2 路由测试的标志。
* `include_internet_dependent`：用于包含那些要求部署环境具备互联网访问权限的测试的标志。
* `include_isolation_segments`：用于包含隔离段测试的标志。
* `include_private_docker_registry`：用于运行依赖私有 Docker 镜像的测试的标志。[详见下方](#private-docker)。
* `include_route_services`：用于包含路由服务测试的标志。这些测试需在已部署 Diego 的前提下才能通过。
* `include_routing`：用于包含路由测试的标志。
* `include_ipv6`：用于包含 IPv6 验证测试组的标志。
* `include_routing_isolation_segments`：用于包含路由隔离段测试的标志。[详见下方](#routing-isolation-segments)。该测试无法与日志隔离段测试同时运行。
* `include_security_groups`：用于包含安全组相关测试的标志。[详见下方](#container-networking-and-application-security-groups)。
* `dynamic_asgs_enabled`：默认值为 `true`。如果在测试环境中禁用了动态 ASG，则将其设置为 `false`。
* `comma_delim_asgs_enabled`：默认值为 `false`。如果在测试环境中启用了以逗号分隔的 ASG 目标，则将其设置为 `true`。
* `include_services`：用于决定是否包含针对 services API 的测试。
* `include_service_instance_sharing`：用于决定是否包含针对不同空间之间服务实例共享功能的测试。要运行这些测试，必须先设置 `include_services`，同时还需启用 `service_instance_sharing` 功能标志才能通过测试。
* `include_service_credential_binding_rotation`：用于执行针对多个服务绑定的测试。详情请参阅 [RFC-0040](https://github.com/cloudfoundry/community/blob/main/toc/rfc/rfc-0040-service-binding-rotation.md)。此测试要求使用 CF CLI v8.18.0 或更高版本，且后端必须支持每个应用和服务实例至少 2 个服务绑定。
* `include_ssh`：用于决定是否包含针对 Diego 容器 ssh 功能的测试。
* `include_sso`：用于决定是否包含与单点登录功能集成的服务测试。
* `include_tasks`：用于决定是否包含 v3 版本的任务测试。要运行这些测试，还必须设置 `include_v3`，同时还需启用 CC API 的 task_creation 功能标志才能通过测试。
* `include_tcp_routing`：用于决定是否包含 TCP 路由测试。这些测试等同于路由验收测试中的 [TCP 路由测试](https://github.com/cloudfoundry/routing-acceptance-tests/blob/master/tcp_routing/tcp_routing_test.go)。
* `tcp_domain`：用于具有 TCP 路由的应用的域名。
* `include_user_provided_services`：用于决定是否包含针对用户提供的服务的测试。
* `include_v3`：用于决定是否包含针对 v3 API 的测试。
* `include_zipkin`：用于决定是否包含针对 Zipkin 追踪功能的测试。要运行这些测试，还必须设置 `include_routing`，同时 CF 部署时必须设置 `router.tracing.enable_zipkin` 才能通过测试。
* `use_http`：如果希望 CF 接受测试在发送 API 和应用请求时使用 HTTP，则将其设置为 `true`（默认为 HTTPS）。
* `use_existing_organization`：当需要指定现有组织而非创建新组织时，将其设置为 `true`。
* `existing_organization`：要使用的现有组织的名称。
* `use_existing_user`：通常会使用上面配置的管理员用户来创建一个权限较低的临时用户，以便在测试期间执行相关操作（如推送应用），并在测试完成后删除该用户；如果希望使用通过以下属性配置的现有用户，则将此值设置为 `true`。
* `staticfile_`：用于指定静态文件的处理方式。

* `staticfile_buildpack_name` [见下文](#buildpack-names)
* `java_buildpack_name` [见下文](#buildpack-names)
* `ruby_buildpack_name` [见下文](#buildpack-names)
* `nginx_buildpack_name` [见下文](#buildpack-names)
* `nodejs_buildpack_name` [见下文](#buildpack-names)
* `go_buildpack_name` [见下文](#buildpack-names)
* `r_buildpack_name` [见下文](#buildpack-names)
* `binary_buildpack_name` [见下文](#buildpack-names)
* `cnb_nodejs_buildpack_name` [见下文](#buildpack-names)
* `python_buildpack_name: python_buildpack` [见下文](#buildpack-names)

* `include_windows`：用于决定是否包含针对 Windows 环境运行的测试的标志。
* `use_windows_test_task`：用于决定是否包含在 Windows 环境下对任务进行的测试的标志。默认值为 `false`。
* `use_windows_context_path`：用于决定是否包含 Windows 上下文路径路由测试的标志。默认值为 `false`。
* `windows_stack`：用于针对其运行测试的 Windows 栈。

* `include_service_discovery`：用于决定是否包含服务发现相关测试的标志。这些测试会使用 `apps.internal` 域名，该域名是 `cf-networking-release` 中的默认值，目前内部域名不可配置。
* `stacks`：需要针对其进行测试的堆栈数组。目前支持 `cflinuxfs4` 和 `cflinuxfs5` 这两种堆栈，默认值为 `[cflinuxfs4]`。

* `include_volume_services`：用于决定是否包含针对卷服务的测试。要运行此测试套件，必须满足以下要求：必须已部署 tcp-routing。
* `volume_service_name`：由卷服务代理提供的卷服务名称。
* `volume_service_plan_name`：由卷服务代理提供的服务的计划名称。
* `volume_service_create_config`：创建卷服务时使用的 JSON 配置。
* `volume_service_bind_config`：卷服务绑定配置的 JSON 配置。

有一个名为 `bin/catsconfiggenerator` 的 Golang 工具，能够根据代码中的现有默认值按需生成完整的 config.json 文件。您可以使用它，并根据自身环境的需求对生成的 JSON 文件进行修改。

#### Buildpack 名称
许多测试在推送应用时会指定对应的 Buildpack，以便在 Diego 中更快地完成应用预发布流程。这些 Buildpack 的默认名称如下；如果您的 Buildpack 名称不同，可以通过设置不同的名称来覆盖默认值。

* `staticfile_buildpack_name: staticfile_buildpack`
* `java_buildpack_name: java_buildpack`
* `ruby_buildpack_name: ruby_buildpack`
* `nginx_buildpack_name: nginx_buildpack`
* `nodejs_buildpack_name: nodejs_buildpack`
* `go_buildpack_name: go_buildpack`
* `r_buildpack_name: r_buildpack`
* `binary_buildpack_name: binary_buildpack`
* `hwc_buildpack_name: hwc_buildpack`
* `python_buildpack_name: python_buildpack`

对于云原生构建包的生命周期，您可以通过设置不同的名称来覆盖默认值：

* `cnb_nodejs_buildpack_name: docker://docker.io/paketobuildpacks/nodejs:latest`

#### Route Services 测试组配置
`route_services` 测试组用于部署那些必须能够访问 Cloud Foundry 部署中的负载均衡器的应用程序。为此需要配置应用安全组以支持此类访问。如果您正在运行 `route_services` 组，部署清单中应包含以下数据：

```yaml
...
properties:
 ...
  cc:
   ...
    security_group_definitions:
      - name: load_balancer
        rules:
        - protocol: all
          destination: IP_OF_YOUR_LOAD_BALANCER # （例如，在基于BOSH-Lite的标准Cloud Foundry部署中为10.244.0.34）
    default_running_security_groups: ["load_balancer"]
```

#### 容器网络与应用安全组
若要运行用于测试容器网络及应用安全组功能的测试，必须将 `include_security_groups` 标志设置为 true。

Windows ASG 测试要求在 CF 部署所使用的私有网络中设置一个未分配的 IP 地址，该地址可通过 `unallocated_ip_for_security_group` 配置值来指定。通过 bbl 在公有云上创建的环境可使用默认值 10.0.244.255。而 vSphere 和 Openstack 环境则可能需要自定义的 IP 地址。

#### 私有 Docker
若要运行测试来验证使用凭据访问私有 Docker 注册表的功能，必须将 `include_private_docker_registry` 标志设置为 true，并且还需提供以下配置值：

* `private_docker_registry_image`：私有 Docker 镜像地址
* `private_docker_registry_username`：私有 Docker 镜像的用户名
* `private_docker_registry_password`：私有 Docker 镜像的密码

这些测试假定指定的私有 Docker 镜像为 cloudfoundry/diego-docker-app:latest 的私有版本。若要将私有版本上传到您的 DockerHub 账户，首先在 DockerHub 上创建一个私有仓库，然后在命令行中登录 Docker。接着运行以下命令：

```bash
docker pull cloudfoundry/diego-docker-app:latest
docker tag cloudfoundry/diego-docker-app:latest <your-private-repo>:<some-tag>
docker push <your-private-repo>:<some-tag>
```

此时，`private_docker_registry_image` 配置值的值应为“<your-private-repo>:<some-tag>”。

#### 路由隔离段
要运行涉及路由隔离段的测试，必须提供以下配置值：
* `isolation_segment_name`
* `isolation_segment_domain`

如需了解更多配置细节，请阅读[此处](http://docs.cloudfoundry.org/adminguide/routing-is.html)的文档。

#### Credhub 模式
- `non-assisted` 模式意味着应用程序需自行负责解析 Credhub 中的凭据引用。当用户将服务绑定到应用程序时，服务代理会将凭据存储在 Credhub 中，并将该引用返回给 Cloud Controller。当用户重新启动应用程序时，Cloud Controller 会通过 `VCAP_SERVICES` 环境变量将该 Credhub 引用传递给应用程序，此时应用程序可直接向 Credhub 发起请求以解析引用并获取凭据。此模式在 `cc.credential_references.interpolate_service_bindings` 的值为 false 时启用——这并非默认配置。
- `assisted` 模式意味着会在应用程序开始运行之前就解析 Credhub 引用。与前述流程相同，当用户将服务绑定到应用程序时，服务代理仍会将凭据存储在 Credhub 中并将引用返回给 Cloud Controller。不同的是，当用户重新启动应用程序时，Cloud Controller 会将该 Credhub 引用传递给 Diego 运行时，随后启动器（来自 buildpackapplifecycle 或 dockerapplifecycle 组件）会解析该引用，并将凭据存储在 `VCAP_SERVICES` 中供应用程序使用。此模式在 `cc.credential_references.interpolate_service_bindings` 的值为 true 时启用——这是默认配置。

#### 捕获测试输出
当测试失败时，请在测试输出中查找测试组名称（如下例中的 `[services]`）：

```bash
• 规范设置（BeforeEach）阶段失败 [34.662秒]
[services] 服务实例生命周期
```

如果在 `$CONFIG` 文件中为 `artifacts_directory` 设置了值，那么就可以获取失败测试运行产生的 `cf` 跟踪输出，当常规测试输出不足以用于问题排查时，这类输出会非常有用。这些测试用例的 `cf` 跟踪输出会保存在 `artifacts_directory` 下的 `CF-TRACE-Applications-*.txt` 文件中。

## 测试执行
要按照配置执行测试，请运行 [bin/test](./bin/test) 脚本，并将 `$CONFIG` 设置为您的 [`integration_config.json`](#test-configuration) 文件路径。

##### 并行执行
可以同时运行所有测试组，让测试在多个进程间并行执行。这种并行处理方式能大幅缩短 CAT 的运行时间。

不过，设置该数值时需格外谨慎，因为它实际上决定了“一次要推送多少个应用”，而几乎所有示例都是推送一个应用。

使用 `--procs` 参数来设置并行进程的数量，例如：
```bash
./bin/test --procs=12
```

供参考，Release Integration 团队运行的并行进程数量如下：

| 基础设施类型 | 并行进程数 |
| ----------- | ----------- |
| [Vanilla CF](https://github.com/cloudfoundry/cf-deployment/blob/master/cf-deployment.yml) | 12 |
| [BOSH Lite](https://github.com/cloudfoundry/cf-deployment/blob/master/operations/bosh-lite.yml) | 4 |


##### 定向执行测试组
如果您已经熟悉 CATs，那么想必知道其中存在许多测试组。您可能并不想在所有环境中运行所有的测试，有时还希望定向执行特定的测试组以定位故障原因。要执行某一组特定的验收测试，例如 `routing/`，请编辑您的 [`integration_config.json`](#test-configuration) 文件，将除 `include_routing` 之外的所有 `include_*` 值设置为 `false`，然后再运行以下命令：

```bash
./bin/test
```

若要执行单个文件中的测试，需在该文件中的测试代码周围使用 `FDescribe` 块：
```go
var _ = AppsDescribe("Apps", func() {
  FDescribe("Focused tests", func() { // 在此处添加该行
  //... 文件的其余部分
  }) // 在此处结束
})
```

测试组名称与目录名称相对应。

##### 详细输出
若要查看 `ginkgo` 的详细输出，可使用 `-v` 标志。

```bash
./bin/test -v
```

当然，您也可以将 `-v` 标志与 `--procs=N` 标志一起使用。

##### 整体测试超时设置
从 Ginkgo 2.0 开始，整个测试的默认超时时间已改为 1 小时（参见：[Ginkgo 2.0 迁移指南 - 超时行为](https://onsi.github.io/ginkgo/MIGRATING_TO_V2#timeout-behavior)）。

根据测试所处的环境及并行度不同，测试运行时间可能会超过一小时，甚至存在失败的可能。

可使用 `--timeout` 标志根据需要调整测试超时时间：

```bash
./bin/test --timeout=24h
```

关于各个测试的单独超时时间，例如 `cf push` 的超时设置，请参阅[测试配置](#test-configuration)。

## 测试组说明

测试组名称 | 描述
--- | ---
`app_syslog_tcp` | 测试配置应用系统日志转发监听器的功能。
`apps` | 测试 Cloud Foundry 的核心功能：暂存、运行、日志记录、路由、构建包等。在正常的 Cloud Foundry 部署环境中，该测试组应始终通过测试。
`credhub` | 测试服务绑定中通过 CredHub 提供的安全服务凭证。要运行这些测试，需要[配置 CredHub](https://docs.cloudfoundry.org/adminguide/credhub-secure-service-credentials.html)。除了选择 `credhub_mode` 外，还需设置 `credhub_client` 和 `credhub_secret` 值。
`cnb` | 测试我们使用云原生构建包的能力。
`detect` | 测试在未明确指定构建包时，平台能否自动检测到用于编译应用的正确构建包。
`docker` | 测试我们在 Diego 上运行 Docker 容器的能力，以及是否正确处理了 Docker 元数据。
`ipv6` | 测试 IPv6 出站调用功能。
`file-based service bindings` | 测试针对构建包应用、CNB 应用和 Docker 应用的基于文件的服务绑定功能。该测试组会在两种不同的堆栈（Windows、Linux）上，通过两个不同的特性标志进行测试。有关这些特性标志的更多详细信息，请查看 [RFC0030](https://github.com/cloudfoundry/community/blob/main/toc/rfc/rfc-0030-add-support-for-file-based-service-binding.md)。
`internet_dependent` | 测试通过 Github URL 指定构建包的功能。因此，这依赖于你的 Cloud Foundry 应用容器能够访问互联网。你需要考虑所部署 Cloud Foundry 的网络配置，以及应用于应用容器的任何安全组设置。
`isolation_segments` | 运行此测试组要求 Diego 至少部署有 2 个节点，且其中一个节点必须已设置了 `placement_tag`。如果部署采用了路由隔离段，则还必须设置 `isolation_segment_domain`。更多信息请参考 [隔离段文档](https://docs.cloudfoundry.org/adminguide/isolation-segments.html)。
`route_services` | 测试 Cloud Foundry 的[路由服务](https://docs.cloudfoundry.org/services/route-services.html)功能。
`routing` | 该测试包包含与路由相关的验收测试（包括上下文路径、通配符、SSL 终结、粘性会话以及 Zipkin 追踪功能）。
`routing_isolation_segments` | 测试指向隔离应用的请求仅通过隔离路由器路由，反之亦然。该测试需要隔离段测试套件的所有配置，同时还必须部署至少两个Gorouter实例。其中一个实例需配置属性`routing_table_sharding_mode: shared-and-segments`，另一个实例则需具备属性`routing_table_sharding_mode: segments`以及`isolation_segments: [YOUR_PLACEMENT_TAG_HERE]`。CATs属性中的`isolation_segment_name`必须与`placement_tag`相匹配，且`isolation_segment`.`isolation_segment_domain`必须已设置，流向该域名的流量应被发送至隔离路由器。
`security_groups`| 测试Cloud Foundry的[安全组](https://docs.cloudfoundry.org/concepts/asg.html)功能。
`service_discovery`| 测试在Cloud Foundry上运行的应用程序的[服务发现](https://docs.cloudfoundry.org/devguide/deploy-apps/cf-networking.html#discovery)功能。
`services`| 测试与服务相关的各种功能，例如通过服务代理API注册服务代理。其中部分测试涉及特殊集成，比如单点登录认证；如果您尚未配置所需的集成（可在配置中将`include_sso`参数设置为`false`），可以选择性地运行该包中的部分测试而跳过其他测试。
`ssh`| 测试通过ssh、scp和sftp与Diego应用进行通信的功能。
`tasks`| 测试Cloud Foundry的[任务](https://docs.cloudfoundry.org/devguide/using-tasks.html)功能。
`tcp_routing`| 测试Cloud Foundry的TCP路由功能。您需要按照[此处](https://docs.cloudfoundry.org/adminguide/enabling-tcp-routing.html)的说明设置一个名为`tcp.<SYSTEM_DOMAIN>`的TCP域名。如果您使用的是`bbl`（BOSH引导程序），则TCP域名会自动为您设置。
`user_provided_services` | 测试与创建及绑定用户提供的服务以安全存储应用凭证相关的功能。
`v3`| 该测试组包含针对下一代v3 Cloud Controller API的测试。
`volume_services` | 测试Cloud Foundry的[卷服务](https://docs.cloudfoundry.org/devguide/services/using-vol-services.html)功能。

## 贡献方式

该仓库使用[go mod](https://golang.org/cmd/go/#hdr-Modules__module_versions__and_more)来管理`go`依赖项。

CATs 所需的所有 `go` 依赖项均已打包到 `vendor` 目录中。

在对测试套件进行修改并引入额外的 `go` 包时，应遵循以下工作流程：

如果可以使用依赖项的最新版本，请使用 `go mod tidy`，否则请使用 `go get <dependency>@<version>`。这两种方法都需要通过 [envrc](.envrc) 启用 go modules。最后使用 `go mod vendor` 将这些依赖项添加到 `vendor` 目录中。

对于工具和资源文件，请通过[go mod tool工作流](https://github.com/go-modules-by-example/index/tree/master/010_tools)使用[helpers/assets/tools.go]文件。

如需更多信息，请参阅[官方Wiki](https://github.com/golang/go/wiki/Modules)以及[官方示例仓库](https://github.com/go-modules-by-example/index)。

虽然该仓库的默认分支是 `main`，但我们要求所有拉取请求都基于 `develop` 分支提交。在提交之前，请先运行单元测试并确保其全部通过。可使用 `./bin/run_units` 命令来运行这些单元测试。

**注意**：必须从仓库的根目录运行测试。

### 代码规范

我们建议 CF 接受测试的开发者遵循若干编码规范：

1. 在推送应用时：
  * 设置**内存**需求，通常使用该套件的 `DEFAULT_MEMORY_LIMIT`（在 `windows` 目录中的测试则使用 `DEFAULT_WINDOWS_MEMORY_LIMIT`），除非测试明确需要测试其他数值；
  * 设置**构建包**，除非测试明确需要测试未指定构建包的情况，此时应使用 `Config.GetRubyBuildpackName()`、`Config.GetJavaBuildpackName()` 等方法获取对应的值，
  * 除非测试明确需要使用特定于该测试的构建包名称或 URL。

  例如：

  ```go
  Expect(cf.Cf("push", appName,
      "-b", buildpackName,                  // 指定构建包
      "-m", DEFAULT_MEMORY_LIMIT,           // 指定内存限制
      "-d", Config.AppsDomain,              // 指定应用域
  ).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
  ```
1. 删除所有已创建的资源，如应用、路由、配额等。这样做的目的是让系统恢复到初始状态。例如，要删除应用及其关联的路由：
    ```
		Expect(cf.Cf("delete", myAppName, "-f", "-r").Wait()).To(Exit(0))
    ```
1. 对于应用而言，在将其销毁之前，先打印出应用的GUID以及最近的运行日志。`app_helpers`包中提供了`AppReport`这个辅助方法来实现这一功能。

    ```go
    AfterEach(func() {
      app_helpers.AppReport(appName)
    })
    ```
1. 在该仓库的README.md中记录各测试组的用途。在修改现有测试组的明确行为或新增测试组时，这一点尤为重要。
1. 在该仓库的README.md中记录对配置对象所做的所有更改。
1. 如果你添加了需要更高最低`cf` CLI版本的测试，请更新`cats_suite_test.go`文件中的`minCliVersion`值。

[networking-releases]: https://github.com/cloudfoundry-incubator/cf-networking-release/releases
[credhub-secure-service-credentials]: https://github.com/pivotal-cf/credhub-release/blob/master/docs/secure-service-credentials.md
