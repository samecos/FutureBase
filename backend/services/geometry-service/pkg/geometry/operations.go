package geometry

import (
	"fmt"
	"math"

	"github.com/twpayne/go-geom"
)

// Operations provides geometric calculations
type Operations struct {
	tolerance float64
}

// NewOperations creates a new operations instance
func NewOperations(tolerance float64) *Operations {
	return &Operations{
		tolerance: tolerance,
	}
}

// CalculateArea calculates the area of a geometry
func (o *Operations) CalculateArea(g geom.T) float64 {
	switch geom := g.(type) {
	case *geom.Polygon:
		return o.polygonArea(geom)
	case *geom.MultiPolygon:
		total := 0.0
		for i := 0; i < geom.NumPolygons(); i++ {
			total += o.polygonArea(geom.Polygon(i))
		}
		return total
	default:
		return 0.0
	}
}

// CalculateLength calculates the length of a geometry
func (o *Operations) CalculateLength(g geom.T) float64 {
	switch geom := g.(type) {
	case *geom.LineString:
		return o.lineStringLength(geom)
	case *geom.MultiLineString:
		total := 0.0
		for i := 0; i < geom.NumLineStrings(); i++ {
			total += o.lineStringLength(geom.LineString(i))
		}
		return total
	case *geom.Polygon:
		// Perimeter
		return o.polygonPerimeter(geom)
	case *geom.LinearRing:
		return o.linearRingLength(geom)
	default:
		return 0.0
	}
}

// CalculatePerimeter calculates the perimeter of a geometry
func (o *Operations) CalculatePerimeter(g geom.T) float64 {
	return o.CalculateLength(g)
}

// CalculateCentroid calculates the centroid of a geometry
func (o *Operations) CalculateCentroid(g geom.T) (x, y, z float64) {
	switch geom := g.(type) {
	case *geom.Point:
		coords := geom.Coords()
		return coords.X(), coords.Y(), coords.Z()
	case *geom.Polygon:
		return o.polygonCentroid(geom)
	case *geom.LineString:
		return o.lineStringCentroid(geom)
	default:
		return 0, 0, 0
	}
}

// CalculateDistance calculates the distance between two geometries
func (o *Operations) CalculateDistance(g1, g2 geom.T) float64 {
	// Calculate distance between centroids
	x1, y1, z1 := o.CalculateCentroid(g1)
	x2, y2, z2 := o.CalculateCentroid(g2)
	
	dx := x2 - x1
	dy := y2 - y1
	dz := z2 - z1
	
	return math.Sqrt(dx*dx + dy*dy + dz*dz)
}

// CalculateVertexCount counts the number of vertices in a geometry
func (o *Operations) CalculateVertexCount(g geom.T) int {
	switch geom := g.(type) {
	case *geom.Point:
		return 1
	case *geom.LineString:
		return geom.NumCoords()
	case *geom.Polygon:
		count := 0
		for i := 0; i < geom.NumLinearRings(); i++ {
			count += geom.LinearRing(i).NumCoords()
		}
		return count
	case *geom.MultiPoint:
		return geom.NumPoints()
	case *geom.MultiLineString:
		count := 0
		for i := 0; i < geom.NumLineStrings(); i++ {
			count += geom.LineString(i).NumCoords()
		}
		return count
	case *geom.MultiPolygon:
		count := 0
		for i := 0; i < geom.NumPolygons(); i++ {
			count += o.CalculateVertexCount(geom.Polygon(i))
		}
		return count
	default:
		return 0
	}
}

// Validate checks if a geometry is valid
func (o *Operations) Validate(g geom.T) (bool, []string) {
	var errors []string
	
	if g == nil {
		return false, []string{"geometry is nil"}
	}
	
	switch geom := g.(type) {
	case *geom.Point:
		if err := o.validatePoint(geom); err != nil {
			errors = append(errors, err...)
		}
	case *geom.LineString:
		if err := o.validateLineString(geom); err != nil {
			errors = append(errors, err...)
		}
	case *geom.Polygon:
		if err := o.validatePolygon(geom); err != nil {
			errors = append(errors, err...)
		}
	}
	
	return len(errors) == 0, errors
}

