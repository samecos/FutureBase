package com.archplatform.property.dto;

import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;

import java.util.List;

@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class PropertyValidationResult {
    private boolean valid;
    private List<PropertyError> errors;

    @Data
    @Builder
    @NoArgsConstructor
    @AllArgsConstructor
    public static class PropertyError {
        private String propertyName;
        private String errorCode;
        private String errorMessage;
    }
}
