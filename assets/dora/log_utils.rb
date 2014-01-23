class LogUtils < Sinatra::Base

  get "/loglines/:linecount" do
    produce_log_output(params[:linecount])
    "logged #{params[:linecount]} line to stdout"
  end

  get "/loglines/:linecount/:tag" do
    produce_log_output(params[:linecount], params[:tag])
    "logged #{params[:linecount]} line with tag #{params[:tag]} to stdout"
  end

  STDOUT.sync = true

  $run = false

  get '/log/sleep/:logspeed' do
    $run = true
    sequence_number = 0
    STDOUT.puts("Muahaha... let's go. Waiting #{params[:logspeed].to_f/1000000.to_f} seconds between loglines. Logging 'Muahaha...' every time.")
    while $run do
      sleep(params[:logspeed].to_f/1000000.to_f)
      STDOUT.puts("Log: #{request.host} Muahaha...#{sequence_number}...#{Time.now}")
      sequence_number += 1
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
    STDOUT.puts("Stopped logs #{Time.now}")
  end

  private
  def produce_log_output(linecount, tag="")
    linecount.to_i.times do |i|
      puts "#{Time.now.strftime("%FT%T.%N%:z")} line #{i} #{tag}"
      $stdout.flush
    end
  end
end
