using Newtonsoft.Json;
using System.Collections.Generic;

namespace Nora.helpers
{
    public class Service
    {
        [JsonProperty("name")]
        public string Name { get; internal set; }
        [JsonProperty("label")]
        public string Label { get; internal set; }
        [JsonProperty("tags")]
        public List<string> Tags { get; internal set; }
        [JsonProperty("credentials")]
        public IDictionary<string, string> Credentials { get; internal set; }
    }


    public class Services
    {
        [JsonProperty("user-provided")]
        public List<Service> UserProvided { get; private set; }

        [JsonProperty("p-mysql")]
        public List<Service> PMySQL { get; private set; }
    }
}