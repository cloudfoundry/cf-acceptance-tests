require 'rubygems'
require 'sinatra'

# Unique identifier for the app's lifecycle.
#
# Useful for checking that an app doesn't die and come back up.
ID = SecureRandom.uuid

$stdout.sync = true
$stderr.sync = true

get '/' do
  "Hello, world!"
end

get '/id' do
  ID
end

get '/find/:filename' do
  `find / -name #{params[:filename]}`
end

get '/sigterm' do
  "Available sigterms #{`man -k signal | grep list`}"
end

get '/delay/:seconds' do
  sleep params[:seconds].to_i
  "YAWN! Slept so well for #{params[:seconds].to_i} seconds"
end

get '/sigterm/:signal' do
  pid = Process.pid
  signal = params[:signal]
  puts "Killing process #{pid} with signal #{signal}"
  Process.kill(signal, pid)
end

get '/logspew/:bytes' do
  system "cat /dev/urandom | head -c #{params[:bytes].to_i}"
  "Just wrote #{params[:bytes]} random bytes to the log"
end

get '/echo/:destination/:output' do
  redirect =
    case params[:destination]
    when "stdout"
      ""
    when "stderr"
      " 1>&2"
    else
      " > #{params[:destination]}"
    end

  system "echo '#{params[:output]}'#{redirect}"

  "Printed '#{params[:output]}' to #{params[:destination]}!"
end

get '/env/:name' do
  ENV[params[:name]]
end

