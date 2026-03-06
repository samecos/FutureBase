package com.archplatform.user.dto;

import com.fasterxml.jackson.annotation.JsonInclude;
import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;

import java.time.LocalDateTime;
import java.util.Set;
import java.util.UUID;

@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
@JsonInclude(JsonInclude.Include.NON_NULL)
public class UserDTO {
    private UUID id;
    private UUID tenantId;
    private String email;
    private String username;
    private String firstName;
    private String lastName;
    private String fullName;
    private String avatarUrl;
    private String status;
    private Boolean emailVerified;
    private Boolean mfaEnabled;
    private LocalDateTime lastLoginAt;
    private String authProvider;
    private LocalDateTime createdAt;
    private LocalDateTime updatedAt;
    private Set<String> roles;
    private Set<String> permissions;
}
