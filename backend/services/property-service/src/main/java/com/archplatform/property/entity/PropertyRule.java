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
@Table(name = "property_rules", schema = "core")
@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class PropertyRule {

    @Id
    @Column(name = "id")
    private UUID id;

    @Column(name = "tenant_id", nullable = false)
    private UUID tenantId;

    @Column(name = "project_id")
    private UUID projectId;

    @Column(name = "name", nullable = false, length = 100)
    private String name;

    @Column(name = "description", length = 1000)
    private String description;

    @Enumerated(EnumType.STRING)
    @Column(name = "rule_type", nullable = false, length = 20)
    private RuleType ruleType;

    @Enumerated(EnumType.STRING)
    @Column(name = "trigger_event", nullable = false, length = 20)
    private TriggerEvent triggerEvent;

    @Column(name = "condition_expression", length = 2000)
    private String conditionExpression;

    @Column(name = "action_expression", length = 2000)
    private String actionExpression;

    @Column(name = "target_properties", length = 500)
    private String targetProperties;

    @Column(name = "source_properties", length = 500)
    private String sourceProperties;

    @Column(name = "applies_to_types", length = 500)
    private String appliesToTypes;

    @Column(name = "priority")
    @Builder.Default
    private Integer priority = 100;

    @Column(name = "is_active")
    @Builder.Default
    private Boolean isActive = true;

    @Column(name = "error_message", length = 500)
    private String errorMessage;

    @Column(name = "execution_count")
    @Builder.Default
    private Long executionCount = 0L;

    @Column(name = "last_executed_at")
    private LocalDateTime lastExecutedAt;

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

    public enum RuleType {
        CALCULATION, VALIDATION, DEFAULT, CASCADE, CONSTRAINT
    }

    public enum TriggerEvent {
        VALUE_CHANGED, ELEMENT_CREATED, ELEMENT_UPDATED, MANUAL, SCHEDULED
    }

    @PrePersist
    public void prePersist() {
        if (id == null) {
            id = UUID.randomUUID();
        }
    }

    public void recordExecution() {
        this.executionCount++;
        this.lastExecutedAt = LocalDateTime.now();
    }
}
