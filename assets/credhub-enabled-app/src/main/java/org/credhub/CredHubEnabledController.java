package org.credhub;

import com.fasterxml.jackson.databind.ObjectMapper;
import org.springframework.credhub.core.CredHubTemplate;
import org.springframework.credhub.support.ServicesData;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RestController;

import java.io.IOException;
import java.util.List;
import java.util.Map;

@RestController
public class CredHubEnabledController {
  private CredHubTemplate credHubTemplate;
  private ServicesData servicesData;

  public CredHubEnabledController(CredHubTemplate credHubTemplate) {
    this.credHubTemplate = credHubTemplate;
  }

  @GetMapping({"/test"})
  public Object runTests() throws Exception {
    String vcapServices = System.getenv("VCAP_SERVICES");
    String serviceOfferingName = System.getenv("SERVICE_NAME") != null ? System.getenv("SERVICE_NAME") : "credhub-read";
    return ((Map)((List)this.interpolateServiceData(vcapServices).get(serviceOfferingName)).get(0)).get("credentials");
  }

  private ServicesData interpolateServiceData(String vcapServices) throws IOException {
    servicesData = this.buildServicesData(vcapServices);
    return this.credHubTemplate.interpolateServiceData(servicesData);
  }

  private ServicesData buildServicesData(String vcapServices) throws IOException {
    ObjectMapper mapper = new ObjectMapper();
    return mapper.readValue(vcapServices, ServicesData.class);
  }
}
