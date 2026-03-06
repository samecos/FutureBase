package com.archplatform.property.dto;

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
public class PropertyValueDTO {
    private UUID id;
    private UUID templateId;
    private String templateName;
    private String templateDisplayName;
    private String dataType;
    private String entityType;
    private UUID entityId;
    private String value;
    private String displayValue;
    private String unit;
    private Boolean isCalculated;
    private String calculationSource;
    private Boolean isInherited;
    private UUID inheritedFrom;
    private String overrideReason;
    private LocalDateTime createdAt;
    private LocalDateTime updatedAt;
    private UUID updatedBy;
}
