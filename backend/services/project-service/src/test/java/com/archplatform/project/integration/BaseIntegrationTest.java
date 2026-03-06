package com.archplatform.project.integration;

import com.archplatform.project.ProjectServiceApplication;
import com.fasterxml.jackson.databind.ObjectMapper;
import org.junit.jupiter.api.BeforeEach;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.autoconfigure.web.servlet.AutoConfigureMockMvc;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.test.annotation.DirtiesContext;
import org.springframework.test.context.ActiveProfiles;
import org.springframework.test.context.DynamicPropertyRegistry;
import org.springframework.test.context.DynamicPropertySource;
import org.springframework.test.web.servlet.MockMvc;

/**
 * Base class for all integration tests in Project Service.
 */
@SpringBootTest(classes = ProjectServiceApplication.class)
@AutoConfigureMockMvc
@ActiveProfiles("test")
@DirtiesContext(classMode = DirtiesContext.ClassMode.AFTER_EACH_TEST_METHOD)
public abstract class BaseIntegrationTest {

    @Autowired
    protected MockMvc mockMvc;

    @Autowired
    protected ObjectMapper objectMapper;

    @Autowired
    protected TestDataHelper testDataHelper;

    @DynamicPropertySource
    static void configureProperties(DynamicPropertyRegistry registry) {
        // H2 in-memory database
        registry.add("spring.datasource.url", () -> "jdbc:h2:mem:testdb;DB_CLOSE_DELAY=-1;DB_CLOSE_ON_EXIT=FALSE");
        registry.add("spring.datasource.driver-class-name", () -> "org.h2.Driver");
        registry.add("spring.datasource.username", () -> "sa");
        registry.add("spring.datasource.password", () -> "");
        registry.add("spring.jpa.hibernate.ddl-auto", () -> "create-drop");
        
        // Disable external services
        registry.add("spring.redis.enabled", () -> "false");
        registry.add("spring.kafka.enabled", () -> "false");
        
        // Mock User Service URL
        registry.add("user.service.url", () -> "http://localhost:8081");
    }

    @BeforeEach
    void setUp() {
        testDataHelper.clearAll();
    }

    protected String asJsonString(Object obj) throws Exception {
        return objectMapper.writeValueAsString(obj);
    }

    protected <T> T fromJsonString(String json, Class<T> clazz) throws Exception {
        return objectMapper.readValue(json, clazz);
    }

    /**
     * Generate a mock JWT token for testing.
     */
    protected String generateMockJwtToken(String userId, String username, String... roles) {
        // Simplified mock token for tests
        return "Bearer mock-jwt-token-" + userId;
    }
}
