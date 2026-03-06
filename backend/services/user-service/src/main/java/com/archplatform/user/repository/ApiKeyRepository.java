package com.archplatform.user.repository;

import com.archplatform.user.entity.ApiKey;
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
public interface ApiKeyRepository extends JpaRepository<ApiKey, UUID> {

    Optional<ApiKey> findByKeyHash(String keyHash);

    @Query("SELECT ak FROM ApiKey ak WHERE ak.user.id = :userId AND ak.revoked = false")
    List<ApiKey> findAllValidByUserId(@Param("userId") UUID userId);

    @Query("SELECT ak FROM ApiKey ak WHERE ak.user.id = :userId")
    List<ApiKey> findAllByUserId(@Param("userId") UUID userId);

    @Query("SELECT ak FROM ApiKey ak WHERE ak.revoked = false AND ak.expiresAt > :now")
    List<ApiKey> findAllValid(@Param("now") LocalDateTime now);

    @Modifying
    @Query("UPDATE ApiKey ak SET ak.revoked = true, ak.revokedAt = :revokedAt, ak.revokedReason = :reason WHERE ak.id = :keyId")
    void revokeById(@Param("keyId") UUID keyId, @Param("revokedAt") LocalDateTime revokedAt, @Param("reason") String reason);

    @Modifying
    @Query("UPDATE ApiKey ak SET ak.lastUsedAt = :usedAt WHERE ak.id = :keyId")
    void updateLastUsed(@Param("keyId") UUID keyId, @Param("usedAt") LocalDateTime usedAt);

    @Modifying
    @Query("DELETE FROM ApiKey ak WHERE ak.expiresAt < :now OR (ak.revoked = true AND ak.revokedAt < :cleanupThreshold)")
    void deleteExpiredOrRevokedKeys(@Param("now") LocalDateTime now, @Param("cleanupThreshold") LocalDateTime cleanupThreshold);

    long countByUserIdAndRevokedFalse(UUID userId);
}
