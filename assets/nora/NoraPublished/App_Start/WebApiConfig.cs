using System.Web.Http;

namespace Nora
{
    public static class WebApiConfig
    {
        public static void Register(HttpConfiguration config)
        {
            // Web API configuration and services

            // Web API routes
            config.MapHttpAttributeRoutes();
            config.Routes.MapHttpRoute("DefaultApi", "api/{controller}/{id}", new {id = RouteParameter.Optional});

           // Remove XML formatter
           var json = config.Formatters.JsonFormatter;
           json.SerializerSettings.PreserveReferencesHandling = Newtonsoft.Json.PreserveReferencesHandling.None;
           config.Formatters.Remove(config.Formatters.XmlFormatter);
        }
    }
}