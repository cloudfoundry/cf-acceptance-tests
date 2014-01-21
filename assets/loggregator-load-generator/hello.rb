require 'sinatra'
STDOUT.sync = true

$run = false

get '/' do
  time = Time.now
<<-RESPONSE
  Endpoints:<br><br>
  <ul>
  <li>/log/sleep/:logspeed - set the pause between loglines to a millionth fraction of a second</li>
  <li>/log/bytesize/:bytesize - set the size of each logline in bytes</li>
  <li>/log/stop - stops any running logging</li>
  </ul>
RESPONSE
end

get '/log/sleep/:logspeed' do
  $run = true
  time = Time.now
  STDOUT.puts("Muahaha... let's go. Waiting #{params[:logspeed].to_f/1000000.to_f} seconds between loglines. Logging 'Muahaha...' every time.")
  while $run do
    sleep(params[:logspeed].to_f/1000000.to_f)
    STDOUT.puts("Log: #{request.host} Muahaha...")
  end
end

get '/log/bytesize/:bytesize' do
  $run = true
  logString = "0" * params[:bytesize].to_i
  STDOUT.puts("Muahaha... let's go. No wait. Logging #{params[:bytesize]} bytes per logline.")
  while $run do
    STDOUT.puts(logString)
  end
end

get '/log/stop' do
  $run = false
  time = Time.now
  STDOUT.puts("Sopped logs #{time}")
end
