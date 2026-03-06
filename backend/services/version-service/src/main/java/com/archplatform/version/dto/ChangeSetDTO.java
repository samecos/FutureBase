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
public class ChangeSetDTO {
    private UUID id;
    private UUID versionId;
    private String changeType;
    private String entityType;
    private UUID entityId;
    private String propertyName;
    private String oldValue;
    private String newValue;
    private Object diffData;
    private String description;
    private UUID createdBy;
    private String createdByName;
    private LocalDateTime createdAt;
}
