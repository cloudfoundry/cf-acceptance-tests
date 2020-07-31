#!/bin/sh

cf push dora --no-start
time cf delete dora -f -r -v 
#> /tmp/job-queue-delete-no-worker.txt

cf push dora6 --no-start
time cf delete dora6 -f -r -v 



