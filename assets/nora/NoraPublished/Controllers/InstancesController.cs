using System.IO.MemoryMappedFiles;
using MySql.Data.MySqlClient;
using Newtonsoft.Json;
using Nora.helpers;
using System;
using System.Collections.Generic;
using System.Diagnostics;
using System.IO;
using System.Net;
using System.Net.Http;
using System.Net.Sockets;
using System.Web;
using System.Web.Http;
using System.Web.Http.Results;
using System.Security;

namespace nora.Controllers
{
    public class InstancesController : ApiController
    {
        private static Services services;

        static InstancesController()
        {
            var env = Environment.GetEnvironmentVariable("VCAP_SERVICES");
            if (env != null)
            {
                services = JsonConvert.DeserializeObject<Services>(env);
            }
        }

        private static string FileAccessStatus(string path)
        {
            try
            {
                Directory.EnumerateFiles(path);
                return "ACCESS_ALLOWED";
            }
            catch (UnauthorizedAccessException)
            {
                return "ACCESS_DENIED";
            }
            catch (SecurityException)
            {
                return "ACCESS_DENIED";
            }
            catch (Exception)
            {
                if (File.Exists(path))
                {
                    return "ACCESS_ALLOWED";
                }
                try
                {
                    var stream = File.OpenRead(path);
                    stream.Close();
                }
                catch (UnauthorizedAccessException)
                {
                    return "ACCESS_DENIED";
                }
                catch (FileNotFoundException)
                {
                    return "NOT_EXIST";
                }
                catch (Exception ex)
                {
                    return "EXCEPTION: " + ex.ToString();
                }
                return "ACCESS_ALLOWED";
            }
        }

        [Route("~/healthcheck")]
        [HttpGet]
        public IHttpActionResult Healthcheck()
        {
            return Ok("Healthcheck passed");
        }

        [Route("~/redirect/{path}")]
        [HttpGet]
        public RedirectResult RedirectTo(string path)
        {
            return Redirect(Url.Content("~/" + path));
        }

        [Route("~/inaccessible_file")]
        [HttpPost]
        public IHttpActionResult InaccessibleFiles()
        {
            var result = Request.Content.ReadAsStringAsync().GetAwaiter().GetResult();
            return Ok(FileAccessStatus(result));
        }

        [Route("~/")]
        [HttpGet]
        public IHttpActionResult Root()
        {
            return Ok(String.Format("hello i am nora running on {0}", Request.RequestUri.AbsoluteUri));
        }

        [Route("~/headers")]
        [HttpGet]
        public IHttpActionResult Headers()
        {
            return Ok(Request.Headers);
        }

        [Route("~/print/{output}")]
        [HttpGet]
        public IHttpActionResult Print(string output)
        {
            System.Console.WriteLine(output);
            return Ok(Request.Headers);
        }

        [Route("~/print_err/{output}")]
        [HttpGet]
        public IHttpActionResult PrintErr(string output)
        {
            Console.Error.WriteLine(output);
            return Ok(Request.Headers);
        }

        [Route("~/id")]
        [HttpGet]
        public IHttpActionResult Id()
        {
            var uuid = Environment.GetEnvironmentVariable("INSTANCE_GUID");
            return Ok(uuid);
        }

        [Route("~/env")]
        [HttpGet]
        public IHttpActionResult Env()
        {
            return Ok(Environment.GetEnvironmentVariables());
        }

        [Route("~/curl/{host}/{port}")]
        [HttpGet]
        public IHttpActionResult Curl(string host, int port)
        {
            Console.WriteLine("Starting /curl handling...");
            var req = WebRequest.Create("http://" + host + ":" + port);
            Console.WriteLine("Created request...");
            req.Timeout = 10000;
            try
            {
                var resp = (HttpWebResponse)req.GetResponse();
                Console.WriteLine("Got response...");
                return Json(new
                {
                    stdout = new StreamReader(resp.GetResponseStream()).ReadToEnd(),
                    return_code = 0,
                });
            }
            catch (WebException ex)
            {
                Console.WriteLine("Got an exception: ", ex);
                return Json(new
                {
                    stderr = ex.Message,
                    // ex.Response != null if the response status code wasn't a success,
                    // null if the operation timedout
                    return_code = ex.Response != null ? 0 : 1,
                });
            }
        }

        [Route("~/connect/{host}/{port}")]
        [HttpGet]
        public IHttpActionResult Connect(string host, int port)
        {
            string stdout = "", stderr = "";
            int return_code = 0;
            TcpClient client = new TcpClient();
            try
            {
                client.Connect(host, port);
                return_code = 0;
                stdout = string.Format("Successful TCP connection to {0}:{1}", host, port);
            }
            catch (SocketException)
            {
                stderr = string.Format("Unable to make TCP connection to {0}:{1}", host, port);
                return_code = 1;
            }
            catch (Exception e)
            {
                stderr = e.Message;
                return_code = 2;
            }

            return Json(new
            {
                stdout = stdout,
                stderr = stderr,
                return_code = return_code
            });
        }

