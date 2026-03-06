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
public class CreateRuleRequest {

    @NotBlank(message = "Rule name is required")
    @Size(min = 1, max = 100, message = "Name must be between 1 and 100 characters")
    private String name;

    @Size(max = 1000, message = "Description must not exceed 1000 characters")
    private String description;

    @NotBlank(message = "Rule type is required")
    private String ruleType;

    @NotBlank(message = "Trigger event is required")
    private String triggerEvent;

    private String conditionExpression;

    @NotBlank(message = "Action expression is required")
    private String actionExpression;

    private List<String> targetProperties;

    private List<String> sourceProperties;

    private List<String> appliesToTypes;

    private Integer priority;

    private UUID projectId;

    private String errorMessage;
}
