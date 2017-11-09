require 'sinatra/base'
require 'vault'

module Nimbus
  class ServicesApp < Sinatra::Base

    configure do
      Vault.address = Nimbus::Config.vault['vault_addr']
    end

    get '/vault/insert/:key/:value' do |key, value|
      Vault.auth.userpass(Nimbus::Config.vault['username'], Nimbus::Config.vault['password'])
      Vault.logical.write("#{Nimbus::Config.vault['base_path']}/val", {key.to_sym => value})
      'OK'
    end

    get '/vault/read/:key/:value' do |key, value|
      Vault.auth.userpass(Nimbus::Config.vault['username'], Nimbus::Config.vault['password'])
      secret = Vault.logical.read("#{Nimbus::Config.vault['base_path']}/val")
      secret.data[key.to_sym] == value ? 'OK' : 'FAIL'
    end

  end
end