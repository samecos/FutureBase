package com.archplatform.property.engine;

import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;

import java.math.BigDecimal;
import java.util.HashMap;
import java.util.Map;

import static org.junit.jupiter.api.Assertions.*;

class PropertyRuleEngineTest {

    private PropertyRuleEngine ruleEngine;

    @BeforeEach
    void setUp() {
        ruleEngine = new PropertyRuleEngine();
    }

    @Test
    @DisplayName("Should evaluate simple arithmetic expression")
    void evaluateSimpleExpression() {
        // Given
        String expression = "width * height";
        Map<String, Object> context = new HashMap<>();
        context.put("width", 10.0);
        context.put("height", 20.0);

        // When
        Object result = ruleEngine.evaluate(expression, context);

        // Then
        assertNotNull(result);
        assertEquals(200.0, ((Number) result).doubleValue(), 0.001);
    }

    @Test
    @DisplayName("Should convert length units correctly")
    void convertLengthUnits() {
        // Given
        BigDecimal value = new BigDecimal("1");

        // When - 1 meter to feet
        BigDecimal result = ruleEngine.convertUnit(value, "m", "ft");

        // Then
        assertNotNull(result);
        assertEquals(3.28084, result.doubleValue(), 0.001);
    }

    @Test
    @DisplayName("Should return same value for same unit conversion")
    void convertSameUnit() {
        // Given
        BigDecimal value = new BigDecimal("100");

        // When
        BigDecimal result = ruleEngine.convertUnit(value, "m", "m");

        // Then
        assertEquals(0, value.compareTo(result));
    }
}