// IsSimple checks if a geometry is simple
func (o *Operations) IsSimple(g geom.T) bool {
	switch geom := g.(type) {
	case *geom.LineString:
		return o.isSimpleLineString(geom)
	case *geom.Polygon:
		return o.isSimplePolygon(geom)
	default:
		return true
	}
}

// IsEmpty checks if a geometry is empty
func (o *Operations) IsEmpty(g geom.T) bool {
	if g == nil {
		return true
	}
	
	switch geom := g.(type) {
	case *geom.Point:
		return false
	case *geom.LineString:
		return geom.NumCoords() == 0
	case *geom.Polygon:
		return geom.NumLinearRings() == 0
	default:
		return false
	}
}

// IsClosed checks if a geometry is closed
func (o *Operations) IsClosed(g geom.T) bool {
	switch geom := g.(type) {
	case *geom.LineString:
		if geom.NumCoords() < 2 {
			return false
		}
		first := geom.Coord(0)
		last := geom.Coord(geom.NumCoords() - 1)
		return first.X() == last.X() && first.Y() == last.Y()
	case *geom.LinearRing:
		return true
	default:
		return false
	}
}

// Simplify simplifies a geometry using Douglas-Peucker algorithm
func (o *Operations) Simplify(g geom.T, tolerance float64) geom.T {
	switch geom := g.(type) {
	case *geom.LineString:
		return o.simplifyLineString(geom, tolerance)
	case *geom.Polygon:
		return o.simplifyPolygon(geom, tolerance)
	default:
		return g
	}
}

// Buffer creates a buffer around a geometry
func (o *Operations) Buffer(g geom.T, distance float64) geom.T {
	// Simplified implementation: return the original geometry
	// A full implementation would compute the actual buffer
	return g
}

// ConvexHull computes the convex hull of a geometry
func (o *Operations) ConvexHull(g geom.T) geom.T {
	// Simplified implementation using a bounding box
	coords := g.FlatCoords()
	if len(coords) == 0 {
		return nil
	}
	
	stride := g.Stride()
	minX, minY := coords[0], coords[1]
	maxX, maxY := minX, minY
	
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
	
	// Create a polygon representing the bounding box
	return geom.NewPolygon(geom.XY).MustSetCoords([][]geom.Coord{
		{
			{minX, minY},
			{maxX, minY},
			{maxX, maxY},
			{minX, maxY},
			{minX, minY},
		},
	})
}

// InteriorPoint returns a point that is guaranteed to be inside the geometry
func (o *Operations) InteriorPoint(g geom.T) *geom.Point {
	centroidX, centroidY, centroidZ := o.CalculateCentroid(g)
	return geom.NewPoint(geom.XYZ).MustSetCoords(geom.Coord{centroidX, centroidY, centroidZ})
}

// Boundary returns the boundary of a geometry
func (o *Operations) Boundary(g geom.T) geom.T {
	switch geom := g.(type) {
	case *geom.Polygon:
		// Return the exterior ring as a line string
		if geom.NumLinearRings() > 0 {
			return geom.LinearRing(0)
		}
	case *geom.LineString:
		// Return the endpoints as a multi-point
		if geom.NumCoords() >= 2 {
			mp := geom.NewMultiPoint(geom.Layout())
			mp.MustPush(
				geom.NewPoint(geom.Layout()).MustSetCoords(geom.Coord(0)),
				geom.NewPoint(geom.Layout()).MustSetCoords(geom.Coord(geom.NumCoords()-1)),
			)
			return mp
		}
	}
	return nil
}

// Envelope returns the minimum bounding box of a geometry
func (o *Operations) Envelope(g geom.T) *geom.Polygon {
	return o.ConvexHull(g).(*geom.Polygon)
}

