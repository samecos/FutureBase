package com.archplatform.property.service;

import com.archplatform.property.dto.*;
import com.archplatform.property.engine.RuleEngine;
import com.archplatform.property.engine.UnitConverter;
import com.archplatform.property.entity.*;
import com.archplatform.property.exception.*;
import com.archplatform.property.repository.*;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.cache.annotation.CacheEvict;
import org.springframework.cache.annotation.Cacheable;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;
import org.springframework.util.StringUtils;

import java.time.LocalDateTime;
import java.util.*;
import java.util.stream.Collectors;

@Slf4j
@Service
@RequiredArgsConstructor
public class PropertyService {

    private final PropertyTemplateRepository templateRepository;
    private final PropertyValueRepository valueRepository;
    private final PropertyGroupRepository groupRepository;
    private final PropertyRuleRepository ruleRepository;
    private final RuleEngine ruleEngine;
    private final UnitConverter unitConverter;

    // Template Operations
    @Transactional(readOnly = true)
    public PropertyTemplateDTO getTemplate(UUID templateId) {
        PropertyTemplate template = templateRepository.findActiveById(templateId)
            .orElseThrow(() -> new TemplateNotFoundException("Template not found: " + templateId));
        return mapToTemplateDTO(template);
    }

    @Transactional(readOnly = true)
    @Cacheable(value = "templates", key = "#tenantId")
    public List<PropertyTemplateDTO> getTemplatesByTenant(UUID tenantId) {
        return templateRepository.findAllByTenantId(tenantId).stream()
            .map(this::mapToTemplateDTO)
            .collect(Collectors.toList());
    }

    @Transactional(readOnly = true)
    public List<PropertyTemplateDTO> getTemplatesByProject(UUID tenantId, UUID projectId) {
        return templateRepository.findAllByTenantIdAndProjectId(tenantId, projectId).stream()
            .map(this::mapToTemplateDTO)
            .collect(Collectors.toList());
    }

    @Transactional(readOnly = true)
    public List<PropertyTemplateDTO> getTemplatesByScope(UUID tenantId, PropertyTemplate.PropertyScope scope) {
        return templateRepository.findAllByTenantIdAndScope(tenantId, scope).stream()
            .map(this::mapToTemplateDTO)
            .collect(Collectors.toList());
    }

    @Transactional
    @CacheEvict(value = "templates", key = "#tenantId")
    public PropertyTemplateDTO createTemplate(UUID tenantId, UUID userId, CreateTemplateRequest request) {
        if (templateRepository.existsByTenantIdAndNameAndDeletedAtIsNull(tenantId, request.getName())) {
            throw new PropertyValidationException("Template with name '" + request.getName() + "' already exists");
        }

        PropertyTemplate template = PropertyTemplate.builder()
            .tenantId(tenantId)
            .projectId(request.getProjectId())
            .name(request.getName())
            .displayName(request.getDisplayName())
            .description(request.getDescription())
            .dataType(PropertyTemplate.DataType.valueOf(request.getDataType().toUpperCase()))
            .unit(request.getUnit())
            .unitCategory(request.getUnitCategory())
            .defaultValue(request.getDefaultValue())
            .minValue(request.getMinValue())
            .maxValue(request.getMaxValue())
            .allowedValues(request.getAllowedValues() != null ? String.join(",", request.getAllowedValues()) : null)
            .regexPattern(request.getRegexPattern())
            .isRequired(request.getIsRequired() != null ? request.getIsRequired() : false)
            .isReadOnly(request.getIsReadOnly() != null ? request.getIsReadOnly() : false)
            .isHidden(request.getIsHidden() != null ? request.getIsHidden() : false)
            .groupName(request.getGroupName())
            .sortOrder(request.getSortOrder() != null ? request.getSortOrder() : 0)
            .scope(request.getScope() != null ? PropertyTemplate.PropertyScope.valueOf(request.getScope().toUpperCase()) : PropertyTemplate.PropertyScope.GLOBAL)
            .appliesTo(request.getAppliesTo())
            .calculationRule(request.getCalculationRule())
            .validationRules(request.getValidationRules())
            .dependsOn(request.getDependsOn() != null ? String.join(",", request.getDependsOn()) : null)
            .createdBy(userId)
            .build();

        PropertyTemplate saved = templateRepository.save(template);
        return mapToTemplateDTO(saved);
    }

