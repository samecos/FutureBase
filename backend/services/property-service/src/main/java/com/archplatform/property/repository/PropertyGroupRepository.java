package com.archplatform.property.repository;

import com.archplatform.property.entity.PropertyGroup;
import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.data.jpa.repository.Query;
import org.springframework.data.repository.query.Param;
import org.springframework.stereotype.Repository;

import java.util.List;
import java.util.Optional;
import java.util.UUID;

@Repository
public interface PropertyGroupRepository extends JpaRepository<PropertyGroup, UUID> {

    @Query("SELECT pg FROM PropertyGroup pg WHERE pg.tenantId = :tenantId ORDER BY pg.sortOrder")
    List<PropertyGroup> findAllByTenantId(@Param("tenantId") UUID tenantId);

    @Query("SELECT pg FROM PropertyGroup pg WHERE pg.tenantId = :tenantId AND pg.projectId = :projectId ORDER BY pg.sortOrder")
    List<PropertyGroup> findAllByTenantIdAndProjectId(@Param("tenantId") UUID tenantId, @Param("projectId") UUID projectId);

    Optional<PropertyGroup> findByTenantIdAndName(UUID tenantId, String name);

    @Query("SELECT pg FROM PropertyGroup pg WHERE pg.tenantId = :tenantId AND pg.isSystem = true")
    List<PropertyGroup> findAllSystemGroupsByTenantId(@Param("tenantId") UUID tenantId);

    boolean existsByTenantIdAndName(UUID tenantId, String name);
}
