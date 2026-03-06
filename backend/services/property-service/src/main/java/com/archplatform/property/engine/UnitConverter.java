package com.archplatform.property.engine;

import lombok.extern.slf4j.Slf4j;
import org.springframework.stereotype.Component;

import javax.measure.Unit;
import javax.measure.quantity.*;
import javax.measure.spi.ServiceProvider;
import java.math.BigDecimal;
import java.math.RoundingMode;
import java.util.HashMap;
import java.util.Map;

import static tech.units.indriya.unit.Units.*;

@Slf4j
@Component
public class UnitConverter {

    private final Map<String, Unit<?>> unitRegistry = new HashMap<>();

    public UnitConverter() {
        initializeUnits();
    }

    private void initializeUnits() {
        // Length
        unitRegistry.put("m", METRE);
        unitRegistry.put("mm", MILLI(METRE));
        unitRegistry.put("cm", CENTI(METRE));
        unitRegistry.put("km", KILO(METRE));
        unitRegistry.put("ft", FOOT);
        unitRegistry.put("in", INCH);
        unitRegistry.put("yd", YARD);

        // Area
        unitRegistry.put("m2", SQUARE_METRE);
        unitRegistry.put("cm2", CENTI(METRE).multiply(CENTI(METRE)));
        unitRegistry.put("mm2", MILLI(METRE).multiply(MILLI(METRE)));
        unitRegistry.put("ft2", SQUARE_FOOT);
        unitRegistry.put("in2", SQUARE_INCH);

        // Volume
        unitRegistry.put("m3", CUBIC_METRE);
        unitRegistry.put("l", LITRE);
        unitRegistry.put("ml", MILLI(LITRE));
        unitRegistry.put("gal", GALLON_LIQUID_US);
        unitRegistry.put("ft3", CUBIC_FOOT);

        // Angle
        unitRegistry.put("deg", DEGREE_ANGLE);
        unitRegistry.put("rad", RADIAN);

        // Temperature
        unitRegistry.put("c", CELSIUS);
        unitRegistry.put("f", FAHRENHEIT);
        unitRegistry.put("k", KELVIN);

        // Pressure
        unitRegistry.put("pa", PASCAL);
        unitRegistry.put("kpa", KILO(PASCAL));
        unitRegistry.put("mpa", MEGA(PASCAL));
        unitRegistry.put("psi", POUND_FORCE_PER_SQUARE_INCH);
        unitRegistry.put("bar", BAR);
    }

    public String convert(String value, String fromUnit, String toUnit) {
        if (value == null || value.isEmpty() || fromUnit == null || toUnit == null) {
            return value;
        }

        if (fromUnit.equalsIgnoreCase(toUnit)) {
            return value;
        }

        try {
            Unit<?> sourceUnit = unitRegistry.get(fromUnit.toLowerCase());
            Unit<?> targetUnit = unitRegistry.get(toUnit.toLowerCase());

            if (sourceUnit == null || targetUnit == null) {
                log.warn("Unknown unit: {} or {}", fromUnit, toUnit);
                return value;
            }

            BigDecimal numericValue = new BigDecimal(value);

            javax.measure.UnitConverter converter = sourceUnit.getConverterToAny(targetUnit);
            Number converted = converter.convert(numericValue);

            return new BigDecimal(converted.toString()).stripTrailingZeros().toPlainString();

        } catch (Exception e) {
            log.error("Unit conversion failed: {} {} to {}", value, fromUnit, toUnit, e);
            return value;
        }
    }

    public boolean isValidUnit(String unit) {
        return unit == null || unitRegistry.containsKey(unit.toLowerCase());
    }

    public boolean areUnitsCompatible(String unit1, String unit2) {
        if (unit1 == null || unit2 == null) {
            return unit1 == null && unit2 == null;
        }

        Unit<?> u1 = unitRegistry.get(unit1.toLowerCase());
        Unit<?> u2 = unitRegistry.get(unit2.toLowerCase());

        if (u1 == null || u2 == null) {
            return false;
        }

        return u1.isCompatible(u2);
    }

    public String getBaseUnit(String unitCategory) {
        return switch (unitCategory.toLowerCase()) {
            case "length" -> "m";
            case "area" -> "m2";
            case "volume" -> "m3";
            case "angle" -> "deg";
            case "temperature" -> "c";
            case "pressure" -> "pa";
            default -> null;
        };
    }
}
