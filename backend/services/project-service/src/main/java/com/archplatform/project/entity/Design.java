package com.archplatform.project.entity;

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
@Table(name = "designs", schema = "core")
@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class Design {

    @Id
    @Column(name = "id")
    private UUID id;

    @ManyToOne(fetch = FetchType.LAZY)
    @JoinColumn(name = "project_id", nullable = false)
    private Project project;

    @Column(name = "tenant_id", nullable = false)
    private UUID tenantId;

    @Column(name = "name", nullable = false, length = 255)
    private String name;

    @Column(name = "description", length = 2000)
    private String description;

    @Enumerated(EnumType.STRING)
    @Column(name = "design_type", length = 50)
    private DesignType designType;

    @Column(name = "file_format", length = 20)
    private String fileFormat;

    @Column(name = "version")
    @Builder.Default
    private Integer version = 1;

    @Column(name = "status", length = 20)
    @Builder.Default
    private String status = "draft";

    @Column(name = "thumbnail_url", length = 500)
    private String thumbnailUrl;

    @Column(name = "file_url", length = 500)
    private String fileUrl;

    @Column(name = "file_size_bytes")
    private Long fileSizeBytes;

    @Column(name = "file_hash", length = 64)
    private String fileHash;

    @Column(name = "created_by", nullable = false)
    private UUID createdBy;

    @Column(name = "updated_by")
    private UUID updatedBy;

    @Column(name = "locked_by")
    private UUID lockedBy;

    @Column(name = "locked_at")
    private LocalDateTime lockedAt;

    @Column(name = "lock_expires_at")
    private LocalDateTime lockExpiresAt;

    @Column(name = "parent_design_id")
    private UUID parentDesignId;

    @Column(name = "folder_id")
    private UUID folderId;

    @Column(name = "tags")
    private String tags;

    @Column(name = "metadata", columnDefinition = "jsonb")
    private String metadata;

    @CreationTimestamp
    @Column(name = "created_at", nullable = false, updatable = false)
    private LocalDateTime createdAt;

    @UpdateTimestamp
    @Column(name = "updated_at", nullable = false)
    private LocalDateTime updatedAt;

    @Column(name = "deleted_at")
    private LocalDateTime deletedAt;

    @OneToMany(mappedBy = "design", cascade = CascadeType.ALL, orphanRemoval = true)
    @Builder.Default
    private Set<DesignVersion> versions = new HashSet<>();

    public enum DesignType {
        ARCHITECTURAL, STRUCTURAL, MEP, LANDSCAPE, INTERIOR, MASTER_PLAN, OTHER
    }

    @PrePersist
    public void prePersist() {
        if (id == null) {
            id = UUID.randomUUID();
        }
    }

    public boolean isLocked() {
        if (lockedBy == null) {
            return false;
        }
        if (lockExpiresAt != null && lockExpiresAt.isBefore(LocalDateTime.now())) {
            return false;
        }
        return true;
    }

    public void lock(UUID userId, int durationMinutes) {
        this.lockedBy = userId;
        this.lockedAt = LocalDateTime.now();
        this.lockExpiresAt = LocalDateTime.now().plusMinutes(durationMinutes);
    }

    public void unlock() {
        this.lockedBy = null;
        this.lockedAt = null;
        this.lockExpiresAt = null;
    }
}
