package com.archplatform.user.integration;

import com.archplatform.user.dto.LoginRequest;
import com.archplatform.user.entity.User;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;
import org.springframework.http.MediaType;

import static org.hamcrest.Matchers.*;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.post;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.*;

/**
 * Integration tests for user login flow.
 */
class UserLoginIntegrationTest extends BaseIntegrationTest {

    @Test
    @DisplayName("Should login successfully with valid credentials")
    void login_Success() throws Exception {
        // Given - Create a test user
        User user = testDataHelper.createTestUser("logintest", "login@example.com", "password123");

        LoginRequest request = new LoginRequest();
        request.setUsername("logintest");
        request.setPassword("password123");

        // When & Then
        mockMvc.perform(post("/api/v1/auth/login")
                .contentType(MediaType.APPLICATION_JSON)
                .content(asJsonString(request)))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.accessToken").exists())
                .andExpect(jsonPath("$.refreshToken").exists())
                .andExpect(jsonPath("$.tokenType").value("Bearer"))
                .andExpect(jsonPath("$.expiresIn").exists())
                .andExpect(jsonPath("$.user.username").value("logintest"))
                .andExpect(jsonPath("$.user.email").value("login@example.com"))
                .andExpect(jsonPath("$.mfaRequired").value(false));
    }

    @Test
    @DisplayName("Should return unauthorized for invalid password")
    void login_InvalidPassword_ReturnsUnauthorized() throws Exception {
        // Given - Create a test user
        testDataHelper.createTestUser("logintest2", "login2@example.com", "password123");

        LoginRequest request = new LoginRequest();
        request.setUsername("logintest2");
        request.setPassword("wrongpassword");

        // When & Then
        mockMvc.perform(post("/api/v1/auth/login")
                .contentType(MediaType.APPLICATION_JSON)
                .content(asJsonString(request)))
                .andExpect(status().isUnauthorized())
                .andExpect(jsonPath("$.code").value("AUTHENTICATION_FAILED"))
                .andExpect(jsonPath("$.message").exists());
    }

    @Test
    @DisplayName("Should return unauthorized for non-existent user")
    void login_NonExistentUser_ReturnsUnauthorized() throws Exception {
        // Given
        LoginRequest request = new LoginRequest();
        request.setUsername("nonexistent");
        request.setPassword("password123");

        // When & Then
        mockMvc.perform(post("/api/v1/auth/login")
                .contentType(MediaType.APPLICATION_JSON)
                .content(asJsonString(request)))
                .andExpect(status().isUnauthorized())
                .andExpect(jsonPath("$.code").value("AUTHENTICATION_FAILED"));
    }

    @Test
    @DisplayName("Should return bad request when username is blank")
    void login_BlankUsername_ReturnsBadRequest() throws Exception {
        // Given
        LoginRequest request = new LoginRequest();
        request.setUsername("");
        request.setPassword("password123");

        // When & Then
        mockMvc.perform(post("/api/v1/auth/login")
                .contentType(MediaType.APPLICATION_JSON)
                .content(asJsonString(request)))
                .andExpect(status().isBadRequest());
    }

    @Test
    @DisplayName("Should return bad request when password is blank")
    void login_BlankPassword_ReturnsBadRequest() throws Exception {
        // Given
        LoginRequest request = new LoginRequest();
        request.setUsername("logintest");
        request.setPassword("");

        // When & Then
        mockMvc.perform(post("/api/v1/auth/login")
                .contentType(MediaType.APPLICATION_JSON)
                .content(asJsonString(request)))
                .andExpect(status().isBadRequest());
    }

    @Test
    @DisplayName("Should track failed login attempts")
    void login_MultipleFailedAttempts_TracksFailures() throws Exception {
        // Given - Create a test user
        testDataHelper.createTestUser("locktest", "lock@example.com", "password123");

        LoginRequest request = new LoginRequest();
        request.setUsername("locktest");
        request.setPassword("wrongpassword");

        // When - Attempt login 3 times with wrong password
        for (int i = 0; i < 3; i++) {
            mockMvc.perform(post("/api/v1/auth/login")
                    .contentType(MediaType.APPLICATION_JSON)
                    .content(asJsonString(request)))
                    .andExpect(status().isUnauthorized());
        }

        // Then - Verify failed attempts were tracked
        var user = testDataHelper.getUserRepository().findByUsername("locktest").orElseThrow();
        assert user.getFailedLoginAttempts() >= 3;
    }
}
