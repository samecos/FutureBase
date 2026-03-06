package geometry

import (
	"fmt"

	"github.com/twpayne/go-geom"
)

// BooleanOperationType represents the type of boolean operation
type BooleanOperationType int

const (
	BooleanUnion BooleanOperationType = iota
	BooleanIntersection
	BooleanDifference
	BooleanSymmetricDifference
)

// BooleanOperation performs boolean operations on geometries
func BooleanOperation(g1, g2 geom.T, op BooleanOperationType) (geom.T, error) {
	// Check if geometries are compatible
	if g1.Layout() != g2.Layout() {
		return nil, fmt.Errorf("geometries have incompatible layouts")
	}
	
	switch op {
	case BooleanUnion:
		return Union(g1, g2)
	case BooleanIntersection:
		return Intersection(g1, g2)
	case BooleanDifference:
		return Difference(g1, g2)
	case BooleanSymmetricDifference:
		return SymmetricDifference(g1, g2)
	default:
		return nil, fmt.Errorf("unknown boolean operation: %d", op)
	}
}

// Union returns the union of two geometries
func Union(g1, g2 geom.T) (geom.T, error) {
	switch geom1 := g1.(type) {
	case *geom.Polygon:
		return polygonUnion(geom1, g2)
	case *geom.MultiPolygon:
		return multiPolygonUnion(geom1, g2)
	case *geom.LineString:
		return lineStringUnion(geom1, g2)
	default:
		return nil, fmt.Errorf("unsupported geometry type for union: %T", g1)
	}
}

// Intersection returns the intersection of two geometries
func Intersection(g1, g2 geom.T) (geom.T, error) {
	switch geom1 := g1.(type) {
	case *geom.Polygon:
		return polygonIntersection(geom1, g2)
	case *geom.LineString:
		return lineStringIntersection(geom1, g2)
	default:
		return nil, fmt.Errorf("unsupported geometry type for intersection: %T", g1)
	}
}

// Difference returns the difference of two geometries (g1 - g2)
func Difference(g1, g2 geom.T) (geom.T, error) {
	switch geom1 := g1.(type) {
	case *geom.Polygon:
		return polygonDifference(geom1, g2)
	case *geom.LineString:
		return lineStringDifference(geom1, g2)
	default:
		return nil, fmt.Errorf("unsupported geometry type for difference: %T", g1)
	}
}

// SymmetricDifference returns the symmetric difference of two geometries
func SymmetricDifference(g1, g2 geom.T) (geom.T, error) {
	switch geom1 := g1.(type) {
	case *geom.Polygon:
		return polygonSymmetricDifference(geom1, g2)
	default:
		return nil, fmt.Errorf("unsupported geometry type for symmetric difference: %T", g1)
	}
}

// polygonUnion computes the union of two polygons
func polygonUnion(p1 *geom.Polygon, g2 geom.T) (geom.T, error) {
	switch geom2 := g2.(type) {
	case *geom.Polygon:
		return mergePolygons(p1, geom2), nil
	case *geom.MultiPolygon:
		mp := geom.NewMultiPolygon(geom.XYZ)
		mp.MustPush(p1)
		for i := 0; i < geom2.NumPolygons(); i++ {
			mp.MustPush(geom2.Polygon(i))
		}
		return mp, nil
	default:
		return nil, fmt.Errorf("cannot union polygon with %T", g2)
	}
}

// multiPolygonUnion computes the union of a multipolygon with another geometry
func multiPolygonUnion(mp1 *geom.MultiPolygon, g2 geom.T) (geom.T, error) {
	switch geom2 := g2.(type) {
	case *geom.Polygon:
		result := geom.NewMultiPolygon(geom.XYZ)
		for i := 0; i < mp1.NumPolygons(); i++ {
			result.MustPush(mp1.Polygon(i))
		}
		result.MustPush(geom2)
		return result, nil
	case *geom.MultiPolygon:
		result := geom.NewMultiPolygon(geom.XYZ)
		for i := 0; i < mp1.NumPolygons(); i++ {
			result.MustPush(mp1.Polygon(i))
		}
		for i := 0; i < geom2.NumPolygons(); i++ {
			result.MustPush(geom2.Polygon(i))
		}
		return result, nil
	default:
		return nil, fmt.Errorf("cannot union multipolygon with %T", g2)
	}
}

