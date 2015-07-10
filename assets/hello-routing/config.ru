require 'json'

app = lambda do |env|
  json = ENV['VCAP_APPLICATION']
 
  vcapApp = JSON.parse(json)

  body = "Hello, " + vcapApp['name'] + "!"

  [ 200,
    { "Content-Type" => "text/plain",
      "Content-Length" => body.length.to_s,
      "JESSIONID" => "12345"
    },
    [body]
  ]
end

run app
