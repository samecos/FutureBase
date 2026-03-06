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
public class PropertyGroupDTO {
    private UUID id;
    private UUID tenantId;
    private UUID projectId;
    private String name;
    private String displayName;
    private String description;
    private String icon;
    private String color;
    private Integer sortOrder;
    private Boolean isCollapsed;
    private Boolean isSystem;
    private List<PropertyTemplateDTO> templates;
    private LocalDateTime createdAt;
    private UUID createdBy;
}
