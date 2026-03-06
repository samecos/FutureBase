package com.archplatform.project.dto;

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
public class ProjectStatsDTO {
    private UUID projectId;
    private String projectName;
    private Integer totalDesigns;
    private Integer totalMembers;
    private Long totalStorageBytes;
    private Map<String, Integer> designsByType;
    private Map<String, Integer> designsByStatus;
    private LocalDateTime lastActivityAt;
    private Integer activityScore;
}
