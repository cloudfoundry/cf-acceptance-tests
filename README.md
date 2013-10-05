# CF Runtime Integration Tests

## Usage

```sh
cat > integration_config.json <<EOF
{ "apps_domain": "10.244.0.22.xip.io" }
EOF

export CONFIG=$PWD/integration_config.json

ginkgo -r -v -slowSpecThreshold=300
```
