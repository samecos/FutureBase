package com.archplatform.search.dto;

import jakarta.validation.constraints.NotBlank;
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
public class SearchRequest {

    @NotBlank(message = "Query is required")
    private String query;

    private List<String> indices;

    private UUID tenantId;

    private UUID projectId;

    private Map<String, String> filters;

    private String sortBy;

    private String sortOrder;

    @Builder.Default
    private Integer page = 0;

    @Builder.Default
    private Integer size = 20;

    private Boolean highlight;
}
