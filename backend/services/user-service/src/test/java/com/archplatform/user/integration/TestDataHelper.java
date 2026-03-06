package com.archplatform.user.integration;

import com.archplatform.user.entity.Role;
import com.archplatform.user.entity.User;
import com.archplatform.user.repository.RoleRepository;
import com.archplatform.user.repository.UserRepository;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.security.crypto.password.PasswordEncoder;
import org.springframework.stereotype.Component;

import java.util.HashSet;
import java.util.Set;
import java.util.UUID;

/**
 * Helper class for creating test data in integration tests.
 */
@Component
public class TestDataHelper {

    @Autowired
    private UserRepository userRepository;

    @Autowired
    private RoleRepository roleRepository;

    @Autowired
    private PasswordEncoder passwordEncoder;

    private final Set<UUID> createdUserIds = new HashSet<>();
    private final Set<UUID> createdRoleIds = new HashSet<>();

    /**
     * Create a test user with default values.
     */
    public User createTestUser() {
        return createTestUser("testuser", "test@example.com");
    }

    /**
     * Create a test user with specified username and email.
     */
    public User createTestUser(String username, String email) {
        return createTestUser(username, email, "password123");
    }

    /**
     * Create a test user with all specified values.
     */
    public User createTestUser(String username, String email, String password) {
        Role userRole = roleRepository.findByName("USER")
                .orElseGet(() -> createRole("USER"));

        User user = User.builder()
                .id(UUID.randomUUID())
                .username(username)
                .email(email)
                .password(passwordEncoder.encode(password))
                .firstName("Test")
                .lastName("User")
                .roles(Set.of(userRole))
                .enabled(true)
                .accountNonLocked(true)
                .accountNonExpired(true)
                .credentialsNonExpired(true)
                .mfaEnabled(false)
                .failedLoginAttempts(0)
                .build();

        User saved = userRepository.save(user);
        createdUserIds.add(saved.getId());
        return saved;
    }

    /**
     * Create a test admin user.
     */
    public User createAdminUser() {
        Role userRole = roleRepository.findByName("USER")
                .orElseGet(() -> createRole("USER"));
        Role adminRole = roleRepository.findByName("ADMIN")
                .orElseGet(() -> createRole("ADMIN"));

        User user = User.builder()
                .id(UUID.randomUUID())
                .username("admin")
                .email("admin@example.com")
                .password(passwordEncoder.encode("admin123"))
                .firstName("Admin")
                .lastName("User")
                .roles(Set.of(userRole, adminRole))
                .enabled(true)
                .accountNonLocked(true)
                .accountNonExpired(true)
                .credentialsNonExpired(true)
                .mfaEnabled(false)
                .failedLoginAttempts(0)
                .build();

        User saved = userRepository.save(user);
        createdUserIds.add(saved.getId());
        return saved;
    }

    /**
     * Create a role.
     */
    public Role createRole(String name) {
        Role role = Role.builder()
                .id(UUID.randomUUID())
                .name(name)
                .description(name + " role")
                .build();

        Role saved = roleRepository.save(role);
        createdRoleIds.add(saved.getId());
        return saved;
    }

    /**
     * Clear all test data.
     */
    public void clearAll() {
        createdUserIds.forEach(userRepository::deleteById);
        createdRoleIds.forEach(roleRepository::deleteById);
        createdUserIds.clear();
        createdRoleIds.clear();
    }

    /**
     * Get user repository for direct access.
     */
    public UserRepository getUserRepository() {
        return userRepository;
    }

    /**
     * Get role repository for direct access.
     */
    public RoleRepository getRoleRepository() {
        return roleRepository;
    }
}
