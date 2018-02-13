package org.credhub;

import org.json.JSONObject;
import org.springframework.http.HttpEntity;
import org.springframework.http.HttpHeaders;
import org.springframework.http.MediaType;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RestController;
import org.springframework.web.client.RestTemplate;


@RestController
public class CredHubEnabledController {

  private RestTemplate restTemplate;
  private String interpolateEndpoint;
  private String serviceOfferingName;

  public CredHubEnabledController(RestTemplate restTemplate) {
    this.restTemplate = restTemplate;
    this.interpolateEndpoint = System.getenv("CREDHUB_API") + "/api/v1/interpolate";
    this.serviceOfferingName = System.getenv("SERVICE_NAME") != null ? System.getenv("SERVICE_NAME") : "credhub-read";
  }

  @GetMapping({"/test"})
  public String runTests() throws Exception {
    String vcapServices = System.getenv("VCAP_SERVICES");
    return this.interpolateServiceData(vcapServices).toString();
  }

  private JSONObject interpolateServiceData(String vcapServices) {
    HttpHeaders headers = new HttpHeaders();
    headers.setContentType(MediaType.APPLICATION_JSON);
    HttpEntity<String> entity = new HttpEntity<>(vcapServices,headers);
    JSONObject jsonObj = new JSONObject(restTemplate.postForObject(interpolateEndpoint, entity,  String.class));

    return jsonObj.getJSONArray(serviceOfferingName).getJSONObject(0).getJSONObject("credentials");
  }

}
