package com.archplatform.version.dto;

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
public class VersionDiffDTO {
    private UUID sourceVersionId;
    private Integer sourceVersionNumber;
    private UUID targetVersionId;
    private Integer targetVersionNumber;
    private List<ChangeDiff> changes;
    private Integer addedCount;
    private Integer modifiedCount;
    private Integer deletedCount;

    @Data
    @Builder
    @NoArgsConstructor
    @AllArgsConstructor
    public static class ChangeDiff {
        private String operation;
        private String path;
        private Object oldValue;
        private Object newValue;
        private String entityType;
        private UUID entityId;
    }
}
