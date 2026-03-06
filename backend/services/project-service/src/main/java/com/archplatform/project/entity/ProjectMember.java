package com.archplatform.project.entity;

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
@Table(name = "project_members", schema = "core", 
       uniqueConstraints = @UniqueConstraint(columnNames = {"project_id", "user_id"}))
@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class ProjectMember {

    @Id
    @Column(name = "id")
    private UUID id;

    @ManyToOne(fetch = FetchType.LAZY)
    @JoinColumn(name = "project_id", nullable = false)
    private Project project;

    @Column(name = "user_id", nullable = false)
    private UUID userId;

    @Enumerated(EnumType.STRING)
    @Column(name = "role", nullable = false, length = 20)
    @Builder.Default
    private MemberRole role = MemberRole.VIEWER;

    @Column(name = "invited_by")
    private UUID invitedBy;

    @Column(name = "invited_at")
    private LocalDateTime invitedAt;

    @Column(name = "joined_at")
    private LocalDateTime joinedAt;

    @Column(name = "last_accessed_at")
    private LocalDateTime lastAccessedAt;

    @CreationTimestamp
    @Column(name = "created_at", nullable = false, updatable = false)
    private LocalDateTime createdAt;

    @UpdateTimestamp
    @Column(name = "updated_at", nullable = false)
    private LocalDateTime updatedAt;

    public enum MemberRole {
        OWNER, ADMIN, EDITOR, VIEWER
    }

    @PrePersist
    public void prePersist() {
        if (id == null) {
            id = UUID.randomUUID();
        }
        if (joinedAt == null) {
            joinedAt = LocalDateTime.now();
        }
    }

    public boolean hasPermission(MemberPermission permission) {
        return switch (role) {
            case OWNER -> true;
            case ADMIN -> permission != MemberPermission.DELETE_PROJECT;
            case EDITOR -> permission == MemberPermission.READ || 
                          permission == MemberPermission.WRITE ||
                          permission == MemberPermission.COMMENT;
            case VIEWER -> permission == MemberPermission.READ || 
                          permission == MemberPermission.COMMENT;
        };
    }

    public enum MemberPermission {
        READ, WRITE, DELETE, COMMENT, MANAGE_MEMBERS, DELETE_PROJECT
    }
}
