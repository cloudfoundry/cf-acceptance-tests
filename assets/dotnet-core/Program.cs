using System;
using System.Collections.Generic;
using System.Linq;
using System.Threading.Tasks;
using Microsoft.AspNetCore.Hosting;
using Microsoft.Extensions.Configuration;
using Microsoft.Extensions.Hosting;
using Microsoft.Extensions.Logging;

namespace dotnet_core
{
    public class Program
    {
        public static void Main(string[] args)
        {
            AppDomain.CurrentDomain.ProcessExit +=
                 (sender, eventArgs) => {
                     Console.WriteLine("Goodbye, cruel world!");
                 };
            CreateHostBuilder(args).Build().Run();
        }

        public static IHostBuilder CreateHostBuilder(string[] args) =>
            Host.CreateDefaultBuilder(args)
                .ConfigureWebHostDefaults(webBuilder =>
                {
                    webBuilder.ConfigureKestrel((options) =>
                    {
                        string port = Environment.GetEnvironmentVariable("PORT") ?? "8080";
                        options.ListenAnyIP(Int32.Parse(port));
                    });
                    webBuilder.UseStartup<Startup>();
                });
    }
}
