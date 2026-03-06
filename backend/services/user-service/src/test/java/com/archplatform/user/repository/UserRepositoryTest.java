package com.archplatform.user.repository;

import com.archplatform.user.entity.Role;
import com.archplatform.user.entity.User;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.autoconfigure.orm.jpa.DataJpaTest;
import org.springframework.boot.test.autoconfigure.orm.jpa.TestEntityManager;
import org.springframework.security.crypto.bcrypt.BCryptPasswordEncoder;
import org.springframework.security.crypto.password.PasswordEncoder;

import java.util.Optional;
import java.util.Set;
import java.util.UUID;

import static org.junit.jupiter.api.Assertions.*;

@DataJpaTest
class UserRepositoryTest {

    @Autowired
    private TestEntityManager entityManager;

    @Autowired
    private UserRepository userRepository;

    @Autowired
    private RoleRepository roleRepository;

    private PasswordEncoder passwordEncoder = new BCryptPasswordEncoder();
    private Role userRole;

    @BeforeEach
    void setUp() {
        userRole = Role.builder()
                .name("USER")
                .description("Standard user role")
                .build();
        roleRepository.save(userRole);
    }

    @Test
    @DisplayName("Should save and retrieve user by ID")
    void saveAndFindById_Success() {
        // Given
        User user = User.builder()
                .id(UUID.randomUUID())
                .username("testuser")
                .email("test@example.com")
                .password(passwordEncoder.encode("password"))
                .firstName("Test")
                .lastName("User")
                .roles(Set.of(userRole))
                .enabled(true)
                .build();

        // When
        User saved = entityManager.persistAndFlush(user);
        Optional<User> found = userRepository.findById(saved.getId());

        // Then
        assertTrue(found.isPresent());
        assertEquals("testuser", found.get().getUsername());
        assertEquals("test@example.com", found.get().getEmail());
    }

    @Test
    @DisplayName("Should find user by username")
    void findByUsername_Success() {
        // Given
        User user = User.builder()
                .username("uniqueuser")
                .email("unique@example.com")
                .password(passwordEncoder.encode("password"))
                .roles(Set.of(userRole))
                .enabled(true)
                .build();
        entityManager.persistAndFlush(user);

        // When
        Optional<User> found = userRepository.findByUsername("uniqueuser");

        // Then
        assertTrue(found.isPresent());
        assertEquals("uniqueuser", found.get().getUsername());
    }

    @Test
    @DisplayName("Should find user by email")
    void findByEmail_Success() {
        // Given
        User user = User.builder()
                .username("emailuser")
                .email("findme@example.com")
                .password(passwordEncoder.encode("password"))
                .roles(Set.of(userRole))
                .enabled(true)
                .build();
        entityManager.persistAndFlush(user);

        // When
        Optional<User> found = userRepository.findByEmail("findme@example.com");

        // Then
        assertTrue(found.isPresent());
        assertEquals("findme@example.com", found.get().getEmail());
    }

    @Test
    @DisplayName("Should check if username exists")
    void existsByUsername_Success() {
        // Given
        User user = User.builder()
                .username("existinguser")
                .email("existing@example.com")
                .password(passwordEncoder.encode("password"))
                .roles(Set.of(userRole))
                .enabled(true)
                .build();
        entityManager.persistAndFlush(user);

        // When & Then
        assertTrue(userRepository.existsByUsername("existinguser"));
        assertFalse(userRepository.existsByUsername("nonexistentuser"));
    }

    @Test
    @DisplayName("Should check if email exists")
    void existsByEmail_Success() {
        // Given
        User user = User.builder()
                .username("emailcheck")
                .email("check@example.com")
                .password(passwordEncoder.encode("password"))
                .roles(Set.of(userRole))
                .enabled(true)
                .build();
        entityManager.persistAndFlush(user);

        // When & Then
        assertTrue(userRepository.existsByEmail("check@example.com"));
        assertFalse(userRepository.existsByEmail("notfound@example.com"));
    }

    @Test
    @DisplayName("Should update user")
    void updateUser_Success() {
        // Given
        User user = User.builder()
                .username("updateuser")
                .email("update@example.com")
                .password(passwordEncoder.encode("password"))
                .firstName("Original")
                .roles(Set.of(userRole))
                .enabled(true)
                .build();
        User saved = entityManager.persistAndFlush(user);

        // When
        saved.setFirstName("Updated");
        saved.setLastName("Name");
        userRepository.save(saved);
        entityManager.flush();

        // Then
        Optional<User> updated = userRepository.findById(saved.getId());
        assertTrue(updated.isPresent());
        assertEquals("Updated", updated.get().getFirstName());
        assertEquals("Name", updated.get().getLastName());
    }

    @Test
    @DisplayName("Should soft delete user")
    void softDeleteUser_Success() {
        // Given
        User user = User.builder()
                .username("deleteuser")
                .email("delete@example.com")
                .password(passwordEncoder.encode("password"))
                .roles(Set.of(userRole))
                .enabled(true)
                .build();
        User saved = entityManager.persistAndFlush(user);

        // When
        userRepository.delete(saved);
        entityManager.flush();

        // Then
        Optional<User> found = userRepository.findById(saved.getId());
        assertFalse(found.isPresent());
    }

    @Test
    @DisplayName("Should find user by refresh token")
    void findByRefreshToken_Success() {
        // Given
        String refreshToken = "test-refresh-token-123";
        User user = User.builder()
                .username("tokenuser")
                .email("token@example.com")
                .password(passwordEncoder.encode("password"))
                .refreshToken(refreshToken)
                .roles(Set.of(userRole))
                .enabled(true)
                .build();
        entityManager.persistAndFlush(user);

        // When
        Optional<User> found = userRepository.findByRefreshToken(refreshToken);

        // Then
        assertTrue(found.isPresent());
        assertEquals("tokenuser", found.get().getUsername());
    }
}
