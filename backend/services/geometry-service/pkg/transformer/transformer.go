package transformer

import (
	"math"

	"github.com/archplatform/geometry-service/pkg/models"
	"github.com/twpayne/go-geom"
)

// Transformer handles geometric transformations
type Transformer struct {
	tolerance float64
}

// NewTransformer creates a new transformer
func NewTransformer(tolerance float64) *Transformer {
	return &Transformer{
		tolerance: tolerance,
	}
}

// Translate translates a geometry by the given offset
func (t *Transformer) Translate(g geom.T, dx, dy, dz float64) (geom.T, error) {
	// Create translation matrix
	matrix := IdentityMatrix()
	matrix[12] = dx
	matrix[13] = dy
	matrix[14] = dz
	
	return t.applyMatrix(g, matrix)
}

// Rotate rotates a geometry around a center point
func (t *Transformer) Rotate(g geom.T, center models.Vector3, axis models.Vector3, angle float64) (geom.T, error) {
	// Normalize axis
	length := math.Sqrt(axis.X*axis.X + axis.Y*axis.Y + axis.Z*axis.Z)
	if length == 0 {
		return g, nil
	}
	nx, ny, nz := axis.X/length, axis.Y/length, axis.Z/length
	
	// Compute rotation matrix (Rodrigues' rotation formula)
	c := math.Cos(angle)
	s := math.Sin(angle)
	t_val := 1 - c
	
	matrix := [16]float64{
		t_val*nx*nx + c,    t_val*nx*ny - s*nz, t_val*nx*nz + s*ny, 0,
		t_val*nx*ny + s*nz, t_val*ny*ny + c,    t_val*ny*nz - s*nx, 0,
		t_val*nx*nz - s*ny, t_val*ny*nz + s*nx, t_val*nz*nz + c,    0,
		0,                  0,                  0,                  1,
	}
	
	// Translate to origin, rotate, then translate back
	translated, err := t.Translate(g, -center.X, -center.Y, -center.Z)
	if err != nil {
		return nil, err
	}
	
	rotated, err := t.applyMatrix(translated, matrix)
	if err != nil {
		return nil, err
	}
	
	return t.Translate(rotated, center.X, center.Y, center.Z)
}

// Scale scales a geometry around a center point
func (t *Transformer) Scale(g geom.T, center models.Vector3, sx, sy, sz float64) (geom.T, error) {
	// Translate to origin
	translated, err := t.Translate(g, -center.X, -center.Y, -center.Z)
	if err != nil {
		return nil, err
	}
	
	// Apply scaling
	matrix := [16]float64{
		sx, 0,  0,  0,
		0,  sy, 0,  0,
		0,  0,  sz, 0,
		0,  0,  0,  1,
	}
	
	scaled, err := t.applyMatrix(translated, matrix)
	if err != nil {
		return nil, err
	}
	
	// Translate back
	return t.Translate(scaled, center.X, center.Y, center.Z)
}

// Mirror mirrors a geometry across a plane
func (t *Transformer) Mirror(g geom.T, point models.Vector3, normal models.Vector3) (geom.T, error) {
	// Normalize normal
	length := math.Sqrt(normal.X*normal.X + normal.Y*normal.Y + normal.Z*normal.Z)
	if length == 0 {
		return g, nil
	}
	nx, ny, nz := normal.X/length, normal.Y/length, normal.Z/length
	
	// Reflection matrix
	matrix := [16]float64{
		1-2*nx*nx,   -2*nx*ny,    -2*nx*nz,    0,
		-2*nx*ny,    1-2*ny*ny,   -2*ny*nz,    0,
		-2*nx*nz,    -2*ny*nz,    1-2*nz*nz,   0,
		0,           0,           0,           1,
	}
	
	// Translate to origin
	translated, err := t.Translate(g, -point.X, -point.Y, -point.Z)
	if err != nil {
		return nil, err
	}
	
	// Apply reflection
	reflected, err := t.applyMatrix(translated, matrix)
	if err != nil {
		return nil, err
	}
	
	// Translate back
	return t.Translate(reflected, point.X, point.Y, point.Z)
}

