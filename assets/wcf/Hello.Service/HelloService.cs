using System;
using System.Collections.Generic;
using System.Linq;
using System.Text;
using System.Threading.Tasks;
using System.ServiceModel;
using System.ServiceModel.Web;

namespace Hello.Service
{
    [ServiceContract]
    public interface IHelloService
    {
        [WebGet]
        string Echo(string msg);
    }

    public class HelloService : IHelloService
    {
        public string Echo(string msg)
        {
            return String.Format("{0},{1},{2}",
                msg,
                Environment.GetEnvironmentVariable("CF_INSTANCE_IP"),
                Environment.GetEnvironmentVariable("INSTANCE_GUID"));
        }
    }
}
