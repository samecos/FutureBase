package com.archplatform.user.repository;

import com.archplatform.user.entity.Tenant;
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
public interface TenantRepository extends JpaRepository<Tenant, UUID> {

    Optional<Tenant> findBySlug(String slug);

    Optional<Tenant> findByName(String name);

    @Query("SELECT t FROM Tenant t WHERE t.slug = :slug AND t.deletedAt IS NULL")
    Optional<Tenant> findActiveBySlug(@Param("slug") String slug);

    @Query("SELECT t FROM Tenant t WHERE t.id = :id AND t.deletedAt IS NULL")
    Optional<Tenant> findActiveById(@Param("id") UUID id);

    @Query("SELECT t FROM Tenant t WHERE t.status = :status AND t.deletedAt IS NULL")
    List<Tenant> findAllByStatus(@Param("status") Tenant.TenantStatus status);

    @Query("SELECT t FROM Tenant t WHERE t.deletedAt IS NULL")
    List<Tenant> findAllActive();

    @Query("SELECT t FROM Tenant t WHERE t.subscriptionEndsAt < :now AND t.plan != 'FREE' AND t.status = 'ACTIVE'")
    List<Tenant> findAllWithExpiredSubscription(@Param("now") LocalDateTime now);

    @Query("SELECT t FROM Tenant t WHERE t.trialEndsAt < :now AND t.status = 'ACTIVE'")
    List<Tenant> findAllWithExpiredTrial(@Param("now") LocalDateTime now);

    boolean existsBySlug(String slug);

    boolean existsByName(String name);

    @Modifying
    @Query("UPDATE Tenant t SET t.status = :status WHERE t.id = :tenantId")
    void updateStatus(@Param("tenantId") UUID tenantId, @Param("status") Tenant.TenantStatus status);

    @Modifying
    @Query("UPDATE Tenant t SET t.deletedAt = :deletedAt WHERE t.id = :tenantId")
    void softDelete(@Param("tenantId") UUID tenantId, @Param("deletedAt") LocalDateTime deletedAt);
}
