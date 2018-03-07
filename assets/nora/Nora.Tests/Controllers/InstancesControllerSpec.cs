using System;
using System.Collections;
using System.Net;
using System.Net.Http;
using System.Runtime.CompilerServices;
using System.Threading;
using System.Web.Http;
using nora.Controllers;
using Newtonsoft.Json;
using NSpec;

namespace nora.Tests.Controllers
{
    internal class InstancesControllerSpec : nspec
    {
        private void describe_()
        {
            InstancesController instancesController = null;

            before = () =>
            {
                instancesController = new InstancesController
                {
                    Request = new HttpRequestMessage(HttpMethod.Get, "http://example.com"),
                    Configuration = new HttpConfiguration()
                };
            };

            describe["Get /"] = () =>
            {
                it["should return the hello message"] = () =>
                {
                    var response = instancesController.Root();
                    string resp;
                    response.ExecuteAsync(new CancellationToken()).Result.TryGetContentValue(out resp);
                    resp.should_be("hello i am nora running on http://example.com/");
                };
            };

            describe["GET /id"] = () =>
            {
                it["should get the instance id from the INSTANCE_GUID"] = () =>
                {
                    var instanceGuid = Guid.NewGuid().ToString();
                    Environment.SetEnvironmentVariable("INSTANCE_GUID", instanceGuid);

                    var response = instancesController.Id();
                    string resp;
                    response.ExecuteAsync(new CancellationToken()).Result.TryGetContentValue(out resp);
                    resp.should_be(instanceGuid);
                };
            };

            describe["Get /env"] = () =>
            {
                it["should return a list of ENV VARS"] = () =>
                {
                    var response = instancesController.Env();
                    Hashtable resp;
                    response.ExecuteAsync(new CancellationToken()).Result.TryGetContentValue(out resp);
                    resp.should_be(Environment.GetEnvironmentVariables());
                };
            };

            describe["Get /env/:name"] = () =>
            {
                it["should return the desired named ENV VAR"] = () =>
                {
                    Environment.SetEnvironmentVariable("FRED", "JANE");

                    var response = instancesController.EnvName("FRED");
                    string resp;
                    response.ExecuteAsync(new CancellationToken()).Result.TryGetContentValue(out resp);

                    resp.should_be("JANE");
                };
            };

            describe["Get /connect/:ip/:port"] = () =>
            {
                it["should make a tcp connection to the specified ip:port"] = () =>
                {
                    var response = instancesController.Connect("8.8.8.8", 53);
                    var json = response.ExecuteAsync(new CancellationToken()).Result.Content.ReadAsStringAsync();
                    json.Wait();
                    json.Result.should_be("{\"stdout\":\"Successful TCP connection to 8.8.8.8:53\",\"stderr\":\"\",\"return_code\":0}");
                };

                context["when the the ip:port specified are not reachable"] = () =>
                {
                    it["returns an error"] = () =>
                    {
                        var response = instancesController.Connect("127.0.0.1", 20);
                        var json = response.ExecuteAsync(new CancellationToken()).Result.Content.ReadAsStringAsync();
                        json.Wait();
                        json.Result.should_be("{\"stdout\":\"\",\"stderr\":\"Unable to make TCP connection to 127.0.0.1:20\",\"return_code\":1}");
                    };
                };

                context["when the the ip specified is null"] = () =>
                {
                    it["returns an error"] = () =>
                    {
                        var response = instancesController.Connect(null, 53);
                        var json = response.ExecuteAsync(new CancellationToken()).Result.Content.ReadAsStringAsync();
                        json.Wait();
                        json.Result.Contains("\"return_code\":2").should_be_true();
                    };
                };

                context["when the the port specified is invalid"] = () =>
                {
                    it["returns an error"] = () =>
                    {
                        var response = instancesController.Connect("127.0.0.1", IPEndPoint.MinPort - 1);
                        var json = response.ExecuteAsync(new CancellationToken()).Result.Content.ReadAsStringAsync();
                        json.Wait();
                        json.Result.Contains("\"return_code\":2").should_be_true();

                        response = instancesController.Connect("127.0.0.1", IPEndPoint.MaxPort + 1);
                        json = response.ExecuteAsync(new CancellationToken()).Result.Content.ReadAsStringAsync();
                        json.Wait();
                        json.Result.Contains("\"return_code\":2").should_be_true();
                    };
                };
            };
        }
    }
}