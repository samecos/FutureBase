# Integration Tests

This directory contains integration tests that verify the interaction between multiple services.

## Structure

```
tests/integration/
├── README.md              # This file
├── e2e/                   # End-to-end tests
├── contracts/             # Consumer-driven contract tests
└── performance/           # Performance tests (k6 scripts)
```

## Running Integration Tests

### Prerequisites

- Java 17+
- Maven 3.8+
- Go 1.21+ (for Go service tests)

### Running Tests

```bash
# Run all integration tests
make test-integration

# Run specific service integration tests
cd services/user-service && ./mvnw test -P integration-test
cd services/project-service && ./mvnw test -P integration-test

# Run Go service integration tests
cd services/collaboration-service && go test -tags=integration ./...
```

## Test Categories

### 1. API Integration Tests

Test HTTP endpoints with full Spring context:

- **Location**: `services/*/src/test/java/**/integration/`
- **Base Class**: `BaseIntegrationTest`
- **Database**: H2 in-memory
- **External Services**: Mocked

### 2. End-to-End Tests

Test complete user flows across multiple services:

- **Location**: `tests/integration/e2e/`
- **Tools**: Postman, k6, or custom scripts
- **Environment**: Docker Compose (when available)

### 3. Contract Tests

Verify API contracts between services:

- **Location**: `tests/integration/contracts/`
- **Tools**: Pact or Spring Cloud Contract

## Writing Integration Tests

### Java Services

```java
@SpringBootTest
@AutoConfigureMockMvc
@ActiveProfiles("test")
class MyIntegrationTest extends BaseIntegrationTest {
    
    @Test
    void shouldCreateResource() throws Exception {
        // Given
        CreateRequest request = new CreateRequest();
        request.setName("Test");
        
        // When & Then
        mockMvc.perform(post("/api/v1/resource")
                .contentType(MediaType.APPLICATION_JSON)
                .content(asJsonString(request)))
                .andExpect(status().isCreated())
                .andExpect(jsonPath("$.id").exists());
    }
}
```

### Go Services

```go
func TestIntegrationCreateResource(t *testing.T) {
    // Setup test server
    srv := setupTestServer()
    defer srv.Close()
    
    // Make request
    resp, err := http.Post(srv.URL+"/api/v1/resource", ...)
    
    // Assert
    assert.Equal(t, http.StatusCreated, resp.StatusCode)
}
```

## Test Data Management

Use `TestDataHelper` to create and clean up test data:

```java
@Autowired
private TestDataHelper testDataHelper;

@BeforeEach
void setUp() {
    testDataHelper.clearAll();
}

@Test
void test() {
    User user = testDataHelper.createTestUser();
    Project project = testDataHelper.createTestProject(user.getId());
    // ... test
}
```

## Mocking External Services

### User Service Mock

```java
@MockBean
private UserServiceClient userServiceClient;

@BeforeEach
void mockUserService() {
    when(userServiceClient.getUser(any()))
        .thenReturn(new UserDTO("test-user", "test@example.com"));
}
```

## CI Integration

Integration tests run in CI pipeline:

```yaml
# .github/workflows/ci.yml
- name: Run Integration Tests
  run: make test-integration
```

## Best Practices

1. **Isolation**: Each test should be independent
2. **Cleanup**: Always clean up test data in `@AfterEach`
3. **Performance**: Keep integration tests fast (< 1s per test)
4. **Coverage**: Focus on critical paths and edge cases
5. **Documentation**: Document test scenarios and expected outcomes
