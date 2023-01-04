package org.cloudfoundry;

import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.EnableAutoConfiguration;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
@EnableAutoConfiguration
public class SpringBootTrivialApplication {
    @RequestMapping("/")
    String home() {
        return "ok";
    }

    public static void main(String[] args) {
        SpringApplication.run(SpringBootTrivialApplication.class, args);
    }
}
