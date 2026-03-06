package com.archplatform.user.dto;

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
public class ApiKeyDTO {
    private UUID id;
    private String name;
    private String description;
    private String scopes;
    private String keyPreview;
    private LocalDateTime lastUsedAt;
    private LocalDateTime expiresAt;
    private boolean revoked;
    private LocalDateTime createdAt;
    private String plainKey;
}
