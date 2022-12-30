CATs will build catnip in its SynchronizedBeforeSuite, 
so if you've run CATs you should have the binary 
at `./assets/catnip/bin/catnip`

To build and push `catnip` locally
```
GOOS=linux GOARCH=amd64 go build -o bin/catnip
cd bin
cf push catnip -b binary_buildpack -c "./catnip"
```

## Sticky Sessions

To set up a sticky session manually:

1. Get your sticky session by running:
```bash
curl -d '' dora.yourdomain.com/session -c instance_1
```
2. Run with a different filename for each instance, and repeat the curl command until you get a new ID
```bash
curl -d '' dora.yourdomain.com/session -c instance_2
```
3. Then you can target whatever instance you want for example:
```bash
curl dora.yourdomain.com/stress_testers -b instance_2
```
