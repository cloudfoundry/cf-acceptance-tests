require 'sinatra/base'

module Nimbus
  class ServicesApp < Sinatra::Base

    DATA_FOLDER = '../data'

    post '/sharedfs/save/:file_name' do
      file_name = params['file_name']
      contents = request.body.read
      File.open("#{DATA_FOLDER}/#{file_name}", 'w') do |f|
        f.write(contents)
      end
      'OK'
    end

    get '/sharedfs/read/:file_name' do
      file_name = params['file_name']
      contents = File.read("#{DATA_FOLDER}/#{file_name}")
      contents
    end

  end
end