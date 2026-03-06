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
@Table(name = "version_tags", schema = "versioning", 
       uniqueConstraints = @UniqueConstraint(columnNames = {"design_id", "tag_name"}))
@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class VersionTag {

    @Id
    @Column(name = "id")
    private UUID id;

    @Column(name = "tenant_id", nullable = false)
    private UUID tenantId;

    @Column(name = "design_id", nullable = false)
    private UUID designId;

    @Column(name = "version_id", nullable = false)
    private UUID versionId;

    @Column(name = "tag_name", nullable = false, length = 100)
    private String tagName;

    @Column(name = "description", length = 1000)
    private String description;

    @Column(name = "is_protected")
    @Builder.Default
    private Boolean isProtected = false;

    @Column(name = "created_by", nullable = false)
    private UUID createdBy;

    @CreationTimestamp
    @Column(name = "created_at", nullable = false, updatable = false)
    private LocalDateTime createdAt;

    @PrePersist
    public void prePersist() {
        if (id == null) {
            id = UUID.randomUUID();
        }
    }
}
