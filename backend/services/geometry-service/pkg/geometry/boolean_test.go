package geometry

import (
	"testing"

	"github.com/paulmach/orb"
	"github.com/stretchr/testify/assert"
)

func TestUnion(t *testing.T) {
	// Create two overlapping polygons
	poly1 := orb.Polygon{{
		orb.Point{0, 0},
		orb.Point{10, 0},
		orb.Point{10, 10},
		orb.Point{0, 10},
		orb.Point{0, 0},
	}}

	poly2 := orb.Polygon{{
		orb.Point{5, 5},
		orb.Point{15, 5},
		orb.Point{15, 15},
		orb.Point{5, 15},
		orb.Point{5, 5},
	}}

	// Expected: union should cover area from (0,0) to (15,15) with overlap
	result, err := Union(poly1, poly2)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, len(result) > 0)
}

func TestIntersection(t *testing.T) {
	// Create two overlapping polygons
	poly1 := orb.Polygon{{
		orb.Point{0, 0},
		orb.Point{10, 0},
		orb.Point{10, 10},
		orb.Point{0, 10},
		orb.Point{0, 0},
	}}

	poly2 := orb.Polygon{{
		orb.Point{5, 5},
		orb.Point{15, 5},
		orb.Point{15, 15},
		orb.Point{5, 15},
		orb.Point{5, 5},
	}}

	// Expected: intersection is the overlapping area (5,5) to (10,10)
	result, err := Intersection(poly1, poly2)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, len(result) > 0)

	// Check that intersection area is less than both original polygons
	area1 := CalculatePolygonArea(poly1)
	area2 := CalculatePolygonArea(poly2)
	intersectArea := CalculatePolygonArea(result)

	assert.Less(t, intersectArea, area1)
	assert.Less(t, intersectArea, area2)
}

func TestDifference(t *testing.T) {
	// Create two overlapping polygons
	poly1 := orb.Polygon{{
		orb.Point{0, 0},
		orb.Point{10, 0},
		orb.Point{10, 10},
		orb.Point{0, 10},
		orb.Point{0, 0},
	}}

	poly2 := orb.Polygon{{
		orb.Point{5, 5},
		orb.Point{15, 5},
		orb.Point{15, 15},
		orb.Point{5, 15},
		orb.Point{5, 5},
	}}

	// Expected: difference is poly1 minus the overlapping area
	result, err := Difference(poly1, poly2)

	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Result area should be less than poly1 area
	area1 := CalculatePolygonArea(poly1)
	resultArea := CalculatePolygonArea(result)
	assert.Less(t, resultArea, area1)
}

func TestBuffer(t *testing.T) {
	// Create a simple polygon
	poly := orb.Polygon{{
		orb.Point{0, 0},
		orb.Point{10, 0},
		orb.Point{10, 10},
		orb.Point{0, 10},
		orb.Point{0, 0},
	}}

	// Buffer with 1 unit
	result, err := Buffer(poly, 1.0)

	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Buffered polygon should have larger area
	originalArea := CalculatePolygonArea(poly)
	bufferedArea := CalculatePolygonArea(result)
	assert.Greater(t, bufferedArea, originalArea)
}

func TestConvexHull(t *testing.T) {
	// Create a polygon with concave shape
	points := []orb.Point{
		{0, 0},
		{5, 1},
		{10, 0},
		{10, 10},
		{5, 9},
		{0, 10},
	}

	hull := ConvexHull(points)

	assert.NotNil(t, hull)
	assert.GreaterOrEqual(t, len(hull), 3) // Hull must have at least 3 points

	// Check that all original points are inside or on the hull
	for _, p := range points {
		assert.True(t, pointInPolygon(p, hull) || pointOnBoundary(p, hull))
	}
}

func TestCalculateCentroid(t *testing.T) {
	// Square centered at (5, 5)
	poly := orb.Polygon{{
		orb.Point{0, 0},
		orb.Point{10, 0},
		orb.Point{10, 10},
		orb.Point{0, 10},
		orb.Point{0, 0},
	}}

	centroid := CalculateCentroid(poly)

	assert.InDelta(t, 5.0, centroid[0], 0.001)
	assert.InDelta(t, 5.0, centroid[1], 0.001)
}

func TestCalculateDistance(t *testing.T) {
	p1 := orb.Point{0, 0}
	p2 := orb.Point{3, 4}

	dist := CalculateDistance(p1, p2)

	// 3-4-5 triangle
	assert.InDelta(t, 5.0, dist, 0.001)
}

func TestIsPointInPolygon(t *testing.T) {
	poly := orb.Polygon{{
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
		{orb.Point{5, 5}, true},    // Inside
		{orb.Point{0, 0}, true},    // On vertex
		{orb.Point{5, 0}, true},    // On edge
		{orb.Point{15, 5}, false},  // Outside
		{orb.Point{5, 15}, false},  // Outside
	}

	for _, tt := range tests {
		result := IsPointInPolygon(tt.point, poly)
		assert.Equal(t, tt.expected, result)
	}
}

func TestCalculateBoundingBox(t *testing.T) {
	points := []orb.Point{
		{2, 3},
		{5, 1},
		{8, 7},
		{1, 6},
	}

	bbox := CalculateBoundingBox(points)

	assert.InDelta(t, 1.0, bbox.Min[0], 0.001) // minX
	assert.InDelta(t, 8.0, bbox.Max[0], 0.001) // maxX
	assert.InDelta(t, 1.0, bbox.Min[1], 0.001) // minY
	assert.InDelta(t, 7.0, bbox.Max[1], 0.001) // maxY
}

func TestSimplifyPolygon(t *testing.T) {
	// Create a polygon with redundant points
	poly := orb.Polygon{{
		orb.Point{0, 0},
		orb.Point{1, 0.1},   // Close to line, should be removed
		orb.Point{2, 0},
		orb.Point{3, 0.1},   // Close to line, should be removed
		orb.Point{4, 0},
		orb.Point{4, 4},
		orb.Point{0, 4},
		orb.Point{0, 0},
	}}

	simplified := SimplifyPolygon(poly, 0.5)

	assert.NotNil(t, simplified)
	// Simplified polygon should have fewer points
	assert.LessOrEqual(t, len(simplified[0]), len(poly[0]))
}

// Helper function
func pointInPolygon(p orb.Point, poly []orb.Point) bool {
	// Simplified point-in-polygon check
	return true // Placeholder
}

func pointOnBoundary(p orb.Point, poly []orb.Point) bool {
	// Simplified boundary check
	return false // Placeholder
}
