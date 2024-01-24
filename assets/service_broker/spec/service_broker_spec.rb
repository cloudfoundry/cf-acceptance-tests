require 'spec_helper'
require 'json'
require 'timeout'

describe ServiceBroker do
  before do
    post '/config/reset'
  end

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

  describe 'GET /v2/service_instances/:id' do
    context 'service instance exists' do
      before do
        put '/v2/service_instances/fake-guid', {service_id: 'fake-service', plan_id: 'fake-plan'}.to_json
      end

      it 'returns 200 with the provisioning data in the body  ' do
        get '/v2/service_instances/fake-guid'
        expect(last_response.status).to eq(200)
        expect(last_response.body).to eq({service_id: 'fake-service', plan_id: 'fake-plan'}.to_json)
      end
    end
    context 'service instance does not exist' do
      it 'returns 404' do
        get '/v2/service_instances/non-existent'
        expect(last_response.status).to eq(404)
        expect(last_response.body).to eq("Broker could not find service instance by the given id non-existent")
      end
    end
  end

  describe 'PUT /v2/service_instances/:id' do
    it 'returns 200 with an empty JSON body' do
      put '/v2/service_instances/fakeIDThough', {}.to_json
      expect(last_response.status).to eq(200)
      expect(JSON.parse(last_response.body)).to be_empty
    end

    context 'when the plan is configured as async_only' do
      before do
        config = {
            max_fetch_service_instance_requests: 1,
            behaviors: {
                provision: {
                    'fake-async-plan-guid' => {
                        sleep_seconds: 0,
                        async_only: true,
                        status: 202,
                        body: {}
                    },
                    default: {
                        sleep_seconds: 0,
                        status: 202,
                        body: {}
                    }
                }
            }
        }.to_json

        post '/config', config
      end


      context 'request is for an async plan' do
        it 'returns as usual if it does include accepts_incomplete' do
          put '/v2/service_instances/fake-guid?accepts_incomplete=true', {plan_id: 'fake-async-plan-guid'}.to_json

          expect(last_response.status).to eq(202)
        end

        it 'rejects request if it does not include accepts_incomplete' do
          put '/v2/service_instances/fake-guid', {plan_id: 'fake-async-plan-guid'}.to_json

          expect(last_response.status).to eq(422)
          expect(last_response.body).to eq(
                                            {
                                                'error' => 'AsyncRequired',
                                                'description' => 'This service plan requires client support for asynchronous service operations.'
                                            }.to_json
                                        )
        end
      end

    end
  end

  describe 'PATCH /v2/service_instance/:id' do
    context 'when updating to an async plan' do
      it 'returns a 202' do
        patch '/v2/service_instances/fake-guid?accepts_incomplete=true', {plan_id: 'fake-async-plan-guid'}.to_json
        expect(last_response.status).to eq(202)
      end
    end

    context 'when updating to a sync plan' do
      it 'returns a 200' do
        patch '/v2/service_instances/fake-guid?accepts_incomplete=true', {plan_id: 'fake-plan-guid'}.to_json
        expect(last_response.status).to eq(200)
      end
    end

    context 'when the plan is configured as async_only' do
      before do
        config = {
            max_fetch_service_instance_requests: 1,
            behaviors: {
                update: {
                    'fake-async-plan-guid' => {
                        sleep_seconds: 0,
                        async_only: true,
                        status: 202,
                        body: {}
                    },
                    default: {
                        sleep_seconds: 0,
                        status: 202,
                        body: {}
                    }
                }
            }
        }.to_json

        post '/config', config
      end


      context 'request is for an async plan' do
        it 'returns as usual if it does include accepts_incomplete' do
          patch '/v2/service_instances/fake-guid?accepts_incomplete=true', {plan_id: 'fake-async-plan-guid'}.to_json

          expect(last_response.status).to eq(202)
        end

        it 'rejects request if it does not include accepts_incomplete' do
          patch '/v2/service_instances/fake-guid', {plan_id: 'fake-async-plan-guid'}.to_json

          expect(last_response.status).to eq(422)
          expect(last_response.body).to eq(
              {
                  'error' => 'AsyncRequired',
                  'description' => 'This service plan requires client support for asynchronous service operations.'
              }.to_json
          )
        end
      end

    end
  end

  describe 'DELETE /v2/service_instances/:id' do
    before do
      put '/v2/service_instances/fake-guid?accepts_incomplete=true', {plan_id: 'fake-async-plan-guid'}.to_json
      expect(last_response.status).to eq(202)
    end

    context 'when the plan is configured as async_only' do
      before do
        config = {
            max_fetch_service_instance_requests: 1,
            behaviors: {
                deprovision: {
                    'fake-async-plan-guid' => {
                        sleep_seconds: 0,
                        async_only: true,
                        status: 202,
                        body: {}
                    },
                    default: {
                        sleep_seconds: 0,
                        status: 202,
                        body: {}
                    }
                }
            }
        }.to_json

        post '/config', config
      end


      context 'request is for an async plan' do
        it 'returns as usual if it does include accepts_incomplete' do
          delete '/v2/service_instances/fake-guid?accepts_incomplete=true'

          expect(last_response.status).to eq(202)
        end

        it 'rejects request if it does not include accepts_incomplete' do
          delete '/v2/service_instances/fake-guid'

          expect(last_response.status).to eq(422)
          expect(last_response.body).to eq(
              {
                  'error' => 'AsyncRequired',
                  'description' => 'This service plan requires client support for asynchronous service operations.'
              }.to_json
          )
        end
      end
    end
  end

  describe 'GET /v2/service_instances/:id/service_bindings/:id' do
    context 'service binding exists' do
      before do
        put '/v2/service_instances/fake-guid', {service_id: 'fake-service', plan_id: 'fake-plan'}.to_json
        put '/v2/service_instances/fake-guid/service_bindings/binding-guid', {plan_id: 'fake-plan'}.to_json
      end

      it 'returns 200 with the provisioning data in the body  ' do
        get '/v2/service_instances/fake-guid/service_bindings/binding-guid'
        expect(last_response.status).to eq(200)
        expect(last_response.body).to eq({plan_id: 'fake-plan'}.to_json)
      end

      context 'service binding is not associated with given service instance' do
        it 'returns 404' do
          get '/v2/service_instances/non-existent/service_bindings/binding-guid'
          expect(last_response.status).to eq(404)
          expect(last_response.body).to eq("Broker could not find the service binding `binding-guid` for service instance `non-existent`")
        end
      end
    end
    context 'service binding does not exist' do
      it 'returns 404' do
        get '/v2/service_instances/non-existent/service_bindings/non-existent-binding'
        expect(last_response.status).to eq(404)
        expect(last_response.body).to eq("Broker could not find the service binding `non-existent-binding` for service instance `non-existent`")
      end
    end
  end

  describe 'PUT /v2/service_instances/:id/service_bindings/:id' do
    before do
      config = {
        max_fetch_service_binding_requests: 1,
        behaviors: {
          bind: {
            'fake-plan-guid' => {
              sleep_seconds: 0,
              status: 201,
              body: {}
            },
            default: {
              sleep_seconds: 0,
              status: 202,
              body: {}
            }
          }
        }
      }.to_json

      post '/config', config

      put '/v2/service_instances/fake-guid', {service_id: 'fake-service', plan_id: 'fake-plan-guid'}.to_json
    end

    it 'returns 201 with an empty JSON body' do
      put '/v2/service_instances/fake-guid/service_bindings/binding-guid', {plan_id: 'fake-plan-guid'}.to_json
      expect(last_response.status).to eq(201)
      expect(JSON.parse(last_response.body)).to be_empty
    end

    context 'service instance does not exist' do
      it 'returns 400' do
        put '/v2/service_instances/non-existent/service_bindings/binding-guid', {plan_id: 'fake-plan-guid'}.to_json
        expect(last_response.status).to eq(400)
        expect(last_response.body).to eq('Broker could not find service instance by the given id non-existent')
      end
    end

    context 'when the plan is configured as async_only' do
      before do
        config = {
          max_fetch_service_binding_requests: 1,
          behaviors: {
            bind: {
              'fake-async-plan-guid' => {
                sleep_seconds: 0,
                async_only: true,
                status: 202,
                body: {}
              },
              default: {
                sleep_seconds: 0,
                status: 202,
                body: {}
              }
            }
          }
        }.to_json

        post '/config', config

        put '/v2/service_instances/fake-guid?accepts_incomplete=true', {service_id: 'fake-service', plan_id: 'fake-async-plan-guid'}.to_json
      end


      context 'request is for an async plan' do
        it 'returns as usual if it does include accepts_incomplete' do
          put '/v2/service_instances/fake-guid/service_bindings/binding-guid?accepts_incomplete=true', {plan_id: 'fake-async-plan-guid'}.to_json

          expect(last_response.status).to eq(202)
        end

        it 'rejects request if it does not include accepts_incomplete' do
          put '/v2/service_instances/fake-guid/service_bindings/binding-guid', {plan_id: 'fake-async-plan-guid'}.to_json

          expect(last_response.status).to eq(422)
          expect(last_response.body).to eq(
                                          {
                                            'error' => 'AsyncRequired',
                                            'description' => 'This service plan requires client support for asynchronous service operations.'
                                          }.to_json
                                        )
        end
      end

    end
  end

  describe 'DELETE /v2/service_instances/:id/service_bindings/:id' do
    before do
      put '/v2/service_instances/fake-guid?accepts_incomplete=true', {plan_id: 'fake-async-plan-guid'}.to_json
      put '/v2/service_instances/fake-guid/service_bindings/binding-guid?accepts_incomplete=true', {plan_id: 'fake-async-plan-guid'}.to_json
    end

    context 'when the plan is configured as async_only' do
      before do
        config = {
          max_fetch_service_binding_requests: 1,
          behaviors: {
            unbind: {
              'fake-async-plan-guid' => {
                sleep_seconds: 0,
                async_only: true,
                status: 202,
                body: {}
              },
              default: {
                sleep_seconds: 0,
                status: 202,
                body: {}
              }
            }
          }
        }.to_json

        post '/config', config
      end


      context 'request is for an async plan' do
        it 'returns as usual if it does include accepts_incomplete' do
          delete '/v2/service_instances/fake-guid/service_bindings/binding-guid?accepts_incomplete=true'

          expect(last_response.status).to eq(202)
        end

        it 'rejects request if it does not include accepts_incomplete' do
          delete '/v2/service_instances/fake-guid/service_bindings/binding-guid'

          expect(last_response.status).to eq(422)
          expect(last_response.body).to eq(
                                          {
                                            'error' => 'AsyncRequired',
                                            'description' => 'This service plan requires client support for asynchronous service operations.'
                                          }.to_json
                                        )
        end
      end
    end
  end

  describe 'GET /v2/service_instances/:id/last_operation' do
    before do
      put '/v2/service_instances/fake-guid', {plan_id: 'fake-plan-guid'}.to_json
    end

    it 'should return 200 and the current status in the body' do
      get '/v2/service_instances/fake-guid/last_operation'
      expect(last_response.status).to eq(200)
      expect(last_response.body).to eq({state: "in progress"}.to_json)
    end

    context 'service instance does not exist' do
      it 'should return 410 - gone' do
        get '/v2/service_instances/non-existent/last_operation'
        expect(last_response.status).to eq(410)
        expect(last_response.body).to eq("Broker could not find service instance by the given id non-existent")
      end
    end
  end

  describe 'GET /v2/service_instances/:id/service_bindings/:id/last_operation' do
    before do
      put '/v2/service_instances/fake-guid', {plan_id: 'fake-plan-guid'}.to_json
      put '/v2/service_instances/fake-guid/service_bindings/binding-guid', {plan_id: 'fake-plan-guid'}.to_json
    end

    it 'should return 200 and the current status in the body' do
      get '/v2/service_instances/fake-guid/service_bindings/binding-guid/last_operation'
      expect(last_response.status).to eq(200)
      expect(last_response.body).to eq({state: "in progress"}.to_json)
    end

    context 'service binding does not exist' do
      it 'should return 410 - gone' do
        get '/v2/service_instances/fake-guid/service_bindings/non-existent/last_operation'
        expect(last_response.status).to eq(410)
        expect(last_response.body).to eq("Broker could not find the service binding `non-existent` for service instance `fake-guid`")
      end
    end

    context 'service binding is not associated with given service instance' do
      it 'should return 410 - gone' do
        get '/v2/service_instances/non-existent/service_bindings/binding-guid/last_operation'
        expect(last_response.status).to eq(410)
        expect(last_response.body).to eq("Broker could not find the service binding `binding-guid` for service instance `non-existent`")
      end
    end
  end

  describe 'cf api info location' do
    api_not_known_error = JSON.pretty_generate({
      "error" => true,
      "message" => "CF API info URL not known - either the cloud controller has not called the broker API yet, or it has failed to include a X-Api-Info-Location header that was a valid URL",
      "path" => "http://example.org/cf_api_info_url",
      "type" => '503'
    })

    context "no request to a /v2 endpoint has been made yet" do
      it 'responds with internal server error' do
        get '/cf_api_info_url'
        expect(last_response.status).to eq(503)
        expect(last_response.body).to eq(api_not_known_error)
      end
    end

    context "a /v2 request from the cloud controller had the X-Api-Info-Location header set to a valid url" do
      it 'receives a url in response' do
        info_endpoint = 'system-domain.com/v2/info'
        get '/v2/catalog', nil, { 'HTTP_X_API_INFO_LOCATION' => info_endpoint }
        get '/cf_api_info_url'
        expect(last_response.body).to eq(info_endpoint)
        expect(last_response.status).to eq(200)
      end
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

    context 'for a fetch service instance last operation' do
      before do
        put '/v2/service_instances/fake-guid', {plan_id: 'fake-plan-guid'}.to_json
      end

      it 'should change the response using a json body' do
        config = {
          max_fetch_service_instance_requests: 1,
          behaviors: {
            fetch_service_instance_last_operation: {
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
            fetch_service_instance_last_operation: {
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
            fetch_service_instance_last_operation: {
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
            fetch_service_instance_last_operation: {
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
            fetch_service_instance_last_operation: {
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

    context 'for a fetch service binding last operation' do
      before do
        put '/v2/service_instances/fake-guid', {plan_id: 'fake-plan-guid'}.to_json
        put '/v2/service_instances/fake-guid/service_bindings/binding-guid', {plan_id: 'fake-plan-guid'}.to_json
      end

      it 'should change the response using a json body' do
        config = {
          max_fetch_service_binding_requests: 1,
          behaviors: {
            fetch_service_binding_last_operation: {
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

        get '/v2/service_instances/fake-guid/service_bindings/binding-guid/last_operation'
        expect(last_response.status).to eq(200)
        expect(last_response.body).to eq('{}')

        get '/v2/service_instances/fake-guid/service_bindings/binding-guid/last_operation'
        expect(last_response.status).to eq(400)
        expect(last_response.body).to eq({ foo: :bar }.to_json)
      end

      it 'should change the response using an invalid json body' do
        config = {
          max_fetch_service_binding_requests: 1,
          behaviors: {
            fetch_service_binding_last_operation: {
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

        get '/v2/service_instances/fake-guid/service_bindings/binding-guid/last_operation'
        expect(last_response.status).to eq(200)
        expect(last_response.body).to eq 'cheese'

        get '/v2/service_instances/fake-guid/service_bindings/binding-guid/last_operation'
        expect(last_response.status).to eq(400)
        expect(last_response.body).to eq 'cake'
      end

      it 'should cause the action to sleep' do
        config = {
          max_fetch_service_binding_requests: 1,
          behaviors: {
            fetch_service_binding_last_operation: {
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
            get '/v2/service_instances/fake-guid/service_bindings/binding-guid/last_operation'
          end
        end.to raise_error(TimeoutError)

        expect do
          Timeout::timeout(0.5) do
            get '/v2/service_instances/fake-guid/service_bindings/binding-guid/last_operation'
          end
        end.to raise_error(TimeoutError)
      end

      it 'honors max_fetch_service_bindings_request' do
        config = {
          max_fetch_service_binding_requests: 2,
          behaviors: {
            fetch_service_binding_last_operation: {
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

        get '/v2/service_instances/fake-guid/service_bindings/binding-guid/last_operation'
        expect(last_response.status).to eq(200)
        expect(last_response.body).to eq('{}')

        get '/v2/service_instances/fake-guid/service_bindings/binding-guid/last_operation'
        expect(last_response.status).to eq(200)
        expect(last_response.body).to eq('{}')

        get '/v2/service_instances/fake-guid/service_bindings/binding-guid/last_operation'
        expect(last_response.status).to eq(400)
        expect(last_response.body).to eq({ foo: :bar }.to_json)
      end

      it 'can be customized on a per-plan basis' do
        config = {
          max_fetch_service_binding_requests: 1,
          behaviors: {
            fetch_service_binding_last_operation: {
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

        get '/v2/service_instances/fake-guid/service_bindings/binding-guid/last_operation'
        expect(last_response.status).to eq(200)
        expect(last_response.body).to eq({ foo: 'bar' }.to_json)

        get '/v2/service_instances/fake-guid/service_bindings/binding-guid/last_operation'
        expect(last_response.status).to eq(201)
        expect(last_response.body).to eq({ foo: 'baz' }.to_json)
      end
    end

    context 'for a fetch service instance' do
      before do
        put '/v2/service_instances/fake-guid', {plan_id: 'fake-plan-guid', context: {organization_guid: 'some-org-guid'}}.to_json
      end

      it 'should return the provision data and merge in the body from config' do
        config = {
          behaviors: {
            fetch_service_instance: {
              default: {
                status: 200,
                sleep_seconds: 0,
                body: {}
              }
            }
          }
        }.to_json

        post '/config', config

        get '/v2/service_instances/fake-guid'
        expect(last_response.status).to eq(200)
        expect(last_response.body).to eq({plan_id: 'fake-plan-guid', context: {organization_guid: 'some-org-guid'}}.to_json)
      end

      it 'should return an empty response if no body provided in the config' do
        config = {
          behaviors: {
            fetch_service_instance: {
              default: {
                status: 200,
                sleep_seconds: 0
              }
            }
          }
        }.to_json

        post '/config', config

        get '/v2/service_instances/fake-guid'
        expect(last_response.status).to eq(200)
        expect(last_response.body).to be_empty
      end

      it 'should overwrite provision data with configured behaviour' do
        config = {
          behaviors: {
            fetch_service_instance: {
              default: {
                status: 200,
                sleep_seconds: 0,
                body: {
                  context: {
                    organization_guid: "some-new-org-guid"
                  }
                }
              }
            }
          }
        }.to_json

        post '/config', config

        get '/v2/service_instances/fake-guid'
        expect(last_response.status).to eq(200)
        expect(last_response.body).to eq({plan_id: 'fake-plan-guid', context: {organization_guid: 'some-new-org-guid'}}.to_json)
      end

      it 'should change the response using an invalid json body' do
        config = {
          behaviors: {
            fetch_service_instance: {
              default: {
                status: 200,
                sleep_seconds: 0,
                raw_body: 'cheese'
              }
            }
          }
        }.to_json

        post '/config', config

        get '/v2/service_instances/fake-guid'
        expect(last_response.status).to eq(200)
        expect(last_response.body).to eq 'cheese'
      end

      it 'should cause the action to sleep' do
        config = {
          behaviors: {
            fetch_service_instance: {
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
            get '/v2/service_instances/fake-guid'
          end
        end.to raise_error(TimeoutError)
      end

      it 'can be customized on a per-plan basis' do
        config = {
          behaviors: {
            fetch_service_instance: {
              'fake-plan-guid' => {
                status: 200,
                sleep_seconds: 0,
                body: { foo: 'bar' }
              },
              default: {
                status: 200,
                sleep_seconds: 0,
                body: {}
              }
            }
          }
        }.to_json

        post '/config', config

        get '/v2/service_instances/fake-guid'
        expect(last_response.status).to eq(200)
        expect(last_response.body).to eq({plan_id: 'fake-plan-guid', context: {organization_guid: 'some-org-guid'}, foo: 'bar'}.to_json)
      end
    end

    context 'for a fetch service binding' do
      before do
        put '/v2/service_instances/fake-guid', {plan_id: 'fake-plan-guid'}.to_json
        put '/v2/service_instances/fake-guid/service_bindings/binding-guid', {plan_id: 'fake-plan-guid', context: {organization_guid: 'some-org-guid'}}.to_json

      end

      it 'should return the provision data and merge in the body from config' do
        config = {
          behaviors: {
            fetch_service_binding: {
              default: {
                status: 200,
                sleep_seconds: 0,
                body: {}
              }
            }
          }
        }.to_json

        post '/config', config

        get '/v2/service_instances/fake-guid/service_bindings/binding-guid'
        expect(last_response.status).to eq(200)
        expect(last_response.body).to eq({plan_id: 'fake-plan-guid', context: {organization_guid: 'some-org-guid'}}.to_json)
      end

      it 'should return an empty response if no body provided in the config' do
        config = {
          behaviors: {
            fetch_service_binding: {
              default: {
                status: 200,
                sleep_seconds: 0
              }
            }
          }
        }.to_json

        post '/config', config

        get '/v2/service_instances/fake-guid/service_bindings/binding-guid'
        expect(last_response.status).to eq(200)
        expect(last_response.body).to be_empty
      end

      it 'should overwrite provision data with configured behaviour' do
        config = {
          behaviors: {
            fetch_service_binding: {
              default: {
                status: 200,
                sleep_seconds: 0,
                body: {
                  context: {
                    organization_guid: "some-new-org-guid"
                  }
                }
              }
            }
          }
        }.to_json

        post '/config', config

        get '/v2/service_instances/fake-guid/service_bindings/binding-guid'
        expect(last_response.status).to eq(200)
        expect(last_response.body).to eq({plan_id: 'fake-plan-guid', context: {organization_guid: 'some-new-org-guid'}}.to_json)
      end

      it 'should change the response using an invalid json body' do
        config = {
          behaviors: {
            fetch_service_binding: {
              default: {
                status: 200,
                sleep_seconds: 0,
                raw_body: 'cheese'
              }
            }
          }
        }.to_json

        post '/config', config

        get '/v2/service_instances/fake-guid/service_bindings/binding-guid'
        expect(last_response.status).to eq(200)
        expect(last_response.body).to eq 'cheese'
      end

      it 'should cause the action to sleep' do
        config = {
          behaviors: {
            fetch_service_binding: {
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
            get '/v2/service_instances/fake-guid/service_bindings/binding-guid'
          end
        end.to raise_error(TimeoutError)
      end

      it 'can be customized on a per-plan basis' do
        config = {
          behaviors: {
            fetch_service_binding: {
              'fake-plan-guid' => {
                status: 200,
                sleep_seconds: 0,
                body: { foo: 'bar' }
              },
              default: {
                status: 200,
                sleep_seconds: 0,
                body: {}
              }
            }
          }
        }.to_json

        post '/config', config

        get '/v2/service_instances/fake-guid/service_bindings/binding-guid'
        expect(last_response.status).to eq(200)
        expect(last_response.body).to eq({plan_id: 'fake-plan-guid', context: {organization_guid: 'some-org-guid'}, foo: 'bar'}.to_json)
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
