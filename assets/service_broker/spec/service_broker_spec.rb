require 'spec_helper'
require 'json'

describe ServiceBroker do
  describe 'GET /v2/catalog' do
    it 'returns a non-empty catalog' do
      get '/v2/catalog'
      response = last_response
      expect(response.body).to be
      expect(JSON.parse(response.body)).to be
    end
  end

  describe 'POST /v2/catalog' do
    it 'changes the catalog' do
      get '/v2/catalog'
      first_response = last_response
      expect(first_response.body).to be

      post '/v2/catalog'

      get '/v2/catalog'
      second_response = last_response
      expect(second_response.body).to eq(first_response.body)
    end
  end

  describe 'PUT /v2/service_instances/:id' do
    it 'returns 200 with an empty JSON body' do
      put '/v2/service_instances/fakeIDThough', {}.to_json
      expect(last_response.status).to eq(200)
      expect(JSON.parse(last_response.body)).to be_empty
    end
  end

  describe 'configuration management' do
    before do
      post '/config/reset'
    end

    def provision
      put '/v2/service_instances/fake-guid', {plan_id: 'fake-plan-guid'}.to_json
    end

    def deprovision
      delete '/v2/service_instances/fake-guid?plan_id=fake-plan-guid', {}.to_json
    end

    def update
      patch '/v2/service_instances/fake-guid', {plan_id: 'fake-plan-guid'}.to_json
    end

    def bind
      put '/v2/service_instances/fake-guid/service_bindings/binding-gui', {plan_id: 'fake-plan-guid'}.to_json
    end

    def unbind
      delete '/v2/service_instances/fake-guid/service_bindings/binding-gui?plan_id=fake-plan-guid', {}.to_json
    end

    [:provision, :deprovision, :update, :bind, :unbind].each do |action|
      context "for a #{action} operation" do
        before do
          put '/v2/service_instances/fake-guid', {plan_id: 'fake-plan-guid'}.to_json unless action == :provision
          put '/v2/service_instances/fake-guid/service_bindings/binding-gui', {plan_id: 'fake-plan-guid'}.to_json if action == :unbind
        end

        it 'should change the response using a json body' do
          config = {
            behaviors: {
              action => {
                default: {
                  status: 400,
                  sleep_seconds: 0,
                  body: {}
                }
              }
            }
          }.to_json

          post '/config', config

          send(action)
          expect(last_response.status).to eq(400)
          expect(last_response.body).to eq('{}')
        end

        it 'should change the response using an invalid json body' do
          config = {
            behaviors: {
              action => {
                default: {
                  status: 201,
                  sleep_seconds: 0,
                  raw_body: 'foo'
                }
              }
            }
          }.to_json

          post '/config', config

          send(action)
          expect(last_response.status).to eq(201)
          expect(last_response.body).to eq 'foo'
        end

        it 'should cause the action to sleep' do
          config = {
            behaviors: {
              action => {
                default: {
                  status: 200,
                  sleep_seconds: 1.1,
                  body: {}
                }
              }
            }
          }.to_json

          post '/config', config


          expect do
            Timeout::timeout(1) do
              send(action)
            end
          end.to raise_error(TimeoutError)
        end

        it 'can be customized on a per-plan basis' do
          config = {
            behaviors: {
              action => {
                'fake-plan-guid' => {
                  status: 200,
                  sleep_seconds: 0,
                  raw_body: 'fake-plan body'
                },
                default: {
                  status: 400,
                  sleep_seconds: 0,
                  body: {}
                }
              }
            }
          }.to_json

          post '/config', config

          send(action)
          expect(last_response.status).to eq(200)
          expect(last_response.body).to eq('fake-plan body')
        end
      end
    end

    context 'for a fetch operation' do
      before do
        put '/v2/service_instances/fake-guid', {plan_id: 'fake-plan-guid'}.to_json
      end

      it 'should change the response using a json body' do
        config = {
          max_fetch_service_instance_requests: 1,
          behaviors: {
            fetch: {
              default: {
                in_progress: {
                  status: 200,
                  sleep_seconds: 0,
                  body: {}
                },
                finished: {
                  status: 400,
                  sleep_seconds: 0,
                  body: { foo: :bar }
                }
              }
            }
          }
        }.to_json

        post '/config', config

        get '/v2/service_instances/fake-guid/last_operation'
        expect(last_response.status).to eq(200)
        expect(last_response.body).to eq('{}')

        get '/v2/service_instances/fake-guid/last_operation'
        expect(last_response.status).to eq(400)
        expect(last_response.body).to eq({ foo: :bar }.to_json)
      end

      it 'should change the response using an invalid json body' do
        config = {
          max_fetch_service_instance_requests: 1,
          behaviors: {
            fetch: {
              default: {
                in_progress: {
                  status: 200,
                  sleep_seconds: 0,
                  raw_body: 'cheese'
                },
                finished: {
                  status: 400,
                  sleep_seconds: 0,
                  raw_body: 'cake'
                }
              }
            }
          }
        }.to_json

        post '/config', config

        get '/v2/service_instances/fake-guid/last_operation'
        expect(last_response.status).to eq(200)
        expect(last_response.body).to eq 'cheese'

        get '/v2/service_instances/fake-guid/last_operation'
        expect(last_response.status).to eq(400)
        expect(last_response.body).to eq 'cake'
      end

      it 'should cause the action to sleep' do
        config = {
          max_fetch_service_instance_requests: 1,
          behaviors: {
            fetch: {
              default: {
                in_progress: {
                  status: 200,
                  sleep_seconds: 1.1,
                  body: {}
                },
                finished: {
                  status: 200,
                  sleep_seconds: 0.6,
                  body: { }
                }
              }
            }
          }
        }.to_json

        post '/config', config

        expect do
          Timeout::timeout(1) do
            get '/v2/service_instances/fake-guid/last_operation'
          end
        end.to raise_error(TimeoutError)

        expect do
          Timeout::timeout(0.5) do
            get '/v2/service_instances/fake-guid/last_operation'
          end
        end.to raise_error(TimeoutError)
      end

      it 'honors max_fetch_service_instance_request' do
        config = {
          max_fetch_service_instance_requests: 2,
          behaviors: {
            fetch: {
              default: {
                in_progress: {
                  status: 200,
                  sleep_seconds: 0,
                  body: {}
                },
                finished: {
                  status: 400,
                  sleep_seconds: 0,
                  body: { foo: :bar }
                }
              }
            }
          }
        }.to_json

        post '/config', config

        get '/v2/service_instances/fake-guid/last_operation'
        expect(last_response.status).to eq(200)
        expect(last_response.body).to eq('{}')

        get '/v2/service_instances/fake-guid/last_operation'
        expect(last_response.status).to eq(200)
        expect(last_response.body).to eq('{}')

        get '/v2/service_instances/fake-guid/last_operation'
        expect(last_response.status).to eq(400)
        expect(last_response.body).to eq({ foo: :bar }.to_json)
      end

      it 'can be customized on a per-plan basis' do
        config = {
          max_fetch_service_instance_requests: 1,
          behaviors: {
            fetch: {
              'fake-plan-guid' => {
                in_progress: {
                  status: 200,
                  sleep_seconds: 0,
                  body: { foo: 'bar' }
                },
                finished: {
                  status: 201,
                  sleep_seconds: 0,
                  body: { foo: 'baz' }
                }
              },
              default: {
                in_progress: {
                  status: 200,
                  sleep_seconds: 0,
                  body: {}
                },
                finished: {
                  status: 400,
                  sleep_seconds: 0,
                  body: { foo: :bar }
                }
              }
            }
          }
        }.to_json

        post '/config', config

        get '/v2/service_instances/fake-guid/last_operation'
        expect(last_response.status).to eq(200)
        expect(last_response.body).to eq({ foo: 'bar' }.to_json)

        get '/v2/service_instances/fake-guid/last_operation'
        expect(last_response.status).to eq(201)
        expect(last_response.body).to eq({ foo: 'baz' }.to_json)
      end
    end

    it 'should allow resetting the configuration to its defaults' do
      get '/config'
      data = last_response.body

      config = {
        behaviors: {
          provision: {
            default: {
              status: 400,
              sleep_seconds: 0,
              body: {}
            }
          }
        }
      }.to_json
      post '/config', config

      post '/config/reset'
      expect(last_response.status).to eq(200)

      get '/config'
      expect(last_response.body).to eq(data)
    end

    it 'should be able to restore a previously saved configuration' do
      get '/config'
      data = last_response.body

      post '/config', data
      expect(last_response.status).to eq(200)
      expect(last_response.body).to eq(data)
    end
  end
end
