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
public class BranchDTO {
    private UUID id;
    private UUID designId;
    private String name;
    private String description;
    private String status;
    private UUID parentBranchId;
    private String parentBranchName;
    private UUID parentVersionId;
    private Integer parentVersionNumber;
    private UUID headVersionId;
    private Integer headVersionNumber;
    private Integer baseVersionNumber;
    private Integer versionCount;
    private Boolean isDefault;
    private Boolean isProtected;
    private UUID createdBy;
    private String createdByName;
    private LocalDateTime createdAt;
    private LocalDateTime updatedAt;
}
