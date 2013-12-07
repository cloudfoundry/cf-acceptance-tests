# Dora the Explorer

## Endpoints

1. `GET /` Hello Dora
1. `GET /id` The id of the instance
1. `POST /session` Sets up the cookies for a sticky session
1. `POST /stress_tester?cpu=1&io=1` Starts the stress tester with 1 cpu and 1 io process
1. `GET /stress_tester` Gets all the stress testers processes
1. `DELETE /stress_tester` Kill all the stress testers processes
1. `GET /find/:filename` Finds a file in your instance
1. `GET /sigterm` Displays all possible sigterms
1. `GET /delay/:seconds` Waits for n seconds
1. `GET /sigterm/:signal` Sends the specfied signal
1. `GET /logspew/:bytes` Spews out n bytes to the logs
1. `GET /echo/:destination/:output` Echos out the output to the destination
1. `GET /env/:name` Prints out the env variable

## Sticky Sessions

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
curl dora.yourdomain.com/stress_tester -b instance_2
```
