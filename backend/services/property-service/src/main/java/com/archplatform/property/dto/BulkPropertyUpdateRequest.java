package com.archplatform.property.dto;

import jakarta.validation.constraints.NotNull;
import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;

import java.util.List;
import java.util.Map;
import java.util.UUID;

@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class BulkPropertyUpdateRequest {

    @NotNull(message = "Entity type is required")
    private String entityType;

    @NotNull(message = "Entity IDs are required")
    private List<UUID> entityIds;

    @NotNull(message = "Properties are required")
    private Map<String, String> properties;
}
