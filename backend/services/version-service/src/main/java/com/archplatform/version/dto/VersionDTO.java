package com.archplatform.version.dto;

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
public class VersionDTO {
    private UUID id;
    private UUID designId;
    private UUID branchId;
    private String branchName;
    private Integer versionNumber;
    private String fullVersionName;
    private String versionName;
    private String description;
    private String status;
    private UUID snapshotId;
    private String snapshotUrl;
    private Long snapshotSizeBytes;
    private String checksum;
    private UUID previousVersionId;
    private Integer previousVersionNumber;
    private List<String> parentVersionIds;
    private String changeSummary;
    private Integer changeCount;
    private UUID createdBy;
    private String createdByName;
    private UUID committedBy;
    private String committedByName;
    private LocalDateTime committedAt;
    private LocalDateTime createdAt;
    private Boolean isTagged;
    private List<String> tags;
}
