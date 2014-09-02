require 'sinatra/base'
require 'json'

$stdout.sync = true
$stderr.sync = true

class Fuse < Sinatra::Base
  get '/' do
    vcap_app  = JSON.parse(ENV.to_hash['VCAP_APPLICATION'])
    port      = vcap_app['port']
    mount_url = "http://127.0.0.1:#{port}/mount"

    `tar zxvf httpfs2-0.1.5.tar.gz -C /tmp`
    `cd /tmp/httpfs2-0.1.5; make`
    `mkdir /tmp/fuse-mount`
    `/tmp/httpfs2-0.1.5/httpfs2 -c /dev/null #{mount_url} /tmp/fuse-mount`

    `cat /tmp/fuse-mount/mount`
  end

  get '/mount' do
    headers['Accept-Ranges'] = 'bytes'
    send_file('mount_response')
  end
end

