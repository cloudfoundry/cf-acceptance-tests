require 'sinatra'
STDOUT.sync = true

$run = false

get '/' do
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
  if $run
    "Already running.  Use /log/stop and then restart."
  else
    $run       = true
    sleep_time = params[:logspeed].to_f/1000000.to_f

    STDOUT.puts("Muahaha... let's go. Waiting #{sleep_time} seconds between loglines. Logging 'Muahaha...' every time.")
    Thread.new do
      while $run do
        sleep(sleep_time)
        STDOUT.puts("Log: #{request.host} Muahaha...")
      end
    end

    "Muahaha... let's go. Waiting #{params[:logspeed].to_f/1000000.to_f} seconds between loglines. Logging 'Muahaha...' every time."
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
  STDOUT.puts("Stopped logs #{time}")
end
