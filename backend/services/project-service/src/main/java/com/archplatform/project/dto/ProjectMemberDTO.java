package com.archplatform.project.dto;

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
public class ProjectMemberDTO {
    private UUID id;
    private UUID userId;
    private String userName;
    private String userEmail;
    private String userAvatarUrl;
    private String role;
    private UUID invitedBy;
    private String invitedByName;
    private LocalDateTime joinedAt;
    private LocalDateTime lastAccessedAt;
}
