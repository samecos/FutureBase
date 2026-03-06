package com.archplatform.user.service;

import com.archplatform.user.dto.*;
import com.archplatform.user.entity.*;
import com.archplatform.user.repository.*;
import com.archplatform.user.security.JwtTokenProvider;
import com.archplatform.user.security.MfaService;
import com.archplatform.user.exception.*;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.data.domain.Page;
import org.springframework.data.domain.Pageable;
import org.springframework.data.redis.core.RedisTemplate;
import org.springframework.security.crypto.password.PasswordEncoder;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;
import org.springframework.util.StringUtils;

import java.time.Duration;
import java.time.LocalDateTime;
import java.util.*;
import java.util.stream.Collectors;

@Slf4j
@Service
@RequiredArgsConstructor
public class UserService {

    private final UserRepository userRepository;
    private final RoleRepository roleRepository;
    private final RefreshTokenRepository refreshTokenRepository;
    private final PasswordEncoder passwordEncoder;
    private final JwtTokenProvider jwtTokenProvider;
    private final MfaService mfaService;
    private final RedisTemplate<String, String> redisTemplate;

    @Transactional(readOnly = true)
    public UserDTO getUserById(UUID userId) {
        User user = userRepository.findActiveById(userId)
            .orElseThrow(() -> new UserNotFoundException("User not found: " + userId));
        return mapToDTO(user);
    }

    @Transactional(readOnly = true)
    public UserDTO getUserByEmail(String email) {
        User user = userRepository.findActiveByEmail(email)
            .orElseThrow(() -> new UserNotFoundException("User not found: " + email));
        return mapToDTO(user);
    }

    @Transactional(readOnly = true)
    public Page<UserDTO> getUsersByTenant(UUID tenantId, Pageable pageable) {
        return userRepository.findAllByTenantId(tenantId, pageable)
            .map(this::mapToDTO);
    }

    @Transactional
    public UserDTO createUser(RegisterRequest request) {
        if (userRepository.existsByEmail(request.getEmail())) {
            throw new UserAlreadyExistsException("Email already registered: " + request.getEmail());
        }
        if (userRepository.existsByUsername(request.getUsername())) {
            throw new UserAlreadyExistsException("Username already taken: " + request.getUsername());
        }

        User user = User.builder()
            .email(request.getEmail().toLowerCase())
            .username(request.getUsername())
            .passwordHash(passwordEncoder.encode(request.getPassword()))
            .firstName(request.getFirstName())
            .lastName(request.getLastName())
            .tenantId(request.getTenantId())
            .status(User.UserStatus.PENDING_VERIFICATION)
            .passwordChangedAt(LocalDateTime.now())
            .build();

        // Assign default role
        Role defaultRole = roleRepository.findSystemRoleByName("USER")
            .orElseThrow(() -> new RoleNotFoundException("Default role not found"));
        user.getRoles().add(defaultRole);

        User savedUser = userRepository.save(user);
        log.info("Created new user: {}", savedUser.getEmail());

        return mapToDTO(savedUser);
    }

    @Transactional
    public AuthResponse authenticate(LoginRequest request, String ipAddress) {
        String cacheKey = "login_attempts:" + request.getEmailOrUsername();
        String attemptsStr = redisTemplate.opsForValue().get(cacheKey);
        int attempts = attemptsStr != null ? Integer.parseInt(attemptsStr) : 0;

        Optional<User> userOpt = userRepository.findActiveByEmail(request.getEmailOrUsername())
            .or(() -> userRepository.findByUsername(request.getEmailOrUsername()));

        if (userOpt.isEmpty()) {
            handleFailedLogin(cacheKey, attempts);
            throw new AuthenticationException("Invalid credentials");
        }

        User user = userOpt.get();

        // Check if user is locked
        if (user.isLocked()) {
            throw new AccountLockedException("Account is locked. Please try again later.");
        }

        // Verify password
        if (!passwordEncoder.matches(request.getPassword(), user.getPasswordHash())) {
            handleFailedLogin(user, cacheKey, attempts);
            throw new AuthenticationException("Invalid credentials");
        }

        // Check if MFA is required
        if (user.getMfaEnabled()) {
            if (!StringUtils.hasText(request.getMfaCode())) {
                String mfaToken = UUID.randomUUID().toString();
                redisTemplate.opsForValue().set(
                    "mfa_pending:" + mfaToken,
                    user.getId().toString(),
                    Duration.ofMinutes(5)
                );
                return AuthResponse.builder()
                    .mfaRequired(true)
                    .mfaToken(mfaToken)
                    .build();
            }

            if (!mfaService.verifyCode(user.getMfaSecret(), request.getMfaCode())) {
                throw new MfaException("Invalid MFA code");
            }
        }

        // Clear failed attempts
        redisTemplate.delete(cacheKey);
        userRepository.resetFailedLoginAttempts(user.getId());

        // Update last login
        userRepository.updateLastLogin(user.getId(), LocalDateTime.now(), ipAddress);

        // Generate tokens
        return generateAuthResponse(user, request.getDeviceInfo(), ipAddress);
    }

