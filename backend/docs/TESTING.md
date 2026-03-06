# Testing Guide

Complete guide for testing the Architecture Platform backend.

## Test Structure

```
backend/
├── services/
│   ├── user-service/
│   │   └── src/test/java/
│   │       ├── com/archplatform/user/
│   │       │   ├── controller/     # Controller tests (WebMvcTest)
│   │       │   ├── service/        # Service tests (Mockito)
│   │       │   ├── repository/     # Repository tests (DataJpaTest)
│   │       │   ├── security/       # Security tests
│   │       │   └── integration/    # Integration tests
│   │       └── resources/
│   │           └── application-test.yml
│   └── ... (other services)
└── tests/
    └── integration/
        ├── e2e/                    # Postman/Newman tests
        └── performance/            # k6 load tests
```

## Test Types

### 1. Unit Tests

Test individual components in isolation.

#### Java (JUnit 5 + Mockito)

```java
@ExtendWith(MockitoExtension.class)
class UserServiceTest {
    @Mock
    private UserRepository userRepository;
    
    @InjectMocks
    private UserService userService;
    
    @Test
    void shouldCreateUser() {
        // given
        when(userRepository.save(any())).thenReturn(testUser);
        
        // when
        User result = userService.createUser(request);
        
        // then
        assertNotNull(result);
        verify(userRepository).save(any());
    }
}
```

#### Go (testing + testify)

```go
func TestCreateUser(t *testing.T) {
    // given
    mockRepo := new(MockUserRepository)
    service := NewUserService(mockRepo)
    
    // when
    result, err := service.CreateUser(ctx, request)
    
    // then
    assert.NoError(t, err)
    assert.NotNil(t, result)
    mockRepo.AssertExpectations(t)
}
```

### 2. Integration Tests

Test component interactions with real (in-memory) database.

#### Java (@SpringBootTest)

```java
@SpringBootTest
@AutoConfigureMockMvc
@ActiveProfiles("test")
class UserRegistrationIntegrationTest extends BaseIntegrationTest {
    
    @Test
    void shouldRegisterUser() throws Exception {
        mockMvc.perform(post("/api/v1/auth/register")
                .contentType(MediaType.APPLICATION_JSON)
                .content(asJsonString(request)))
                .andExpect(status().isCreated())
                .andExpect(jsonPath("$.username").value("testuser"));
    }
}
```

### 3. Controller Tests

Test HTTP layer with mocked services.

```java
@WebMvcTest(UserController.class)
class UserControllerTest {
    @Autowired
    private MockMvc mockMvc;
    
    @MockBean
    private UserService userService;
    
    @Test
    @WithMockUser
    void shouldGetUser() throws Exception {
        when(userService.findById(id)).thenReturn(Optional.of(user));
        
        mockMvc.perform(get("/api/v1/users/{id}", id))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.id").value(id.toString()));
    }
}
```

### 4. Repository Tests

Test database layer with in-memory H2.

```java
@DataJpaTest
class UserRepositoryTest {
    @Autowired
    private TestEntityManager entityManager;
    
    @Autowired
    private UserRepository userRepository;
    
    @Test
    void shouldFindByUsername() {
        entityManager.persist(user);
        
        Optional<User> found = userRepository.findByUsername("testuser");
        
        assertTrue(found.isPresent());
    }
}
```

## Running Tests

### Quick Commands

```bash
# Run all tests
make test

# Run only Java tests
make test-java

# Run only Go tests
make test-go

# Run only unit tests (exclude integration)
make test-java-unit

# Run only integration tests
make test-java-integration

# Run all tests including integration
make test-all

# Run tests with coverage
make test-coverage
```

### Manual Commands

#### Java

```bash
# All tests in a service
cd services/user-service
./mvnw test

# Specific test class
./mvnw test -Dtest=UserServiceTest

# Integration tests only
./mvnw test -Dtest="*IntegrationTest"

# With coverage
./mvnw test jacoco:report
# Report: target/site/jacoco/index.html
```

#### Go