    @Transactional
    @CacheEvict(value = "templates", key = "#tenantId")
    public PropertyTemplateDTO updateTemplate(UUID tenantId, UUID templateId, UUID userId, CreateTemplateRequest request) {
        PropertyTemplate template = templateRepository.findActiveById(templateId)
            .orElseThrow(() -> new TemplateNotFoundException("Template not found: " + templateId));

        if (!template.getName().equals(request.getName()) && 
            templateRepository.existsByTenantIdAndNameAndDeletedAtIsNull(tenantId, request.getName())) {
            throw new PropertyValidationException("Template with name '" + request.getName() + "' already exists");
        }

        template.setName(request.getName());
        template.setDisplayName(request.getDisplayName());
        template.setDescription(request.getDescription());
        template.setUnit(request.getUnit());
        template.setDefaultValue(request.getDefaultValue());
        template.setMinValue(request.getMinValue());
        template.setMaxValue(request.getMaxValue());
        template.setAllowedValues(request.getAllowedValues() != null ? String.join(",", request.getAllowedValues()) : null);
        template.setIsRequired(request.getIsRequired() != null ? request.getIsRequired() : false);
        template.setIsReadOnly(request.getIsReadOnly() != null ? request.getIsReadOnly() : false);
        template.setIsHidden(request.getIsHidden() != null ? request.getIsHidden() : false);
        template.setGroupName(request.getGroupName());
        template.setSortOrder(request.getSortOrder() != null ? request.getSortOrder() : 0);
        template.setAppliesTo(request.getAppliesTo());
        template.setCalculationRule(request.getCalculationRule());
        template.setValidationRules(request.getValidationRules());
        template.setDependsOn(request.getDependsOn() != null ? String.join(",", request.getDependsOn()) : null);
        template.setUpdatedBy(userId);

        PropertyTemplate saved = templateRepository.save(template);
        return mapToTemplateDTO(saved);
    }

    @Transactional
    @CacheEvict(value = "templates", key = "#tenantId")
    public void deleteTemplate(UUID tenantId, UUID templateId) {
        PropertyTemplate template = templateRepository.findActiveById(templateId)
            .orElseThrow(() -> new TemplateNotFoundException("Template not found: " + templateId));

        template.setDeletedAt(LocalDateTime.now());
        templateRepository.save(template);
        
        log.info("Deleted template: {}", templateId);
    }

    // Value Operations
    @Transactional(readOnly = true)
    @Cacheable(value = "propertyValues", key = "#entityType + ':' + #entityId")
    public List<PropertyValueDTO> getPropertyValues(String entityType, UUID entityId) {
        return valueRepository.findAllByEntity(entityType, entityId).stream()
            .map(this::mapToValueDTO)
            .collect(Collectors.toList());
    }

    @Transactional(readOnly = true)
    public PropertyValueDTO getPropertyValue(UUID templateId, String entityType, UUID entityId) {
        PropertyValue value = valueRepository.findByTemplateIdAndEntityTypeAndEntityId(templateId, entityType, entityId)
            .orElse(null);
        
        if (value == null) {
            // Return default value from template
            PropertyTemplate template = templateRepository.findActiveById(templateId).orElse(null);
            if (template != null && template.getDefaultValue() != null) {
                return PropertyValueDTO.builder()
                    .templateId(templateId)
                    .templateName(template.getName())
                    .templateDisplayName(template.getDisplayName())
                    .entityType(entityType)
                    .entityId(entityId)
                    .value(template.getDefaultValue())
                    .displayValue(template.getDefaultValue())
                    .unit(template.getUnit())
                    .isInherited(true)
                    .build();
            }
            return null;
        }
        
        return mapToValueDTO(value);
    }

    @Transactional
    @CacheEvict(value = "propertyValues", key = "#request.entityType + ':' + #request.entityId")
    public PropertyValueDTO setPropertyValue(UUID tenantId, UUID userId, SetPropertyValueRequest request) {
        PropertyTemplate template = templateRepository.findActiveById(request.getTemplateId())
            .orElseThrow(() -> new TemplateNotFoundException("Template not found: " + request.getTemplateId()));

        if (template.getIsReadOnly()) {
            throw new PropertyValidationException("Property '" + template.getName() + "' is read-only");
        }

        // Validate value
        String valueToSet = request.getValue();
        if (valueToSet != null && !valueToSet.isEmpty()) {
            if (!ruleEngine.validateValue(template, valueToSet)) {
                throw new PropertyValidationException("Invalid value for property '" + template.getName() + "'");
            }
        }

        // Unit conversion if needed
        String targetUnit = request.getUnit() != null ? request.getUnit() : template.getUnit();
        if (template.hasUnit() && request.getUnit() != null && !request.getUnit().equals(template.getUnit())) {
            valueToSet = unitConverter.convert(valueToSet, request.getUnit(), template.getUnit());
        }

        PropertyValue value = valueRepository.findByTemplateIdAndEntityTypeAndEntityId(
            request.getTemplateId(), request.getEntityType(), request.getEntityId()).orElse(null);

        if (value == null) {
            value = PropertyValue.builder()
                .template(template)
                .tenantId(tenantId)
                .entityType(request.getEntityType())
                .entityId(request.getEntityId())
                .createdBy(userId)
                .build();
        }

        value.setValue(valueToSet);
        value.setUnit(targetUnit);
        value.setDisplayValue(valueToSet);
        value.setUpdatedBy(userId);
        value.setOverrideReason(request.getOverrideReason());

        PropertyValue saved = valueRepository.save(value);

        // Execute dependent calculations
        executeDependentCalculations(tenantId, template.getName(), request.getEntityType(), request.getEntityId(), userId);

        return mapToValueDTO(saved);
    }

