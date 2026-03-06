package geometry

import (
	"math"
	"testing"

	"github.com/paulmach/orb"
	"github.com/stretchr/testify/assert"
)

func TestDouglasPeucker_SimpleLine(t *testing.T) {
	// Create a simple line with redundant points
	line := orb.LineString{
		orb.Point{0, 0},
		orb.Point{1, 0.1},   // This point should be removed (close to line)
		orb.Point{2, 0},
	}
	
	simplified := DouglasPeucker(line, 0.5)
	
	// Should keep start and end points
	assert.GreaterOrEqual(t, len(simplified), 2)
	assert.Equal(t, line[0], simplified[0])
	assert.Equal(t, line[len(line)-1], simplified[len(simplified)-1])
}

func TestDouglasPeucker_ComplexLine(t *testing.T) {
	// Create a line with some important points and some redundant
	line := orb.LineString{
		orb.Point{0, 0},
		orb.Point{1, 0.1},   // Close to line, should be removed
		orb.Point{2, 5},     // Important deviation, should be kept
		orb.Point{3, 4.9},   // Close to line, should be removed
		orb.Point{4, 0},
	}
	
	simplified := DouglasPeucker(line, 1.0)
	
	// Should keep the point with deviation
	assert.GreaterOrEqual(t, len(simplified), 3)
}

func TestDouglasPeucker_SmallTolerance(t *testing.T) {
	// With very small tolerance, almost all points should be kept
	line := orb.LineString{
		orb.Point{0, 0},
		orb.Point{1, 0.01},
		orb.Point{2, 0.02},
		orb.Point{3, 0},
	}
	
	simplified := DouglasPeucker(line, 0.001)
	
	// Most points should be kept with small tolerance
	assert.Equal(t, len(line), len(simplified))
}

func TestDouglasPeucker_LargeTolerance(t *testing.T) {
	// With large tolerance, many points should be removed
	line := orb.LineString{
		orb.Point{0, 0},
		orb.Point{1, 0.1},
		orb.Point{2, 0.2},
		orb.Point{3, 0.1},
		orb.Point{4, 0},
	}
	
	simplified := DouglasPeucker(line, 10.0)
	
	// Only start and end should remain with large tolerance
	assert.Equal(t, 2, len(simplified))
}

func TestDouglasPeucker_MinimumPoints(t *testing.T) {
	// Lines with 2 or fewer points should remain unchanged
	line1 := orb.LineString{orb.Point{0, 0}}
	line2 := orb.LineString{orb.Point{0, 0}, orb.Point{1, 1}}
	
	result1 := DouglasPeucker(line1, 1.0)
	result2 := DouglasPeucker(line2, 1.0)
	
	assert.Equal(t, len(line1), len(result1))
	assert.Equal(t, len(line2), len(result2))
}

func TestPerpendicularDistance(t *testing.T) {
	tests := []struct {
		name     string
		point    orb.Point
		lineStart orb.Point
		lineEnd  orb.Point
		expected float64
	}{
		{
			name:     "Point on line",
			point:    orb.Point{1, 1},
			lineStart: orb.Point{0, 0},
			lineEnd:  orb.Point{2, 2},
			expected: 0,
		},
		{
			name:     "Point off diagonal line",
			point:    orb.Point{1, 2},
			lineStart: orb.Point{0, 0},
			lineEnd:  orb.Point{2, 0},
			expected: 2,
		},
		{
			name:     "Point off horizontal line",
			point:    orb.Point{5, 3},
			lineStart: orb.Point{0, 0},
			lineEnd:  orb.Point{10, 0},
			expected: 3,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dist := perpendicularDistance(tt.point, tt.lineStart, tt.lineEnd)
			assert.InDelta(t, tt.expected, dist, 0.0001)
		})
	}
}

func TestCalculatePolygonArea(t *testing.T) {
	tests := []struct {
		name     string
		polygon  orb.Polygon
		expected float64
	}{
		{
			name: "Square 10x10",
			polygon: orb.Polygon{{
				orb.Point{0, 0},
				orb.Point{10, 0},
				orb.Point{10, 10},
				orb.Point{0, 10},
				orb.Point{0, 0},
			}},
			expected: 100,
		},
		{
			name: "Rectangle 5x10",
			polygon: orb.Polygon{{
				orb.Point{0, 0},
				orb.Point{10, 0},
				orb.Point{10, 5},
				orb.Point{0, 5},
				orb.Point{0, 0},
			}},
			expected: 50,
		},
		{
			name: "Triangle",
			polygon: orb.Polygon{{
				orb.Point{0, 0},
				orb.Point{10, 0},
				orb.Point{5, 10},
				orb.Point{0, 0},
			}},
			expected: 50,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			area := CalculatePolygonArea(tt.polygon)
			assert.InDelta(t, tt.expected, area, 0.0001)
		})
	}
}

func TestCalculateBoundingBox(t *testing.T) {
	points := []orb.Point{
		{1, 5},
		{3, 2},
		{5, 8},
		{2, 1},
	}
	
	bbox := CalculateBoundingBox(points)
	
	assert.InDelta(t, 1.0, bbox.Min[0], 0.0001) // minX
	assert.InDelta(t, 5.0, bbox.Max[0], 0.0001) // maxX
	assert.InDelta(t, 1.0, bbox.Min[1], 0.0001) // minY
	assert.InDelta(t, 8.0, bbox.Max[1], 0.0001) // maxY
}

func TestCalculateDistance(t *testing.T) {
	// Test distance between two points
	p1 := orb.Point{0, 0}
	p2 := orb.Point{3, 4}
	
	dist := CalculateDistance(p1, p2)
	
	// 3-4-5 triangle
	assert.InDelta(t, 5.0, dist, 0.0001)
}

func TestIsPointInPolygon(t *testing.T) {
	polygon := orb.Polygon{{
		orb.Point{0, 0},
		orb.Point{10, 0},
		orb.Point{10, 10},
		orb.Point{0, 10},
		orb.Point{0, 0},
	}}
	
	tests := []struct {
		point    orb.Point
		expected bool
	}{
		{orb.Point{5, 5}, true},   // Inside
		{orb.Point{0, 0}, true},   // On edge
		{orb.Point{15, 5}, false}, // Outside
		{orb.Point{5, 15}, false}, // Outside
	}
	
	for _, tt := range tests {
		result := IsPointInPolygon(tt.point, polygon)
		assert.Equal(t, tt.expected, result)
	}
}

func TestDegreesToRadians(t *testing.T) {
	assert.InDelta(t, math.Pi, DegreesToRadians(180), 0.0001)
	assert.InDelta(t, math.Pi/2, DegreesToRadians(90), 0.0001)
	assert.InDelta(t, 0.0, DegreesToRadians(0), 0.0001)
}

func TestRadiansToDegrees(t *testing.T) {
	assert.InDelta(t, 180.0, RadiansToDegrees(math.Pi), 0.0001)
	assert.InDelta(t, 90.0, RadiansToDegrees(math.Pi/2), 0.0001)
	assert.InDelta(t, 0.0, RadiansToDegrees(0), 0.0001)
}
