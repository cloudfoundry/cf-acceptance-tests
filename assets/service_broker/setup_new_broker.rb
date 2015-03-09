require 'json'
require 'pp'
require 'securerandom'

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

uniquify_config
puts `cf push async-broker`

outuput = `cf create-service-broker async-broker user password http://async-broker.10.244.0.34.xip.io`
if outuput =~ /service broker url is taken/
  puts "Broker already exists. Updating..."
  puts `cf update-service-broker async-broker user password http://async-broker.10.244.0.34.xip.io`
end

puts `cf enable-service-access #{$service_name}`