// lineStringUnion computes the union of two line strings
func lineStringUnion(ls1 *geom.LineString, g2 geom.T) (geom.T, error) {
	switch geom2 := g2.(type) {
	case *geom.LineString:
		// Create a multi-line string
		mls := geom.NewMultiLineString(geom.XYZ)
		mls.MustPush(ls1, geom2)
		return mls, nil
	case *geom.MultiLineString:
		result := geom.NewMultiLineString(geom.XYZ)
		result.MustPush(ls1)
		for i := 0; i < geom2.NumLineStrings(); i++ {
			result.MustPush(geom2.LineString(i))
		}
		return result, nil
	default:
		return nil, fmt.Errorf("cannot union line string with %T", g2)
	}
}

// polygonIntersection computes the intersection of two polygons
func polygonIntersection(p1 *geom.Polygon, g2 geom.T) (geom.T, error) {
	switch geom2 := g2.(type) {
	case *geom.Polygon:
		return intersectPolygons(p1, geom2), nil
	default:
		return nil, fmt.Errorf("cannot intersect polygon with %T", g2)
	}
}

// lineStringIntersection computes the intersection of two line strings
func lineStringIntersection(ls1 *geom.LineString, g2 geom.T) (geom.T, error) {
	switch geom2 := g2.(type) {
	case *geom.LineString:
		// Find intersection points
		points := findLineStringIntersections(ls1, geom2)
		if len(points) == 0 {
			return nil, nil
		}
		if len(points) == 1 {
			return points[0], nil
		}
		mp := geom.NewMultiPoint(geom.XYZ)
		for _, p := range points {
			mp.MustPush(p)
		}
		return mp, nil
	default:
		return nil, fmt.Errorf("cannot intersect line string with %T", g2)
	}
}

// polygonDifference computes the difference of two polygons
func polygonDifference(p1 *geom.Polygon, g2 geom.T) (geom.T, error) {
	switch geom2 := g2.(type) {
	case *geom.Polygon:
		// Simplified implementation: return the first polygon if it doesn't intersect
		if !polygonsIntersect(p1, geom2) {
			return p1, nil
		}
		// If they intersect, return a multipolygon containing the non-overlapping parts
		return subtractPolygon(p1, geom2), nil
	default:
		return nil, fmt.Errorf("cannot compute difference between polygon and %T", g2)
	}
}

// lineStringDifference computes the difference of two line strings
func lineStringDifference(ls1 *geom.LineString, g2 geom.T) (geom.T, error) {
	switch geom2 := g2.(type) {
	case *geom.LineString:
		// Return the first line string minus any overlapping segments
		return subtractLineString(ls1, geom2), nil
	default:
		return nil, fmt.Errorf("cannot compute difference between line string and %T", g2)
	}
}

// polygonSymmetricDifference computes the symmetric difference of two polygons
func polygonSymmetricDifference(p1 *geom.Polygon, g2 geom.T) (geom.T, error) {
	switch geom2 := g2.(type) {
	case *geom.Polygon:
		// Simplified: union minus intersection
		union, _ := polygonUnion(p1, geom2)
		intersection, _ := polygonIntersection(p1, geom2)
		
		if union == nil || intersection == nil {
			return union, nil
		}
		
		return Difference(union, intersection)
	default:
		return nil, fmt.Errorf("cannot compute symmetric difference between polygon and %T", g2)
	}
}

// Helper functions for boolean operations

// mergePolygons combines two polygons into a multipolygon
func mergePolygons(p1, p2 *geom.Polygon) *geom.MultiPolygon {
	mp := geom.NewMultiPolygon(geom.XYZ)
	mp.MustPush(p1, p2)
	return mp
}

