package storage

import (
	"context"
	"database/sql"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/archplatform/geometry-service/pkg/models"
	"github.com/lib/pq"
	"github.com/twpayne/go-geom"
	"github.com/twpayne/go-geom/encoding/ewkb"
)

// PostGISStorage implements geometry storage using PostGIS
type PostGISStorage struct {
	db     *sql.DB
	srid   int
}

// NewPostGISStorage creates a new PostGIS storage
func NewPostGISStorage(db *sql.DB, srid int) (*PostGISStorage, error) {
	storage := &PostGISStorage{
		db:   db,
		srid: srid,
	}
	
	// Verify PostGIS extension
	if err := storage.verifyPostGIS(); err != nil {
		return nil, fmt.Errorf("postgis verification failed: %w", err)
	}
	
	return storage, nil
}

// verifyPostGIS checks if PostGIS extension is available
func (s *PostGISStorage) verifyPostGIS() error {
	var version string
	err := s.db.QueryRow("SELECT PostGIS_Version()").Scan(&version)
	if err != nil {
		return fmt.Errorf("postgis not available: %w", err)
	}
	return nil
}

// CreateGeometry creates a new geometry in the database
func (s *PostGISStorage) CreateGeometry(ctx context.Context, g *models.Geometry) error {
	g.BeforeInsert()
	
	// Convert geometry to WKB
	var geom2DWKB, geom3DWKB, bboxWKB []byte
	var err error
	
	if g.Geom2D != nil {
		geom2DWKB, err = ewkb.Marshal(g.Geom2D, binary.LittleEndian)
		if err != nil {
			return fmt.Errorf("failed to marshal 2d geometry: %w", err)
		}
	}
	
	if g.Geom3D != nil {
		geom3DWKB, err = ewkb.Marshal(g.Geom3D, binary.LittleEndian)
		if err != nil {
			return fmt.Errorf("failed to marshal 3d geometry: %w", err)
		}
	}
	
	if g.BBox != nil {
		bbox := s.createBBoxGeometry(g.BBox)
		bboxWKB, err = ewkb.Marshal(bbox, binary.LittleEndian)
		if err != nil {
			return fmt.Errorf("failed to marshal bbox: %w", err)
		}
	}
	
	query := `
		INSERT INTO geometry.geometries (
			id, element_id, design_id, project_id, tenant_id, geometry_type,
			geom_2d, geom_3d, bbox, area, length, perimeter, vertex_count,
			properties, metadata, srid, precision_mm, version, created_at, updated_at,
			created_by, updated_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22)
	`
	
	_, err = s.db.ExecContext(ctx, query,
		g.ID, g.ElementID, g.DesignID, g.ProjectID, g.TenantID, g.Type,
		geom2DWKB, geom3DWKB, bboxWKB, g.Area, g.Length, g.Perimeter, g.VertexCount,
		g.Properties, g.Metadata, g.SRID, g.Precision, g.Version,
		g.CreatedAt, g.UpdatedAt, g.CreatedBy, g.UpdatedBy,
	)
	
	return err
}

// GetGeometry retrieves a geometry by ID
func (s *PostGISStorage) GetGeometry(ctx context.Context, id string) (*models.Geometry, error) {
	query := `
		SELECT 
			id, element_id, design_id, project_id, tenant_id, geometry_type,
			ST_AsBinary(geom_2d), ST_AsBinary(geom_3d), ST_AsBinary(bbox),
			area, length, perimeter, vertex_count,
			properties, metadata, srid, precision_mm, version,
			created_at, updated_at, created_by, updated_by
		FROM geometry.geometries
		WHERE id = $1
	`
	
	g := &models.Geometry{}
	var geom2DWKB, geom3DWKB, bboxWKB sql.NullString
	
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&g.ID, &g.ElementID, &g.DesignID, &g.ProjectID, &g.TenantID, &g.Type,
		&geom2DWKB, &geom3DWKB, &bboxWKB,
		&g.Area, &g.Length, &g.Perimeter, &g.VertexCount,
		&g.Properties, &g.Metadata, &g.SRID, &g.Precision, &g.Version,
		&g.CreatedAt, &g.UpdatedAt, &g.CreatedBy, &g.UpdatedBy,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	
	// Parse geometries
	if geom2DWKB.Valid && geom2DWKB.String != "" {
		g.Geom2D, err = ewkb.Unmarshal([]byte(geom2DWKB.String))
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal 2d geometry: %w", err)
		}
	}
	
	if geom3DWKB.Valid && geom3DWKB.String != "" {
		g.Geom3D, err = ewkb.Unmarshal([]byte(geom3DWKB.String))
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal 3d geometry: %w", err)
		}
	}
	
	return g, nil
}

