package com.archplatform.property.repository;

import com.archplatform.property.entity.PropertyRule;
import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.data.jpa.repository.Query;
import org.springframework.data.repository.query.Param;
import org.springframework.stereotype.Repository;

import java.util.List;
import java.util.Optional;
import java.util.UUID;

@Repository
public interface PropertyRuleRepository extends JpaRepository<PropertyRule, UUID> {

    Optional<PropertyRule> findByIdAndIsActiveTrue(UUID id);

    @Query("SELECT pr FROM PropertyRule pr WHERE pr.tenantId = :tenantId AND pr.isActive = true ORDER BY pr.priority")
    List<PropertyRule> findAllActiveByTenantId(@Param("tenantId") UUID tenantId);

    @Query("SELECT pr FROM PropertyRule pr WHERE pr.tenantId = :tenantId AND pr.projectId = :projectId AND pr.isActive = true")
    List<PropertyRule> findAllActiveByTenantIdAndProjectId(@Param("tenantId") UUID tenantId, @Param("projectId") UUID projectId);

    @Query("SELECT pr FROM PropertyRule pr WHERE pr.tenantId = :tenantId AND pr.ruleType = :ruleType AND pr.isActive = true")
    List<PropertyRule> findAllActiveByTenantIdAndType(@Param("tenantId") UUID tenantId, @Param("ruleType") PropertyRule.RuleType ruleType);

    @Query("SELECT pr FROM PropertyRule pr WHERE pr.tenantId = :tenantId AND pr.triggerEvent = :triggerEvent AND pr.isActive = true")
    List<PropertyRule> findAllActiveByTenantIdAndTrigger(@Param("tenantId") UUID tenantId, @Param("triggerEvent") PropertyRule.TriggerEvent triggerEvent);

    @Query("SELECT pr FROM PropertyRule pr WHERE pr.tenantId = :tenantId AND pr.targetProperties LIKE %:propertyName% AND pr.isActive = true")
    List<PropertyRule> findAllActiveByTargetProperty(@Param("tenantId") UUID tenantId, @Param("propertyName") String propertyName);

    @Query("SELECT pr FROM PropertyRule pr WHERE pr.tenantId = :tenantId AND pr.sourceProperties LIKE %:propertyName% AND pr.isActive = true")
    List<PropertyRule> findAllActiveBySourceProperty(@Param("tenantId") UUID tenantId, @Param("propertyName") String propertyName);

    @Query("SELECT pr FROM PropertyRule pr WHERE pr.tenantId = :tenantId AND (pr.appliesToTypes IS NULL OR pr.appliesToTypes LIKE %:elementType%) AND pr.isActive = true")
    List<PropertyRule> findAllActiveByTenantIdAndElementType(@Param("tenantId") UUID tenantId, @Param("elementType") String elementType);

    long countByTenantIdAndIsActiveTrue(UUID tenantId);
}
