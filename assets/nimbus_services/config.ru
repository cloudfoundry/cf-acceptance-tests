require_relative 'nimbus_services'

require_relative 'rabbit'   if Nimbus::Config.rabbit
require_relative 'mongo'    if Nimbus::Config.mongo
require_relative 'postres'  if Nimbus::Config.postgres
require_relative 'proxy'    if Nimbus::Config.proxy
require_relative 'redis'    if Nimbus::Config.redis

run Nimbus::ServicesApp

