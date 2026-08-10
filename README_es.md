# Pruebas de aceptación de CF (CATs)

<!-- hy-mt2-i18n:start -->
[English](./README.md) | [中文](./README_zh-CN.md) | [日本語](./README_ja.md) | **Español**
<!-- hy-mt2-i18n:end -->

Este conjunto de pruebas utiliza la CLI `cf` y `curl` para probar una implementación de [Cloud Foundry](https://github.com/cloudfoundry/cf-deployment).
Está destinado exclusivamente a probar las funcionalidades de extremo a extremo orientadas al usuario.

Por ejemplo, una prueba envía una aplicación mediante `cf push`, hace una solicitud a un endpoint de dicha aplicación con `curl` lo que provoca su caída, y verifica que aparezca un evento de fallo en `cf events`.

Las pruebas que _no se incluirán_ aquí son, por ejemplo, las operaciones básicas de CRUD de un objeto en el Cloud Controller. Dichas pruebas pertenecen al componente al que están relacionadas.

Estas pruebas no están destinadas para utilizarse en sistemas en producción. Están pensadas para entornos de aceptación que emplean quienes desarrollan las versiones de Cloud Foundry. Aunque estas pruebas intentan dejar todo en orden tras ejecutarse, no hay garantía de que no modifiquen el estado de su sistema de manera indeseada. Para pruebas ligeras de sistema que se pueden ejecutar sin riesgo en un entorno de producción, utilice las [Pruebas de humo de CF](https://github.com/cloudfoundry/cf-smoke-tests).

**NOTA:** Dado que queremos paralelizar la ejecución,  
los tests deben estar escritos de manera que puedan ejecutarse de forma independiente.  
No deben depender del estado de otros tests,  
ni modificar el estado de Cloud Foundry de tal forma que afecte a otros tests.

# Restricciones estrictas
1. **Bloqueo estructural**: Mantener absolutamente intacta la estructura de datos Markdown original, los sangrados, los niveles de título, las tablas, los enlaces, las URL, las insignias, los bloques de código y el código inline.
2. **Traducción selectiva**: Solo traducir el contenido de lenguaje natural visible para el usuario.
3. **Prohibición de modificaciones**: Está **estrictamente prohibido** traducir o cambiar etiquetas de código, nombres de claves, placeholders de variables (como {{var}}, ${var}, %s, %d, etc.), ejemplos de comandos, rutas de archivos, nombres de proyectos, nombres de API, nombres de paquetes, nombres de modelos, identificadores y símbolos de código; a menos que la información de contexto ya proporcione una traducción correspondiente.
4. La traducción de términos, estilos y nombres propios debe ser coherente con la información de contexto proporcionada.

## Configuración de pruebas
### Requisitos previos para ejecutar CATS
- Instale golang >= `1.24.0`. Configure su entorno de desarrollo de golang siguiendo las instrucciones en
  [golang.org](http://golang.org/doc/install).
- Instale la versión [`cf CLI`](https://github.com/cloudfoundry/cli) >= `8.5.0`. Asegúrese de que esté disponible en su `$PATH`.
- Instale [curl](http://curl.haxx.se/).
- Descargue una copia de `cf-acceptance-tests`. Dado que utiliza módulos Go, no es necesario colocarla en `$GOPATH`.
- Instale una implementación de Cloud Foundry en ejecución contra la cual realizar estas pruebas de aceptación; por ejemplo, bosh-lite.

### Actualización de las dependencias de `go`
Todas las dependencias de `go` que requiere CATS
se encuentran en el directorio `vendor`.

Asegúrese de tener Golang 1.24 o superior.

Para actualizar una dependencia actual a una versión específica, siga estos pasos:

```bash
cd cf-acceptance-tests
source.envrc
go get <import_path>@<revision_number>
go mod vendor
```

Si desea agregar una nueva dependencia, simplemente ejecute:

```bash
go mod tidy
go mod vendor
```

## Configuración de pruebas
Debe establecer la variable de entorno `$CONFIG`, que apunta a un archivo JSON que contiene varios datos que se utilizarán para configurar las pruebas de aceptación; por ejemplo, para indicarle a las pruebas cómo dirigirse a su implementación de Cloud Foundry en ejecución y qué pruebas ejecutar.

Se puede pegar lo siguiente en una terminal,
lo cual configurará un `$CONFIG` adecuado
para ejecutar los conjuntos de pruebas principales
contra una implementación de CF basada en [BOSH-Lite](https://github.com/cloudfoundry/bosh-lite).

Consulte [`example-cats-config.json`](example-cats-config.json) como referencia.

De forma predeterminada, solo se ejecutan los siguientes grupos de pruebas:
```
include_apps
include_detect
include_routing
include_v3
include_app_syslog_tcp
```

#### A continuación se explica el conjunto completo de parámetros de configuración:
##### Parámetros obligatorios:
* `api`: Punto final de la API del Cloud Controller, sin especificar el esquema (HTTP/S).
* `admin_user`: Nombre de un usuario en su instancia de CF con credenciales de administrador. Este usuario administrador debe tener el ámbito `doppler.firehose`.
* `admin_password`: Contraseña del usuario administrador mencionado anteriormente.
* `apps_domain`: Un dominio compartido que las pruebas pueden utilizar para crear subdominios que redirijan a aplicaciones también creadas en las pruebas, sin especificar el esquema (HTTP/S).
* `skip_ssl_validation`: Establezca este valor en `true` si utiliza un certificado inválido (por ejemplo, autofirmado) para el tráfico dirigido a su instancia de CF; generalmente este valor siempre es `true` en las implementaciones BOSH-Lite de CF.
* `skip_dns_validation`: Omitir la validación DNS para la API de CF y el dominio de las aplicaciones. Utilice `true` en entornos de proxy. Valor por defecto: `false`.

##### Parámetros opcionales:
Los parámetros `include_*` se utilizan para especificar si se deben omitir pruebas en función de cómo está configurada la implementación.
* `include_app_syslog_tcp`: Bandera para incluir el grupo de pruebas de drenaje de syslog de la aplicación por TCP.
* `include_apps`: Bandera para incluir el grupo de pruebas de las aplicaciones.
* `readiness_health_checks_enabled`: Tiene como valor predeterminado `true`. Establezca `false` si utiliza un entorno sin comprobaciones de salud de preparación.
* `include_cnb`: Bandera para incluir pruebas relacionadas con la construcción de aplicaciones mediante Cloud Native Buildpacks. Para que estas pruebas se completen, debe estar desplegado Diego y habilitada la bandera de características CC API diego_cnb. La versión de CF CLI debe ser al menos v8.14.0.
* `include_container_networking`: Bandera para incluir pruebas relacionadas con las redes de contenedores.
* `credhub_mode`: Los valores válidos son `assisted` o `non-assisted`. [Véase más abajo](#credhub-modes).
* `credhub_location`: Ubicación de la instancia de CredHub; el valor predeterminado es `https://credhub.service.cf.internal:8844`.
* `credhub_client`: Credenciales del cliente UAA para que Service Broker tenga acceso de escritura a CredHub (necesario para las pruebas de CredHub); el valor predeterminado es `credhub_admin_client`.
* `credhub_secret`: Secreto del cliente UAA para que Service Broker tenga acceso de escritura a CredHub (necesario para las pruebas de CredHub).
* `include_deployments`: Bandera para incluir pruebas para las implementaciones por rollo del controlador en la nube. También debe estar habilitada la versión V3.
* `include_detect`: Bandera para incluir pruebas en el grupo detect.
* `include_docker`: Bandera para incluir pruebas relacionadas con la ejecución de aplicaciones Docker en Diego. Para que estas pruebas se completen, debe estar desplegado Diego y habilitada la bandera de características CC API diego_docker.
* `include_file_based_service_bindings`: Bandera para incluir pruebas de vinculación de servicios basadas en archivos. Para más detalles, consulte [RFC0030](https://github.com/cloudfoundry/community/blob/main/toc/rfc/rfc-0030-add-support-for-file-based-service-binding.md).
* `include_http2_routing`: Bandera para incluir las pruebas de enrutamiento HTTP/2.
* `include_internet_dependent`: Bandera para incluir pruebas que requieren que la implementación tenga acceso a Internet.
* `include_isolation_segments`: Bandera para incluir pruebas de segmentos de aislamiento.
* `include_private_docker_registry`: Bandera para ejecutar pruebas que dependen de una imagen Docker privada. [Véase más abajo](#private-docker).
* `include_route_services`: Bandera para incluir pruebas de servicios de enrutamiento. Para que estas pruebas se completen, debe estar desplegado Diego.
* `include_routing`: Bandera para incluir pruebas de enrutamiento.
* `include_ipv6`: Bandera para incluir el grupo de pruebas de validación de IPv6.
* `include_routing_isolation_segments`: Bandera para incluir pruebas de segmentos de aislamiento de enrutamiento. [Véase más abajo](#routing-isolation-segments). No se pueden ejecutar junto con las pruebas de segmentos de aislamiento de registro.
* `include_security_groups`: Bandera para incluir pruebas para los grupos de seguridad. [Véase más abajo](#container-networking-and-application-security-groups).
* `dynamic_asgs_enabled`: Tiene como valor por defecto `true`. Establezca este valor en `false` si los ASGs dinámicos están deshabilitados en el entorno de pruebas.  
* `comma_delim_asgs_enabled`: Tiene como valor por defecto `false`. Establezca este valor en `true` si los destinos de ASG delimitados por comas están habilitados en el entorno de pruebas.  
* `include_services`: Bandera para incluir pruebas para la API de servicios.  
* `include_service_instance_sharing`: Bandera para incluir pruebas de intercambio de instancias de servicio entre espacios. Es necesario establecer `include_services` para que estas pruebas se ejecuten. Además, la bandera de característica `service_instance_sharing` también debe estar habilitada para que estas pruebas aprueben.  
* `include_service_credential_binding_rotation`: Ejecuta pruebas para múltiples vinculaciones de servicios. Consulte [RFC-0040](https://github.com/cloudfoundry/community/blob/main/toc/rfc/rfc-0040-service-binding-rotation.md) para obtener detalles. Esta prueba requiere CF CLI v8.18.0 o una versión posterior. El backend debe soportar al menos 2 vinculaciones de servicio por aplicación e instancia de servicio.  
* `include_ssh`: Bandera para incluir pruebas para la función de ssh de contenedores Diego.  
* `include_sso`: Bandera para incluir las pruebas de servicios que se integran con el inicio de sesión único.  
* `include_tasks`: Bandera para incluir pruebas de tareas de versión v3. También es necesario establecer `include_v3` para que las pruebas se ejecuten. La bandera de característica CC API task_creation debe estar habilitada para que estas pruebas aprueben.  
* `include_tcp_routing`: Bandera para incluir pruebas de enrutamiento TCP. Estas pruebas son equivalentes a las [pruebas de enrutamiento TCP](https://github.com/cloudfoundry/routing-acceptance-tests/blob/master/tcp_routing/tcp_routing_test.go) de las Pruebas de Aceptación de Enrutamiento.  
* `tcp_domain`: Dominio que se utilizará para las aplicaciones con rutas TCP.  
* `include_user_provided_services`: Bandera para incluir pruebas para servicios proporcionados por el usuario.  
* `include_v3`: Bandera para incluir pruebas para la API de versión v3.  
* `include_zipkin`: Bandera para incluir pruebas de seguimiento Zipkin. También es necesario establecer `include_routing` para que las pruebas se ejecuten. CF debe estar desplegado con `router.tracing.enable_zipkin` habilitado para que las pruebas aprueben.  
* `use_http`: Establezca este valor en `true` si desea que las Pruebas de Aceptación de CF utilicen HTTP al realizar solicitudes a APIs y aplicaciones. (El valor por defecto es HTTPS).  
* `use_existing_organization`: Establezca este valor en `true` cuando necesite especificar una organización existente en lugar de crear una nueva.  
* `existing_organization`: Nombre de la organización existente que se utilizará.  
* `use_existing_user`: Normalmente, el usuario administrador configurado anteriormente se utiliza para crear un usuario temporal (con menos permisos) con el fin de realizar acciones (como enviar aplicaciones) durante las pruebas, y luego se elimina dicho usuario una vez finalizadas las pruebas; establezca este valor en `true` si desea utilizar un usuario existente, configurado a través de las propiedades siguientes.
* `staticfile_`:

* `staticfile_buildpack_name` [Véase más abajo](#buildpack-names)
* `java_buildpack_name` [Véase más abajo](#buildpack-names)
* `ruby_buildpack_name` [Véase más abajo](#buildpack-names)
* `nginx_buildpack_name` [Véase más abajo](#buildpack-names)
* `nodejs_buildpack_name` [Véase más abajo](#buildpack-names)
* `go_buildpack_name` [Véase más abajo](#buildpack-names)
* `r_buildpack_name` [Véase más abajo](#buildpack-names)
* `binary_buildpack_name` [Véase más abajo](#buildpack-names)
* `cnb_nodejs_buildpack_name` [Véase más abajo](#buildpack-names)
* `python_buildpack_name: python_buildpack` [Véase más abajo](#buildpack-names)

* `include_windows`: Bandera para incluir las pruebas que se ejecutan en celdas Windows.  
* `use_windows_test_task`: Bandera para incluir las tareas de prueba en celdas Windows. El valor predeterminado es `false`.  
* `use_windows_context_path`: Bandera para incluir las pruebas de enrutamiento con la ruta de contexto de Windows. El valor predeterminado es `false`.  
* `windows_stack`: Pila Windows contra la cual se ejecutarán las pruebas.

* `include_service_discovery`: Bandera para incluir pruebas de detección de servicios. Estas pruebas utilizan el dominio `apps.internal`, que es el predeterminado en `cf-networking-release`. Actualmente, el dominio interno no es configurable.
* `stacks`: Un array de stacks contra los cuales se realizarán las pruebas. Actualmente se admiten los stacks `cflinuxfs4` y `cflinuxfs5`. El valor predeterminado es `[cflinuxfs4]`.

* `include_volume_services`: Bandera para incluir las pruebas relacionadas con los servicios de volumen. Se deben cumplir los siguientes requisitos para ejecutar este conjunto de pruebas: debe estar desplegado tcp-routing.
* `volume_service_name`: El nombre del servicio de volumen proporcionado por el broker de servicios de volumen.
* `volume_service_plan_name`: El nombre del plan del servicio proporcionado por el broker de servicios de volumen.
* `volume_service_create_config`: La configuración en formato JSON que se utiliza al crear un servicio de volumen.
* `volume_service_bind_config`: La configuración en formato JSON para la asociación del servicio de volumen.

Existe una utilidad en Golang llamada `bin/catsconfiggenerator` que genera un archivo config.json completo según sea necesario, a partir de los valores predeterminados existentes en el código. Puede utilizarla y modificar el archivo JSON resultante como lo considere adecuado para su entorno.

#### Nombres de los buildpack
Muchas pruebas especifican un buildpack al publicar una aplicación, para que el proceso de preparación de la misma en Diego se complete en menos tiempo. Los nombres predeterminados de los buildpack son los siguientes; si cuenta con buildpacks con nombres diferentes, puede sobrescribirlos estableciendo nombres distintos:

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

Para el ciclo de vida de los Cloud Native Buildpacks, puede sobrescribirlos estableciendo nombres diferentes:

* `cnb_nodejs_buildpack_name: docker://docker.io/paketobuildpacks/nodejs:latest`

#### Configuración del grupo de pruebas Route Services  
El grupo de pruebas `route_services` envía aplicaciones que deben poder conectarse al balanceador de carga de su implementación de Cloud Foundry. Para ello es necesario configurar los grupos de seguridad de las aplicaciones para que lo permitan. Si está ejecutando el grupo `route_services`, su manifiesto de implementación debe incluir los siguientes datos:

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
          destination: IP_DE_TU/load_balancera # (p. ej., 10.244.0.34 para una implementación estándar de Cloud Foundry en BOSH-Lite)
    default_running_security_groups: ["load_balancer"]
```

#### Redes de contenedores y grupos de seguridad de aplicaciones
Para ejecutar pruebas que prueben las redes de contenedores y el funcionamiento de los grupos de seguridad de aplicaciones, la opción `include_security_groups` debe estar configurada en true.

Las pruebas de ASG en Windows requieren una IP no asignada en la red privada utilizada por la implementación de CF, la cual se establece mediante el valor de configuración `unallocated_ip_for_security_group`. Los entornos creados por bbl en nubes públicas pueden utilizar el valor predeterminado de 10.0.244.255. Los entornos de vSphere y OpenStack podrían necesitar una IP personalizada.

#### Docker privado
Para ejecutar pruebas que prueben el uso de credenciales para acceder a un registro Docker privado, la marca `include_private_docker_registry` debe estar activada, y se deben proporcionar los siguientes valores de configuración:

* `private_docker_registry_image`  
* `private_docker_registry_username`  
* `private_docker_registry_password`

Estas pruebas suponen que la imagen Docker privada especificada es una versión privada de cloudfoundry/diego-docker-app:latest. Para cargar una versión privada en su cuenta de DockerHub, primero cree un repositorio privado en DockerHub e inicie sesión en Docker desde la línea de comandos. Luego ejecute los siguientes comandos:

```bash
docker pull cloudfoundry/diego-docker-app:latest
docker tag cloudfoundry/diego-docker-app:latest <your-private-repo>:<some-tag>
docker push <your-private-repo>:<some-tag>
```

En este caso, el valor de la configuración `private_docker_registry_image` sería “<your-private-repo>:<some-tag>”.

#### Segmentos de Aislamiento de Enrutamiento
Para ejecutar pruebas que involucren segmentos de aislamiento de enrutamiento, es necesario proporcionar los siguientes valores de configuración:
* `isolation_segment_name`
* `isolation_segment_domain`

Lea la documentación [aquí](http://docs.cloudfoundry.org/adminguide/routing-is.html) para obtener más detalles sobre la configuración.

#### Modos de Credhub
- El modo `non-assisted` implica que las aplicaciones son responsables de resolver las referencias de Credhub para las credenciales. Cuando el usuario vincula un servicio a una aplicación, el service broker almacenará la credencial en Credhub y devolverá la referencia al Cloud Controller. Al reiniciar la aplicación, el Cloud Controller pasará la referencia de Credhub a la aplicación a través de la variable de entorno `VCAP_SERVICES`, permitiendo así que la aplicación haga una solicitud directa a Credhub para resolver la referencia y obtener la credencial. Este modo se activa cuando `cc.credential_references.interpolate_service_bindings` es falso, lo cual corresponde a la configuración no predeterminada.
- El modo `assisted` significa que la referencia de Credhub se resolverá antes de que comience a ejecutarse la aplicación. Al igual que antes, cuando el usuario vincula un servicio a una aplicación, el service broker almacenará la credencial en Credhub y devolverá la referencia al Cloud Controller. Esta vez, al reiniciar la aplicación, el Cloud Controller pasará la referencia de Credhub al entorno de ejecución de Diego; a continuación, el iniciador (proveniente de los componentes buildpackapplifecycle o dockerapplifecycle) resolverá la referencia de Credhub y almacenará la credencial en `VCAP_SERVICES` para que la aplique la aplicación. Este modo se activa cuando `cc.credential_references.interpolate_service_bindings` es verdadero, lo cual es la configuración predeterminada.

#### Captura de la salida de las pruebas
Cuando una prueba falla, busque el nombre del grupo de pruebas («[services]» en el ejemplo a continuación) en la salida de dicha prueba:

```bash
• Fallo en la configuración de las pruebas (BeforeEach) [34.662 segundos]
[services] Ciclo de vida de la instancia del servicio
```

Si establece un valor para `artifacts_directory` en su archivo `$CONFIG`, podrá capturar la salida de seguimiento `cf` de las ejecuciones de pruebas fallidas; esta salida puede ser útil cuando la salida normal de las pruebas no es suficiente para depurar un problema. La salida de seguimiento `cf` de las pruebas de estas especificaciones se encontrará en los archivos `CF-TRACE-Applications-*.txt` dentro del `artifacts_directory`.

## Ejecución de pruebas
Para ejecutar las pruebas según su configuración, ejecute el script [bin/test](./bin/test)
con `$CONFIG` establecido en la ruta de su archivo
[`integration_config.json`](#test-configuration).

##### Ejecución en paralelo
Es posible ejecutar todos los grupos de pruebas y hacer que estas se realicen en paralelo a través de múltiples procesos. Esta paralelización puede reducir significativamente el tiempo de ejecución de los CATs.

Sin embargo, tenga cuidado con este valor, ya que en la práctica representa “cuántas aplicaciones se deben enviar al mismo tiempo”, dado que casi todos los ejemplos envían una aplicación.

Utilice la bandera `--procs` para establecer la cantidad de procesos en paralelo, por ejemplo:  
```bash
./bin/test --procs=12
```

A modo de referencia, aquí se indica cuántos procesos en paralelo utiliza el equipo de integración de lanzamientos:

| Tipo de Foundation | Número de procesos en paralelo |
| ----------- | ----------- |
| [Vanilla CF](https://github.com/cloudfoundry/cf-deployment/blob/master/cf-deployment.yml) | 12 |
| [BOSH Lite](https://github.com/cloudfoundry/cf-deployment/blob/master/operations/bosh-lite.yml) | 4 |


##### Enfoque en grupos de pruebas
Si ya está familiarizado con CATs, probablemente sepa que existen muchos grupos de pruebas. Es posible que no desee ejecutar todas las pruebas en todos los contextos, y a veces querrá enfocarse en grupos de pruebas individuales para identificar una falla específica. Para ejecutar un grupo concreto de pruebas de aceptación, por ejemplo `routing/`, edite su archivo [`integration_config.json`](#test-configuration) y establezca todos los valores de `include_*` en `false`, excepto `include_routing`, y luego ejecute lo siguiente:

```bash
./bin/test
```

Para ejecutar pruebas en un único archivo, utilice un bloque `FDescribe` alrededor de las pruebas de dicho archivo:  
```go
var _ = AppsDescribe("Apps", func() {
  FDescribe("Pruebas específicas", func() { // Añada esta línea aquí
  //... resto del archivo
  }) // Cierre aquí
})
```

Los nombres de los grupos de pruebas corresponden a los nombres de las carpetas.

##### Salida detallada
Para ver la salida detallada de `ginkgo`, utilice la opción `-v`.

```bash
./bin/test -v
```

Por supuesto, puede combinar la opción `-v` con la opción `--procs=N`.

##### Configuración general del tiempo de espera de las pruebas
A partir de Ginkgo 2.0, el tiempo de espera predeterminado para todas las pruebas se ha cambiado a 1 hora (consulte: [Guía de migración a Ginkgo 2.0 - Comportamiento del tiempo de espera](https://onsi.github.io/ginkgo/MIGRATING_TO_V2#timeout-behavior)).

Dependiendo del entorno y el nivel de paralelismo, las pruebas pueden ejecutarse durante más de una hora, lo que puede conllevar fallos.

Ajuste el tiempo de espera de las pruebas según sea necesario con la opción `--timeout`:

```bash
./bin/test --timeout=24h
```

Para los tiempos de espera individuales, como el tiempo de espera de `cf push`, consulte [Configuración de pruebas](#test-configuration).

## Explicación de los grupos de pruebas

Nombre del grupo de pruebas | Descripción
--- | ---
`app_syslog_tcp` | Prueba la capacidad de configurar un receptor de logs syslog para una aplicación.
`apps` | Prueba las funcionalidades principales de Cloud Foundry: entorno de preparación, ejecución, registro de logs, enrutamiento, buildpacks, etc. Este grupo de pruebas siempre debe superarse en una implementación correcta de Cloud Foundry.
`credhub` | Prueba las credenciales de servicios seguros proporcionadas por CredHub en los vínculos de servicio. Se requiere [configuración de CredHub](https://docs.cloudfoundry.org/adminguide/credhub-secure-service-credentials.html) para ejecutar estas pruebas. Además de seleccionar un `credhub_mode`, también se necesitan valores para `credhub_client` y `credhub_secret`.
`cnb` | Prueba nuestra capacidad para utilizar buildpacks nativos de la nube.
`detect` | Prueba la capacidad de la plataforma para detectar el buildpack adecuado para compilar una aplicación cuando no se especifica explícitamente ninguno.
`docker` | Prueba nuestra capacidad para ejecutar contenedores Docker en Diego y que gestionemos correctamente sus metadatos.
`ipv6` | Prueba las llamadas de salida mediante IPv6.
`file-based service bindings` | Prueba los vínculos de servicio basados en archivos para aplicaciones con buildpacks, aplicaciones CNB y aplicaciones Docker. Este grupo de pruebas se ejecuta con 2 indicadores de función diferentes en 2 entornos distintos (Windows, Linux). Para más detalles sobre estos indicadores de función, consulte [RFC0030](https://github.com/cloudfoundry/community/blob/main/toc/rfc/rfc-0030-add-support-for-file-based-service-binding.md).
`internet_dependent` | Prueba la posibilidad de especificar un buildpack a través de una URL de Github. Por lo tanto, esto depende de que los contenedores de la aplicación Cloud Foundry tengan acceso a Internet. Debe tener en cuenta la configuración de la red en la que ha desplegado su Cloud Foundry, así como cualquier configuración de grupos de seguridad aplicada a los contenedores de la aplicación.
`isolation_segments` | Este grupo de pruebas requiere que Diego esté desplegado con al menos 2 celdas. Una de esas celdas debe haberse desplegado con una `placement_tag`. Si la implementación se realizó con un segmento de aislamiento de enrutamiento, también se debe establecer `isolation_segment_domain`. Para más información, consulte la [documentación sobre segmentos de aislamiento](https://docs.cloudfoundry.org/adminguide/isolation-segments.html).
`route_services` | Prueba la función [Route Services](https://docs.cloudfoundry.org/services/route-services.html) de Cloud Foundry.
`routing` | Este paquete contiene pruebas de aceptación específicas relacionadas con el enrutamiento (rutas de contexto, comodines, terminación SSL, sesiones persistentes y seguimiento con Zipkin).
`routing_isolation_segments` | Prueba que las solicitudes dirigidas a aplicaciones aisladas solo se enruten a través de routers aislados, y viceversa. Requiere toda la configuración necesaria para el conjunto de pruebas de segmentos de aislamiento. Además, deben desplegarse al menos dos instancias de Gorouter. Una instancia debe configurarse con la propiedad `routing_table_sharding_mode: shared-and-segments`. La otra instancia debe tener las propiedades `routing_table_sharding_mode: segments` y `isolation_segments: [YOUR_PLACEMENT_TAG_HERE]`. El `isolation_segment_name` en las propiedades de CATs debe coincidir con el `placement_tag`, y `isolation_segment`.`isolation_segment_domain` debe estar definido; el tráfico dirigido a ese dominio debe llegar al router aislado.
`security_groups`| Prueba la función [Security Groups](https://docs.cloudfoundry.org/concepts/asg.html) de Cloud Foundry.
`service_discovery`| Prueba la función [Service Discovery](https://docs.cloudfoundry.org/devguide/deploy-apps/cf-networking.html#discovery) para aplicaciones que se ejecutan en Cloud Foundry.
`services`| Prueba diversas funciones relacionadas con los servicios, como por ejemplo, el registro de un broker de servicios mediante su API. Algunas de estas pruebas utilizan integraciones especiales, como la autenticación de inicio de sesión único; es posible que desee ejecutar algunas pruebas de este paquete pero omitir otras selectivamente si no ha configurado las integraciones requeridas (configurando el parámetro `include_sso` en `false` en su configuración).
`ssh`| Prueba la comunicación con aplicaciones de Diego mediante ssh, scp y sftp.
`tasks`| Prueba la función [Tasks](https://docs.cloudfoundry.org/devguide/using-tasks.html) de Cloud Foundry.
`tcp_routing`| Prueba la función de enrutamiento TCP de Cloud Foundry. Debe asegurarse de haber configurado un dominio TCP `tcp.<SYSTEM_DOMAIN>` tal como se describe [aquí](https://docs.cloudfoundry.org/adminguide/enabling-tcp-routing.html). Si utiliza `bbl` (BOSH Bootloader), el dominio TCP se configura automáticamente para usted.
`user_provided_services` | Prueba funciones relacionadas con la creación y vinculación de servicios proporcionados por el usuario para almacenar de forma segura las credenciales de las aplicaciones.
`v3`| Este grupo de pruebas contiene pruebas para la API del Controlador Cloud de próxima generación, v3.
`volume_services` | Prueba la función [Volume Services](https://docs.cloudfoundry.org/devguide/services/using-vol-services.html) de Cloud Foundry.

## Contribuir

Este repositorio utiliza [go mod](https://golang.org/cmd/go/#hdr-Modules__module_versions__and_more) para gestionar las dependencias en `go`.

Todas las dependencias `go` requeridas por los CATs se almacenan en el directorio `vendor`.

Al realizar cambios en el conjunto de pruebas que introduzcan paquetes `go` adicionales, debe seguir el siguiente flujo de trabajo:

Si puede utilizar la versión más reciente de una dependencia, use `go mod tidy`; de lo contrario, utilice `go get <dependency>@<version>`. Ambas opciones requieren que los módulos Go estén habilitados a través de [envrc](.envrc). Finalmente, use `go mod vendor` para agregar las dependencias al directorio `vendor`.

Para herramientas y recursos, utilice [helpers/assets/tools.go] a través del [flujo de trabajo de herramientas de go mod](https://github.com/go-modules-by-example/index/tree/master/010_tools).

Para obtener información adicional, consulte la [wiki oficial](https://github.com/golang/go/wiki/Modules) y el [repo de ejemplos oficiales](https://github.com/go-modules-by-example/index).

Aunque la rama por defecto de este repositorio es `main`, solicitamos que todas las solicitudes de pull se realicen en la rama `develop`. Por favor, ejecute las pruebas unitarias y asegúrese de que pasen antes de enviarlas. Utilice `./bin/run_units` para ejecutar estas pruebas unitarias.

**Nota**: es necesario ejecutar las pruebas desde la raíz del repositorio.

### Convenciones de código

Existen varias convenciones que recomendamos que los desarrolladores de pruebas de aceptación de CF adopten:

1. Al publicar una aplicación:  
  * establezca el requisito de **memoria**, utilizando el valor `DEFAULT_MEMORY_LIMIT` de la suite (`DEFAULT_WINDOWS_MEMORY_LIMIT` para pruebas en el directorio `windows`), a menos que la prueba necesite específicamente probar un valor diferente;  
  * establezca el **buildpack**, a menos que la prueba necesite específicamente probar el caso en el que no se especifique ningún buildpack, utilizando uno de los valores como `Config.GetRubyBuildpackName()`, `Config.GetJavaBuildpackName()`, etc., a menos que la prueba requiera explícitamente utilizar un nombre o URL de buildpack específico para ella.

  Por ejemplo:

  ```go
  Expect(cf.Cf("push", appName,
      "-b", buildpackName,                  // especificar buildpack
      "-m", DEFAULT_MEMORY_LIMIT,           // especificar límite de memoria
      "-d", Config.AppsDomain,              // especificar dominio de la aplicación
  ).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
  ```
1. Elimine todos los recursos que se hayan creado, como aplicaciones, rutas, cuotas, etc. Esto se hace para dejar el sistema en el mismo estado en el que se encontró. Por ejemplo, para eliminar aplicaciones y sus rutas asociadas:
    ```
		Expect(cf.Cf("delete", myAppName, "-f", "-r").Wait()).To(Exit(0))
    ```
1. Específicamente para las aplicaciones, antes de desmantelarlas, imprima el GUID de la aplicación y los registros recientes de ella. Existe un método auxiliar `AppReport` en el paquete `app_helpers` para este propósito.

    ```go
    AfterEach(func() {
      app_helpers.AppReport(appName)
    })
    ```
1. Documente el propósito de los grupos de pruebas en el README.md de este repositorio. Esto es especialmente importante al cambiar el comportamiento explícito de grupos de pruebas existentes o al agregar nuevos grupos de pruebas.
1. Documente todos los cambios realizados en el objeto de configuración en el README.md de este repositorio.
1. Si agrega una prueba que requiera una nueva versión mínima de la CLI `cf`, actualice el valor de `minCliVersion` en `cats_suite_test.go`.

[networking-releases]: https://github.com/cloudfoundry-incubator/cf-networking-release/releases
[credhub-secure-service-credentials]: https://github.com/pivotal-cf/credhub-release/blob/master/docs/secure-service-credentials.md
