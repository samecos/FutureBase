package com.archplatform.user;

import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;

@SpringBootApplication(scanBasePackages = "com.archplatform.user", exclude = {
    net.devh.boot.grpc.server.autoconfigure.GrpcServerSecurityAutoConfiguration.class
})
public class UserServiceApplication {

    public static void main(String[] args) {
        SpringApplication.run(UserServiceApplication.class, args);
    }
}
