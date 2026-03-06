package com.archplatform.version.dto;

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
public class CreateVersionRequest {

    @NotNull(message = "Branch ID is required")
    private UUID branchId;

    @Size(max = 100, message = "Version name must not exceed 100 characters")
    private String versionName;

    @Size(max = 2000, message = "Description must not exceed 2000 characters")
    private String description;

    private UUID snapshotId;

    private String snapshotUrl;

    private Long snapshotSizeBytes;

    private String checksum;

    private String changeSummary;

    private Integer changeCount;

    private List<String> tags;
}
