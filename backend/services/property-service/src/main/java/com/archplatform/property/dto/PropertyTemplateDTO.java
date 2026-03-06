package com.archplatform.property.dto;

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
public class PropertyTemplateDTO {
    private UUID id;
    private UUID tenantId;
    private UUID projectId;
    private String name;
    private String displayName;
    private String description;
    private String dataType;
    private String unit;
    private String unitCategory;
    private String defaultValue;
    private String minValue;
    private String maxValue;
    private List<String> allowedValues;
    private String regexPattern;
    private Boolean isRequired;
    private Boolean isReadOnly;
    private Boolean isHidden;
    private String groupName;
    private Integer sortOrder;
    private String scope;
    private String appliesTo;
    private String calculationRule;
    private String validationRules;
    private List<String> dependsOn;
    private LocalDateTime createdAt;
    private LocalDateTime updatedAt;
    private UUID createdBy;
}
