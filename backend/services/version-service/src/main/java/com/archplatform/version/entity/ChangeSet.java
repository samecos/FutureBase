package com.archplatform.version.entity;

import jakarta.persistence.*;
import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;
import org.hibernate.annotations.CreationTimestamp;

import java.time.LocalDateTime;
import java.util.UUID;

@Entity
@Table(name = "change_sets", schema = "versioning")
@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class ChangeSet {

    @Id
    @Column(name = "id")
    private UUID id;

    @Column(name = "tenant_id", nullable = false)
    private UUID tenantId;

    @Column(name = "version_id", nullable = false)
    private UUID versionId;

    @Column(name = "design_id", nullable = false)
    private UUID designId;

    @Enumerated(EnumType.STRING)
    @Column(name = "change_type", nullable = false, length = 20)
    private ChangeType changeType;

    @Column(name = "entity_type", length = 50)
    private String entityType;

    @Column(name = "entity_id")
    private UUID entityId;

    @Column(name = "property_name", length = 100)
    private String propertyName;

    @Column(name = "old_value", columnDefinition = "text")
    private String oldValue;

    @Column(name = "new_value", columnDefinition = "text")
    private String newValue;

    @Column(name = "diff_data", columnDefinition = "jsonb")
    private String diffData;

    @Column(name = "description", length = 1000)
    private String description;

    @Column(name = "created_by", nullable = false)
    private UUID createdBy;

    @CreationTimestamp
    @Column(name = "created_at", nullable = false, updatable = false)
    private LocalDateTime createdAt;

    public enum ChangeType {
        CREATE, UPDATE, DELETE,
        GEOMETRY_ADDED, GEOMETRY_MODIFIED, GEOMETRY_DELETED,
        PROPERTY_CHANGED, LAYER_ADDED, LAYER_REMOVED,
        ELEMENT_ADDED, ELEMENT_MODIFIED, ELEMENT_REMOVED
    }

    @PrePersist
    public void prePersist() {
        if (id == null) {
            id = UUID.randomUUID();
        }
    }
}
