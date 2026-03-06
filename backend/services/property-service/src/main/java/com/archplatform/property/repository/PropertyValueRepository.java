package com.archplatform.property.repository;

import com.archplatform.property.entity.PropertyValue;
import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.data.jpa.repository.Modifying;
import org.springframework.data.jpa.repository.Query;
import org.springframework.data.repository.query.Param;
import org.springframework.stereotype.Repository;

import java.util.List;
import java.util.Optional;
import java.util.UUID;

@Repository
public interface PropertyValueRepository extends JpaRepository<PropertyValue, UUID> {

    Optional<PropertyValue> findByTemplateIdAndEntityTypeAndEntityId(UUID templateId, String entityType, UUID entityId);

    @Query("SELECT pv FROM PropertyValue pv WHERE pv.entityType = :entityType AND pv.entityId = :entityId")
    List<PropertyValue> findAllByEntity(@Param("entityType") String entityType, @Param("entityId") UUID entityId);

    @Query("SELECT pv FROM PropertyValue pv JOIN pv.template pt WHERE pv.entityType = :entityType AND pv.entityId = :entityId AND pt.groupName = :groupName")
    List<PropertyValue> findAllByEntityAndGroupName(@Param("entityType") String entityType, @Param("entityId") UUID entityId, @Param("groupName") String groupName);

    @Query("SELECT pv FROM PropertyValue pv WHERE pv.template.id = :templateId")
    List<PropertyValue> findAllByTemplateId(@Param("templateId") UUID templateId);

    @Query("SELECT pv FROM PropertyValue pv WHERE pv.tenantId = :tenantId AND pv.isCalculated = true")
    List<PropertyValue> findAllCalculatedValues(@Param("tenantId") UUID tenantId);

    @Query("SELECT pv FROM PropertyValue pv WHERE pv.inheritedFrom = :templateId")
    List<PropertyValue> findAllInheritedFrom(@Param("templateId") UUID templateId);

    @Query("SELECT COUNT(pv) FROM PropertyValue pv WHERE pv.template.id = :templateId AND pv.value IS NOT NULL")
    long countNonEmptyByTemplateId(@Param("templateId") UUID templateId);

    @Modifying
    @Query("DELETE FROM PropertyValue pv WHERE pv.entityType = :entityType AND pv.entityId = :entityId")
    void deleteAllByEntity(@Param("entityType") String entityType, @Param("entityId") UUID entityId);

    @Modifying
    @Query("UPDATE PropertyValue pv SET pv.isCalculated = false, pv.calculationSource = NULL WHERE pv.id = :valueId")
    void clearCalculatedFlag(@Param("valueId") UUID valueId);

    boolean existsByTemplateIdAndEntityTypeAndEntityId(UUID templateId, String entityType, UUID entityId);
}
