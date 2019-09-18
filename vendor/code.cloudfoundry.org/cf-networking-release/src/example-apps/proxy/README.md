#### To use the example app:

Push two differently named instances of the app
```bash
cd ~/workspace/cf-networking-release/src/example-apps/proxy
cf push appA
cf push appB
```

See that they are reachable and what their IPs are
```bash
curl appa.<system-domain>
curl appb.<system-domain>
```

See that proxying from A to B works over both the router and overlay
```bash
curl appa.<system-domain>/proxy/appb.<system-domain>
curl appa.<system-domain>/proxy/<overlay-ip-of-appB>:8080
```