    @Transactional
    @CacheEvict(value = "propertyValues", key = "#entityType + ':' + #entityId")
    public void deletePropertyValues(String entityType, UUID entityId) {
        valueRepository.deleteAllByEntity(entityType, entityId);
    }

    @Transactional
    public Map<String, PropertyValueDTO> bulkUpdateProperties(UUID tenantId, UUID userId, BulkPropertyUpdateRequest request) {
        Map<String, PropertyValueDTO> results = new HashMap<>();
        
        for (UUID entityId : request.getEntityIds()) {
            for (Map.Entry<String, String> entry : request.getProperties().entrySet()) {
                PropertyTemplate template = templateRepository.findByTenantIdAndName(tenantId, entry.getKey()).orElse(null);
                if (template != null) {
                    try {
                        SetPropertyValueRequest valueRequest = SetPropertyValueRequest.builder()
                            .templateId(template.getId())
                            .entityType(request.getEntityType())
                            .entityId(entityId)
                            .value(entry.getValue())
                            .build();
                        
                        PropertyValueDTO result = setPropertyValue(tenantId, userId, valueRequest);
                        results.put(entityId + ":" + entry.getKey(), result);
                    } catch (Exception e) {
                        log.warn("Failed to set property {} for entity {}: {}", entry.getKey(), entityId, e.getMessage());
                    }
                }
            }
        }
        
        return results;
    }

    // Group Operations
    @Transactional(readOnly = true)
    public List<PropertyGroupDTO> getPropertyGroups(UUID tenantId) {
        return groupRepository.findAllByTenantId(tenantId).stream()
            .map(this::mapToGroupDTO)
            .collect(Collectors.toList());
    }

    @Transactional
    public PropertyGroupDTO createPropertyGroup(UUID tenantId, UUID userId, PropertyGroupDTO request) {
        if (groupRepository.existsByTenantIdAndName(tenantId, request.getName())) {
            throw new PropertyValidationException("Group with name '" + request.getName() + "' already exists");
        }

        PropertyGroup group = PropertyGroup.builder()
            .tenantId(tenantId)
            .projectId(request.getProjectId())
            .name(request.getName())
            .displayName(request.getDisplayName())
            .description(request.getDescription())
            .icon(request.getIcon())
            .color(request.getColor())
            .sortOrder(request.getSortOrder() != null ? request.getSortOrder() : 0)
            .createdBy(userId)
            .build();

        PropertyGroup saved = groupRepository.save(group);
        return mapToGroupDTO(saved);
    }

    // Validation
    @Transactional(readOnly = true)
    public PropertyValidationResult validateProperties(String entityType, UUID entityId) {
        List<PropertyValue> values = valueRepository.findAllByEntity(entityType, entityId);
        List<PropertyValidationResult.PropertyError> errors = new ArrayList<>();

        for (PropertyValue value : values) {
            PropertyTemplate template = value.getTemplate();
            
            if (template.getIsRequired() && (value.getValue() == null || value.getValue().isEmpty())) {
                errors.add(PropertyValidationResult.PropertyError.builder()
                    .propertyName(template.getName())
                    .errorCode("REQUIRED")
                    .errorMessage("Property '" + template.getDisplayName() + "' is required")
                    .build());
                continue;
            }

            if (value.getValue() != null && !value.getValue().isEmpty()) {
                if (!ruleEngine.validateValue(template, value.getValue())) {
                    errors.add(PropertyValidationResult.PropertyError.builder()
                        .propertyName(template.getName())
                        .errorCode("INVALID_VALUE")
                        .errorMessage("Invalid value for property '" + template.getDisplayName() + "'")
                        .build());
                }
            }
        }

        return PropertyValidationResult.builder()
            .valid(errors.isEmpty())
            .errors(errors)
            .build();
    }

