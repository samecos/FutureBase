package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/twpayne/go-geom"
)

// GeometryType represents the type of geometry
type GeometryType string

const (
	GeometryTypePoint           GeometryType = "point"
	GeometryTypeLine            GeometryType = "line"
	GeometryTypePolyline        GeometryType = "polyline"
	GeometryTypePolygon         GeometryType = "polygon"
	GeometryTypeCircle          GeometryType = "circle"
	GeometryTypeArc             GeometryType = "arc"
	GeometryTypeMesh            GeometryType = "mesh"
	GeometryTypeBrep            GeometryType = "brep"
	GeometryTypeNurbsSurface    GeometryType = "nurbs_surface"
	GeometryTypeCompound        GeometryType = "compound"
)

// JSONB is a custom type for JSONB database columns
type JSONB map[string]interface{}

// Value implements the driver.Valuer interface
func (j JSONB) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// Scan implements the sql.Scanner interface
func (j *JSONB) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}
	
	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return fmt.Errorf("cannot scan type %T into JSONB", value)
	}
	
	return json.Unmarshal(bytes, j)
}

// Geometry represents a geometric entity
type Geometry struct {
	ID           string       `db:"id" json:"id"`
	ElementID    string       `db:"element_id" json:"element_id"`
	DesignID     string       `db:"design_id" json:"design_id"`
	ProjectID    string       `db:"project_id" json:"project_id"`
	TenantID     string       `db:"tenant_id" json:"tenant_id"`
	Type         GeometryType `db:"geometry_type" json:"type"`
	
	// 2D Geometry (WGS84 - SRID 4326)
	Geom2D       geom.T       `db:"-" json:"geom_2d,omitempty"`
	Geom2DEWKB   []byte       `db:"geom_2d" json:"-"`
	
	// 3D Geometry (Local coordinate system)
	Geom3D       geom.T       `db:"-" json:"geom_3d,omitempty"`
	Geom3DEWKB   []byte       `db:"geom_3d" json:"-"`
	
	// Simplified geometry for fast rendering
	GeomSimplified []byte     `db:"geom_simplified" json:"-"`
	
	// Bounding box
	BBox         *BoundingBox `db:"-" json:"bbox,omitempty"`
	BBoxEWKB     []byte       `db:"bbox" json:"-"`
	
	// Computed properties
	Area         *float64     `db:"area" json:"area,omitempty"`
	Length       *float64     `db:"length" json:"length,omitempty"`
	Perimeter    *float64     `db:"perimeter" json:"perimeter,omitempty"`
	VertexCount  int          `db:"vertex_count" json:"vertex_count"`
	
	// Metadata
	Properties   JSONB        `db:"properties" json:"properties"`
	Metadata     JSONB        `db:"metadata" json:"metadata"`
	SRID         int          `db:"srid" json:"srid"`
	Precision    float64      `db:"precision_mm" json:"precision_mm"`
	
	// Versioning
	Version      int          `db:"version" json:"version"`
	CreatedAt    time.Time    `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time    `db:"updated_at" json:"updated_at"`
	CreatedBy    string       `db:"created_by" json:"created_by"`
	UpdatedBy    string       `db:"updated_by" json:"updated_by"`
}

// BoundingBox represents a 3D bounding box
type BoundingBox struct {
	MinX, MinY, MinZ float64
	MaxX, MaxY, MaxZ float64
}

// Center returns the center point of the bounding box
func (b *BoundingBox) Center() (x, y, z float64) {
	return (b.MinX + b.MaxX) / 2,
		(b.MinY + b.MaxY) / 2,
		(b.MinZ + b.MaxZ) / 2
}

// Width returns the width of the bounding box
func (b *BoundingBox) Width() float64 {
	return b.MaxX - b.MinX
}

// Height returns the height of the bounding box
func (b *BoundingBox) Height() float64 {
	return b.MaxY - b.MinY
}

// Depth returns the depth of the bounding box
func (b *BoundingBox) Depth() float64 {
	return b.MaxZ - b.MinZ
}

// Contains checks if a point is inside the bounding box
func (b *BoundingBox) Contains(x, y, z float64) bool {
	return x >= b.MinX && x <= b.MaxX &&
		y >= b.MinY && y <= b.MaxY &&
		z >= b.MinZ && z <= b.MaxZ
}

// GeometrySnapshot represents a snapshot of a geometry at a specific version
type GeometrySnapshot struct {
	ID           string    `db:"id" json:"id"`
	SnapshotID   string    `db:"snapshot_id" json:"snapshot_id"`
	GeometryID   string    `db:"geometry_id" json:"geometry_id"`
	ElementID    string    `db:"element_id" json:"element_id"`
	DesignID     string    `db:"design_id" json:"design_id"`
	Geom2D       []byte    `db:"geom_2d" json:"-"`
	Geom3D       []byte    `db:"geom_3d" json:"-"`
	BBox         []byte    `db:"bbox" json:"-"`
	Area         *float64  `db:"area" json:"area,omitempty"`
	Length       *float64  `db:"length" json:"length,omitempty"`
	VertexCount  int       `db:"vertex_count" json:"vertex_count"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
}

