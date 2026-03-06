package com.archplatform.property.dto;

import jakarta.validation.constraints.NotBlank;
import jakarta.validation.constraints.NotNull;
import jakarta.validation.constraints.Size;
import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;

import java.util.List;
import java.util.UUID;

@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class CreateTemplateRequest {

    @NotBlank(message = "Property name is required")
    @Size(min = 1, max = 100, message = "Name must be between 1 and 100 characters")
    private String name;

    @NotBlank(message = "Display name is required")
    @Size(min = 1, max = 200, message = "Display name must be between 1 and 200 characters")
    private String displayName;

    @Size(max = 1000, message = "Description must not exceed 1000 characters")
    private String description;

    @NotBlank(message = "Data type is required")
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

    private UUID projectId;

    private String appliesTo;

    private String calculationRule;

    private String validationRules;

    private List<String> dependsOn;
}
