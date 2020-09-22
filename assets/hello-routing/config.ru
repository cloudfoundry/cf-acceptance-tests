require 'json'

app = lambda do |env|
  json = ENV['VCAP_APPLICATION']
  vcapApp = {}
  vcapApp = JSON.parse(json) unless json.nil?

  body = "Hello, " + vcapApp['name'].to_s + " at index: " + ENV['CF_INSTANCE_INDEX'].to_s + "!"

  # log headers
  puts JSON.pretty_generate(env)
  $stdout.flush

  [ 200,
    { "Content-Type" => "text/plain",
      "Content-Length" => body.length.to_s,
      "Set-Cookie" => "JSESSIONID=12345",
    },
    [body]
  ]
end

run app