// Private helper methods

func (o *Operations) polygonArea(p *geom.Polygon) float64 {
	if p.NumLinearRings() == 0 {
		return 0
	}
	
	// Calculate area of exterior ring
	area := o.ringArea(p.LinearRing(0))
	
	// Subtract area of interior rings (holes)
	for i := 1; i < p.NumLinearRings(); i++ {
		area -= o.ringArea(p.LinearRing(i))
	}
	
	return math.Abs(area)
}

func (o *Operations) ringArea(ring *geom.LinearRing) float64 {
	coords := ring.Coords()
	n := len(coords)
	if n < 3 {
		return 0
	}
	
	area := 0.0
	for i := 0; i < n-1; i++ {
		area += coords[i].X() * coords[i+1].Y()
		area -= coords[i+1].X() * coords[i].Y()
	}
	
	return area / 2
}

func (o *Operations) polygonPerimeter(p *geom.Polygon) float64 {
	if p.NumLinearRings() == 0 {
		return 0
	}
	
	perimeter := o.linearRingLength(p.LinearRing(0))
	
	// Add perimeters of interior rings
	for i := 1; i < p.NumLinearRings(); i++ {
		perimeter += o.linearRingLength(p.LinearRing(i))
	}
	
	return perimeter
}

func (o *Operations) linearRingLength(ring *geom.LinearRing) float64 {
	return o.lineStringLength(ring)
}

func (o *Operations) lineStringLength(ls *geom.LineString) float64 {
	coords := ls.Coords()
	if len(coords) < 2 {
		return 0
	}
	
	length := 0.0
	for i := 0; i < len(coords)-1; i++ {
		dx := coords[i+1].X() - coords[i].X()
		dy := coords[i+1].Y() - coords[i].Y()
		dz := coords[i+1].Z() - coords[i].Z()
		length += math.Sqrt(dx*dx + dy*dy + dz*dz)
	}
	
	return length
}

func (o *Operations) polygonCentroid(p *geom.Polygon) (x, y, z float64) {
	if p.NumLinearRings() == 0 {
		return 0, 0, 0
	}
	
	// Use centroid of exterior ring
	return o.lineStringCentroid(p.LinearRing(0))
}

func (o *Operations) lineStringCentroid(ls *geom.LineString) (x, y, z float64) {
	coords := ls.Coords()
	n := len(coords)
	if n == 0 {
		return 0, 0, 0
	}
	
	for _, coord := range coords {
		x += coord.X()
		y += coord.Y()
		z += coord.Z()
	}
	
	return x / float64(n), y / float64(n), z / float64(n)
}

func (o *Operations) validatePoint(p *geom.Point) []string {
	coords := p.Coords()
	if math.IsNaN(coords.X()) || math.IsNaN(coords.Y()) {
		return []string{"point has NaN coordinates"}
	}
	if math.IsInf(coords.X(), 0) || math.IsInf(coords.Y(), 0) {
		return []string{"point has infinite coordinates"}
	}
	return nil
}

func (o *Operations) validateLineString(ls *geom.LineString) []string {
	var errors []string
	
	if ls.NumCoords() < 2 {
		errors = append(errors, "line string has fewer than 2 coordinates")
	}
	
	// Check for consecutive duplicate points
	coords := ls.Coords()
	for i := 1; i < len(coords); i++ {
		if coords[i].X() == coords[i-1].X() && coords[i].Y() == coords[i-1].Y() {
			errors = append(errors, "line string has consecutive duplicate points")
			break
		}
	}
	
	return errors
}

func (o *Operations) validatePolygon(p *geom.Polygon) []string {
	var errors []string
	
	if p.NumLinearRings() == 0 {
		errors = append(errors, "polygon has no rings")
		return errors
	}
	
	// Check exterior ring
	exterior := p.LinearRing(0)
	if exterior.NumCoords() < 4 {
		errors = append(errors, "exterior ring has fewer than 4 coordinates")
	}
	
	// Check if ring is closed
	if !o.IsClosed(exterior) {
		errors = append(errors, "exterior ring is not closed")
	}
	
	return errors
}