    @Transactional
    public AuthResponse refreshToken(String refreshToken) {
        RefreshToken token = refreshTokenRepository.findByTokenHash(hashToken(refreshToken))
            .orElseThrow(() -> new TokenException("Invalid refresh token"));

        if (!token.isValid()) {
            throw new TokenException("Refresh token is expired or revoked");
        }

        User user = token.getUser();
        if (!user.isActive()) {
            throw new AuthenticationException("User account is not active");
        }

        // Revoke old token
        refreshTokenRepository.revokeById(token.getId(), LocalDateTime.now());

        // Generate new tokens
        return generateAuthResponse(user, token.getDeviceInfo(), token.getIpAddress());
    }

    @Transactional
    public void logout(UUID userId, String refreshToken) {
        if (refreshToken != null) {
            refreshTokenRepository.findByTokenHash(hashToken(refreshToken))
                .ifPresent(token -> refreshTokenRepository.revokeById(token.getId(), LocalDateTime.now()));
        }
        refreshTokenRepository.revokeAllByUserId(userId, LocalDateTime.now());
        log.info("User logged out: {}", userId);
    }

    @Transactional
    public UserDTO updateUser(UUID userId, UpdateUserRequest request) {
        User user = userRepository.findActiveById(userId)
            .orElseThrow(() -> new UserNotFoundException("User not found: " + userId));

        if (StringUtils.hasText(request.getFirstName())) {
            user.setFirstName(request.getFirstName());
        }
        if (StringUtils.hasText(request.getLastName())) {
            user.setLastName(request.getLastName());
        }
        if (StringUtils.hasText(request.getAvatarUrl())) {
            user.setAvatarUrl(request.getAvatarUrl());
        }

        User updatedUser = userRepository.save(user);
        return mapToDTO(updatedUser);
    }

    @Transactional
    public void changePassword(UUID userId, ChangePasswordRequest request) {
        User user = userRepository.findActiveById(userId)
            .orElseThrow(() -> new UserNotFoundException("User not found: " + userId));

        if (!passwordEncoder.matches(request.getCurrentPassword(), user.getPasswordHash())) {
            throw new AuthenticationException("Current password is incorrect");
        }

        user.setPasswordHash(passwordEncoder.encode(request.getNewPassword()));
        user.setPasswordChangedAt(LocalDateTime.now());
        userRepository.save(user);

        // Revoke all refresh tokens after password change
        refreshTokenRepository.revokeAllByUserId(userId, LocalDateTime.now());
        log.info("Password changed for user: {}", userId);
    }

    @Transactional
    public void deleteUser(UUID userId) {
        User user = userRepository.findActiveById(userId)
            .orElseThrow(() -> new UserNotFoundException("User not found: " + userId));

        userRepository.softDelete(userId, LocalDateTime.now());
        refreshTokenRepository.revokeAllByUserId(userId, LocalDateTime.now());
        log.info("User deleted: {}", userId);
    }

    // MFA methods
    @Transactional
    public MfaSetupResponse setupMfa(UUID userId) {
        User user = userRepository.findActiveById(userId)
            .orElseThrow(() -> new UserNotFoundException("User not found: " + userId));

        if (user.getMfaEnabled()) {
            throw new MfaException("MFA is already enabled");
        }

        String secret = mfaService.generateSecret();
        String qrCodeUrl = mfaService.getQrCodeUrl(user.getEmail(), secret);

        // Store secret temporarily
        redisTemplate.opsForValue().set(
            "mfa_setup:" + userId,
            secret,
            Duration.ofMinutes(10)
        );

        return MfaSetupResponse.builder()
            .secret(secret)
            .qrCodeUrl(qrCodeUrl)
            .enabled(false)
            .build();
    }

