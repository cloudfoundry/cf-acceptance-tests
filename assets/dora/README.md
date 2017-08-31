# Dora the Explorer

## Endpoints

1. `GET /` Hello Dora
1. `GET /id` The id of the instance
1. `POST /session` Sets up the cookies for a sticky session
1. `POST /stress_testers?cpu=1&io=1` Starts the stress tester with 1 cpu and 1 io process
1. `GET /stress_testers` Gets all the stress testers processes
1. `DELETE /stress_testers` Kill all the stress testers processes
1. `GET /find/:filename` Finds a file in your instance
1. `GET /sigterm` Displays all possible sigterms
1. `GET /delay/:seconds` Waits for n seconds
1. `GET /sigterm/:signal` Sends the specfied signal
1. `GET /logspew/:bytes` Spews out n bytes to the logs
1. `GET /loglines/:linecount` Writes n lines to stdout, each line contains a timestamp with nanoseconds
1. `GET /loglines/:linecount/:tag` Writes n lines to stdout, each line contains a timestamp with nanoseconds and the given tag
1. `GET /log/sleep/count` Returns a count of the log messages logged by the log service
1. `GET /log/sleep/running` Returns whether the log service is running
1. `GET /log/sleep/:logspeed/limit/:limit` Produces logspeed output with the given parameters
1. `GET /log/sleep/:logspeed` Produces logspeed output without limit
1. `GET /log/stop` Stops the log service
1. `GET /log/bytesize/:bytesize` Produces continuous log entries of the given bytesize
1. `GET /echo/:destination/:output` Echos out the output to the destination
1. `GET /env/:name` Prints out the environment variable `:name`
1. `GET /env` Prints out the entire environment as a serialized Ruby hash
1. `GET /env.json` Prints out the entire environment as a JSON object
1. `GET /largetext/:kbytes` Returns a dummy response of size `:kbytes`. For testing large payloads.
1. `GET /health` Returns 500 the first 3 times you call it, "I'm alive" thereafter
1. `GET /ping/:address` Pings the given address 4 times
1. `GET /lsb_release` Returns information about the Linux distribution of the container
1. `GET /dpkg/:package` Returns the output of `dpkg -l` for the given packange
1. `GET /myip` Returns the IP of the app container
1. `GET /curl/:host/?:port?` cURLs the given host and port and returns the stdout, stderr, and status as JSON

## Sticky Sessions

There is a helper script in this directory: `get_instance_cookie_jars.sh`

- specify number of expected instances with `-e #`
- specify maximum number of tries with `-m #`

The script will create cookie jars in the current directory, using the filename pattern `cookie_jar_<instance_id>.cjar`

To direct a curl request to a particular instance, specify `-b <cookie_jar_file>` on the curl command line.

Or, to set up a sticky session manually:

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
```

