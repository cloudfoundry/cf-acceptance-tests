using System;
using System.Web;
using System.Web.Http;

namespace Nora
{
    public class WebApiApplication : HttpApplication
    {
        protected void Application_Start()
        {
            GlobalConfiguration.Configure(WebApiConfig.Register);
        }

        void Application_Error(Object sender, EventArgs e)
        {
            var exception = Server.GetLastError();
            if (exception == null)
                return;

            Console.WriteLine("error: " + exception);
        }
    }
}