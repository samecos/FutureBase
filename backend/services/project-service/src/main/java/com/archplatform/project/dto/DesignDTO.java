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
public class DesignDTO {
    private UUID id;
    private UUID projectId;
    private String name;
    private String description;
    private String designType;
    private String fileFormat;
    private Integer version;
    private String status;
    private String thumbnailUrl;
    private Long fileSizeBytes;
    private UUID createdBy;
    private String createdByName;
    private UUID updatedBy;
    private String updatedByName;
    private UUID lockedBy;
    private String lockedByName;
    private LocalDateTime lockedAt;
    private LocalDateTime lockExpiresAt;
    private Boolean isLocked;
    private UUID folderId;
    private String folderName;
    private List<String> tags;
    private LocalDateTime createdAt;
    private LocalDateTime updatedAt;
}
