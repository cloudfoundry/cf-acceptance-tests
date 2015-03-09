#!/usr/bin/env ruby

require 'CSV'
require 'json'
require 'benchmark'
require 'securerandom'

class ProvisionCommand
  def setup(instance_name)
  end

  def run(instance_name)
    `cf create-service fake-service fake-plan #{instance_name}`
  end

  def cleanup(instance_name)
  end
end

class UpdateCommand
  def setup(instance_name)
    `cf create-service fake-service fake-plan #{instance_name}`
  end

  def run(instance_name)
    `cf update-service #{instance_name} -p fake-async-plan`
  end

  def cleanup(instance_name)
  end
end

class DeprovisionCommand
  def setup(instance_name)
    `cf create-service fake-service fake-plan #{instance_name}`
  end

  def run(instance_name)
    `cf delete-service #{instance_name} -f`
  end

  def cleanup(instance_name)
  end
end

class CleanupCommandWrapper
  def initialize(command)
    @command = command
  end

  def setup(instance_name)
    @command.setup(instance_name)
  end

  def run(instance_name)
    @command.run(instance_name)
  end

  def cleanup(instance_name)
    @command.cleanup(instance_name)
    if attempt_delete(instance_name)
      -> {
        until attempt_delete(instance_name)
        end
      }
    end
  end

  private

  def attempt_delete(instance_name)
    output = `cf delete-service #{instance_name} -f`
    !output.include?('Another operation for this service instance is in progress')
  end
end

deferred_deletions = []

action_to_cmd_mapping = {
  provision: CleanupCommandWrapper.new(ProvisionCommand.new),
  update: CleanupCommandWrapper.new(UpdateCommand.new),
  deprovision: CleanupCommandWrapper.new(DeprovisionCommand.new),
}

DEFAULT_BROKER_URL = 'http://async-broker.10.244.0.34.xip.io'

if ARGV.length < 1
  puts "Usage: #{$PROGRAM_NAME} CSV_FILE [BROKER_URL]"
  puts
  puts "Broker URL defaults to #{DEFAULT_BROKER_URL}"
  exit(1)
end

input_file = ARGV[0]

name = File.basename(input_file, '.*')
extension = File.extname(input_file)
output_file = name + "-out" + extension

broker_url = ARGV.length > 1 ? ARGV[1] : DEFAULT_BROKER_URL
rows = []

report = Benchmark.measure do
  CSV.foreach(input_file, headers: true) do |row|
    rows << row

    action, status, body = row['action'], row['status'], row['body']

    next unless action

    command = action_to_cmd_mapping[action.to_sym]

    next unless command

    json_config = {
      behaviors: {
        action => {
          default: {
            status: status,
            raw_body: body,
            sleep_seconds: row['sleep seconds'].to_f
          }
        }
      }
    }

    `curl -s #{broker_url}/config/reset -X POST`
    `curl -s #{broker_url}/config -d '#{json_config.to_json}'`

    instance_name = "si-#{SecureRandom.uuid}"

    command.setup(instance_name)
    output = command.run(instance_name)
    deferred_deletions << command.cleanup(instance_name)

    row['output'] = output
    STDOUT.write('.')
    STDOUT.flush
  end

  puts

  CSV.open(output_file, 'w') do |csv|
    csv << rows[0].headers
    rows.each do |row|
      csv << row
    end
  end

  STDOUT.write("Cleaning up service instances... ")
  STDOUT.flush
  deferred_deletions.compact.each do |callback|
    callback.call
  end
  puts "Done"
end

puts "Took #{report.real} seconds"