#!/bin/bash

instances_json=`CF_TRACE=true gcf app xyz | grep fds_quota`
eval set `echo $instances_json | \
                 ruby -ne '
                 require "json"; 
                 require "pp"; 
                 x=JSON.parse($_); 
                 x.keys.each {|k| 
                   v = x[k]["stats"]["host"]; 
                   puts "#{k} #{v}"
                 }'`
while [ $# -gt 0 ]
do
	INSTANCES[$1]=$2
	shift 2
done


for cjarfile in cookie_jars/*.cjar
do
  output=`curl -s dora.sunset.cf-app.com/env/VCAP_APPLICATION -b $cjarfile`
  instance=`echo $output | sed -e 's/^.*nstance_index":\([0-9]*\),.*$/\1/'`
  #appguid=`CF_TRACE=true gcf app dora | grep GET | grep /stats | head -1 | cut -d/ -f4`
  #dea=`gcf curl -v /v2/apps/$appguid/instances/$instance/files | grep Host: | grep -v 'Host: api' | awk '{print $2}'`
  dea=${INSTANCES[$instance]}
  echo $cjarfile is for index $instance running on dea $dea
done




