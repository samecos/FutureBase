package com.archplatform.project.dto;

import com.fasterxml.jackson.annotation.JsonInclude;
import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;

import java.time.LocalDateTime;
import java.util.List;
import java.util.UUID;

@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
@JsonInclude(JsonInclude.Include.NON_NULL)
public class ProjectDTO {
    private UUID id;
    private UUID tenantId;
    private String name;
    private String description;
    private String slug;
    private String status;
    private String visibility;
    private UUID ownerId;
    private String ownerName;
    private String thumbnailUrl;
    private List<String> tags;
    private LocalDateTime startDate;
    private LocalDateTime targetEndDate;
    private Integer designCount;
    private Integer memberCount;
    private Long totalStorageBytes;
    private LocalDateTime createdAt;
    private LocalDateTime updatedAt;
    private List<ProjectMemberDTO> members;
}
