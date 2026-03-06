package com.archplatform.user.security;

import io.jsonwebtoken.Claims;
import io.jsonwebtoken.Jwts;
import io.jsonwebtoken.security.Keys;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;

import javax.crypto.SecretKey;
import java.nio.charset.StandardCharsets;
import java.util.Base64;
import java.util.Date;
import java.util.UUID;

import static org.junit.jupiter.api.Assertions.*;

class JwtTokenProviderTest {

    private JwtTokenProvider tokenProvider;
    private static final String SECRET = "test-secret-key-that-is-at-least-256-bits-long-for-security-purposes-only";
    private static final long ACCESS_EXPIRATION = 3600000; // 1 hour
    private static final long REFRESH_EXPIRATION = 86400000; // 24 hours

    @BeforeEach
    void setUp() {
        tokenProvider = new JwtTokenProvider(SECRET, ACCESS_EXPIRATION, REFRESH_EXPIRATION);
    }

    @Test
    @DisplayName("Should generate valid access token")
    void generateAccessToken_Success() {
        // Given
        UUID userId = UUID.randomUUID();
        String username = "testuser";

        // When
        String token = tokenProvider.generateAccessToken(userId, username);

        // Then
        assertNotNull(token);
        assertTrue(token.length() > 0);

        Claims claims = parseToken(token);
        assertEquals(userId.toString(), claims.getSubject());
        assertEquals(username, claims.get("username"));
        assertEquals("ACCESS", claims.get("type"));
    }

    @Test
    @DisplayName("Should generate valid refresh token")
    void generateRefreshToken_Success() {
        // Given
        UUID userId = UUID.randomUUID();
        String username = "testuser";

        // When
        String token = tokenProvider.generateRefreshToken(userId, username);

        // Then
        assertNotNull(token);

        Claims claims = parseToken(token);
        assertEquals(userId.toString(), claims.getSubject());
        assertEquals("REFRESH", claims.get("type"));
    }

    @Test
    @DisplayName("Should validate token successfully")
    void validateToken_Success() {
        // Given
        UUID userId = UUID.randomUUID();
        String token = tokenProvider.generateAccessToken(userId, "testuser");

        // When
        boolean isValid = tokenProvider.validateToken(token);

        // Then
        assertTrue(isValid);
    }

    @Test
    @DisplayName("Should reject invalid token")
    void validateToken_InvalidToken_ReturnsFalse() {
        // Given
        String invalidToken = "invalid.token.here";

        // When
        boolean isValid = tokenProvider.validateToken(invalidToken);

        // Then
        assertFalse(isValid);
    }

    @Test
    @DisplayName("Should reject expired token")
    void validateToken_ExpiredToken_ReturnsFalse() {
        // Given - Create provider with short expiration
        JwtTokenProvider shortLivedProvider = new JwtTokenProvider(SECRET, 1, 1);
        String token = shortLivedProvider.generateAccessToken(UUID.randomUUID(), "testuser");

        // Wait for token to expire
        try {
            Thread.sleep(10);
        } catch (InterruptedException e) {
            Thread.currentThread().interrupt();
        }

        // When
        boolean isValid = shortLivedProvider.validateToken(token);

        // Then
        assertFalse(isValid);
    }

    @Test
    @DisplayName("Should extract user ID from token")
    void getUserIdFromToken_Success() {
        // Given
        UUID userId = UUID.randomUUID();
        String token = tokenProvider.generateAccessToken(userId, "testuser");

        // When
        UUID extractedUserId = tokenProvider.getUserIdFromToken(token);

        // Then
        assertEquals(userId, extractedUserId);
    }

    @Test
    @DisplayName("Should extract username from token")
    void getUsernameFromToken_Success() {
        // Given
        String username = "testuser";
        String token = tokenProvider.generateAccessToken(UUID.randomUUID(), username);

        // When
        String extractedUsername = tokenProvider.getUsernameFromToken(token);

        // Then
        assertEquals(username, extractedUsername);
    }

    @Test
    @DisplayName("Should calculate correct expiration time")
    void getExpirationTime_Success() {
        // Given
        UUID userId = UUID.randomUUID();
        String token = tokenProvider.generateAccessToken(userId, "testuser");

        // When
        Date expiration = tokenProvider.getExpirationDateFromToken(token);

        // Then
        assertNotNull(expiration);
        assertTrue(expiration.after(new Date()));

        // Should be approximately 1 hour from now
        long expectedExpiration = System.currentTimeMillis() + ACCESS_EXPIRATION;
        long actualExpiration = expiration.getTime();
        long diff = Math.abs(expectedExpiration - actualExpiration);
        assertTrue(diff < 5000); // Within 5 seconds
    }

    private Claims parseToken(String token) {
        SecretKey key = Keys.hmacShaKeyFor(SECRET.getBytes(StandardCharsets.UTF_8));
        return Jwts.parser()
                .verifyWith(key)
                .build()
                .parseSignedClaims(token)
                .getPayload();
    }
}
