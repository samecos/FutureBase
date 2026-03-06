package com.archplatform.property.controller;

import com.archplatform.property.dto.*;
import com.archplatform.property.entity.PropertyTemplate;
import com.archplatform.property.security.JwtAuthenticationFilter;
import com.archplatform.property.service.PropertyService;
import jakarta.validation.Valid;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.http.ResponseEntity;
import org.springframework.security.core.annotation.AuthenticationPrincipal;
import org.springframework.web.bind.annotation.*;

import java.util.List;
import java.util.Map;
import java.util.UUID;

@Slf4j
@RestController
@RequestMapping("/properties")
@RequiredArgsConstructor
public class PropertyController {

    private final PropertyService propertyService;

    // Templates
    @GetMapping("/templates")
    public ResponseEntity<List<PropertyTemplateDTO>> getTemplates(
            @RequestParam UUID tenantId,
            @RequestParam(required = false) UUID projectId,
            @RequestParam(required = false) String scope,
            @AuthenticationPrincipal JwtAuthenticationFilter.UserPrincipal principal) {
        
        List<PropertyTemplateDTO> templates;
        if (projectId != null) {
            templates = propertyService.getTemplatesByProject(tenantId, projectId);
        } else if (scope != null) {
            templates = propertyService.getTemplatesByScope(tenantId, PropertyTemplate.PropertyScope.valueOf(scope.toUpperCase()));
        } else {
            templates = propertyService.getTemplatesByTenant(tenantId);
        }
        return ResponseEntity.ok(templates);
    }

    @GetMapping("/templates/{templateId}")
    public ResponseEntity<PropertyTemplateDTO> getTemplate(@PathVariable UUID templateId) {
        PropertyTemplateDTO template = propertyService.getTemplate(templateId);
        return ResponseEntity.ok(template);
    }

    @PostMapping("/templates")
    public ResponseEntity<PropertyTemplateDTO> createTemplate(
            @RequestParam UUID tenantId,
            @Valid @RequestBody CreateTemplateRequest request,
            @AuthenticationPrincipal JwtAuthenticationFilter.UserPrincipal principal) {
        
        PropertyTemplateDTO template = propertyService.createTemplate(tenantId, principal.id(), request);
        return ResponseEntity.ok(template);
    }

    @PutMapping("/templates/{templateId}")
    public ResponseEntity<PropertyTemplateDTO> updateTemplate(
            @RequestParam UUID tenantId,
            @PathVariable UUID templateId,
            @Valid @RequestBody CreateTemplateRequest request,
            @AuthenticationPrincipal JwtAuthenticationFilter.UserPrincipal principal) {
        
        PropertyTemplateDTO template = propertyService.updateTemplate(tenantId, templateId, principal.id(), request);
        return ResponseEntity.ok(template);
    }

    @DeleteMapping("/templates/{templateId}")
    public ResponseEntity<Void> deleteTemplate(
            @RequestParam UUID tenantId,
            @PathVariable UUID templateId,
            @AuthenticationPrincipal JwtAuthenticationFilter.UserPrincipal principal) {
        
        propertyService.deleteTemplate(tenantId, templateId);
        return ResponseEntity.ok().build();
    }

    // Property Values
    @GetMapping("/values")
    public ResponseEntity<List<PropertyValueDTO>> getPropertyValues(
            @RequestParam String entityType,
            @RequestParam UUID entityId,
            @AuthenticationPrincipal JwtAuthenticationFilter.UserPrincipal principal) {
        
        List<PropertyValueDTO> values = propertyService.getPropertyValues(entityType, entityId);
        return ResponseEntity.ok(values);
    }

    @GetMapping("/values/{templateId}")
    public ResponseEntity<PropertyValueDTO> getPropertyValue(
            @PathVariable UUID templateId,
            @RequestParam String entityType,
            @RequestParam UUID entityId,
            @AuthenticationPrincipal JwtAuthenticationFilter.UserPrincipal principal) {
        
        PropertyValueDTO value = propertyService.getPropertyValue(templateId, entityType, entityId);
        return ResponseEntity.ok(value);
    }

    @PostMapping("/values")
    public ResponseEntity<PropertyValueDTO> setPropertyValue(
            @RequestParam UUID tenantId,
            @Valid @RequestBody SetPropertyValueRequest request,
            @AuthenticationPrincipal JwtAuthenticationFilter.UserPrincipal principal) {
        
        PropertyValueDTO value = propertyService.setPropertyValue(tenantId, principal.id(), request);
        return ResponseEntity.ok(value);
    }

    @DeleteMapping("/values")
    public ResponseEntity<Void> deletePropertyValues(
            @RequestParam String entityType,
            @RequestParam UUID entityId,
            @AuthenticationPrincipal JwtAuthenticationFilter.UserPrincipal principal) {
        
        propertyService.deletePropertyValues(entityType, entityId);
        return ResponseEntity.ok().build();
    }

    @PostMapping("/values/bulk")
    public ResponseEntity<Map<String, PropertyValueDTO>> bulkUpdateProperties(
            @RequestParam UUID tenantId,
            @Valid @RequestBody BulkPropertyUpdateRequest request,
            @AuthenticationPrincipal JwtAuthenticationFilter.UserPrincipal principal) {
        
        Map<String, PropertyValueDTO> results = propertyService.bulkUpdateProperties(tenantId, principal.id(), request);
        return ResponseEntity.ok(results);
    }

    // Groups
    @GetMapping("/groups")
    public ResponseEntity<List<PropertyGroupDTO>> getPropertyGroups(
            @RequestParam UUID tenantId,
            @AuthenticationPrincipal JwtAuthenticationFilter.UserPrincipal principal) {
        
        List<PropertyGroupDTO> groups = propertyService.getPropertyGroups(tenantId);
        return ResponseEntity.ok(groups);
    }

    @PostMapping("/groups")
    public ResponseEntity<PropertyGroupDTO> createPropertyGroup(
            @RequestParam UUID tenantId,
            @RequestBody PropertyGroupDTO request,
            @AuthenticationPrincipal JwtAuthenticationFilter.UserPrincipal principal) {
        
        PropertyGroupDTO group = propertyService.createPropertyGroup(tenantId, principal.id(), request);
        return ResponseEntity.ok(group);
    }

    // Validation
    @PostMapping("/validate")
    public ResponseEntity<PropertyValidationResult> validateProperties(
            @RequestParam String entityType,
            @RequestParam UUID entityId,
            @AuthenticationPrincipal JwtAuthenticationFilter.UserPrincipal principal) {
        
        PropertyValidationResult result = propertyService.validateProperties(entityType, entityId);
        return ResponseEntity.ok(result);
    }
}