// ApplyTransform applies a full transform (translation, rotation, scale)
func (t *Transformer) ApplyTransform(g geom.T, transform *models.Transform) (geom.T, error) {
	if transform.Matrix != [16]float64{} {
		return t.applyMatrix(g, transform.Matrix)
	}
	
	result := g
	var err error
	
	// Apply translation
	if transform.Translation != nil {
		result, err = t.Translate(result, transform.Translation.X, transform.Translation.Y, transform.Translation.Z)
		if err != nil {
			return nil, err
		}
	}
	
	// Apply rotation
	if transform.Rotation != nil {
		// Convert rotation to axis-angle if needed
		if transform.Rotation.AxisAngle != nil {
			center := models.Vector3{X: 0, Y: 0, Z: 0}
			result, err = t.Rotate(result, center, transform.Rotation.AxisAngle.Axis, transform.Rotation.AxisAngle.Angle)
			if err != nil {
				return nil, err
			}
		}
		// TODO: Handle quaternion and euler rotations
	}
	
	// Apply scale
	if transform.Scale != nil {
		center := models.Vector3{X: 0, Y: 0, Z: 0}
		result, err = t.Scale(result, center, transform.Scale.X, transform.Scale.Y, transform.Scale.Z)
		if err != nil {
			return nil, err
		}
	}
	
	return result, nil
}

// applyMatrix applies a 4x4 transformation matrix to a geometry
func (t *Transformer) applyMatrix(g geom.T, matrix [16]float64) (geom.T, error) {
	switch geom := g.(type) {
	case *geom.Point:
		return t.transformPoint(geom, matrix), nil
	case *geom.LineString:
		return t.transformLineString(geom, matrix), nil
	case *geom.Polygon:
		return t.transformPolygon(geom, matrix), nil
	case *geom.MultiPoint:
		return t.transformMultiPoint(geom, matrix), nil
	case *geom.MultiLineString:
		return t.transformMultiLineString(geom, matrix), nil
	case *geom.MultiPolygon:
		return t.transformMultiPolygon(geom, matrix), nil
	case *geom.GeometryCollection:
		return t.transformGeometryCollection(geom, matrix), nil
	default:
		return g, nil
	}
}

// transformPoint transforms a point
func (t *Transformer) transformPoint(p *geom.Point, matrix [16]float64) *geom.Point {
	coords := p.Coords()
	x, y, z := coords.X(), coords.Y(), coords.Z()
	
	x2 := matrix[0]*x + matrix[4]*y + matrix[8]*z + matrix[12]
	y2 := matrix[1]*x + matrix[5]*y + matrix[9]*z + matrix[13]
	z2 := matrix[2]*x + matrix[6]*y + matrix[10]*z + matrix[14]
	w := matrix[3]*x + matrix[7]*y + matrix[11]*z + matrix[15]
	
	if w != 1 && w != 0 {
		x2 /= w
		y2 /= w
		z2 /= w
	}
	
	return geom.NewPoint(geom.XYZ).MustSetCoords(geom.Coord{x2, y2, z2})
}

// transformLineString transforms a line string
func (t *Transformer) transformLineString(ls *geom.LineString, matrix [16]float64) *geom.LineString {
	coords := ls.Coords()
	newCoords := make([]geom.Coord, len(coords))
	
	for i, coord := range coords {
		p := geom.NewPoint(geom.XYZ).MustSetCoords(coord)
		transformed := t.transformPoint(p, matrix)
		newCoords[i] = transformed.Coords()
	}
	
	return geom.NewLineString(geom.XYZ).MustSetCoords(newCoords)
}

// transformPolygon transforms a polygon
func (t *Transformer) transformPolygon(p *geom.Polygon, matrix [16]float64) *geom.Polygon {
	coords := p.Coords()
	newCoords := make([][]geom.Coord, len(coords))
	
	for i, ring := range coords {
		newRing := make([]geom.Coord, len(ring))
		for j, coord := range ring {
			pt := geom.NewPoint(geom.XYZ).MustSetCoords(coord)
			transformed := t.transformPoint(pt, matrix)
			newRing[j] = transformed.Coords()
		}
		newCoords[i] = newRing
	}
	
	return geom.NewPolygon(geom.XYZ).MustSetCoords(newCoords)
}