// SpatialIndex represents a spatial index entry for fast queries
type SpatialIndex struct {
	ID           string    `db:"id" json:"id"`
	GeometryID   string    `db:"geometry_id" json:"geometry_id"`
	ProjectID    string    `db:"project_id" json:"project_id"`
	DesignID     string    `db:"design_id" json:"design_id"`
	ElementType  string    `db:"element_type" json:"element_type"`
	
	// Grid coordinates for multi-level indexing
	GridX        int       `db:"grid_x" json:"grid_x"`
	GridY        int       `db:"grid_y" json:"grid_y"`
	GridLevel    int       `db:"grid_level" json:"grid_level"`
	
	// Bounding box for non-spatial queries
	MinX         float64   `db:"min_x" json:"min_x"`
	MinY         float64   `db:"min_y" json:"min_y"`
	MaxX         float64   `db:"max_x" json:"max_x"`
	MaxY         float64   `db:"max_y" json:"max_y"`
}

// SpatialRelation represents a pre-computed spatial relationship
type SpatialRelation struct {
	ID               string    `db:"id" json:"id"`
	SourceGeometryID string    `db:"source_geometry_id" json:"source_geometry_id"`
	TargetGeometryID string    `db:"target_geometry_id" json:"target_geometry_id"`
	RelationType     string    `db:"relation_type" json:"relation_type"`
	DistanceMM       *float64  `db:"distance_mm" json:"distance_mm,omitempty"`
	OverlapArea      *float64  `db:"overlap_area" json:"overlap_area,omitempty"`
	ComputedAt       time.Time `db:"computed_at" json:"computed_at"`
}

// Transform represents a 3D transformation
type Transform struct {
	// 4x4 transformation matrix
	Matrix [16]float64 `json:"matrix"`
	
	// Decomposed transform (optional)
	Translation *Vector3   `json:"translation,omitempty"`
	Rotation    *Rotation  `json:"rotation,omitempty"`
	Scale       *Vector3   `json:"scale,omitempty"`
}

// Vector3 represents a 3D vector
type Vector3 struct {
	X, Y, Z float64
}

// Rotation represents a 3D rotation
type Rotation struct {
	Quaternion *Quaternion `json:"quaternion,omitempty"`
	Euler      *EulerAngles `json:"euler,omitempty"`
	AxisAngle  *AxisAngle   `json:"axis_angle,omitempty"`
}

// Quaternion represents a quaternion rotation
type Quaternion struct {
	X, Y, Z, W float64
}

// EulerAngles represents rotation in Euler angles
type EulerAngles struct {
	Pitch float64 `json:"pitch"` // X axis
	Yaw   float64 `json:"yaw"`   // Y axis
	Roll  float64 `json:"roll"`  // Z axis
}

// AxisAngle represents rotation around an axis
type AxisAngle struct {
	Axis  Vector3 `json:"axis"`
	Angle float64 `json:"angle"` // radians
}

// BIMMetadata stores metadata from BIM files
type BIMMetadata struct {
	ID              string    `db:"id" json:"id"`
	FilePath        string    `db:"file_path" json:"file_path"`
	Format          string    `db:"format" json:"format"`
	SchemaVersion   string    `db:"schema_version" json:"schema_version"`
	ApplicationName string    `db:"application_name" json:"application_name"`
	ApplicationVersion string `db:"application_version" json:"application_version"`
	CreatedAt       time.Time `db:"created_at" json:"created_at"`
	Author          string    `db:"author" json:"author"`
	Organization    string    `db:"organization" json:"organization"`
	ElementCount    int       `db:"element_count" json:"element_count"`
	ElementTypes    []string  `db:"element_types" json:"element_types"`
	ProjectBounds   []byte    `db:"project_bounds" json:"-"`
}

