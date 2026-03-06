package com.archplatform.property.entity;

import jakarta.persistence.*;
import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;
import org.hibernate.annotations.CreationTimestamp;
import org.hibernate.annotations.UpdateTimestamp;

import java.time.LocalDateTime;
import java.util.UUID;

@Entity
@Table(name = "property_values", schema = "core", 
       uniqueConstraints = @UniqueConstraint(columnNames = {"template_id", "entity_type", "entity_id"}))
@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class PropertyValue {

    @Id
    @Column(name = "id")
    private UUID id;

    @ManyToOne(fetch = FetchType.LAZY)
    @JoinColumn(name = "template_id", nullable = false)
    private PropertyTemplate template;

    @Column(name = "tenant_id", nullable = false)
    private UUID tenantId;

    @Column(name = "entity_type", nullable = false, length = 50)
    private String entityType;

    @Column(name = "entity_id", nullable = false)
    private UUID entityId;

    @Column(name = "value", length = 1000)
    private String value;

    @Column(name = "display_value", length = 1000)
    private String displayValue;

    @Column(name = "unit", length = 50)
    private String unit;

    @Column(name = "is_calculated")
    @Builder.Default
    private Boolean isCalculated = false;

    @Column(name = "calculation_source", length = 500)
    private String calculationSource;

    @Column(name = "is_inherited")
    @Builder.Default
    private Boolean isInherited = false;

    @Column(name = "inherited_from")
    private UUID inheritedFrom;

    @Column(name = "override_reason", length = 500)
    private String overrideReason;

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

    @PrePersist
    public void prePersist() {
        if (id == null) {
            id = UUID.randomUUID();
        }
    }

    public boolean isEmpty() {
        return value == null || value.isEmpty();
    }

    public void setValueWithConversion(String newValue, String targetUnit) {
        this.value = newValue;
        this.unit = targetUnit;
        // Display value would be formatted based on template settings
        this.displayValue = newValue;
    }
}
