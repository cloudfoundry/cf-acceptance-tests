require_relative 'nimbus_services'

require_relative 'rabbit'   if Nimbus::Config.rabbit
require_relative 'mysql'    if Nimbus::Config.mysql
require_relative 'mongo'    if Nimbus::Config.mongo
require_relative 'postres'  if Nimbus::Config.postgres
require_relative 'proxy'    if Nimbus::Config.proxy
require_relative 'internalproxy'    if Nimbus::Config.internalproxy
require_relative 'sharedfs' if Nimbus::Config.sharedfs
require_relative 'redis'    if Nimbus::Config.redis
require_relative 'memcache' if Nimbus::Config.memcache
require_relative 'vault'    if Nimbus::Config.vault
require_relative 'cassandra'    if Nimbus::Config.cassandra

run Nimbus::ServicesApp

