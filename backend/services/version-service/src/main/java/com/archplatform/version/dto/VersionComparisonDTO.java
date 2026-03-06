package com.archplatform.version.dto;

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
public class VersionComparisonDTO {
    private UUID baseVersionId;
    private Integer baseVersionNumber;
    private String baseVersionName;
    private UUID compareVersionId;
    private Integer compareVersionNumber;
    private String compareVersionName;
    private List<PropertyComparison> propertyChanges;
    private List<GeometryComparison> geometryChanges;
    private LocalDateTime comparisonTime;

    @Data
    @Builder
    @NoArgsConstructor
    @AllArgsConstructor
    public static class PropertyComparison {
        private String propertyName;
        private String oldValue;
        private String newValue;
        private String changeType;
    }

    @Data
    @Builder
    @NoArgsConstructor
    @AllArgsConstructor
    public static class GeometryComparison {
        private UUID elementId;
        private String elementName;
        private String changeType;
        private String changeDescription;
    }
}