// UpdateGeometry updates a geometry
func (s *PostGISStorage) UpdateGeometry(ctx context.Context, g *models.Geometry) error {
	g.BeforeUpdate()
	
	var geom2DWKB, geom3DWKB, bboxWKB []byte
	var err error
	
	if g.Geom2D != nil {
		geom2DWKB, err = ewkb.Marshal(g.Geom2D, binary.LittleEndian)
		if err != nil {
			return fmt.Errorf("failed to marshal 2d geometry: %w", err)
		}
	}
	
	if g.Geom3D != nil {
		geom3DWKB, err = ewkb.Marshal(g.Geom3D, binary.LittleEndian)
		if err != nil {
			return fmt.Errorf("failed to marshal 3d geometry: %w", err)
		}
	}
	
	if g.BBox != nil {
		bbox := s.createBBoxGeometry(g.BBox)
		bboxWKB, err = ewkb.Marshal(bbox, binary.LittleEndian)
		if err != nil {
			return fmt.Errorf("failed to marshal bbox: %w", err)
		}
	}
	
	query := `
		UPDATE geometry.geometries SET
			geom_2d = $1,
			geom_3d = $2,
			bbox = $3,
			area = $4,
			length = $5,
			perimeter = $6,
			vertex_count = $7,
			properties = $8,
			metadata = $9,
			precision_mm = $10,
			version = $11,
			updated_at = $12,
			updated_by = $13
		WHERE id = $14
	`
	
	_, err = s.db.ExecContext(ctx, query,
		geom2DWKB, geom3DWKB, bboxWKB,
		g.Area, g.Length, g.Perimeter, g.VertexCount,
		g.Properties, g.Metadata, g.Precision,
		g.Version, g.UpdatedAt, g.UpdatedBy,
		g.ID,
	)
	
	return err
}

// DeleteGeometry deletes a geometry
func (s *PostGISStorage) DeleteGeometry(ctx context.Context, id string) error {
	query := `DELETE FROM geometry.geometries WHERE id = $1`
	_, err := s.db.ExecContext(ctx, query, id)
	return err
}