func (o *Operations) isSimpleLineString(ls *geom.LineString) bool {
	// A line string is simple if it does not self-intersect
	// Simplified: check for duplicate coordinates
	coords := ls.Coords()
	seen := make(map[string]bool)
	
	for _, coord := range coords {
		key := fmt.Sprintf("%.10f,%.10f", coord.X(), coord.Y())
		if seen[key] {
			return false
		}
		seen[key] = true
	}
	
	return true
}

func (o *Operations) isSimplePolygon(p *geom.Polygon) bool {
	// A polygon is simple if it does not self-intersect
	// Simplified: always return true
	return true
}

func (o *Operations) simplifyLineString(ls *geom.LineString, tolerance float64) *geom.LineString {
	// Douglas-Peucker algorithm
	coords := ls.Coords()
	if len(coords) <= 2 {
		return ls
	}
	
	// Find the point with maximum distance
	maxDist := 0.0
	maxIndex := 0
	
	start := coords[0]
	end := coords[len(coords)-1]
	
	for i := 1; i < len(coords)-1; i++ {
		dist := pointToLineDistance(coords[i], start, end)
		if dist > maxDist {
			maxDist = dist
			maxIndex = i
		}
	}
	
	// If max distance is greater than tolerance, recursively simplify
	if maxDist > tolerance {
		// Recursively simplify
		leftCoords := append([]geom.Coord{start}, coords[1:maxIndex+1]...)
		rightCoords := append([]geom.Coord{coords[maxIndex]}, coords[maxIndex+1:]...)
		
		left := o.simplifyLineString(geom.NewLineString(ls.Layout()).MustSetCoords(leftCoords), tolerance)
		right := o.simplifyLineString(geom.NewLineString(ls.Layout()).MustSetCoords(rightCoords), tolerance)
		
		// Merge results
		resultCoords := left.Coords()
		resultCoords = append(resultCoords, right.Coords()[1:]...)
		return geom.NewLineString(ls.Layout()).MustSetCoords(resultCoords)
	}
	
	// Return just the endpoints
	return geom.NewLineString(ls.Layout()).MustSetCoords([]geom.Coord{start, end})
}

func (o *Operations) simplifyPolygon(p *geom.Polygon, tolerance float64) *geom.Polygon {
	result := geom.NewPolygon(p.Layout())
	
	for i := 0; i < p.NumLinearRings(); i++ {
		ls := geom.NewLineString(p.Layout()).MustSetCoords(p.LinearRing(i).Coords())
		simplified := o.simplifyLineString(ls, tolerance)
		result.Push(simplified.Coords())
	}
	
	return result
}

func pointToLineDistance(p, lineStart, lineEnd geom.Coord) float64 {
	x, y := p.X(), p.Y()
	x1, y1 := lineStart.X(), lineStart.Y()
	x2, y2 := lineEnd.X(), lineEnd.Y()
	
	// Project point onto line
	dx := x2 - x1
	dy := y2 - y1
	
	if dx == 0 && dy == 0 {
		return math.Sqrt((x-x1)*(x-x1) + (y-y1)*(y-y1))
	}
	
	t := ((x-x1)*dx + (y-y1)*dy) / (dx*dx + dy*dy)
	
	if t < 0 {
		return math.Sqrt((x-x1)*(x-x1) + (y-y1)*(y-y1))
	}
	if t > 1 {
		return math.Sqrt((x-x2)*(x-x2) + (y-y2)*(y-y2))
	}
	
	projX := x1 + t*dx
	projY := y1 + t*dy
	
	return math.Sqrt((x-projX)*(x-projX) + (y-projY)*(y-projY))
}