// transformMultiPoint transforms a multi-point
func (t *Transformer) transformMultiPoint(mp *geom.MultiPoint, matrix [16]float64) *geom.MultiPoint {
	coords := mp.Coords()
	newCoords := make([]geom.Coord, len(coords))
	
	for i, coord := range coords {
		p := geom.NewPoint(geom.XYZ).MustSetCoords(coord)
		transformed := t.transformPoint(p, matrix)
		newCoords[i] = transformed.Coords()
	}
	
	return geom.NewMultiPoint(geom.XYZ).MustSetCoords(newCoords)
}

// transformMultiLineString transforms a multi-line string
func (t *Transformer) transformMultiLineString(mls *geom.MultiLineString, matrix [16]float64) *geom.MultiLineString {
	coords := mls.Coords()
	newCoords := make([][]geom.Coord, len(coords))
	
	for i, line := range coords {
		newLine := make([]geom.Coord, len(line))
		for j, coord := range line {
			p := geom.NewPoint(geom.XYZ).MustSetCoords(coord)
			transformed := t.transformPoint(p, matrix)
			newLine[j] = transformed.Coords()
		}
		newCoords[i] = newLine
	}
	
	return geom.NewMultiLineString(geom.XYZ).MustSetCoords(newCoords)
}

// transformMultiPolygon transforms a multi-polygon
func (t *Transformer) transformMultiPolygon(mp *geom.MultiPolygon, matrix [16]float64) *geom.MultiPolygon {
	coords := mp.Coords()
	newCoords := make([][][]geom.Coord, len(coords))
	
	for i, polygon := range coords {
		newPolygon := make([][]geom.Coord, len(polygon))
		for j, ring := range polygon {
			newRing := make([]geom.Coord, len(ring))
			for k, coord := range ring {
				p := geom.NewPoint(geom.XYZ).MustSetCoords(coord)
				transformed := t.transformPoint(p, matrix)
				newRing[k] = transformed.Coords()
			}
			newPolygon[j] = newRing
		}
		newCoords[i] = newPolygon
	}
	
	return geom.NewMultiPolygon(geom.XYZ).MustSetCoords(newCoords)
}

// transformGeometryCollection transforms a geometry collection
func (t *Transformer) transformGeometryCollection(gc *geom.GeometryCollection, matrix [16]float64) *geom.GeometryCollection {
	geometries := gc.Geoms()
	newGeometries := make([]geom.T, len(geometries))
	
	for i, g := range geometries {
		transformed, _ := t.applyMatrix(g, matrix)
		newGeometries[i] = transformed
	}
	
	result := geom.NewGeometryCollection()
	result.MustPush(newGeometries...)
	return result
}

// IdentityMatrix returns a 4x4 identity matrix
func IdentityMatrix() [16]float64 {
	return [16]float64{
		1, 0, 0, 0,
		0, 1, 0, 0,
		0, 0, 1, 0,
		0, 0, 0, 1,
	}
}

// ComposeMatrices multiplies two 4x4 matrices
func ComposeMatrices(a, b [16]float64) [16]float64 {
	var result [16]float64
	
	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			for k := 0; k < 4; k++ {
				result[i*4+j] += a[i*4+k] * b[k*4+j]
			}
		}
	}
	
	return result
}

// ComputeBoundingBox computes the bounding box of a geometry
func (t *Transformer) ComputeBoundingBox(g geom.T) *models.BoundingBox {
	flatCoords := g.FlatCoords()
	stride := g.Stride()
	
	if len(flatCoords) == 0 {
		return &models.BoundingBox{}
	}
	
	minX, minY, minZ := flatCoords[0], flatCoords[1], 0.0
	maxX, maxY, maxZ := minX, minY, minZ
	
	if stride > 2 {
		minZ = flatCoords[2]
		maxZ = minZ
	}
	
	for i := stride; i < len(flatCoords); i += stride {
		x := flatCoords[i]
		y := flatCoords[i+1]
		
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
		
		if stride > 2 {
			z := flatCoords[i+2]
			if z < minZ {
				minZ = z
			}
			if z > maxZ {
				maxZ = z
			}
		}
	}
	
	return &models.BoundingBox{
		MinX: minX, MinY: minY, MinZ: minZ,
		MaxX: maxX, MaxY: maxY, MaxZ: maxZ,
	}
}
