## Sticky Sessions

To set up a sticky session manually:

1. Get your sticky session by running:
```bash
curl -d '' dora.yourdomain.com/session -c instance_1
```
1. Run with a different filename for each instance, and repeat the curl command until you get a new ID
```bash
curl -d '' dora.yourdomain.com/session -c instance_2
```
1. Then you can target whatever instance you want for example:
```bash
curl dora.yourdomain.com/stress_testers -b instance_2