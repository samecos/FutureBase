-- ============================================
-- PostGIS Geometry Database Setup
-- ============================================

-- Connect to main database
\c archdesign_platform;

-- Enable PostGIS extensions
CREATE EXTENSION IF NOT EXISTS postgis;
CREATE EXTENSION IF NOT EXISTS postgis_topology;
CREATE EXTENSION IF NOT EXISTS postgis_raster;
CREATE EXTENSION IF NOT EXISTS fuzzystrmatch;
CREATE EXTENSION IF NOT EXISTS postgis_tiger_geocoder;

-- ============================================
-- Geometry Schema
-- ============================================

-- Geometries table
CREATE TABLE IF NOT EXISTS geometry.geometries (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    element_id          UUID NOT NULL,
    design_id           UUID NOT NULL,
    project_id          UUID NOT NULL,
    tenant_id           UUID NOT NULL,
    geometry_type       VARCHAR(50) NOT NULL
                        CHECK (geometry_type IN (
                            'point', 'line', 'polyline', 'polygon', 'circle', 'arc',
                            'mesh', 'brep', 'nurbs_surface', 'compound'
                        )),
    -- 2D Geometry (WGS84 - SRID 4326)
    geom_2d             GEOMETRY(GEOMETRY, 4326),
    -- 3D Geometry (Local coordinate system)
    geom_3d             GEOMETRY(GEOMETRYZ, 0),
    -- Simplified geometry for fast rendering
    geom_simplified     GEOMETRY(GEOMETRY, 4326),
    -- Bounding box
    bbox                GEOMETRY(POLYGON, 4326),
    -- Computed properties
    area                DECIMAL(18, 6),
    length              DECIMAL(18, 6),
    perimeter           DECIMAL(18, 6),
    vertex_count        INTEGER DEFAULT 0,
    -- Precision settings
    precision_mm        DECIMAL(10, 4) DEFAULT 1.0,
    -- Metadata
    properties          JSONB DEFAULT '{}',
    metadata            JSONB DEFAULT '{}',
    srid                INTEGER DEFAULT 4326,
    -- Versioning
    version             INTEGER NOT NULL DEFAULT 1,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by          UUID,
    updated_by          UUID
);

-- Geometry snapshots table (for version control)
CREATE TABLE IF NOT EXISTS geometry.geometry_snapshots (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    snapshot_id         UUID NOT NULL,
    geometry_id         UUID NOT NULL REFERENCES geometry.geometries(id),
    element_id          UUID NOT NULL,
    design_id           UUID NOT NULL,
    geom_2d             GEOMETRY(GEOMETRY, 4326),
    geom_3d             GEOMETRY(GEOMETRYZ, 0),
    bbox                GEOMETRY(POLYGON, 4326),
    area                DECIMAL(18, 6),
    length              DECIMAL(18, 6),
    vertex_count        INTEGER,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(snapshot_id, geometry_id)
);

-- Spatial index table (for multi-level grid indexing)
CREATE TABLE IF NOT EXISTS geometry.spatial_index (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    geometry_id         UUID NOT NULL REFERENCES geometry.geometries(id) ON DELETE CASCADE,
    project_id          UUID NOT NULL,
    design_id           UUID NOT NULL,
    element_type        VARCHAR(100),
    -- Grid coordinates
    grid_x              INTEGER,
    grid_y              INTEGER,
    grid_level          INTEGER DEFAULT 0,
    -- Bounding box coordinates
    min_x               DECIMAL(18, 6),
    min_y               DECIMAL(18, 6),
    max_x               DECIMAL(18, 6),
    max_y               DECIMAL(18, 6),

    UNIQUE(geometry_id, grid_level)
);

-- Spatial relations table (pre-computed relationships)
CREATE TABLE IF NOT EXISTS geometry.spatial_relations (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_geometry_id  UUID NOT NULL REFERENCES geometry.geometries(id) ON DELETE CASCADE,
    target_geometry_id  UUID NOT NULL REFERENCES geometry.geometries(id) ON DELETE CASCADE,
    relation_type       VARCHAR(50) NOT NULL
                        CHECK (relation_type IN (
                            'intersects', 'contains', 'within', 'touches',
                            'crosses', 'overlaps', 'equals', 'disjoint', 'distance'
                        )),
    distance_mm         DECIMAL(18, 6),
    overlap_area        DECIMAL(18, 6),
    computed_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(source_geometry_id, target_geometry_id, relation_type)
);

