package com.archplatform.project.event;

import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;

import java.time.LocalDateTime;
import java.util.Map;
import java.util.UUID;

@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class ProjectEvent {
    private EventType type;
    private UUID projectId;
    private UUID tenantId;
    private UUID userId;
    private Map<String, String> metadata;
    private LocalDateTime timestamp;

    public ProjectEvent(EventType type, UUID projectId, UUID tenantId, UUID userId, Map<String, String> metadata) {
        this.type = type;
        this.projectId = projectId;
        this.tenantId = tenantId;
        this.userId = userId;
        this.metadata = metadata;
        this.timestamp = LocalDateTime.now();
    }

    public enum EventType {
        PROJECT_CREATED,
        PROJECT_UPDATED,
        PROJECT_DELETED,
        PROJECT_ARCHIVED,
        MEMBER_ADDED,
        MEMBER_REMOVED,
        MEMBER_ROLE_CHANGED,
        DESIGN_CREATED,
        DESIGN_UPDATED,
        DESIGN_DELETED
    }
}
