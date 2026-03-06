package com.archplatform.property.engine;

import com.archplatform.property.entity.PropertyRule;
import com.archplatform.property.entity.PropertyTemplate;
import com.archplatform.property.entity.PropertyValue;
import lombok.extern.slf4j.Slf4j;
import org.mvel2.MVEL;
import org.springframework.stereotype.Component;

import java.io.Serializable;
import java.math.BigDecimal;
import java.util.*;

@Slf4j
@Component
public class RuleEngine {

    private final Map<String, Serializable> compiledExpressions = new HashMap<>();

    public ExecutionResult executeRule(PropertyRule rule, Map<String, Object> context) {
        long startTime = System.currentTimeMillis();
        
        try {
            // Check condition if present
            if (rule.getConditionExpression() != null && !rule.getConditionExpression().isEmpty()) {
                boolean conditionMet = evaluateCondition(rule.getConditionExpression(), context);
                if (!conditionMet) {
                    return ExecutionResult.skipped("Condition not met");
                }
            }

            // Execute action
            Object result = executeAction(rule.getActionExpression(), context);
            
            long executionTime = System.currentTimeMillis() - startTime;
            
            return ExecutionResult.success(result, executionTime);
            
        } catch (Exception e) {
            log.error("Rule execution failed: {}", rule.getName(), e);
            return ExecutionResult.failed(e.getMessage());
        }
    }

    public Object evaluateCalculation(String expression, Map<String, Object> variables) {
        try {
            Serializable compiled = getCompiledExpression(expression);
            return MVEL.executeExpression(compiled, variables);
        } catch (Exception e) {
            log.error("Calculation evaluation failed: {}", expression, e);
            throw new RuleExecutionException("Failed to evaluate calculation: " + e.getMessage());
        }
    }

    public boolean validateValue(PropertyTemplate template, String value) {
        if (value == null || value.isEmpty()) {
            return !template.getIsRequired();
        }

        try {
            // Data type validation
            switch (template.getDataType()) {
                case INTEGER -> Integer.parseInt(value);
                case DECIMAL, LENGTH, AREA, VOLUME, ANGLE -> new BigDecimal(value);
                case BOOLEAN -> Boolean.parseBoolean(value);
                case ENUM -> validateEnumValue(template, value);
                case MULTI_SELECT -> validateMultiSelectValue(template, value);
                case URL -> validateUrl(value);
                case JSON -> validateJson(value);
            }

            // Range validation for numeric types
            if (template.isNumeric()) {
                BigDecimal numericValue = new BigDecimal(value);
                
                if (template.getMinValue() != null) {
                    BigDecimal min = new BigDecimal(template.getMinValue());
                    if (numericValue.compareTo(min) < 0) {
                        return false;
                    }
                }
                
                if (template.getMaxValue() != null) {
                    BigDecimal max = new BigDecimal(template.getMaxValue());
                    if (numericValue.compareTo(max) > 0) {
                        return false;
                    }
                }
            }

            // Regex validation
            if (template.getRegexPattern() != null && !template.getRegexPattern().isEmpty()) {
                if (!value.matches(template.getRegexPattern())) {
                    return false;
                }
            }

            // Allowed values validation
            if (template.getAllowedValues() != null && !template.getAllowedValues().isEmpty()) {
                List<String> allowed = Arrays.asList(template.getAllowedValues().split(","));
                if (!allowed.contains(value)) {
                    return false;
                }
            }

            return true;
            
        } catch (Exception e) {
            return false;
        }
    }

    private boolean evaluateCondition(String condition, Map<String, Object> context) {
        Serializable compiled = getCompiledExpression(condition);
        Object result = MVEL.executeExpression(compiled, context);
        return Boolean.TRUE.equals(result);
    }

    private Object executeAction(String action, Map<String, Object> context) {
        Serializable compiled = getCompiledExpression(action);
        return MVEL.executeExpression(compiled, context);
    }

    private Serializable getCompiledExpression(String expression) {
        return compiledExpressions.computeIfAbsent(expression, MVEL::compileExpression);
    }

    private void validateEnumValue(PropertyTemplate template, String value) {
        if (template.getAllowedValues() != null) {
            List<String> allowed = Arrays.asList(template.getAllowedValues().split(","));
            if (!allowed.contains(value)) {
                throw new IllegalArgumentException("Value not in allowed list");
            }
        }
    }

    private void validateMultiSelectValue(PropertyTemplate template, String value) {
        if (template.getAllowedValues() != null) {
            List<String> allowed = Arrays.asList(template.getAllowedValues().split(","));
            List<String> selected = Arrays.asList(value.split(","));
            if (!allowed.containsAll(selected)) {
                throw new IllegalArgumentException("Some values not in allowed list");
            }
        }
    }

    private void validateUrl(String value) {
        if (!value.startsWith("http://") && !value.startsWith("https://")) {
            throw new IllegalArgumentException("Invalid URL format");
        }
    }

    private void validateJson(String value) {
        // Basic JSON validation - could be enhanced with a proper JSON parser
        if (!value.trim().startsWith("{") && !value.trim().startsWith("[")) {
            throw new IllegalArgumentException("Invalid JSON format");
        }
    }

    public Map<String, Object> buildContext(PropertyValue value, PropertyTemplate template, Map<String, Object> additionalVars) {
        Map<String, Object> context = new HashMap<>();
        
        // Add property value
        context.put("value", value.getValue());
        context.put("unit", value.getUnit());
        
        // Add template info
        context.put("templateName", template.getName());
        context.put("dataType", template.getDataType().name());
        context.put("defaultValue", template.getDefaultValue());
        
        // Add additional variables
        if (additionalVars != null) {
            context.putAll(additionalVars);
        }
        
        return context;
    }

    public record ExecutionResult(boolean success, Object result, String message, long executionTimeMs) {
        public static ExecutionResult success(Object result, long executionTimeMs) {
            return new ExecutionResult(true, result, null, executionTimeMs);
        }
        
        public static ExecutionResult skipped(String reason) {
            return new ExecutionResult(false, null, reason, 0);
        }
        
        public static ExecutionResult failed(String error) {
            return new ExecutionResult(false, null, error, 0);
        }
    }

    public static class RuleExecutionException extends RuntimeException {
        public RuleExecutionException(String message) {
            super(message);
        }
    }
}