```bash
# All tests in a service
cd services/collaboration-service
go test ./...

# Verbose output
go test -v ./...

# With coverage
go test -cover ./...
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Test Configuration

### H2 Database (for tests)

```yaml
# application-test.yml
spring:
  datasource:
    url: jdbc:h2:mem:testdb;DB_CLOSE_DELAY=-1;MODE=PostgreSQL
    driver-class-name: org.h2.Driver
  jpa:
    hibernate:
      ddl-auto: create-drop
```

### Test Profiles

- `@ActiveProfiles("test")` - Use test configuration
- `@TestPropertySource` - Override specific properties

## Test Coverage

### Current Coverage

| Service | Unit Tests | Integration Tests | Coverage |
|---------|-----------|-------------------|----------|
| user-service | 5 | 3 | TBD |
| project-service | 5 | 3 | TBD |
| property-service | 1 | 0 | TBD |
| version-service | 1 | 0 | TBD |
| search-service | 1 | 0 | TBD |
| collaboration-service | 2 | 0 | TBD |
| geometry-service | 2 | 0 | TBD |
| file-service | 1 | 0 | TBD |

### Coverage Goals

- **Unit Tests**: 70%+ line coverage
- **Integration Tests**: Cover all critical paths
- **Total Coverage**: 60%+ combined

## Writing Good Tests

### AAA Pattern

```java
@Test
void shouldDoSomething() {
    // Arrange (Given)
    User user = createTestUser();
    
    // Act (When)
    User result = service.save(user);
    
    // Assert (Then)
    assertNotNull(result.getId());
}
```

### Test Naming

- `should[ExpectedBehavior]When[Condition]`
- Examples:
  - `shouldCreateUserWhenValidRequest`
  - `shouldReturn404WhenUserNotFound`
  - `shouldThrowExceptionWhenDuplicateEmail`

### Best Practices

1. **One concept per test** - Test one thing at a time
2. **Independent tests** - Tests should not depend on each other
3. **Clear assertions** - Use descriptive assertion messages
4. **Setup/Teardown** - Clean up test data after each test
5. **Test data builders** - Use builder pattern for test data

## E2E Tests

### Postman/Newman

```bash
# Run E2E tests
make test-e2e

# Or manually
newman run tests/integration/e2e/user-project-flow.json
```

### Test Scenarios

1. **User Registration Flow**
   - Register → Login → Get Profile → Update Profile → Delete

2. **Project Management Flow**
   - Create Project → Add Members → Update → Archive → Delete

3. **File Upload Flow**
   - Get Upload URL → Upload File → Download File → Delete File

## Performance Tests

### k6

```bash
# Run performance tests
make test-performance

# Or manually
k6 run tests/integration/performance/load-test.js
```

### Scenarios

- **Load Test**: 100 concurrent users for 10 minutes
- **Stress Test**: Ramp up to 1000 users
- **Spike Test**: Sudden increase to 500 users

## CI Integration

Tests run automatically in CI:

```yaml
# .github/workflows/ci.yml
- name: Unit Tests
  run: make test

- name: Integration Tests
  run: make test-integration

- name: Coverage Report
  run: make test-coverage
```

## Troubleshooting

### Common Issues

1. **Tests fail on H2 but pass on PostgreSQL**
   - Use `MODE=PostgreSQL` in H2 URL
   - Avoid PostgreSQL-specific features in tests

2. **MockMvc returns 403**
   - Add `@WithMockUser` or `.with(csrf())`
   - Configure Spring Security for tests

3. **Database connection issues**
   - Check `application-test.yml` configuration
   - Ensure `@ActiveProfiles("test")` is present

4. **Go tests hang**
   - Check for goroutine leaks
   - Use `t.Parallel()` carefully
   - Add timeouts to tests

## Resources

- [JUnit 5 User Guide](https://junit.org/junit5/docs/current/user-guide/)
- [Mockito Documentation](https://javadoc.io/doc/org.mockito/mockito-core/latest/org/mockito/Mockito.html)
- [Go Testing](https://golang.org/pkg/testing/)
- [Testify](https://github.com/stretchr/testify)
