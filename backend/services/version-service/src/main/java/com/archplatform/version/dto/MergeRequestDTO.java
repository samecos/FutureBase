package com.archplatform.version.dto;

import com.fasterxml.jackson.annotation.JsonInclude;
import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;

import java.time.LocalDateTime;
import java.util.UUID;

@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
@JsonInclude(JsonInclude.Include.NON_NULL)
public class MergeRequestDTO {
    private UUID id;
    private UUID designId;
    private UUID sourceBranchId;
    private String sourceBranchName;
    private UUID sourceVersionId;
    private Integer sourceVersionNumber;
    private UUID targetBranchId;
    private String targetBranchName;
    private UUID targetVersionId;
    private Integer targetVersionNumber;
    private String title;
    private String description;
    private String status;
    private Integer conflictCount;
    private Object conflictResolution;
    private UUID createdBy;
    private String createdByName;
    private UUID assignedTo;
    private String assignedToName;
    private UUID mergedBy;
    private String mergedByName;
    private LocalDateTime mergedAt;
    private UUID resultVersionId;
    private LocalDateTime createdAt;
    private LocalDateTime updatedAt;
}
