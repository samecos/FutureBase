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
@Table(name = "merge_requests", schema = "versioning")
@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class MergeRequest {

    @Id
    @Column(name = "id")
    private UUID id;

    @Column(name = "tenant_id", nullable = false)
    private UUID tenantId;

    @Column(name = "design_id", nullable = false)
    private UUID designId;

    @Column(name = "source_branch_id", nullable = false)
    private UUID sourceBranchId;

    @Column(name = "source_version_id", nullable = false)
    private UUID sourceVersionId;

    @Column(name = "target_branch_id", nullable = false)
    private UUID targetBranchId;

    @Column(name = "target_version_id")
    private UUID targetVersionId;

    @Column(name = "title", nullable = false, length = 255)
    private String title;

    @Column(name = "description", length = 2000)
    private String description;

    @Enumerated(EnumType.STRING)
    @Column(name = "status", nullable = false, length = 20)
    @Builder.Default
    private MergeStatus status = MergeStatus.OPEN;

    @Column(name = "conflict_count")
    @Builder.Default
    private Integer conflictCount = 0;

    @Column(name = "conflict_resolution", columnDefinition = "jsonb")
    private String conflictResolution;

    @Column(name = "created_by", nullable = false)
    private UUID createdBy;

    @Column(name = "assigned_to")
    private UUID assignedTo;

    @Column(name = "merged_by")
    private UUID mergedBy;

    @Column(name = "merged_at")
    private LocalDateTime mergedAt;

    @Column(name = "result_version_id")
    private UUID resultVersionId;

    @CreationTimestamp
    @Column(name = "created_at", nullable = false, updatable = false)
    private LocalDateTime createdAt;

    @UpdateTimestamp
    @Column(name = "updated_at", nullable = false)
    private LocalDateTime updatedAt;

    @Column(name = "closed_at")
    private LocalDateTime closedAt;

    public enum MergeStatus {
        OPEN, CONFLICTS, MERGING, MERGED, CLOSED
    }

    @PrePersist
    public void prePersist() {
        if (id == null) {
            id = UUID.randomUUID();
        }
    }

    public boolean isOpen() {
        return status == MergeStatus.OPEN || status == MergeStatus.CONFLICTS;
    }
}
