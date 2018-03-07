using System;
using System.Collections;
using System.Net.Http;
using System.Threading;
using System.Web.Http;
using nora.Controllers;
using NSpec;

namespace nora.Tests.Controllers
{
    internal class InstancesControllerSpec : nspec
    {
        private void describe_()
        {
            InstancesController instancesController = null;
            String ID = null;

            before = () =>
            {
                instancesController = new InstancesController
                {
                    Request = new HttpRequestMessage(),
                    Configuration = new HttpConfiguration()
                };
            };

            describe["Get /"] = () =>
            {
                it["should return hello i am nora"] = () =>
                {
                    IHttpActionResult response = instancesController.Root();
                    String resp = null;
                    response.ExecuteAsync(new CancellationToken()).Result.TryGetContentValue(out resp);
                    resp.should_be("hello i am nora");
                };
            };

            describe["GET /id"] = () =>
            {
                it["should get the instance id from the VCAP_APPLICATION json"] = () =>
                {
                    IHttpActionResult response = instancesController.Id();
                    String resp = null;
                    response.ExecuteAsync(new CancellationToken()).Result.TryGetContentValue(out resp);
                    resp.should_be("A123F285-26B4-45F1-8C31-816DC5F53ECF");
                };
            };

            describe["Get /env"] = () =>
            {
                it["should return a list of ENV VARS"] = () =>
                {
                    IHttpActionResult response = instancesController.Env();
                    Hashtable resp = null;
                    response.ExecuteAsync(new CancellationToken()).Result.TryGetContentValue(out resp);
                    resp.should_be(Environment.GetEnvironmentVariables());
                };
            };
        }
    }
}