    // Helper methods
    private void executeDependentCalculations(UUID tenantId, String propertyName, String entityType, UUID entityId, UUID userId) {
        List<PropertyTemplate> dependentTemplates = templateRepository.findAllDependentOnProperty(tenantId, propertyName);
        
        for (PropertyTemplate template : dependentTemplates) {
            if (template.getCalculationRule() != null && !template.getCalculationRule().isEmpty()) {
                try {
                    Map<String, Object> context = buildCalculationContext(template, entityType, entityId);
                    Object result = ruleEngine.evaluateCalculation(template.getCalculationRule(), context);
                    
                    if (result != null) {
                        SetPropertyValueRequest calcRequest = SetPropertyValueRequest.builder()
                            .templateId(template.getId())
                            .entityType(entityType)
                            .entityId(entityId)
                            .value(result.toString())
                            .build();
                        
                        setPropertyValue(tenantId, userId, calcRequest);
                    }
                } catch (Exception e) {
                    log.warn("Calculation failed for template {}: {}", template.getName(), e.getMessage());
                }
            }
        }
    }

    private Map<String, Object> buildCalculationContext(PropertyTemplate template, String entityType, UUID entityId) {
        Map<String, Object> context = new HashMap<>();
        
        // Add all current property values to context
        List<PropertyValue> values = valueRepository.findAllByEntity(entityType, entityId);
        for (PropertyValue value : values) {
            context.put(value.getTemplate().getName(), value.getValue());
        }
        
        // Add default values for properties not yet set
        if (template.getDependsOn() != null) {
            for (String dep : template.getDependsOn().split(",")) {
                if (!context.containsKey(dep.trim())) {
                    templateRepository.findByTenantIdAndName(template.getTenantId(), dep.trim())
                        .ifPresent(t -> context.put(t.getName(), t.getDefaultValue()));
                }
            }
        }
        
        return context;
    }

    private PropertyTemplateDTO mapToTemplateDTO(PropertyTemplate template) {
        return PropertyTemplateDTO.builder()
            .id(template.getId())
            .tenantId(template.getTenantId())
            .projectId(template.getProjectId())
            .name(template.getName())
            .displayName(template.getDisplayName())
            .description(template.getDescription())
            .dataType(template.getDataType().name())
            .unit(template.getUnit())
            .unitCategory(template.getUnitCategory())
            .defaultValue(template.getDefaultValue())
            .minValue(template.getMinValue())
            .maxValue(template.getMaxValue())
            .allowedValues(template.getAllowedValues() != null ? Arrays.asList(template.getAllowedValues().split(",")) : null)
            .regexPattern(template.getRegexPattern())
            .isRequired(template.getIsRequired())
            .isReadOnly(template.getIsReadOnly())
            .isHidden(template.getIsHidden())
            .groupName(template.getGroupName())
            .sortOrder(template.getSortOrder())
            .scope(template.getScope().name())
            .appliesTo(template.getAppliesTo())
            .calculationRule(template.getCalculationRule())
            .validationRules(template.getValidationRules())
            .dependsOn(template.getDependsOn() != null ? Arrays.asList(template.getDependsOn().split(",")) : null)
            .createdAt(template.getCreatedAt())
            .updatedAt(template.getUpdatedAt())
            .createdBy(template.getCreatedBy())
            .build();
    }

    private PropertyValueDTO mapToValueDTO(PropertyValue value) {
        PropertyTemplate template = value.getTemplate();
        return PropertyValueDTO.builder()
            .id(value.getId())
            .templateId(template.getId())
            .templateName(template.getName())
            .templateDisplayName(template.getDisplayName())
            .dataType(template.getDataType().name())
            .entityType(value.getEntityType())
            .entityId(value.getEntityId())
            .value(value.getValue())
            .displayValue(value.getDisplayValue())
            .unit(value.getUnit())
            .isCalculated(value.getIsCalculated())
            .calculationSource(value.getCalculationSource())
            .isInherited(value.getIsInherited())
            .inheritedFrom(value.getInheritedFrom())
            .overrideReason(value.getOverrideReason())
            .createdAt(value.getCreatedAt())
            .updatedAt(value.getUpdatedAt())
            .updatedBy(value.getUpdatedBy())
            .build();
    }

    private PropertyGroupDTO mapToGroupDTO(PropertyGroup group) {
        return PropertyGroupDTO.builder()
            .id(group.getId())
            .tenantId(group.getTenantId())
            .projectId(group.getProjectId())
            .name(group.getName())
            .displayName(group.getDisplayName())
            .description(group.getDescription())
            .icon(group.getIcon())
            .color(group.getColor())
            .sortOrder(group.getSortOrder())
            .isCollapsed(group.getIsCollapsed())
            .isSystem(group.getIsSystem())
            .createdAt(group.getCreatedAt())
            .createdBy(group.getCreatedBy())
            .build();
    }
}
