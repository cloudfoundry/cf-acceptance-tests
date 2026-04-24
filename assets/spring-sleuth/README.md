# spring-sleuth

A small Spring Boot app that replaces legacy `Spring Cloud Sleuth` `SpanAccessor` usage with modern `Micrometer Tracing` + `OpenTelemetry`.

## What it does

- Exposes `GET /`
- Reads the current span from `io.micrometer.tracing.Tracer`
- Returns current trace/span information in a response string similar to the legacy app
- Extracts parent span IDs from incoming `traceparent`, `b3`, or `X-B3-ParentSpanId` headers for display

## Build and test

```bash
cd /Users/d050987/cf/cf-acceptance-tests/assets/spring-sleuth
mvn test
```

## Run locally

```bash
cd /Users/d050987/cf/cf-acceptance-tests/assets/spring-sleuth
mvn spring-boot:run
```

Then call it with a W3C trace header:

```bash
curl -s \
  -H 'traceparent: 00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01' \
  http://localhost:8080/
```

## Cloud Foundry push example

```bash
cd /Users/d050987/cf/cf-acceptance-tests/assets/spring-sleuth
mvn -DskipTests package
cf push spring-sleuth -p target/spring-sleuth-0.0.1.jar -m 512M
```

