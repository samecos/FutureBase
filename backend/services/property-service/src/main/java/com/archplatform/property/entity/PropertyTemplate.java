package com.archplatform.property.entity;

import jakarta.persistence.*;
import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;
import org.hibernate.annotations.CreationTimestamp;
import org.hibernate.annotations.UpdateTimestamp;

import java.time.LocalDateTime;
import java.util.HashSet;
import java.util.Set;
import java.util.UUID;

@Entity
@Table(name = "property_templates", schema = "core")
@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class PropertyTemplate {

    @Id
    @Column(name = "id")
    private UUID id;

    @Column(name = "tenant_id", nullable = false)
    private UUID tenantId;

    @Column(name = "project_id")
    private UUID projectId;

    @Column(name = "name", nullable = false, length = 100)
    private String name;

    @Column(name = "display_name", nullable = false, length = 200)
    private String displayName;

    @Column(name = "description", length = 1000)
    private String description;

    @Enumerated(EnumType.STRING)
    @Column(name = "data_type", nullable = false, length = 20)
    private DataType dataType;

    @Column(name = "unit", length = 50)
    private String unit;

    @Column(name = "unit_category", length = 50)
    private String unitCategory;

    @Column(name = "default_value", length = 500)
    private String defaultValue;

    @Column(name = "min_value", length = 100)
    private String minValue;

    @Column(name = "max_value", length = 100)
    private String maxValue;

    @Column(name = "allowed_values", length = 2000)
    private String allowedValues;

    @Column(name = "regex_pattern", length = 255)
    private String regexPattern;

    @Column(name = "is_required")
    @Builder.Default
    private Boolean isRequired = false;

    @Column(name = "is_read_only")
    @Builder.Default
    private Boolean isReadOnly = false;

    @Column(name = "is_hidden")
    @Builder.Default
    private Boolean isHidden = false;

    @Column(name = "group_name", length = 100)
    private String groupName;

    @Column(name = "sort_order")
    @Builder.Default
    private Integer sortOrder = 0;

    @Enumerated(EnumType.STRING)
    @Column(name = "scope", nullable = false, length = 20)
    @Builder.Default
    private PropertyScope scope = PropertyScope.GLOBAL;

    @Column(name = "applies_to", length = 100)
    private String appliesTo;

    @Column(name = "calculation_rule", length = 2000)
    private String calculationRule;

    @Column(name = "validation_rules", length = 2000)
    private String validationRules;

    @Column(name = "depends_on", length = 500)
    private String dependsOn;

    @CreationTimestamp
    @Column(name = "created_at", nullable = false, updatable = false)
    private LocalDateTime createdAt;

    @UpdateTimestamp
    @Column(name = "updated_at", nullable = false)
    private LocalDateTime updatedAt;

    @Column(name = "created_by", nullable = false)
    private UUID createdBy;

    @Column(name = "updated_by")
    private UUID updatedBy;

    @Column(name = "deleted_at")
    private LocalDateTime deletedAt;

    public enum DataType {
        STRING, INTEGER, DECIMAL, BOOLEAN, DATE, DATETIME,
        LENGTH, AREA, VOLUME, ANGLE, TEMPERATURE, PRESSURE,
        COLOR, ENUM, MULTI_SELECT, URL, JSON
    }

    public enum PropertyScope {
        GLOBAL, PROJECT, ELEMENT_TYPE, LAYER
    }

    @PrePersist
    public void prePersist() {
        if (id == null) {
            id = UUID.randomUUID();
        }
    }

    public boolean isNumeric() {
        return dataType == DataType.INTEGER || 
               dataType == DataType.DECIMAL ||
               dataType == DataType.LENGTH ||
               dataType == DataType.AREA ||
               dataType == DataType.VOLUME ||
               dataType == DataType.ANGLE ||
               dataType == DataType.TEMPERATURE ||
               dataType == DataType.PRESSURE;
    }

    public boolean hasUnit() {
        return unit != null && !unit.isEmpty();
    }
}
