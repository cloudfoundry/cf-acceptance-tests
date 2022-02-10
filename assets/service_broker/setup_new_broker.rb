#!/usr/bin/env ruby

require 'json'
require 'pp'
require 'securerandom'

broker_name = ARGV[0]
broker_name ||= 'async-broker'

$service_name = nil

def uniquify_config
  puts 'Creating a unique configuration for broker'

  raw_config = File.read('data.json')
  config = JSON.parse(raw_config)
  catalog = config['behaviors']['catalog']['body']

  plan_mapping = {}
  catalog['services'] = catalog['services'].map do |service|
    $service_name = service['name'] = "fake-service-#{SecureRandom.uuid}"
    service['id'] = SecureRandom.uuid

    service['dashboard_client']['id'] = SecureRandom.uuid
    service['dashboard_client']['secret'] = SecureRandom.uuid

    service['plans'] = service['plans'].map do |plan|
      original_id = plan['id']
      plan['id'] = SecureRandom.uuid
      plan_mapping[original_id] = plan['id']
      plan
    end
    service
  end

  config['behaviors'].each do |action, behavior|
    next if action == 'catalog'

    behavior.keys.each do |plan_id|
      next if plan_id == 'default'

      response = behavior[plan_id]
      new_plan_id = plan_mapping[plan_id]
      behavior[new_plan_id] = response
      behavior.delete(plan_id)
    end
  end

  File.open('data.json', 'w') do |file|
    file.write(JSON.pretty_generate(config))
  end
end

def push_broker(broker_name)
  puts "Pushing the broker"
  IO.popen("cf push #{broker_name}") do |cmd_output|
    cmd_output.each { |line| puts line }
  end
  puts
  puts
end

def create_service_broker(broker_name, url)
  output = []
  IO.popen("cf create-service-broker #{broker_name} user password #{url}") do |cmd|
    cmd.each do |line|
      puts line
      output << line
    end
  end
  output
end

def broker_already_exists?(output)
  output.any? { |line| line =~ /service broker url is taken/ }
end

def update_service_broker(broker_name, url)
  puts
  puts "Broker already exists. Updating"
  IO.popen("cf update-service-broker #{broker_name} user password #{url}") do |cmd|
    cmd.each { |line| puts line }
  end
  puts
end

def enable_service_access
  IO.popen("cf enable-service-access #{$service_name}") do |cmd|
    cmd.each { |line| puts line }
  end
end

uniquify_config

push_broker(broker_name)

app_guid, routes_object, url = ""
IO.popen("cf app #{broker_name} --guid") do |cmd|
  app_guid = cmd.read.chomp
end

IO.popen("cf curl /v3/apps/#{app_guid}/routes") do |cmd|
  url = "https://" + JSON.parse(cmd.read)["resources"][0]["url"]
end

output = create_service_broker(broker_name, url)
if broker_already_exists?(output)
  update_service_broker(broker_name, url)
end

enable_service_access

puts
puts 'Setup complete'