-- BIM metadata table
CREATE TABLE IF NOT EXISTS geometry.bim_metadata (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    file_path           VARCHAR(1000) NOT NULL,
    format              VARCHAR(50) NOT NULL,
    schema_version      VARCHAR(50),
    application_name    VARCHAR(255),
    application_version VARCHAR(100),
    created_at          TIMESTAMPTZ,
    author              VARCHAR(255),
    organization        VARCHAR(255),
    element_count       INTEGER DEFAULT 0,
    element_types       TEXT[],
    project_bounds      GEOMETRY(POLYGON, 4326),
    imported_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- BIM elements table
CREATE TABLE IF NOT EXISTS geometry.bim_elements (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bim_metadata_id     UUID NOT NULL REFERENCES geometry.bim_metadata(id) ON DELETE CASCADE,
    element_id          VARCHAR(255) NOT NULL,
    element_type        VARCHAR(100) NOT NULL,
    element_subtype     VARCHAR(100),
    name                VARCHAR(255),
    description         TEXT,
    properties          JSONB DEFAULT '{}',
    geometry_id         UUID REFERENCES geometry.geometries(id),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(bim_metadata_id, element_id)
);

-- Import jobs table
CREATE TABLE IF NOT EXISTS geometry.import_jobs (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL,
    project_id          UUID NOT NULL,
    design_id           UUID,
    file_path           VARCHAR(1000) NOT NULL,
    format              VARCHAR(50) NOT NULL,
    status              VARCHAR(50) NOT NULL DEFAULT 'pending'
                        CHECK (status IN ('pending', 'processing', 'completed', 'failed', 'cancelled')),
    progress_percent    DECIMAL(5, 2) DEFAULT 0,
    total_count         INTEGER DEFAULT 0,
    processed_count     INTEGER DEFAULT 0,
    error_count         INTEGER DEFAULT 0,
    error_message       TEXT,
    started_at          TIMESTAMPTZ,
    completed_at        TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by          UUID
);

-- ============================================
-- Indexes
-- ============================================

-- Geometries indexes
CREATE INDEX IF NOT EXISTS idx_geometries_element_id ON geometry.geometries(element_id);
CREATE INDEX IF NOT EXISTS idx_geometries_design_id ON geometry.geometries(design_id);
CREATE INDEX IF NOT EXISTS idx_geometries_project_id ON geometry.geometries(project_id);
CREATE INDEX IF NOT EXISTS idx_geometries_tenant_id ON geometry.geometries(tenant_id);
CREATE INDEX IF NOT EXISTS idx_geometries_type ON geometry.geometries(geometry_type);

-- PostGIS spatial indexes
CREATE INDEX IF NOT EXISTS idx_geometries_geom_2d ON geometry.geometries USING GIST(geom_2d);
CREATE INDEX IF NOT EXISTS idx_geometries_geom_3d ON geometry.geometries USING GIST(geom_3d);
CREATE INDEX IF NOT EXISTS idx_geometries_bbox ON geometry.geometries USING GIST(bbox);

-- BRIN index for time-series data
CREATE INDEX IF NOT EXISTS idx_geometries_created_brin ON geometry.geometries USING BRIN(created_at);

-- Snapshot indexes
CREATE INDEX IF NOT EXISTS idx_geometry_snapshots_snapshot ON geometry.geometry_snapshots(snapshot_id);
CREATE INDEX IF NOT EXISTS idx_geometry_snapshots_geometry ON geometry.geometry_snapshots(geometry_id);

-- Spatial index indexes
CREATE INDEX IF NOT EXISTS idx_spatial_index_project ON geometry.spatial_index(project_id);
CREATE INDEX IF NOT EXISTS idx_spatial_index_design ON geometry.spatial_index(design_id);
CREATE INDEX IF NOT EXISTS idx_spatial_index_grid ON geometry.spatial_index(grid_x, grid_y, grid_level);
CREATE INDEX IF NOT EXISTS idx_spatial_index_bounds ON geometry.spatial_index(min_x, min_y, max_x, max_y);

-- Spatial relations indexes
CREATE INDEX IF NOT EXISTS idx_spatial_relations_source ON geometry.spatial_relations(source_geometry_id);
CREATE INDEX IF NOT EXISTS idx_spatial_relations_target ON geometry.spatial_relations(target_geometry_id);

-- BIM indexes
CREATE INDEX IF NOT EXISTS idx_bim_metadata_format ON geometry.bim_metadata(format);
CREATE INDEX IF NOT EXISTS idx_bim_metadata_imported ON geometry.bim_metadata(imported_at);
CREATE INDEX IF NOT EXISTS idx_bim_elements_metadata ON geometry.bim_elements(bim_metadata_id);
CREATE INDEX IF NOT EXISTS idx_bim_elements_type ON geometry.bim_elements(element_type);

-- Import job indexes
CREATE INDEX IF NOT EXISTS idx_import_jobs_tenant ON geometry.import_jobs(tenant_id);
CREATE INDEX IF NOT EXISTS idx_import_jobs_project ON geometry.import_jobs(project_id);
CREATE INDEX IF NOT EXISTS idx_import_jobs_status ON geometry.import_jobs(status);

-- ============================================
-- Functions
-- ============================================

-- Update geometry stats function
CREATE OR REPLACE FUNCTION geometry.update_geometry_stats()
RETURNS TRIGGER AS $$
BEGIN
    -- Calculate area (only for polygon types)
    IF GeometryType(NEW.geom_2d) IN ('POLYGON', 'MULTIPOLYGON') THEN
        NEW.area := ST_Area(NEW.geom_2d::GEOGRAPHY)::DECIMAL(18, 6);
    END IF;

    -- Calculate length
    IF GeometryType(NEW.geom_2d) IN ('LINESTRING', 'MULTILINESTRING') THEN
        NEW.length := ST_Length(NEW.geom_2d::GEOGRAPHY)::DECIMAL(18, 6);
    END IF;

    -- Calculate perimeter for polygons
    IF GeometryType(NEW.geom_2d) IN ('POLYGON', 'MULTIPOLYGON') THEN
        NEW.perimeter := ST_Perimeter(NEW.geom_2d::GEOGRAPHY)::DECIMAL(18, 6);
    END IF;

    -- Calculate bounding box
    NEW.bbox := ST_Envelope(NEW.geom_2d);

    -- Calculate vertex count
    NEW.vertex_count := ST_NPoints(NEW.geom_2d);

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger for automatic stats calculation
DROP TRIGGER IF EXISTS trigger_geometries_stats ON geometry.geometries;
CREATE TRIGGER trigger_geometries_stats
    BEFORE INSERT OR UPDATE ON geometry.geometries
    FOR EACH ROW
    EXECUTE FUNCTION geometry.update_geometry_stats();

-- Update spatial index function
CREATE OR REPLACE FUNCTION geometry.update_spatial_index(p_geometry_id UUID)
RETURNS VOID AS $$
DECLARE
    v_rec RECORD;
    v_bbox GEOMETRY;
    v_grid_size INTEGER := 100;  -- Grid size in meters
BEGIN
    SELECT geom_2d, project_id, design_id INTO v_rec
    FROM geometry.geometries WHERE id = p_geometry_id;

    IF v_rec.geom_2d IS NULL THEN
        RETURN;
    END IF;

    v_bbox := ST_Envelope(v_rec.geom_2d);

    -- Delete old indexes
    DELETE FROM geometry.spatial_index WHERE geometry_id = p_geometry_id;

    -- Insert new indexes (multi-level grid)
    FOR i IN 0..3 LOOP
        INSERT INTO geometry.spatial_index (
            geometry_id, project_id, design_id,
            grid_x, grid_y, grid_level,
            min_x, min_y, max_x, max_y
        ) VALUES (
            p_geometry_id, v_rec.project_id, v_rec.design_id,
            FLOOR(ST_XMin(v_bbox) / (v_grid_size * (2^i)))::INTEGER,
            FLOOR(ST_YMin(v_bbox) / (v_grid_size * (2^i)))::INTEGER,
            i,
            ST_XMin(v_bbox), ST_YMin(v_bbox),
            ST_XMax(v_bbox), ST_YMax(v_bbox)
        );
    END LOOP;
END;
$$ LANGUAGE plpgsql;

-- Query by bounding box function
CREATE OR REPLACE FUNCTION geometry.query_by_bbox(
    p_min_x DECIMAL,
    p_min_y DECIMAL,
    p_max_x DECIMAL,
    p_max_y DECIMAL,
    p_project_id UUID DEFAULT NULL,
    p_design_id UUID DEFAULT NULL
)
RETURNS TABLE (
    geometry_id UUID,
    element_id UUID,
    geometry_type VARCHAR(50),
    geom_2d GEOMETRY,
    area DECIMAL,
    length DECIMAL
) AS $$
DECLARE
    v_bbox GEOMETRY;
BEGIN
    v_bbox := ST_MakeEnvelope(p_min_x, p_min_y, p_max_x, p_max_y, 4326);

    RETURN QUERY
    SELECT
        g.id, g.element_id, g.geometry_type, g.geom_2d, g.area, g.length
    FROM geometry.geometries g
    WHERE g.bbox && v_bbox  -- Bounding box intersection
      AND (p_project_id IS NULL OR g.project_id = p_project_id)
      AND (p_design_id IS NULL OR g.design_id = p_design_id)
      AND ST_Intersects(g.geom_2d, v_bbox)  -- Precise intersection test
    ORDER BY ST_Area(g.bbox) DESC;
END;
$$ LANGUAGE plpgsql;

-- Query by radius function
CREATE OR REPLACE FUNCTION geometry.query_by_radius(
    p_center_x DECIMAL,
    p_center_y DECIMAL,
    p_radius_meters DECIMAL,
    p_project_id UUID DEFAULT NULL
)
RETURNS TABLE (
    geometry_id UUID,
    element_id UUID,
    geometry_type VARCHAR(50),
    distance_meters DECIMAL,
    geom_2d GEOMETRY
) AS $$
DECLARE
    v_center GEOMETRY;
    v_radius_degrees DECIMAL;
BEGIN
    v_center := ST_SetSRID(ST_MakePoint(p_center_x, p_center_y), 4326);
    -- Rough conversion: 1 degree ≈ 111km
    v_radius_degrees := p_radius_meters / 111000.0;

    RETURN QUERY
    SELECT
        g.id, g.element_id, g.geometry_type,
        ST_Distance(g.geom_2d::GEOGRAPHY, v_center::GEOGRAPHY)::DECIMAL as distance_meters,
        g.geom_2d
    FROM geometry.geometries g
    WHERE ST_DWithin(g.geom_2d, v_center, v_radius_degrees)
      AND (p_project_id IS NULL OR g.project_id = p_project_id)
    ORDER BY distance_meters;
END;
$$ LANGUAGE plpgsql;

-- Simplify geometry function
CREATE OR REPLACE FUNCTION geometry.simplify_geometry(
    p_geometry_id UUID,
    p_tolerance_meters DECIMAL DEFAULT 1.0
)
RETURNS GEOMETRY AS $$
DECLARE
    v_geom GEOMETRY;
    v_simplified GEOMETRY;
BEGIN
    SELECT geom_2d INTO v_geom FROM geometry.geometries WHERE id = p_geometry_id;

    IF v_geom IS NULL THEN
        RETURN NULL;
    END IF;

    -- Use Douglas-Peucker algorithm
    v_simplified := ST_SimplifyPreserveTopology(v_geom, p_tolerance_meters / 111000.0);

    -- Update simplified geometry
    UPDATE geometry.geometries
    SET geom_simplified = v_simplified
    WHERE id = p_geometry_id;

    RETURN v_simplified;
END;
$$ LANGUAGE plpgsql;

-- Export to GeoJSON function
CREATE OR REPLACE FUNCTION geometry.export_to_geojson(
    p_project_id UUID DEFAULT NULL,
    p_design_id UUID DEFAULT NULL,
    p_element_ids UUID[] DEFAULT NULL
)
RETURNS JSONB AS $$
DECLARE
    v_result JSONB;
BEGIN
    SELECT jsonb_build_object(
        'type', 'FeatureCollection',
        'features', jsonb_agg(
            jsonb_build_object(
                'type', 'Feature',
                'id', g.id,
                'geometry', ST_AsGeoJSON(g.geom_2d)::JSONB,
                'properties', jsonb_build_object(
                    'element_id', g.element_id,
                    'design_id', g.design_id,
                    'geometry_type', g.geometry_type,
                    'area', g.area,
                    'length', g.length
                ) || COALESCE(g.properties, '{}')
            )
        )
    ) INTO v_result
    FROM geometry.geometries g
    WHERE (p_project_id IS NULL OR g.project_id = p_project_id)
      AND (p_design_id IS NULL OR g.design_id = p_design_id)
      AND (p_element_ids IS NULL OR g.element_id = ANY(p_element_ids));

    RETURN COALESCE(v_result, '{"type": "FeatureCollection", "features": []}'::JSONB);
END;
$$ LANGUAGE plpgsql;