// BIMElement represents an element extracted from a BIM file
type BIMElement struct {
	ID           string    `db:"id" json:"id"`
	BIMMetadataID string   `db:"bim_metadata_id" json:"bim_metadata_id"`
	ElementID    string    `db:"element_id" json:"element_id"`
	ElementType  string    `db:"element_type" json:"element_type"`
	ElementSubtype string  `db:"element_subtype" json:"element_subtype"`
	Name         string    `db:"name" json:"name"`
	Description  string    `db:"description" json:"description"`
	Properties   JSONB     `db:"properties" json:"properties"`
	GeometryID   *string   `db:"geometry_id" json:"geometry_id,omitempty"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
}

// ImportJob tracks the status of an import operation
type ImportJob struct {
	ID             string    `db:"id" json:"id"`
	TenantID       string    `db:"tenant_id" json:"tenant_id"`
	ProjectID      string    `db:"project_id" json:"project_id"`
	DesignID       string    `db:"design_id" json:"design_id"`
	FilePath       string    `db:"file_path" json:"file_path"`
	Format         string    `db:"format" json:"format"`
	Status         string    `db:"status" json:"status"` // pending, processing, completed, failed
	ProgressPercent float64  `db:"progress_percent" json:"progress_percent"`
	TotalCount     int       `db:"total_count" json:"total_count"`
	ProcessedCount int       `db:"processed_count" json:"processed_count"`
	ErrorCount     int       `db:"error_count" json:"error_count"`
	ErrorMessage   *string   `db:"error_message" json:"error_message,omitempty"`
	StartedAt      *time.Time `db:"started_at" json:"started_at,omitempty"`
	CompletedAt    *time.Time `db:"completed_at" json:"completed_at,omitempty"`
	CreatedAt      time.Time `db:"created_at" json:"created_at"`
	CreatedBy      string    `db:"created_by" json:"created_by"`
}

// CreateGeometry creates a new geometry with generated ID
func (g *Geometry) BeforeInsert() error {
	if g.ID == "" {
		g.ID = uuid.New().String()
	}
	if g.SRID == 0 {
		g.SRID = 4326 // Default WGS84
	}
	if g.Precision == 0 {
		g.Precision = 1.0 // Default 1mm precision
	}
	if g.Version == 0 {
		g.Version = 1
	}
	g.CreatedAt = time.Now()
	g.UpdatedAt = time.Now()
	return nil
}

// BeforeUpdate updates timestamps
func (g *Geometry) BeforeUpdate() error {
	g.UpdatedAt = time.Now()
	g.Version++
	return nil
}

// TableName returns the table name for Geometry
func (Geometry) TableName() string {
	return "geometry.geometries"
}

// TableName returns the table name for SpatialIndex
func (SpatialIndex) TableName() string {
	return "geometry.spatial_index"
}

// TableName returns the table name for SpatialRelation
func (SpatialRelation) TableName() string {
	return "geometry.spatial_relations"
}

// TableName returns the table name for BIMMetadata
func (BIMMetadata) TableName() string {
	return "geometry.bim_metadata"
}

// TableName returns the table name for BIMElement
func (BIMElement) TableName() string {
	return "geometry.bim_elements"
}

// TableName returns the table name for ImportJob
func (ImportJob) TableName() string {
	return "geometry.import_jobs"
}

// ToBounds returns the bounding box as a flat array [minx, miny, minz, maxx, maxy, maxz]
func (b *BoundingBox) ToBounds() []float64 {
	return []float64{b.MinX, b.MinY, b.MinZ, b.MaxX, b.MaxY, b.MaxZ}
}

// FromBounds creates a BoundingBox from a flat array
func BoundingBoxFromBounds(bounds []float64) *BoundingBox {
	if len(bounds) < 6 {
		return nil
	}
	return &BoundingBox{
		MinX: bounds[0], MinY: bounds[1], MinZ: bounds[2],
		MaxX: bounds[3], MaxY: bounds[4], MaxZ: bounds[5],
	}
}
