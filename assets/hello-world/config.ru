app = lambda do |env|
  body = "Hello, world!"

  [ 200,
    { "Content-Type" => "text/plain",
      "Content-Length" => body.length.to_s
    },
    [body]
  ]
end

run app
