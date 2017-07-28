package org.credhub;

import org.springframework.context.annotation.Configuration;
import org.springframework.context.annotation.Import;
import org.springframework.credhub.configuration.CredHubConfiguration;

@Configuration
@Import({CredHubConfiguration.class})
public class CredHubEnabledConfiguration {
  public CredHubEnabledConfiguration() {
  }
}