        [Route("~/env/{name}")]
        [HttpGet]
        public IHttpActionResult EnvName(string name)
        {
            return Ok(Environment.GetEnvironmentVariable(name));
        }

        [Route("~/logspew/{kbytes}")]
        [HttpGet]
        public IHttpActionResult LogSpew(int kbytes)
        {
            var kb = new string('1', 1024);
            for (var i = 0; i < kbytes; i++)
            {
                Console.WriteLine(kb);
            }
            return Ok(String.Format("Just wrote {0} kbytes to the log", kbytes));
        }

        [Route("~/users")]
        [HttpGet]
        public IHttpActionResult CupsUsers()
        {
            if (services.UserProvided.Count == 0)
            {
                var msg = new HttpResponseMessage();
                msg.StatusCode = HttpStatusCode.NotFound;
                msg.Content = new StringContent("No services found");
                return new ResponseMessageResult(msg);
            }

            var service = services.UserProvided[0];

            var users = UsersFromService(service);
            return Ok(users);
        }

        [Route("~/pmysql")]
        [HttpGet]
        public IHttpActionResult PMysqlUsers()
        {
            if (services.PMySQL.Count == 0)
            {
                var msg = new HttpResponseMessage();
                msg.StatusCode = HttpStatusCode.NotFound;
                msg.Content = new StringContent("No services found");
                return new ResponseMessageResult(msg);
            }

            var service = services.PMySQL[0];

            var users = UsersFromService(service);

            return Ok(users);
        }

        [Route("~/run")]
        [HttpPost]
        public IHttpActionResult Run()
        {
            var result = Request.Content.ReadAsStringAsync().GetAwaiter().GetResult();
            var path = HttpContext.Current.Request.MapPath(result);
            Process.Start(path);
            return Ok("Started: " + path);
        }

        [Route("~/commitcharge")]
        [HttpGet]
        public IHttpActionResult GetCommitCharge()
        {
            var p = new PerformanceCounter("Memory", "Committed Bytes");
            return Ok(p.RawValue);
        }

        private static MemoryMappedFile MmapFile = null;

        [Route("~/mmapleak/{maxbytes}")]
        [HttpGet]
        public IHttpActionResult MmapLeakMax(long maxbytes)
        {
            if (MmapFile != null)
            {
                MmapFile.Dispose();
            }

            MmapFile = MemoryMappedFile.CreateNew(
                Guid.NewGuid().ToString(),
                maxbytes,
                MemoryMappedFileAccess.ReadWrite);

            return Ok();
        }

        [Route("~/exit")]
        [HttpGet]
        public IHttpActionResult Exit()
        {
            Process.GetCurrentProcess().Kill();
            return Ok();
        }

        private static List<IntPtr> _leakedPointers;
        [Route("~/leakmemory/{mb}")]
        [HttpGet]
        public IHttpActionResult Memory(int mb)
        {
            if (_leakedPointers == null)
                _leakedPointers = new List<IntPtr>();

            var bytes = mb * 1024 * 1024;
            _leakedPointers.Add(System.Runtime.InteropServices.Marshal.AllocHGlobal(bytes));
            return Ok();
        }



        private static List<string> UsersFromService(Service service)
        {
            var creds = service.Credentials;
            var username = creds["username"];
            var password = creds["password"];
            var host = creds.ContainsKey("host") ? creds["host"] : creds["hostname"];
            var dbname = creds.ContainsKey("name") ? creds["name"] : "mysql";
            var connString = String.Format("server={0};uid={1};pwd={2};database={3}", host, username, password, dbname);

            Console.WriteLine("Connecting to mysql using {0}", connString);

            var users = new List<string>();

            using (var conn = new MySqlConnection())
            {
                conn.ConnectionString = connString;
                conn.Open();

                new MySqlCommand(
                    "CREATE TABLE IF NOT EXISTS Hits(Id INT PRIMARY KEY AUTO_INCREMENT, CreatedAt DATETIME) ENGINE=INNODB;", conn)
                    .ExecuteNonQuery();

                new MySqlCommand(
                    "INSERT INTO Hits(CreatedAt)VALUES(now());", conn)
                    .ExecuteNonQuery();

                using (var cmd = new MySqlCommand("select CreatedAt from Hits order by id desc limit 10", conn))
                {
                    using (var reader = cmd.ExecuteReader())
                    {
                        var colIdx = reader.GetOrdinal("CreatedAt");
                        while (reader.Read())
                        {
                            users.Add(reader.GetString(colIdx));
                        }
                    }
                }
            }
            return users;
        }
    }
}
