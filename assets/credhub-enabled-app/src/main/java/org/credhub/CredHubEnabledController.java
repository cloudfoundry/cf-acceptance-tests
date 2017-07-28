package org.credhub;

import com.fasterxml.jackson.databind.ObjectMapper;
import org.springframework.credhub.core.CredHubTemplate;
import org.springframework.credhub.support.ServicesData;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RestController;

import java.io.IOException;
import java.util.HashMap;
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
    return ((Map)((List)this.interpolateServiceData(vcapServices).get("credhub-read")).get(0)).get("credentials");
  }

  private ServicesData interpolateServiceData(String vcapServices) throws IOException {
    servicesData = this.buildServicesData(vcapServices);
    return this.credHubTemplate.interpolateServiceData(servicesData);
  }

  private ServicesData buildServicesData(String vcapServices) throws IOException {
    ObjectMapper mapper = new ObjectMapper();
    return (ServicesData)mapper.readValue(vcapServices, ServicesData.class);
  }

  @PostMapping({"/cleanup"})
  public void doCleanup() throws Exception {
    if (servicesData != null) {
      HashMap<String, String> credentials = (HashMap) servicesData.get("credhub-read").get(0)
          .get("credentials");
      String credentialName = credentials.get("credhub-ref");
      credHubTemplate.deleteByName(credentialName);
    }

  }
}
