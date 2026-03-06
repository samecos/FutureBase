package com.archplatform.property.repository;

import com.archplatform.property.entity.PropertyTemplate;
import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.data.jpa.repository.Query;
import org.springframework.data.repository.query.Param;
import org.springframework.stereotype.Repository;

import java.util.List;
import java.util.Optional;
import java.util.UUID;

@Repository
public interface PropertyTemplateRepository extends JpaRepository<PropertyTemplate, UUID> {

    @Query("SELECT pt FROM PropertyTemplate pt WHERE pt.id = :id AND pt.deletedAt IS NULL")
    Optional<PropertyTemplate> findActiveById(@Param("id") UUID id);

    @Query("SELECT pt FROM PropertyTemplate pt WHERE pt.tenantId = :tenantId AND pt.deletedAt IS NULL ORDER BY pt.groupName, pt.sortOrder")
    List<PropertyTemplate> findAllByTenantId(@Param("tenantId") UUID tenantId);

    @Query("SELECT pt FROM PropertyTemplate pt WHERE pt.tenantId = :tenantId AND pt.scope = :scope AND pt.deletedAt IS NULL")
    List<PropertyTemplate> findAllByTenantIdAndScope(@Param("tenantId") UUID tenantId, @Param("scope") PropertyTemplate.PropertyScope scope);

    @Query("SELECT pt FROM PropertyTemplate pt WHERE pt.tenantId = :tenantId AND pt.projectId = :projectId AND pt.deletedAt IS NULL")
    List<PropertyTemplate> findAllByTenantIdAndProjectId(@Param("tenantId") UUID tenantId, @Param("projectId") UUID projectId);

    @Query("SELECT pt FROM PropertyTemplate pt WHERE pt.tenantId = :tenantId AND pt.appliesTo = :appliesTo AND pt.deletedAt IS NULL")
    List<PropertyTemplate> findAllByTenantIdAndAppliesTo(@Param("tenantId") UUID tenantId, @Param("appliesTo") String appliesTo);

    @Query("SELECT pt FROM PropertyTemplate pt WHERE pt.tenantId = :tenantId AND pt.groupName = :groupName AND pt.deletedAt IS NULL")
    List<PropertyTemplate> findAllByTenantIdAndGroupName(@Param("tenantId") UUID tenantId, @Param("groupName") String groupName);

    @Query("SELECT pt FROM PropertyTemplate pt WHERE pt.tenantId = :tenantId AND pt.name = :name AND pt.deletedAt IS NULL")
    Optional<PropertyTemplate> findByTenantIdAndName(@Param("tenantId") UUID tenantId, @Param("name") String name);

    @Query("SELECT pt FROM PropertyTemplate pt WHERE pt.tenantId = :tenantId AND pt.calculationRule IS NOT NULL AND pt.deletedAt IS NULL")
    List<PropertyTemplate> findAllWithCalculationRules(@Param("tenantId") UUID tenantId);

    @Query("SELECT pt FROM PropertyTemplate pt WHERE pt.tenantId = :tenantId AND pt.dependsOn LIKE %:propertyName% AND pt.deletedAt IS NULL")
    List<PropertyTemplate> findAllDependentOnProperty(@Param("tenantId") UUID tenantId, @Param("propertyName") String propertyName);

    boolean existsByTenantIdAndNameAndDeletedAtIsNull(UUID tenantId, String name);

    @Query("SELECT DISTINCT pt.groupName FROM PropertyTemplate pt WHERE pt.tenantId = :tenantId AND pt.deletedAt IS NULL ORDER BY pt.groupName")
    List<String> findAllGroupNamesByTenantId(@Param("tenantId") UUID tenantId);
}
