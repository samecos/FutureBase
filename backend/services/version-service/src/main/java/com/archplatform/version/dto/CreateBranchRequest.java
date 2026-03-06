package com.archplatform.version.dto;

import jakarta.validation.constraints.NotBlank;
import jakarta.validation.constraints.NotNull;
import jakarta.validation.constraints.Size;
import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;

import java.util.UUID;

@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class CreateBranchRequest {

    @NotNull(message = "Design ID is required")
    private UUID designId;

    @NotBlank(message = "Branch name is required")
    @Size(min = 1, max = 100, message = "Branch name must be between 1 and 100 characters")
    private String name;

    @Size(max = 1000, message = "Description must not exceed 1000 characters")
    private String description;

    private UUID parentBranchId;

    private UUID parentVersionId;

    private Boolean isDefault;

    private Boolean isProtected;
}
