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
@Table(name = "versions", schema = "versioning")
@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class Version {

    @Id
    @Column(name = "id")
    private UUID id;

    @Column(name = "tenant_id", nullable = false)
    private UUID tenantId;

    @Column(name = "design_id", nullable = false)
    private UUID designId;

    @ManyToOne(fetch = FetchType.LAZY)
    @JoinColumn(name = "branch_id", nullable = false)
    private Branch branch;

    @Column(name = "version_number", nullable = false)
    private Integer versionNumber;

    @Column(name = "version_name", length = 100)
    private String versionName;

    @Column(name = "description", length = 2000)
    private String description;

    @Enumerated(EnumType.STRING)
    @Column(name = "status", nullable = false, length = 20)
    @Builder.Default
    private VersionStatus status = VersionStatus.DRAFT;

    @Column(name = "snapshot_id")
    private UUID snapshotId;

    @Column(name = "snapshot_url", length = 500)
    private String snapshotUrl;

    @Column(name = "snapshot_size_bytes")
    private Long snapshotSizeBytes;

    @Column(name = "checksum", length = 64)
    private String checksum;

    @Column(name = "previous_version_id")
    private UUID previousVersionId;

    @Column(name = "parent_version_ids", length = 1000)
    private String parentVersionIds;

    @Column(name = "change_summary", length = 1000)
    private String changeSummary;

    @Column(name = "change_count")
    @Builder.Default
    private Integer changeCount = 0;

    @Column(name = "created_by", nullable = false)
    private UUID createdBy;

    @Column(name = "committed_by")
    private UUID committedBy;

    @Column(name = "committed_at")
    private LocalDateTime committedAt;

    @CreationTimestamp
    @Column(name = "created_at", nullable = false, updatable = false)
    private LocalDateTime createdAt;

    @Column(name = "is_tagged")
    @Builder.Default
    private Boolean isTagged = false;

    @Column(name = "tags", length = 500)
    private String tags;

    public enum VersionStatus {
        DRAFT, COMMITTED, ARCHIVED
    }

    @PrePersist
    public void prePersist() {
        if (id == null) {
            id = UUID.randomUUID();
        }
    }

    public boolean isCommitted() {
        return status == VersionStatus.COMMITTED;
    }

    public boolean isDraft() {
        return status == VersionStatus.DRAFT;
    }

    public String getFullVersionName() {
        String branchName = branch != null ? branch.getName() : "unknown";
        return branchName + "/v" + versionNumber;
    }
}
