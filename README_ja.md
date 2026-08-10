# CF受容テスト（CATs）

<!-- hy-mt2-i18n:start -->
[English](./README.md) | [中文](./README_zh-CN.md) | **日本語** | [Español](./README_es.md)
<!-- hy-mt2-i18n:end -->

このテストセットでは、`cf` CLIおよび`curl`を使用して[Cloud Foundry](https://github.com/cloudfoundry/cf-deployment)のデプロイメントを実行します。
ユーザーが利用するエンドツーエンドの機能をテストすることを目的としています。

例えば、あるテストでは`cf push`を使ってアプリをデプロイし、`curl`を使ってアプリのエンドポイントにアクセスしてクラッシュを引き起こし、その結果`cf events`にクラッシュイベントが記録されることを確認します。

ここでは導入されないテストには、Cloud Controller内のオブジェクトに対する基本的なCRUD操作などが含まれます。このようなテストは、関連するコンポーネントに属します。

これらのテストは本番システムでの実行を目的としたものではありません。  
Cloud Foundryのリリースを開発する人々が使用する受け入れ環境向けのものです。  
これらのテストでは自動的に後処理を行おうとしますが、  
システムの状態が望ましくない形で変更されないという保証はありません。  
本番環境で安全に実行できる軽量なシステムテストについては、  
[CF Smoke Tests](https://github.com/cloudfoundry/cf-smoke-tests)をご利用ください。

**注:** 実行を並列化したいため、テストは独立して実行可能な形で記述する必要があります。テストは他のテストの状態に依存してはならず、また他のテストに影響を与えるような方法でCFの状態を変更してはなりません。

# 厳格な制約事項
1. **構造の維持**：元のMarkdownデータ構造、インデント、見出し階層、表、リンク、URL、バッジ、コードブロック、およびインラインコードを一切変更しないこと。
2. **選択的翻訳**：ユーザーに表示される可視的な自然言語コンテンツのみを翻訳すること。
3. **変更禁止**：コードタグ、キー名、変数プレースホルダー（{{var}}、${var}、%s、%dなど）、コマンド例、ファイルパス、プロジェクト名、API名、パッケージ名、モデル名、識別子、コード記号を翻訳または変更することは**厳禁**である。背景情報に対応する訳名が既に記載されている場合を除く。
4. 用語、スタイル、固有名詞の翻訳は、与えられた背景情報と一致させること。

## テストのセットアップ
### CATSを実行するための前提条件
- `golang` を `1.24.0` 以上でインストールします。[golang.org](http://golang.org/doc/install)に従って、
  golangの開発環境を構築してください。
- [`cf CLI`](https://github.com/cloudfoundry/cli) の `8.5.0` 以上をインストールします。このツールが `$PATH` 内から
  アクセス可能であることを確認してください。
- [curl](http://curl.haxx.se/) をインストールします。
- `cf-acceptance-tests` のコピーを取得してください。このプロジェクトはGoモジュールを使用しているため、
  `$GOPATH` に配置する必要はありません。
- これらの受容テストを実行するための、実行中のCloud Foundryデプロイメントを用意します。
  例えばbosh-liteなどです。

### `go`依存関係の更新
CATsで必要とされるすべての`go`依存関係は、
`vendor`ディレクトリ内にヴェンダーファイルとして格納されています。

Golang 1.24以降がインストールされていることを確認してください。

既存の依存関係を特定のバージョンにアップデートするには、次の手順を実行してください：

```bash
cd cf-acceptance-tests
source.envrc
go get <import_path>@<revision_number>
go mod vendor
```

新しい依存関係を追加したい場合は、次のコマンドを実行してください：

```bash
go mod tidy
go mod vendor
```

## テスト設定
受容テストを構成するために使用される
いくつかのデータが含まれたJSONファイルを指す
環境変数`$CONFIG`を必ず設定してください。
例えば、テストがどの実行中のCloud Foundryデプロイメントを対象とし、
どのテストを実行すべきかを指定します。

以下の内容をターミナルに貼り付けると、
CFの[BOSH-Lite](https://github.com/cloudfoundry/bosh-lite)デプロイメントを対象に
コアテストスイートを実行するのに十分な`$CONFIG`が設定されます。

参考までに[`example-cats-config.json`](example-cats-config.json)をご覧ください。

デフォルトでは、以下のテストグループのみが実行されます：
```
include_apps
include_detect
include_routing
include_v3
include_app_syslog_tcp
```

# 厳格な制約事項
1. **構造の維持**：元のMarkdownのデータ構造、インデント、見出しの階層、表、リンク、URL、バッジ、コードブロック、インラインコードを一切変更しないこと。
2. **選択的翻訳**：ユーザーに表示される可視的な自然言語内容のみを翻訳すること。
3. **変更禁止**：コードタグ、キー名、変数プレースホルダー（{{var}}、${var}、%s、%dなど）、コマンド例、ファイルパス、プロジェクト名、API名、パッケージ名、モデル名、識別子、コード記号を翻訳または変更することは**厳禁**である。背景情報に対応する訳名が既に記載されている場合を除く。
4. 用語、スタイル、固有名詞の翻訳は、与えられた背景情報と一致させること。

#### 設定パラメータの全項目については以下で説明します：
##### 必須パラメータ：
* `api`：Cloud Controller APIのエンドポイント。プロトコル（HTTP/S）は指定しない。
* `admin_user`：CFインスタンス内で管理者権限を持つユーザーの名前。この管理者ユーザーは`doppler.firehose`スコープを持っている必要がある。
* `admin_password`：上記の管理者ユーザーのパスワード。
* `apps_domain`：テストでサブドメインを作成する際に使用される共有ドメイン。そのサブドメインはテスト内で作成されたアプリケーションにリダイレクトされる。プロトコル（HTTP/S）は指定しない。
* `skip_ssl_validation`：CFインスタンスに送信されるトラフィックで無効な（例：自己署名済み）証明書を使用している場合にtrueに設定する。CFのBOSH-Liteデプロイメントの場合は通常常にtrueとなる。
* `skip_dns_validation`：CF APIおよびapps_domainのDNS検証をスキップする。プロキシ環境ではtrueを使用する。デフォルト値はfalse。

##### オプションパラメータ:
`include_*` パラメータは、デプロイメントの構成に基づいてテストをスキップするかどうかを指定するために使用されます。
* `include_app_syslog_tcp`: TCP経由でアプリのsyslogを収集するテストグループを含めるためのフラグ。
* `include_apps`: アプリケーション関連のテストグループを含めるためのフラグ。
* `readiness_health_checks_enabled`: デフォルトは `true` です。レディネスヘルスチェックがない環境を使用している場合は `false` に設定してください。
* `include_cnb`: Cloud Native Buildpacksを使用してアプリをビルドする際のテストを含めるためのフラグ。これらのテストが正常に実行されるにはDiegoがデプロイされており、CC APIのdiego_cnb機能フラグが有効になっている必要があります。また、CF CLIのバージョンは少なくともv8.14.0でなければなりません。
* `include_container_networking`: コンテナネットワーキングに関連するテストを含めるためのフラグ。
* `credhub_mode`: 有効な値は `assisted` または `non-assisted` です。[詳細はこちら](#credhub-modes)。
* `credhub_location`: CredHubインスタンスの場所。デフォルトは `https://credhub.service.cf.internal:8844` です。
* `credhub_client`: CredHubへのService Brokerからの書き込みアクセスに必要なUAAクライアント資格情報（CredHubテストに必須）。デフォルトは `credhub_admin_client` です。
* `credhub_secret`: CredHubへのService Brokerからの書き込みアクセスに必要なUAAクライアントシークレット（CredHubテストに必須）。
* `include_deployments`: クラウドコントローラーによるローリングデプロイメントに関するテストを含めるためのフラグ。V3も有効にしておく必要があります。
* `include_detect`: detectグループ内のテストを含めるためのフラグ。
* `include_docker`: Diego上でDockerアプリを実行する際のテストを含めるためのフラグ。これらのテストが正常に実行されるにはDiegoがデプロイされており、CC APIのdiego_docker機能フラグが有効になっている必要があります。
* `include_file_based_service_bindings`: ファイルベースのサービスバインディングに関するテストを含めるためのフラグ。詳細は [RFC0030](https://github.com/cloudfoundry/community/blob/main/toc/rfc/rfc-0030-add-support-for-file-based-service-binding.md) を参照してください。
* `include_http2_routing`: HTTP/2 Routingテストを含めるためのフラグ。
* `include_internet_dependent`: デプロイメントがインターネットアクセスを持つ必要があるテストを含めるためのフラグ。
* `include_isolation_segments`: アイソレーションセグメントに関するテストを含めるためのフラグ。
* `include_private_docker_registry`: プライベートなDockerイメージを利用するテストを実行するためのフラグ。[詳細はこちら](#private-docker)。
* `include_route_services`: route servicesテストを含めるためのフラグ。これらのテストが正常に実行されるにはDiegoがデプロイされている必要があります。
* `include_routing`: Routingテストを含めるためのフラグ。
* `include_ipv6`: IPv6検証テストグループを含めるためのフラグ。
* `include_routing_isolation_segments`: Routingアイソレーションセグメントに関するテストを含めるためのフラグ。[詳細はこちら](#routing-isolation-segments)。ログ記録用のアイソレーションセグメントテストと同時に実行することはできません。
* `include_security_groups`: セキュリティグループに関するテストを含めるためのフラグ。[詳細はこちら](#container-networking-and-application-security-groups)
* `dynamic_asgs_enabled`: デフォルトは `true` です。テスト環境で動的ASGが無効になっている場合は `false` に設定してください。
* `comma_delim_asgs_enabled`: デフォルトは `false` です。テスト環境でカンマ区切りのASG宛先が有効になっている場合は `true` に設定してください。
* `include_services`: services APIのテストを含めるためのフラグです。
* `include_service_instance_sharing`: スペース間でのサービスインスタンス共有に関するテストを含めるためのフラグです。これらのテストを実行するには `include_services` を必ず設定する必要があります。また、テストが合格するには `service_instance_sharing` フィーチャーフラグも有効にする必要があります。
* `include_service_credential_binding_rotation`: 複数のサービスバインディングに関するテストを実行します。詳細は [RFC-0040](https://github.com/cloudfoundry/community/blob/main/toc/rfc/rfc-0040-service-binding-rotation.md) を参照してください。このテストにはCF CLI v8.18.0以降が必要です。バックエンドは、各アプリおよびサービスインスタンスあたり少なくとも2つのサービスバインディングをサポートしている必要があります。
* `include_ssh`: Diegoコンテナのssh機能に関するテストを含めるためのフラグです。
* `include_sso`: シングルサインオンと連携するサービス関連のテストを含めるためのフラグです。
* `include_tasks`: v3タスクに関するテストを含めるためのフラグです。テストを実行するには `include_v3` も必ず設定する必要があります。また、テストが合格するにはCC APIのtask_creationフィーチャーフラグを有効にする必要があります。
* `include_tcp_routing`: TCPルーティングに関するテストを含めるためのフラグです。これらのテストは、Routing Acceptance Testsに含まれている [TCP Routing tests](https://github.com/cloudfoundry/routing-acceptance-tests/blob/master/tcp_routing/tcp_routing_test.go) と同等です。
* `tcp_domain`: TCPルーティングを使用するアプリで利用されるドメインです。
* `include_user_provided_services`: ユーザーが提供するサービスに関するテストを含めるためのフラグです。
* `include_v3`: v3 APIに関するテストを含めるためのフラグです。
* `include_zipkin`: Zipkinトレーシングに関するテストを含めるためのフラグです。テストを実行するには `include_routing` も必ず設定する必要があります。テストが合格するには、CFをデプロイする際に `router.tracing.enable_zipkin` を有効にしておく必要があります。
* `use_http`: CF Acceptance TestsがAPIやアプリケーションのリクエストを送信する際にHTTPを使用するようにしたい場合は `true` に設定してください。（デフォルトはHTTPSです）
* `use_existing_organization`: 新しい組織を作成する代わりに既存の組織を指定して使用したい場合は `true` に設定してください。
* `existing_organization`: 使用する既存の組織の名前です。
* `use_existing_user`: 上記で設定した管理者ユーザーは、通常、テスト中にアプリケーションのプッシュなどの操作を行うために、権限が限定された一時的なユーザーを作成して使用され、テスト終了後にそのユーザーは削除されます。既存のユーザーを使用したい場合は、以下のプロパティを通じて設定し、この値を `true` にしてください。
* `staticfile_`:

* `staticfile_buildpack_name` [詳細はこちら](#buildpack-names)
* `java_buildpack_name` [詳細はこちら](#buildpack-names)
* `ruby_buildpack_name` [詳細はこちら](#buildpack-names)
* `nginx_buildpack_name` [詳細はこちら](#buildpack-names)
* `nodejs_buildpack_name` [詳細はこちら](#buildpack-names)
* `go_buildpack_name` [詳細はこちら](#buildpack-names)
* `r_buildpack_name` [詳細はこちら](#buildpack-names)
* `binary_buildpack_name` [詳細はこちら](#buildpack-names)
* `cnb_nodejs_buildpack_name` [詳細はこちら](#buildpack-names)
* `python_buildpack_name: python_buildpack` [詳細はこちら](#buildpack-names)

* `include_windows`: Windows環境のセルで実行されるテストを含めるかどうかを指定するフラグです。
* `use_windows_test_task`: Windows環境のセルでのタスクテストを含めるかどうかを指定するフラグです。デフォルトは `false` です。
* `use_windows_context_path`: Windowsのコンテキストパスルーティングテストを含めるかどうかを指定するフラグです。デフォルトは `false` です。
* `windows_stack`: テストを実行するためのWindowsスタックです。

* `include_service_discovery`: サービスディスカバリーのテストを含めるかどうかを指定するフラグです。これらのテストでは `cf-networking-release` のデフォルトである `apps.internal` ドメインが使用されます。なお、この内部ドメインは現在設定可能ではありません。
* `stacks`: テスト対象となるスタックの配列です。現在は `cflinuxfs4` および `cflinuxfs5` スタックがサポートされています。デフォルト値は `[cflinuxfs4]` です。

* `include_volume_services`: ボリュームサービスのテストを含めるかどうかを指定するフラグ。このテストセットを実行するには、tcp-routingがデプロイされている必要があります。
* `volume_service_name`: ボリュームサービスブローカーによって提供されるボリュームサービスの名前。
* `volume_service_plan_name`: ボリュームサービスブローカーによって提供されるサービスのプラン名。
* `volume_service_create_config`: ボリュームサービスを作成する際に使用されるJSON設定ファイル。
* `volume_service_bind_config`: ボリュームサービスのバインド設定用のJSON設定ファイル。

Golangのユーティリティ `bin/catsconfiggenerator` があり、コード内にある既存のデフォルト設定をもとに必要に応じて完全なconfig.jsonを生成します。これを利用して、ご自身の環境に合わせて生成されたJSONファイルを自由に修正できます。

#### Buildpackの名前
多くのテストでは、アプリをプッシュする際にBuildpackを指定することで、diego上でのアプリのステージング処理がより短時間で完了します。Buildpackのデフォルト名は以下の通りです。別の名前のBuildpackを使用している場合は、異なる名前を設定することでそれらを上書きできます。

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

Cloud Native Buildpacksのライフサイクルにおいては、異なる名前を設定することでそれらを上書きできます：

* `cnb_nodejs_buildpack_name: docker://docker.io/paketobuildpacks/nodejs:latest`

#### Route Servicesテストグループの設定
`route_services`テストグループは、Cloud Foundryデプロイメントのロードバランサーにアクセスできる必要があるアプリケーションをプッシュします。これを実現するには、アプリケーションのセキュリティグループを適切に設定する必要があります。`route_services`グループを実行している場合、デプロイメントマニフェストには以下のデータを含めるべきです：

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
          destination: IP_OF_YOUR_LOAD_BALANCER # (例：BOSH-Lite上で標準的にCloud Foundryをデプロイした場合は10.244.0.34)
    default_running_security_groups: ["load_balancer"]
```

#### コンテナネットワーキングとアプリケーションセキュリティグループ
コンテナネットワーキングやアプリケーションセキュリティグループの動作をテストするには、`include_security_groups` フラグを `true` に設定する必要があります。

Windows ASGテストを実行するには、CFデプロイメントで使用されるプライベートネットワーク上に
`unallocated_ip_for_security_group`という設定値で指定された未割り当てIPが必要です。
bblによってパブリッククラウド上に作成された環境では、デフォルト値の10.0.244.255を使用できます。
vSphereやOpenStackの環境では、カスタムのIPが必要になる場合があります。

#### プライベート Docker
プライベート Docker レジストリにアクセスするための資格情報の使用をテストするには、`include_private_docker_registry` フラグを `true` に設定し、以下の設定値を指定する必要があります。

* `private_docker_registry_image`  
* `private_docker_registry_username`  
* `private_docker_registry_password`

これらのテストでは、指定されたプライベート Docker イメージが cloudfoundry/diego-docker-app:latest のプライベート版であると仮定しています。DockerHub アカウントにプライベート版をアップロードするには、まず DockerHub でプライベートリポジトリを作成し、コマンドラインから docker にログインします。その後、以下のコマンドを実行してください：

```bash
docker pull cloudfoundry/diego-docker-app:latest
docker tag cloudfoundry/diego-docker-app:latest <your-private-repo>:<some-tag>
docker push <your-private-repo>:<some-tag>
```

この場合、`private_docker_registry_image` 設定値の値は「<your-private-repo>:<some-tag>」となります。

#### Routing Isolation Segments
ルーティング分離セグメントを使用するテストを実行するには、以下の設定値を指定する必要があります：
* `isolation_segment_name`
* `isolation_segment_domain`

詳細な設定手順については、ドキュメントの[こちら](http://docs.cloudfoundry.org/adminguide/routing-is.html)をご覧ください。

#### Credhubモード
- `non-assisted`モードでは、アプリケーションが自らCredhubで資格情報の参照先を解決する責任を負います。  
  ユーザーがサービスをアプリケーションにバインドすると、サービスブローカーはCredhubに資格情報を保存し、その参照先をCloud Controllerに返します。  
  ユーザーがアプリケーションを再起動すると、Cloud Controllerは`VCAP_SERVICES`環境変数を通じてそのCredhubの参照先をアプリケーションに渡し、そこでアプリケーションは直接Credhubにリクエストを送って参照先を解決し、資格情報を取得できます。  
  このモードは`cc.credential_references.interpolate_service_bindings`の値がfalseの場合に有効となり、これはデフォルトではない設定です。  
- `assisted`モードでは、アプリケーションの実行開始前にCredhubの参照先が解決されます。  
  前と同様に、ユーザーがサービスをアプリケーションにバインドすると、サービスブローカーはCredhubに資格情報を保存し、その参照先をCloud Controllerに返します。  
  今回は、ユーザーがアプリケーションを再起動すると、Cloud ControllerはそのCredhubの参照先をDiegoランタイムに渡し、そこでランチャー（buildpackapplifecycleまたはdockerapplifecycleコンポーネントから）がCredhubの参照先を解決し、アプリケーションが利用できるように`VCAP_SERVICES`に資格情報を保存します。  
  このモードは`cc.credential_references.interpolate_service_bindings`の値がtrueの場合に有効となり、これがデフォルトの設定です。

#### テスト出力の取得
テストに失敗した場合、テスト出力内にあるテストグループ名（下記の例では`[services]`）を探してください：

```bash
• Spec Setup (BeforeEach) での失敗 [34.662秒]
[services] サービスインスタンスのライフサイクル
```

`$CONFIG` ファイル内で `artifacts_directory` の値を設定すると、失敗したテスト実行時の `cf` トレース出力を取得できるようになります。通常のテスト出力だけでは問題のデバッグが困難な場合に、この出力は役立つかもしれません。これらのスペックに含まれるテストの `cf` トレース出力は、`artifacts_directory` 内の `CF-TRACE-Applications-*.txt` に格納されます。

## テストの実行
設定に従ってテストを実行するには、`$CONFIG` の値を自分の[`integration_config.json`](#test-configuration)ファイルパスに設定し、[bin/test](./bin/test)スクリプトを実行してください。

##### 同時実行
すべてのテストグループを実行し、複数のプロセスで並行してテストを実行することが可能です。この並列処理により、CATの実行時間を大幅に短縮できます。

ただし、この数値は「一度にプッシュするアプリの数」とほぼ同義なため、十分注意が必要です。というのも、ほとんどすべての例でアプリがプッシュされているからです。

並列処理の数を設定するには `--procs` フラグを使用します。例：
```bash
./bin/test --procs=12
```

参考までに、Release Integrationチームが実行している並列プロセスの数は以下の通りです：

| 基盤タイプ | 同時実行プロセス数 |
| ----------- | ----------- |
| [Vanilla CF](https://github.com/cloudfoundry/cf-deployment/blob/master/cf-deployment.yml) | 12 |
| [BOSH Lite](https://github.com/cloudfoundry/cf-deployment/blob/master/operations/bosh-lite.yml) | 4 |


##### テストグループの絞り込み
CATsに既に慣れている場合、多数のテストグループが存在することはご存知かと思います。すべてのコンテキストで全てのテストを実行したくない場合や、障害の原因を特定するために個々のテストグループに絞り込みたい場合もあります。例えば`routing/`といった特定の受容テストグループを実行するには、[`integration_config.json`](#test-configuration)ファイルを編集し、`include_routing`を除くすべての`include_*`値を`false`に設定した後、以下のコマンドを実行してください。

```bash
./bin/test
```

単一のファイル内のテストを実行するには、そのファイル内のテストを囲むように `FDescribe` ブロックを使用します：
```go
var _ = AppsDescribe("Apps", func() {
  FDescribe("Focused tests", func() { // ここにこの行を追加します
  //... ファイルの残りの部分
  }) // ここで閉じます
})
```

テストグループの名前はディレクトリ名に対応しています。

##### 詳細出力
`ginkgo`の詳細な出力を見るには、`-v`フラグを使用します。

```bash
./bin/test -v
```

もちろん、`-v` フラグと `--procs=N` フラグを組み合わせて使用することもできます。

##### 全体テストのタイムアウト設定
Ginkgo 2.0以降、全テストのデフォルトタイムアウトが1時間に変更されました（参照：[Ginkgo 2.0マイグレーションガイド – タイムアウトの動作](https://onsi.github.io/ginkgo/MIGRATING_TO_V2#timeout-behavior)）。

テストの実行時間は、環境や並列処理の状況によって1時間を超えることがあり、失敗につながる可能性もあります。

`--timeout` フラグを使用して、必要に応じてテストのタイムアウトを調整できます：

```bash
./bin/test --timeout=24h
```

`cf push` のタイムアウトなど、個別のタイムアウトについては、[テスト設定](#test-configuration)を参照してください。

## テストグループの説明

テストグループ名 | 説明
--- | ---
`app_syslog_tcp` | アプリケーションのsyslogドレインリスナーを設定する機能をテストします。
`apps` | Cloud Foundryの核心的な機能、つまりステージング、実行、ログ記録、ルーティング、ビルドパックなどをテストします。正常にデプロイされたCloud Foundry環境では、このテストグループは常に合格すべきです。
`credhub` | サービスバインディングにおけるCredHubが提供するセキュアサービス認証情報をテストします。これらのテストを実行するには[CredHub設定](https://github.com/cloudfoundry/community/blob/main/toc/rfc/rfc-0030-add-support-for-file-based-service-binding.md)が必要です。`credhub_mode`を指定するほか、`credhub_client`および`credhub_secret`の値も必要となります。
`cnb` | クラウドネイティブなビルドパックを使用する能力をテストします。
`detect` | 明示的にビルドパックが指定されていない場合でも、プラットフォームがアプリケーションをコンパイルするための適切なビルドパックを検出できるかをテストします。
`docker` | Diego上でDockerコンテナを実行する能力や、Dockerメタデータを正しく処理できるかをテストします。
`ipv6` | IPv6エグレス呼び出しをテストします。
`file-based service bindings` | ビルドパックアプリ、CNBアプリ、Dockerアプリにおけるファイルベースのサービスバインディングをテストします。このテストグループは、異なる2つのスタック（Windows、Linux）上で2つの異なるフィーチャーフラグを使って実行されます。フィーチャーフラグに関する詳細は[RFC0030](https://github.com/cloudfoundry/community/blob/main/toc/rfc/rfc-0030-add-support-for-file-based-service-binding.md)をご覧ください。
`internet_dependent` | Github URLを通じてビルドパックを指定できる機能をテストします。したがって、これはCloud Foundryアプリケーションコンテナがインターネットにアクセスできることに依存します。Cloud Foundryをデプロイしたネットワークの設定や、アプリケーションコンテナに適用されているセキュリティグループの設定も考慮する必要があります。
`isolation_segments` | このテストグループを実行するには、Diegoを少なくとも2つのセルでデプロイする必要があります。そのうちの1つのセルには`placement_tag`が設定されていなければなりません。デプロイにルーティング分離セグメントが使用されている場合は、`isolation_segment_domain`も設定する必要があります。詳細については[Isolation Segmentsのドキュメント](https://docs.cloudfoundry.org/adminguide/isolation-segments.html)を参照してください。
`route_services` | Cloud Foundryの[Route Services](https://docs.cloudfoundry.org/services/route-services.html)機能をテストします。
`routing` | このパッケージには、ルーティングに関連する受容テスト（コンテキストパス、ワイルドカード、SSL終端処理、スティッキーセッション、Zipkinトレーシングなど）が含まれています。
`routing_isolation_segments` | 分離されたアプリケーションへのリクエストが必ず分離されたルーター経由でのみ送信され、その逆も同様であることをテストします。このテストには分離セグメントテストスイートのための全ての設定が必要です。さらに、少なくとも2つのGorouterインスタンスをデプロイする必要があります。1つのインスタンスには`routing_table_sharding_mode: shared-and-segments`というプロパティを設定し、もう1つのインスタンスには`routing_table_sharding_mode: segments`および`isolation_segments: [YOUR_PLACEMENT_TAG_HERE]`というプロパティを設定します。CATsプロパティ内の`isolation_segment_name`は`placement_tag`と一致していなければならず、`isolation_segment`.`isolation_segment_domain`は必ず設定され、そのドメインへのトラフィックは分離されたルーターに送信されるべきです。
`security_groups`| Cloud Foundryの[Security Groups](https://docs.cloudfoundry.org/concepts/asg.html)機能をテストします。
`service_discovery`| Cloud Foundry上で実行されるアプリケーション向けの[Service Discovery](https://docs.cloudfoundry.org/devguide/deploy-apps/cf-networking.html#discovery)機能をテストします。
`services`| サービスに関連する様々な機能をテストします。例えば、サービスブローカーAPIを通じてサービスブローカーを登録するなどです。これらのテストの中には、シングルサインオン認証などの特別な統合機能を利用するものもあります。必要な統合機能を設定していない場合（設定ファイル内で`include_sso`パラメータを`false`に設定することで）、このパッケージ内の一部のテストのみを実行し、他のテストは選択的にスキップすることも可能です。
`ssh`| ssh、scp、sftpを介したDiegoアプリケーションとの通信をテストします。
`tasks`| Cloud Foundryの[Tasks](https://docs.cloudfoundry.org/devguide/using-tasks.html)機能をテストします。
`tcp_routing`| Cloud FoundryのTCPルーティング機能をテストします。[こちら](https://docs.cloudfoundry.org/adminguide/enabling-tcp-routing.html)に記載されている通り、`tcp.<SYSTEM_DOMAIN>`というTCPドメインを事前に設定しておく必要があります。`bbl`（BOSH Bootloader）を使用している場合は、TCPドメインが自動的に設定されます。
`user_provided_services` | アプリケーションの認証情報を安全に保管するための、ユーザー提供型サービスの作成やバインドに関連する機能をテストします。
`v3`| このテストグループには、次世代のv3 Cloud Controller APIに関するテストが含まれています。
`volume_services` | Cloud Foundryの[Volume Services](https://docs.cloudfoundry.org/devguide/services/using-vol-services.html)機能をテストします。

## 貢献方法

このリポジトリでは、`go`依存関係を管理するために[go mod](https://golang.org/cmd/go/#hdr-Modules__module_versions__and_more)を使用しています。

CATsで必要とされるすべての`go`依存関係は、`vendor`ディレクトリに格納されています。

テストスイートに変更を加えて新たな `go` パッケージを導入する場合は、以下のワークフローを使用すべきです：

依存関係の最新バージョンを利用できる場合は `go mod tidy` を使用し、そうでない場合は `go get <dependency>@<version>` を使用してください。いずれの方法も、[envrc](.envrc) を通じて go modules が有効になっている必要があります。最後に `go mod vendor` を使って、その依存関係を `vendor` ディレクトリに追加してください。

ツールやアセットについては、[go mod tool workflow](https://github.com/go-modules-by-example/index/tree/master/010_tools) を通じて [helpers/assets/tools.go](helpers/assets/tools.go) をご利用ください。

詳細については、[公式Wiki](https://github.com/golang/go/wiki/Modules)および[公式サンプルリポジトリ](https://github.com/go-modules-by-example/index)をご参照ください。

このリポジトリのデフォルトブランチは`main`ですが、すべてのプルリクエストは`develop`ブランチを対象にしていただくようお願いします。提出する前に単体テストを実行し、正常に通過していることを確認してください。これらの単体テストを実行するには`./bin/run_units`を使用してください。

**注**：リポジトリのルートからテストを実行する必要があります。

### コードの規約

CF受け入れテストの開発者には、採用していただきたいいくつかの規約があります：

1. アプリをプッシュする際：
  * **メモリ**要件を設定し、テストで特に別の値をテストする必要がない限り、スイート内の`DEFAULT_MEMORY_LIMIT`（`windows`ディレクトリ内のテストの場合は`DEFAULT_WINDOWS_MEMORY_LIMIT`）を使用します。
  * テストで特にビルドパックを指定しないケースをテストする必要がない限り、**ビルドパック**を設定し、テストで特にそのテスト専用のビルドパック名やURLを使用する必要がない限り、`Config.GetRubyBuildpackName()`、`Config.GetJavaBuildpackName()`などの関数を利用します。

  例：

  ```go
  Expect(cf.Cf("push", appName,
      "-b", buildpackName,                  // ビルドパックを指定
      "-m", DEFAULT_MEMORY_LIMIT,           // メモリ制限を指定
      "-d", Config.AppsDomain,              // アプリケーションドメインを指定
  ).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
  ```
1. アプリケーション、ルート、クォータなど、作成されたすべてのリソースを削除します。これによりシステムは元の状態のままになります。アプリケーションとそれに関連するルートを削除する例：
    ```
		Expect(cf.Cf("delete", myAppName, "-f", "-r").Wait()).To(Exit(0))
    ```
1. アプリケーションの場合、削除する前にそのGUIDと最近のアプリケーションログを出力してください。この目的のために `app_helpers` パッケージに `AppReport` というヘルパーメソッドが用意されています。

    ```go
    AfterEach(func() {
      app_helpers.AppReport(appName)
    })
    ```
1. このリポジトリの README.md に、各テストグループの目的を記載してください。既存のテストグループの動作を変更したり、新しいテストグループを追加したりする際には、これが特に重要です。
1. このリポジトリの README.md に、configオブジェクトに加えられたすべての変更内容を記載してください。
1. 新しい最小 `cf` CLIバージョンが必要なテストを追加する場合は、`cats_suite_test.go` 内の `minCliVersion` を更新してください。

[networking-releases]: https://github.com/cloudfoundry-incubator/cf-networking-release/releases
[credhub-secure-service-credentials]: https://github.com/pivotal-cf/credhub-release/blob/master/docs/secure-service-credentials.md
