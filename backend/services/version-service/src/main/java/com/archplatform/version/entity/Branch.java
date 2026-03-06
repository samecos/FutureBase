package com.archplatform.version.entity;

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
@Table(name = "branches", schema = "versioning")
@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class Branch {

    @Id
    @Column(name = "id")
    private UUID id;

    @Column(name = "tenant_id", nullable = false)
    private UUID tenantId;

    @Column(name = "design_id", nullable = false)
    private UUID designId;

    @Column(name = "name", nullable = false, length = 100)
    private String name;

    @Column(name = "description", length = 1000)
    private String description;

    @Enumerated(EnumType.STRING)
    @Column(name = "status", nullable = false, length = 20)
    @Builder.Default
    private BranchStatus status = BranchStatus.ACTIVE;

    @Column(name = "parent_branch_id")
    private UUID parentBranchId;

    @Column(name = "parent_version_id")
    private UUID parentVersionId;

    @Column(name = "head_version_id")
    private UUID headVersionId;

    @Column(name = "base_version_number")
    @Builder.Default
    private Integer baseVersionNumber = 0;

    @Column(name = "version_count")
    @Builder.Default
    private Integer versionCount = 0;

    @Column(name = "is_default")
    @Builder.Default
    private Boolean isDefault = false;

    @Column(name = "is_protected")
    @Builder.Default
    private Boolean isProtected = false;

    @Column(name = "created_by", nullable = false)
    private UUID createdBy;

    @CreationTimestamp
    @Column(name = "created_at", nullable = false, updatable = false)
    private LocalDateTime createdAt;

    @UpdateTimestamp
    @Column(name = "updated_at", nullable = false)
    private LocalDateTime updatedAt;

    @Column(name = "deleted_at")
    private LocalDateTime deletedAt;

    public enum BranchStatus {
        ACTIVE, ARCHIVED, MERGED
    }

    @PrePersist
    public void prePersist() {
        if (id == null) {
            id = UUID.randomUUID();
        }
    }

    public boolean isActive() {
        return status == BranchStatus.ACTIVE && deletedAt == null;
    }

    public void incrementVersionCount() {
        this.versionCount = (this.versionCount == null ? 0 : this.versionCount) + 1;
    }
}