// QueryByBoundingBox queries geometries within a bounding box
func (s *PostGISStorage) QueryByBoundingBox(ctx context.Context, tenantID, designID string, bbox *models.BoundingBox, limit int) ([]*models.Geometry, error) {
	bboxGeom := s.createBBoxGeometry(bbox)
	bboxWKB, err := ewkb.Marshal(bboxGeom, binary.LittleEndian)
	if err != nil {
		return nil, err
	}
	
	query := `
		SELECT 
			id, element_id, design_id, project_id, tenant_id, geometry_type,
			ST_AsBinary(geom_2d), ST_AsBinary(geom_3d), ST_AsBinary(bbox),
			area, length, perimeter, vertex_count,
			properties, metadata, srid, precision_mm, version,
			created_at, updated_at, created_by, updated_by
		FROM geometry.geometries
		WHERE tenant_id = $1 
		  AND ($2 = '' OR design_id = $2)
		  AND bbox && ST_GeomFromEWKB($3)
		LIMIT $4
	`
	
	if limit <= 0 {
		limit = 1000
	}
	
	rows, err := s.db.QueryContext(ctx, query, tenantID, designID, bboxWKB, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	return s.scanGeometries(rows)
}

// QueryByRadius queries geometries within a radius of a point
func (s *PostGISStorage) QueryByRadius(ctx context.Context, tenantID, projectID string, centerX, centerY, radius float64, limit int) ([]*models.Geometry, error) {
	query := `
		SELECT 
			id, element_id, design_id, project_id, tenant_id, geometry_type,
			ST_AsBinary(geom_2d), ST_AsBinary(geom_3d), ST_AsBinary(bbox),
			area, length, perimeter, vertex_count,
			properties, metadata, srid, precision_mm, version,
			created_at, updated_at, created_by, updated_by
		FROM geometry.geometries
		WHERE tenant_id = $1 
		  AND ($2 = '' OR project_id = $2)
		  AND ST_DWithin(
			  geom_2d::geography,
			  ST_SetSRID(ST_MakePoint($3, $4), $6)::geography,
			  $5
		  )
		ORDER BY ST_Distance(
			geom_2d::geography,
			ST_SetSRID(ST_MakePoint($3, $4), $6)::geography
		)
		LIMIT $7
	`
	
	if limit <= 0 {
		limit = 100
	}
	
	rows, err := s.db.QueryContext(ctx, query, tenantID, projectID, centerX, centerY, radius, s.srid, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	return s.scanGeometries(rows)
}

// QueryNearest finds the nearest geometries to a point
func (s *PostGISStorage) QueryNearest(ctx context.Context, tenantID, projectID string, centerX, centerY float64, limit int, maxDistance float64) ([]*models.Geometry, error) {
	query := `
		SELECT 
			id, element_id, design_id, project_id, tenant_id, geometry_type,
			ST_AsBinary(geom_2d), ST_AsBinary(geom_3d), ST_AsBinary(bbox),
			area, length, perimeter, vertex_count,
			properties, metadata, srid, precision_mm, version,
			created_at, updated_at, created_by, updated_by
		FROM geometry.geometries
		WHERE tenant_id = $1 
		  AND ($2 = '' OR project_id = $2)
		  AND ST_DWithin(
			  geom_2d::geography,
			  ST_SetSRID(ST_MakePoint($3, $4), $7)::geography,
			  $6
		  )
		ORDER BY ST_Distance(
			geom_2d::geography,
			ST_SetSRID(ST_MakePoint($3, $4), $7)::geography
		)
		LIMIT $5
	`
	
	if limit <= 0 {
		limit = 10
	}
	
	rows, err := s.db.QueryContext(ctx, query, tenantID, projectID, centerX, centerY, limit, maxDistance, s.srid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	return s.scanGeometries(rows)
}

// QueryIntersecting queries geometries that intersect with a reference geometry
func (s *PostGISStorage) QueryIntersecting(ctx context.Context, tenantID, projectID string, refGeometry geom.T, limit int) ([]*models.Geometry, error) {
	refWKB, err := ewkb.Marshal(refGeometry, binary.LittleEndian)
	if err != nil {
		return nil, err
	}
	
	query := `
		SELECT 
			id, element_id, design_id, project_id, tenant_id, geometry_type,
			ST_AsBinary(geom_2d), ST_AsBinary(geom_3d), ST_AsBinary(bbox),
			area, length, perimeter, vertex_count,
			properties, metadata, srid, precision_mm, version,
			created_at, updated_at, created_by, updated_by
		FROM geometry.geometries
		WHERE tenant_id = $1 
		  AND ($2 = '' OR project_id = $2)
		  AND ST_Intersects(geom_2d, ST_GeomFromEWKB($3))
		LIMIT $4
	`
	
	if limit <= 0 {
		limit = 1000
	}
	
	rows, err := s.db.QueryContext(ctx, query, tenantID, projectID, refWKB, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	return s.scanGeometries(rows)
}

// CalculateArea calculates the area of a geometry
func (s *PostGISStorage) CalculateArea(ctx context.Context, id string) (float64, error) {
	var area float64
	query := `SELECT ST_Area(geom_2d::geography) FROM geometry.geometries WHERE id = $1`
	err := s.db.QueryRowContext(ctx, query, id).Scan(&area)
	return area, err
}

// CalculateLength calculates the length of a geometry
func (s *PostGISStorage) CalculateLength(ctx context.Context, id string) (float64, error) {
	var length float64
	query := `SELECT ST_Length(geom_2d::geography) FROM geometry.geometries WHERE id = $1`
	err := s.db.QueryRowContext(ctx, query, id).Scan(&length)
	return length, err
}

// CalculateDistance calculates the distance between two geometries
func (s *PostGISStorage) CalculateDistance(ctx context.Context, id1, id2 string) (float64, error) {
	var distance float64
	query := `
		SELECT ST_Distance(
			(SELECT geom_2d::geography FROM geometry.geometries WHERE id = $1),
			(SELECT geom_2d::geography FROM geometry.geometries WHERE id = $2)
		)
	`
	err := s.db.QueryRowContext(ctx, query, id1, id2).Scan(&distance)
	return distance, err
}

// CalculateCentroid calculates the centroid of a geometry
func (s *PostGISStorage) CalculateCentroid(ctx context.Context, id string) (float64, float64, error) {
	var x, y float64
	query := `SELECT ST_X(centroid), ST_Y(centroid) FROM (SELECT ST_Centroid(geom_2d) as centroid FROM geometry.geometries WHERE id = $1) t`
	err := s.db.QueryRowContext(ctx, query, id).Scan(&x, &y)
	return x, y, err
}

// ValidateGeometry checks if a geometry is valid
func (s *PostGISStorage) ValidateGeometry(ctx context.Context, id string) (bool, string, error) {
	var isValid bool
	var reason string
	query := `SELECT ST_IsValid(geom_2d), ST_IsValidReason(geom_2d) FROM geometry.geometries WHERE id = $1`
	err := s.db.QueryRowContext(ctx, query, id).Scan(&isValid, &reason)
	return isValid, reason, err
}

// RepairGeometry attempts to repair an invalid geometry
func (s *PostGISStorage) RepairGeometry(ctx context.Context, id string) error {
	query := `
		UPDATE geometry.geometries 
		SET geom_2d = ST_MakeValid(geom_2d),
		    updated_at = $2
		WHERE id = $1
	`
	_, err := s.db.ExecContext(ctx, query, id, time.Now())
	return err
}

// SimplifyGeometry simplifies a geometry using Douglas-Peucker algorithm
func (s *PostGISStorage) SimplifyGeometry(ctx context.Context, id string, tolerance float64) error {
	// Convert tolerance from meters to degrees (approximate)
	toleranceDegrees := tolerance / 111000.0
	
	query := `
		UPDATE geometry.geometries 
		SET geom_simplified = ST_SimplifyPreserveTopology(geom_2d, $2),
		    updated_at = $3
		WHERE id = $1
	`
	_, err := s.db.ExecContext(ctx, query, id, toleranceDegrees, time.Now())
	return err
}

// UpdateSpatialIndex updates the spatial index for a geometry
func (s *PostGISStorage) UpdateSpatialIndex(ctx context.Context, geometryID string) error {
	query := `SELECT geometry.update_spatial_index($1)`
	_, err := s.db.ExecContext(ctx, query, geometryID)
	return err
}

// BatchCreate creates multiple geometries in a batch
func (s *PostGISStorage) BatchCreate(ctx context.Context, geometries []*models.Geometry) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	
	for _, g := range geometries {
		if err := s.CreateGeometry(ctx, g); err != nil {
			return err
		}
	}
	
	return tx.Commit()
}

// scanGeometries scans multiple geometry rows
func (s *PostGISStorage) scanGeometries(rows *sql.Rows) ([]*models.Geometry, error) {
	var geometries []*models.Geometry
	
	for rows.Next() {
		g := &models.Geometry{}
		var geom2DWKB, geom3DWKB, bboxWKB sql.NullString
		
		err := rows.Scan(
			&g.ID, &g.ElementID, &g.DesignID, &g.ProjectID, &g.TenantID, &g.Type,
			&geom2DWKB, &geom3DWKB, &bboxWKB,
			&g.Area, &g.Length, &g.Perimeter, &g.VertexCount,
			&g.Properties, &g.Metadata, &g.SRID, &g.Precision, &g.Version,
			&g.CreatedAt, &g.UpdatedAt, &g.CreatedBy, &g.UpdatedBy,
		)
		if err != nil {
			return nil, err
		}
		
		// Parse geometries
		if geom2DWKB.Valid && geom2DWKB.String != "" {
			g.Geom2D, err = ewkb.Unmarshal([]byte(geom2DWKB.String))
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal 2d geometry: %w", err)
			}
		}
		
		if geom3DWKB.Valid && geom3DWKB.String != "" {
			g.Geom3D, err = ewkb.Unmarshal([]byte(geom3DWKB.String))
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal 3d geometry: %w", err)
			}
		}
		
		geometries = append(geometries, g)
	}
	
	return geometries, rows.Err()
}

// createBBoxGeometry creates a polygon geometry from a bounding box
func (s *PostGISStorage) createBBoxGeometry(bbox *models.BoundingBox) geom.T {
	return geom.NewPolygon(geom.XY).MustSetCoords([][]geom.Coord{
		{
			{bbox.MinX, bbox.MinY},
			{bbox.MaxX, bbox.MinY},
			{bbox.MaxX, bbox.MaxY},
			{bbox.MinX, bbox.MaxY},
			{bbox.MinX, bbox.MinY},
		},
	})
}

// Close closes the database connection
func (s *PostGISStorage) Close() error {
	return s.db.Close()
}
