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
public class PropertyRuleDTO {
    private UUID id;
    private UUID tenantId;
    private UUID projectId;
    private String name;
    private String description;
    private String ruleType;
    private String triggerEvent;
    private String conditionExpression;
    private String actionExpression;
    private List<String> targetProperties;
    private List<String> sourceProperties;
    private List<String> appliesToTypes;
    private Integer priority;
    private Boolean isActive;
    private String errorMessage;
    private Long executionCount;
    private LocalDateTime lastExecutedAt;
    private LocalDateTime createdAt;
    private LocalDateTime updatedAt;
    private UUID createdBy;
}
