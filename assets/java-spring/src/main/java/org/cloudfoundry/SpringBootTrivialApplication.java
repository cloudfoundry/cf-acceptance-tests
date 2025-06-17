package org.cloudfoundry;

import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;
import org.springframework.web.client.RestTemplate;

import java.net.URI;
import java.net.URISyntaxException;
import java.util.HashMap;
import java.util.Map;
import java.util.logging.Logger;

@SpringBootApplication
public class SpringBootTrivialApplication {

    public static void main(String[] args) {
        SpringApplication.run(SpringBootTrivialApplication.class, args);
    }
}

@RestController
@RequestMapping("")
class IPv6TesterController {

    private static final Logger logger = Logger.getLogger(IPv6TesterController.class.getName());
    private static final Map<String, EndpointInfo> ENDPOINT_TYPE_MAP;

    static {
        ENDPOINT_TYPE_MAP = new HashMap<>();
        ENDPOINT_TYPE_MAP.put("api.ipify.org", new EndpointInfo("/ipv4-test"));
        ENDPOINT_TYPE_MAP.put("api6.ipify.org", new EndpointInfo("/ipv6-test"));
        ENDPOINT_TYPE_MAP.put("api64.ipify.org", new EndpointInfo("/dual-stack-test"));
    }

    @GetMapping("/")
    public String home() {
        return "ok";
    }

    @GetMapping("/ipv4-test")
    public String testIPv4() {
        return testEndpoint("api.ipify.org");
    }

    @GetMapping("/ipv6-test")
    public String testIPv6() {
        return testEndpoint("api6.ipify.org");
    }

    @GetMapping("/dual-stack-test")
    public String testDualStack() {
        return testEndpoint("api64.ipify.org");
    }

    private String testEndpoint(String endpoint) {
        EndpointInfo endpointInfo = ENDPOINT_TYPE_MAP.get(endpoint);
        try {
            logger.info("Testing endpoint: " + endpoint);
            RestTemplate restTemplate = new RestTemplate();
            URI uri = new URI("http://" + endpoint + "/");

            String response = restTemplate.getForObject(uri, String.class);
            if (response == null || response.isEmpty()) {
                throw new RuntimeException("Empty response from " + endpoint);
            }

            return response;

        } catch (URISyntaxException e) {
            logger.severe("URI syntax issue with " + endpoint + ": " + e.getMessage());
            return e.getMessage();
        } catch (Exception e) {
            logger.severe("Failed to reach " + endpoint + ": " + e.getMessage());
            return e.getMessage();
        }
    }

    private static record EndpointInfo(String path) {
    }
}