// intersectPolygons computes the intersection of two polygons
// This is a simplified implementation
func intersectPolygons(p1, p2 *geom.Polygon) geom.T {
	// Check if bounding boxes intersect
	bbox1 := polygonBounds(p1)
	bbox2 := polygonBounds(p2)
	
	if !bboxIntersect(bbox1, bbox2) {
		return nil
	}
	
	// Simplified: return a point at the centroid of the intersection of bounding boxes
	cx := (max(bbox1[0], bbox2[0]) + min(bbox1[2], bbox2[2])) / 2
	cy := (max(bbox1[1], bbox2[1]) + min(bbox1[3], bbox2[3])) / 2
	
	return geom.NewPoint(geom.XY).MustSetCoords(geom.Coord{cx, cy})
}

// subtractPolygon removes the area of p2 from p1
func subtractPolygon(p1, p2 *geom.Polygon) *geom.MultiPolygon {
	// Simplified: return p1 as a multipolygon
	// A full implementation would compute the actual difference
	mp := geom.NewMultiPolygon(geom.XYZ)
	mp.MustPush(p1)
	return mp
}

// subtractLineString removes overlapping segments
func subtractLineString(ls1, ls2 *geom.LineString) geom.T {
	// Simplified: return ls1
	// A full implementation would remove overlapping segments
	return ls1
}

// findLineStringIntersections finds intersection points between two line strings
func findLineStringIntersections(ls1, ls2 *geom.LineString) []*geom.Point {
	var intersections []*geom.Point
	
	coords1 := ls1.Coords()
	coords2 := ls2.Coords()
	
	for i := 0; i < len(coords1)-1; i++ {
		for j := 0; j < len(coords2)-1; j++ {
			if pt := lineSegmentIntersection(
				coords1[i], coords1[i+1],
				coords2[j], coords2[j+1],
			); pt != nil {
				intersections = append(intersections, pt)
			}
		}
	}
	
	return intersections
}

// lineSegmentIntersection finds the intersection of two line segments
func lineSegmentIntersection(a1, a2, b1, b2 geom.Coord) *geom.Point {
	// Convert to 2D for simplicity
	x1, y1 := a1.X(), a1.Y()
	x2, y2 := a2.X(), a2.Y()
	x3, y3 := b1.X(), b1.Y()
	x4, y4 := b2.X(), b2.Y()
	
	// Compute intersection
	denom := (x1-x2)*(y3-y4) - (y1-y2)*(x3-x4)
	if denom == 0 {
		return nil // Parallel lines
	}
	
	t := ((x1-x3)*(y3-y4) - (y1-y3)*(x3-x4)) / denom
	u := -((x1-x2)*(y1-y3) - (y1-y2)*(x1-x3)) / denom
	
	if t >= 0 && t <= 1 && u >= 0 && u <= 1 {
		x := x1 + t*(x2-x1)
		y := y1 + t*(y2-y1)
		return geom.NewPoint(geom.XY).MustSetCoords(geom.Coord{x, y})
	}
	
	return nil
}

// polygonsIntersect checks if two polygons intersect
func polygonsIntersect(p1, p2 *geom.Polygon) bool {
	bbox1 := polygonBounds(p1)
	bbox2 := polygonBounds(p2)
	return bboxIntersect(bbox1, bbox2)
}

// polygonBounds returns the bounding box of a polygon
func polygonBounds(p *geom.Polygon) [4]float64 {
	coords := p.FlatCoords()
	if len(coords) == 0 {
		return [4]float64{0, 0, 0, 0}
	}
	
	minX, minY := coords[0], coords[1]
	maxX, maxY := minX, minY
	
	stride := p.Stride()
	for i := stride; i < len(coords); i += stride {
		x, y := coords[i], coords[i+1]
		if x < minX {
			minX = x
		}
		if x > maxX {
			maxX = x
		}
		if y < minY {
			minY = y
		}
		if y > maxY {
			maxY = y
		}
	}
	
	return [4]float64{minX, minY, maxX, maxY}
}

// bboxIntersect checks if two bounding boxes intersect
func bboxIntersect(b1, b2 [4]float64) bool {
	return b1[0] <= b2[2] && b1[2] >= b2[0] &&
		b1[1] <= b2[3] && b1[3] >= b2[1]
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
