package com.archplatform.user.repository;

import com.archplatform.user.entity.User;
import org.springframework.data.domain.Page;
import org.springframework.data.domain.Pageable;
import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.data.jpa.repository.Modifying;
import org.springframework.data.jpa.repository.Query;
import org.springframework.data.repository.query.Param;
import org.springframework.stereotype.Repository;

import java.time.LocalDateTime;
import java.util.List;
import java.util.Optional;
import java.util.UUID;

@Repository
public interface UserRepository extends JpaRepository<User, UUID> {

    Optional<User> findByEmail(String email);

    Optional<User> findByUsername(String username);

    Optional<User> findByEmailAndDeletedAtIsNull(String email);

    Optional<User> findByUsernameAndDeletedAtIsNull(String username);

    @Query("SELECT u FROM User u WHERE u.email = :email AND u.deletedAt IS NULL")
    Optional<User> findActiveByEmail(@Param("email") String email);

    @Query("SELECT u FROM User u WHERE u.id = :id AND u.deletedAt IS NULL")
    Optional<User> findActiveById(@Param("id") UUID id);

    boolean existsByEmail(String email);

    boolean existsByUsername(String username);

    @Query("SELECT u FROM User u WHERE u.tenantId = :tenantId AND u.deletedAt IS NULL")
    List<User> findAllByTenantId(@Param("tenantId") UUID tenantId);

    @Query("SELECT u FROM User u WHERE u.tenantId = :tenantId AND u.deletedAt IS NULL")
    Page<User> findAllByTenantId(@Param("tenantId") UUID tenantId, Pageable pageable);

    @Query("SELECT u FROM User u WHERE u.tenantId = :tenantId AND u.status = :status AND u.deletedAt IS NULL")
    List<User> findAllByTenantIdAndStatus(@Param("tenantId") UUID tenantId, @Param("status") User.UserStatus status);

    @Query("SELECT COUNT(u) FROM User u WHERE u.tenantId = :tenantId AND u.deletedAt IS NULL")
    long countByTenantId(@Param("tenantId") UUID tenantId);

    @Query("SELECT u FROM User u WHERE u.lockedUntil < :now AND u.lockedUntil IS NOT NULL")
    List<User> findUsersToUnlock(@Param("now") LocalDateTime now);

    @Query("SELECT u FROM User u WHERE u.passwordExpiresAt < :now AND u.passwordExpiresAt IS NOT NULL")
    List<User> findUsersWithExpiredPassword(@Param("now") LocalDateTime now);

    @Modifying
    @Query("UPDATE User u SET u.lastLoginAt = :loginTime, u.lastLoginIp = :ip WHERE u.id = :userId")
    void updateLastLogin(@Param("userId") UUID userId, @Param("loginTime") LocalDateTime loginTime, @Param("ip") String ip);

    @Modifying
    @Query("UPDATE User u SET u.failedLoginAttempts = u.failedLoginAttempts + 1 WHERE u.id = :userId")
    void incrementFailedLoginAttempts(@Param("userId") UUID userId);

    @Modifying
    @Query("UPDATE User u SET u.failedLoginAttempts = 0, u.lockedUntil = NULL WHERE u.id = :userId")
    void resetFailedLoginAttempts(@Param("userId") UUID userId);

    @Modifying
    @Query("UPDATE User u SET u.lockedUntil = :lockedUntil WHERE u.id = :userId")
    void lockUser(@Param("userId") UUID userId, @Param("lockedUntil") LocalDateTime lockedUntil);

    @Modifying
    @Query("UPDATE User u SET u.emailVerified = true WHERE u.id = :userId")
    void verifyEmail(@Param("userId") UUID userId);

    @Modifying
    @Query("UPDATE User u SET u.mfaEnabled = :enabled, u.mfaSecret = :secret WHERE u.id = :userId")
    void updateMfaSettings(@Param("userId") UUID userId, @Param("enabled") boolean enabled, @Param("secret") String secret);

    @Modifying
    @Query("UPDATE User u SET u.deletedAt = :deletedAt WHERE u.id = :userId")
    void softDelete(@Param("userId") UUID userId, @Param("deletedAt") LocalDateTime deletedAt);

    @Query("SELECT u FROM User u WHERE u.authProvider = :provider AND u.providerId = :providerId AND u.deletedAt IS NULL")
    Optional<User> findByAuthProviderAndProviderId(@Param("provider") User.AuthProvider provider, @Param("providerId") String providerId);

    @Query("SELECT CASE WHEN COUNT(u) > 0 THEN true ELSE false END FROM User u WHERE u.tenantId = :tenantId AND u.deletedAt IS NULL")
    boolean existsActiveByTenantId(@Param("tenantId") UUID tenantId);
}