    @Transactional
    public void verifyAndEnableMfa(UUID userId, String code) {
        User user = userRepository.findActiveById(userId)
            .orElseThrow(() -> new UserNotFoundException("User not found: " + userId));

        String secret = redisTemplate.opsForValue().get("mfa_setup:" + userId);
        if (secret == null) {
            throw new MfaException("MFA setup session expired");
        }

        if (!mfaService.verifyCode(secret, code)) {
            throw new MfaException("Invalid verification code");
        }

        userRepository.updateMfaSettings(userId, true, secret);
        redisTemplate.delete("mfa_setup:" + userId);
        log.info("MFA enabled for user: {}", userId);
    }

    @Transactional
    public void disableMfa(UUID userId, String password) {
        User user = userRepository.findActiveById(userId)
            .orElseThrow(() -> new UserNotFoundException("User not found: " + userId));

        if (!passwordEncoder.matches(password, user.getPasswordHash())) {
            throw new AuthenticationException("Password is incorrect");
        }

        userRepository.updateMfaSettings(userId, false, null);
        log.info("MFA disabled for user: {}", userId);
    }

    // Helper methods
    private void handleFailedLogin(String cacheKey, int attempts) {
        attempts++;
        redisTemplate.opsForValue().set(cacheKey, String.valueOf(attempts), Duration.ofMinutes(15));
    }

    private void handleFailedLogin(User user, String cacheKey, int attempts) {
        attempts++;
        userRepository.incrementFailedLoginAttempts(user.getId());

        if (attempts >= 5) {
            LocalDateTime lockUntil = LocalDateTime.now().plusMinutes(30);
            userRepository.lockUser(user.getId(), lockUntil);
            throw new AccountLockedException("Too many failed attempts. Account locked for 30 minutes.");
        }

        redisTemplate.opsForValue().set(cacheKey, String.valueOf(attempts), Duration.ofMinutes(15));
    }

    private AuthResponse generateAuthResponse(User user, String deviceInfo, String ipAddress) {
        // Get user permissions
        Set<String> permissions = user.getRoles().stream()
            .flatMap(role -> role.getPermissions().stream())
            .map(Permission::getPermissionString)
            .collect(Collectors.toSet());

        Set<String> roles = user.getRoles().stream()
            .map(Role::getName)
            .collect(Collectors.toSet());

        // Generate tokens
        String accessToken = jwtTokenProvider.generateAccessToken(user, roles, permissions);
        String refreshTokenValue = jwtTokenProvider.generateRefreshToken();

        // Save refresh token
        RefreshToken refreshToken = RefreshToken.builder()
            .user(user)
            .tokenHash(hashToken(refreshTokenValue))
            .deviceInfo(deviceInfo)
            .ipAddress(ipAddress)
            .expiresAt(LocalDateTime.now().plusDays(7))
            .build();
        refreshTokenRepository.save(refreshToken);

        return AuthResponse.builder()
            .accessToken(accessToken)
            .refreshToken(refreshTokenValue)
            .tokenType("Bearer")
            .expiresIn(900L) // 15 minutes
            .userId(user.getId())
            .email(user.getEmail())
            .username(user.getUsername())
            .fullName(user.getFullName())
            .mfaRequired(false)
            .roles(roles)
            .permissions(permissions)
            .build();
    }

    private UserDTO mapToDTO(User user) {
        Set<String> roles = user.getRoles().stream()
            .map(Role::getName)
            .collect(Collectors.toSet());

        Set<String> permissions = user.getRoles().stream()
            .flatMap(role -> role.getPermissions().stream())
            .map(Permission::getPermissionString)
            .collect(Collectors.toSet());

        return UserDTO.builder()
            .id(user.getId())
            .tenantId(user.getTenantId())
            .email(user.getEmail())
            .username(user.getUsername())
            .firstName(user.getFirstName())
            .lastName(user.getLastName())
            .fullName(user.getFullName())
            .avatarUrl(user.getAvatarUrl())
            .status(user.getStatus().name())
            .emailVerified(user.getEmailVerified())
            .mfaEnabled(user.getMfaEnabled())
            .lastLoginAt(user.getLastLoginAt())
            .authProvider(user.getAuthProvider().name())
            .createdAt(user.getCreatedAt())
            .updatedAt(user.getUpdatedAt())
            .roles(roles)
            .permissions(permissions)
            .build();
    }

    private String hashToken(String token) {
        return Base64.getEncoder().encodeToString(token.getBytes());
    }
}
