package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	pb "github.com/archplatform/geometry-service/api/v1"
	"github.com/archplatform/geometry-service/pkg/config"
	"github.com/archplatform/geometry-service/pkg/geometry"
	"github.com/archplatform/geometry-service/pkg/models"
	"github.com/archplatform/geometry-service/pkg/storage"
	"github.com/archplatform/geometry-service/pkg/transformer"
	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/twpayne/go-geom"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

// GeometryServer implements the GeometryService gRPC server
type GeometryServer struct {
	pb.UnimplementedGeometryServiceServer
	
	storage     *storage.PostGISStorage
	redis       *redis.Client
	transformer *transformer.Transformer
	operations  *geometry.Operations
	logger      *zap.Logger
	config      *config.Config
}

// NewGeometryServer creates a new geometry server
func NewGeometryServer(cfg *config.Config, logger *zap.Logger) (*GeometryServer, error) {
	// Connect to database
	db, err := sql.Open("postgres", cfg.Database.DatabaseDSN())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	
	db.SetMaxOpenConns(cfg.Database.MaxOpenConns)
	db.SetMaxIdleConns(cfg.Database.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.Database.ConnMaxLifetime)
	
	// Connect to Redis
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.RedisAddr(),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
		PoolSize: cfg.Redis.PoolSize,
	})
	
	// Create storage
	pgStorage, err := storage.NewPostGISStorage(db, cfg.Geometry.DefaultSRID)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage: %w", err)
	}
	
	server := &GeometryServer{
		storage:     pgStorage,
		redis:       rdb,
		transformer: transformer.NewTransformer(cfg.Geometry.DefaultTolerance),
		operations:  geometry.NewOperations(cfg.Geometry.DefaultTolerance),
		logger:      logger,
		config:      cfg,
	}
	
	return server, nil
}

// CreateGeometry creates a new geometry
func (s *GeometryServer) CreateGeometry(ctx context.Context, req *pb.CreateGeometryRequest) (*pb.Geometry, error) {
	s.logger.Info("CreateGeometry called",
		zap.String("design_id", req.DesignId),
		zap.String("element_id", req.ElementId),
	)
	
	// Convert protobuf geometry to internal model
	g, err := s.protoToModel(req.Geometry)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid geometry: %v", err)
	}
	
	g.ElementID = req.ElementId
	g.DesignID = req.DesignId
	g.ProjectID = req.ProjectId
	g.TenantID = req.TenantId
	
	// Calculate derived properties
	if g.Geom2D != nil {
		g.Area = floatPtr(s.operations.CalculateArea(g.Geom2D))
		g.Length = floatPtr(s.operations.CalculateLength(g.Geom2D))
		g.VertexCount = s.operations.CalculateVertexCount(g.Geom2D)
		g.BBox = s.transformer.ComputeBoundingBox(g.Geom2D)
	}
	
	// Create in database
	if err := s.storage.CreateGeometry(ctx, g); err != nil {
		s.logger.Error("Failed to create geometry", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to create geometry: %v", err)
	}
	
	return s.modelToProto(g), nil
}

// GetGeometry retrieves a geometry by ID
func (s *GeometryServer) GetGeometry(ctx context.Context, req *pb.GetGeometryRequest) (*pb.Geometry, error) {
	g, err := s.storage.GetGeometry(ctx, req.Id)
	if err != nil {
		s.logger.Error("Failed to get geometry", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to get geometry: %v", err)
	}
	
	if g == nil {
		return nil, status.Errorf(codes.NotFound, "geometry not found: %s", req.Id)
	}
	
	return s.modelToProto(g), nil
}

// UpdateGeometry updates a geometry
func (s *GeometryServer) UpdateGeometry(ctx context.Context, req *pb.UpdateGeometryRequest) (*pb.Geometry, error) {
	g, err := s.protoToModel(req.Geometry)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid geometry: %v", err)
	}
	
	g.ID = req.Id
	
	// Recalculate properties
	if g.Geom2D != nil {
		g.Area = floatPtr(s.operations.CalculateArea(g.Geom2D))
		g.Length = floatPtr(s.operations.CalculateLength(g.Geom2D))
		g.VertexCount = s.operations.CalculateVertexCount(g.Geom2D)
		g.BBox = s.transformer.ComputeBoundingBox(g.Geom2D)
	}
	
	if err := s.storage.UpdateGeometry(ctx, g); err != nil {
		s.logger.Error("Failed to update geometry", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to update geometry: %v", err)
	}
	
	// Update cache
	s.redis.Del(ctx, fmt.Sprintf("geometry:%s", req.Id))
	
	return s.modelToProto(g), nil
}

// DeleteGeometry deletes a geometry
func (s *GeometryServer) DeleteGeometry(ctx context.Context, req *pb.DeleteGeometryRequest) (*emptypb.Empty, error) {
	if err := s.storage.DeleteGeometry(ctx, req.Id); err != nil {
		s.logger.Error("Failed to delete geometry", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to delete geometry: %v", err)
	}
	
	// Delete from cache
	s.redis.Del(ctx, fmt.Sprintf("geometry:%s", req.Id))
	
	return &emptypb.Empty{}, nil
}

// BatchCreateGeometry creates multiple geometries
func (s *GeometryServer) BatchCreateGeometry(ctx context.Context, req *pb.BatchCreateGeometryRequest) (*pb.BatchGeometryResponse, error) {
	var geometries []*pb.Geometry
	var errors []*pb.GeometryError
	
	for _, protoGeom := range req.Geometries {
		g, err := s.protoToModel(protoGeom)
		if err != nil {
			errors = append(errors, &pb.GeometryError{
				Id:          protoGeom.Id,
				ErrorCode:   "INVALID_GEOMETRY",
				ErrorMessage: err.Error(),
			})
			continue
		}
		
		g.DesignID = req.DesignId
		g.ProjectID = req.ProjectId
		g.TenantID = req.TenantId
		
		// Calculate properties
		if g.Geom2D != nil {
			g.Area = floatPtr(s.operations.CalculateArea(g.Geom2D))
			g.Length = floatPtr(s.operations.CalculateLength(g.Geom2D))
			g.VertexCount = s.operations.CalculateVertexCount(g.Geom2D)
			g.BBox = s.transformer.ComputeBoundingBox(g.Geom2D)
		}
		
		if err := s.storage.CreateGeometry(ctx, g); err != nil {
			errors = append(errors, &pb.GeometryError{
				Id:          protoGeom.Id,
				ErrorCode:   "CREATE_FAILED",
				ErrorMessage: err.Error(),
			})
			continue
		}
		
		geometries = append(geometries, s.modelToProto(g))
	}
	
	return &pb.BatchGeometryResponse{
		Geometries:    geometries,
		Errors:        errors,
		SuccessCount:  int32(len(geometries)),
		FailedCount:   int32(len(errors)),
	}, nil
}

// Transform applies a transformation to a geometry
func (s *GeometryServer) Transform(ctx context.Context, req *pb.TransformRequest) (*pb.Geometry, error) {
	// Get the geometry
	g, err := s.storage.GetGeometry(ctx, req.Id)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get geometry: %v", err)
	}
	if g == nil {
		return nil, status.Errorf(codes.NotFound, "geometry not found: %s", req.Id)
	}
	
	if g.Geom2D == nil {
		return nil, status.Errorf(codes.InvalidArgument, "geometry has no 2d representation")
	}
	
	// Apply transformation
	var result geom.T
	switch params := req.Params.(type) {
	case *pb.TransformRequest_Translate:
		result, err = s.transformer.Translate(g.Geom2D, params.Translate.Dx, params.Translate.Dy, params.Translate.Dz)
	case *pb.TransformRequest_Rotate:
		center := models.Vector3{X: params.Rotate.Origin.X, Y: params.Rotate.Origin.Y, Z: params.Rotate.Origin.Z}
		axis := models.Vector3{X: params.Rotate.Axis.X, Y: params.Rotate.Axis.Y, Z: params.Rotate.Axis.Z}
		result, err = s.transformer.Rotate(g.Geom2D, center, axis, params.Rotate.Angle)
	case *pb.TransformRequest_Scale:
		center := models.Vector3{X: params.Scale.Origin.X, Y: params.Scale.Origin.Y, Z: params.Scale.Origin.Z}
		result, err = s.transformer.Scale(g.Geom2D, center, params.Scale.Sx, params.Scale.Sy, params.Scale.Sz)
	case *pb.TransformRequest_Mirror:
		point := models.Vector3{X: params.Mirror.PointOnPlane.X, Y: params.Mirror.PointOnPlane.Y, Z: params.Mirror.PointOnPlane.Z}
		normal := models.Vector3{X: params.Mirror.Normal.X, Y: params.Mirror.Normal.Y, Z: params.Mirror.Normal.Z}
		result, err = s.transformer.Mirror(g.Geom2D, point, normal)
	case *pb.TransformRequest_Matrix:
		var matrix [16]float64
		copy(matrix[:], params.Matrix.Values)
		result, err = s.transformer.ApplyTransform(g.Geom2D, &models.Transform{Matrix: matrix})
	}
	
	if err != nil {
		return nil, status.Errorf(codes.Internal, "transformation failed: %v", err)
	}
	
	// Update geometry
	g.Geom2D = result
	g.Area = floatPtr(s.operations.CalculateArea(result))
	g.Length = floatPtr(s.operations.CalculateLength(result))
	g.BBox = s.transformer.ComputeBoundingBox(result)
	
	if req.CreateNewVersion {
		// Create new geometry
		g.ID = ""
		g.Version = 1
		if err := s.storage.CreateGeometry(ctx, g); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to create transformed geometry: %v", err)
		}
	} else {
		if err := s.storage.UpdateGeometry(ctx, g); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to update geometry: %v", err)
		}
	}
	
	return s.modelToProto(g), nil
}

// BooleanOperation performs a boolean operation on two geometries
func (s *GeometryServer) BooleanOperation(ctx context.Context, req *pb.BooleanOperationRequest) (*pb.Geometry, error) {
	// Get geometries
	g1, err := s.storage.GetGeometry(ctx, req.GeometryId1)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get geometry 1: %v", err)
	}
	if g1 == nil {
		return nil, status.Errorf(codes.NotFound, "geometry 1 not found")
	}
	
	g2, err := s.storage.GetGeometry(ctx, req.GeometryId2)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get geometry 2: %v", err)
	}
	if g2 == nil {
		return nil, status.Errorf(codes.NotFound, "geometry 2 not found")
	}
	
	if g1.Geom2D == nil || g2.Geom2D == nil {
		return nil, status.Errorf(codes.InvalidArgument, "geometries must have 2d representation")
	}
	
	// Convert operation type
	var op geometry.BooleanOperationType
	switch req.Operation {
	case pb.BooleanOperation_BOOLEAN_OPERATION_UNION:
		op = geometry.BooleanUnion
	case pb.BooleanOperation_BOOLEAN_OPERATION_INTERSECTION:
		op = geometry.BooleanIntersection
	case pb.BooleanOperation_BOOLEAN_OPERATION_DIFFERENCE:
		op = geometry.BooleanDifference
	default:
		return nil, status.Errorf(codes.InvalidArgument, "unsupported boolean operation")
	}
	
	// Perform operation
	result, err := geometry.BooleanOperation(g1.Geom2D, g2.Geom2D, op)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "boolean operation failed: %v", err)
	}
	
	// Create new geometry
	newGeom := &models.Geometry{
		ElementID:   g1.ElementID,
		DesignID:    g1.DesignID,
		ProjectID:   g1.ProjectID,
		TenantID:    g1.TenantID,
		Type:        g1.Type,
		Geom2D:      result,
		Area:        floatPtr(s.operations.CalculateArea(result)),
		Length:      floatPtr(s.operations.CalculateLength(result)),
		VertexCount: s.operations.CalculateVertexCount(result),
		BBox:        s.transformer.ComputeBoundingBox(result),
		Properties:  g1.Properties,
	}
	
	if err := s.storage.CreateGeometry(ctx, newGeom); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create result geometry: %v", err)
	}
	
	return s.modelToProto(newGeom), nil
}

// QueryByBoundingBox queries geometries within a bounding box
func (s *GeometryServer) QueryByBoundingBox(ctx context.Context, req *pb.BoundingBoxQueryRequest) (*pb.GeometryCollection, error) {
	bbox := &models.BoundingBox{
		MinX: req.Bbox.MinX,
		MinY: req.Bbox.MinY,
		MinZ: req.Bbox.MinZ,
		MaxX: req.Bbox.MaxX,
		MaxY: req.Bbox.MaxY,
		MaxZ: req.Bbox.MaxZ,
	}
	
	geometries, err := s.storage.QueryByBoundingBox(ctx, req.TenantId, req.DesignId, bbox, int(req.PageSize))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query failed: %v", err)
	}
	
	var pbGeometries []*pb.Geometry
	for _, g := range geometries {
		pbGeometries = append(pbGeometries, s.modelToProto(g))
	}
	
	return &pb.GeometryCollection{
		Geometries:  pbGeometries,
		TotalCount:  int32(len(pbGeometries)),
	}, nil
}

// QueryByRadius queries geometries within a radius
func (s *GeometryServer) QueryByRadius(ctx context.Context, req *pb.RadiusQueryRequest) (*pb.GeometryCollection, error) {
	geometries, err := s.storage.QueryByRadius(ctx, req.TenantId, req.ProjectId, 
		req.Center.X, req.Center.Y, req.Radius, int(req.Limit))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query failed: %v", err)
	}
	
	var pbGeometries []*pb.Geometry
	for _, g := range geometries {
		pbGeometries = append(pbGeometries, s.modelToProto(g))
	}
	
	return &pb.GeometryCollection{
		Geometries:  pbGeometries,
		TotalCount:  int32(len(pbGeometries)),
	}, nil
}

// QueryNearest finds the nearest geometries to a point
func (s *GeometryServer) QueryNearest(ctx context.Context, req *pb.NearestQueryRequest) (*pb.GeometryCollection, error) {
	geometries, err := s.storage.QueryNearest(ctx, req.TenantId, req.ProjectId,
		req.ReferencePoint.X, req.ReferencePoint.Y, int(req.Limit), req.MaxDistance)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query failed: %v", err)
	}
	
	var pbGeometries []*pb.Geometry
	for _, g := range geometries {
		pbGeometries = append(pbGeometries, s.modelToProto(g))
	}
	
	return &pb.GeometryCollection{
		Geometries:  pbGeometries,
		TotalCount:  int32(len(pbGeometries)),
	}, nil
}

// CalculateArea calculates the area of a geometry
func (s *GeometryServer) CalculateArea(ctx context.Context, req *pb.CalculateRequest) (*pb.CalculateResponse, error) {
	area, err := s.storage.CalculateArea(ctx, req.Id)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "calculation failed: %v", err)
	}
	
	return &pb.CalculateResponse{
		Value: area,
		Unit:  "square_meters",
	}, nil
}

// CalculateVolume is not implemented for 2D geometries
func (s *GeometryServer) CalculateVolume(ctx context.Context, req *pb.CalculateRequest) (*pb.CalculateResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "volume calculation not implemented")
}

// CalculateDistance calculates the distance between two geometries
func (s *GeometryServer) CalculateDistance(ctx context.Context, req *pb.DistanceRequest) (*pb.DistanceResponse, error) {
	distance, err := s.storage.CalculateDistance(ctx, req.GeometryId1, req.GeometryId2)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "calculation failed: %v", err)
	}
	
	return &pb.DistanceResponse{
		Distance: distance,
	}, nil
}

// CalculateCentroid calculates the centroid of a geometry
func (s *GeometryServer) CalculateCentroid(ctx context.Context, req *pb.CalculateRequest) (*pb.Point, error) {
	x, y, err := s.storage.CalculateCentroid(ctx, req.Id)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "calculation failed: %v", err)
	}
	
	return &pb.Point{X: x, Y: y}, nil
}

// ValidateGeometry validates a geometry
func (s *GeometryServer) ValidateGeometry(ctx context.Context, req *pb.ValidateRequest) (*pb.ValidateResponse, error) {
	g, err := s.storage.GetGeometry(ctx, req.Id)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get geometry: %v", err)
	}
	if g == nil {
		return nil, status.Errorf(codes.NotFound, "geometry not found")
	}
	
	valid, errors := s.operations.Validate(g.Geom2D)
	
	var pbErrors []*pb.ValidationError
	for _, e := range errors {
		pbErrors = append(pbErrors, &pb.ValidationError{
			Type:     pb.ValidationErrorType_VALIDATION_ERROR_TYPE_UNSPECIFIED,
			Message:  e,
			Severity: pb.Severity_ERROR,
		})
	}
	
	return &pb.ValidateResponse{
		IsValid: valid,
		Errors:  pbErrors,
	}, nil
}

// RepairGeometry repairs an invalid geometry
func (s *GeometryServer) RepairGeometry(ctx context.Context, req *pb.RepairRequest) (*pb.Geometry, error) {
	if err := s.storage.RepairGeometry(ctx, req.Id); err != nil {
		return nil, status.Errorf(codes.Internal, "repair failed: %v", err)
	}
	
	return s.GetGeometry(ctx, &pb.GetGeometryRequest{Id: req.Id})
}

// SimplifyGeometry simplifies a geometry
func (s *GeometryServer) SimplifyGeometry(ctx context.Context, req *pb.SimplifyRequest) (*pb.Geometry, error) {
	if err := s.storage.SimplifyGeometry(ctx, req.Id, req.Tolerance); err != nil {
		return nil, status.Errorf(codes.Internal, "simplification failed: %v", err)
	}
	
	return s.GetGeometry(ctx, &pb.GetGeometryRequest{Id: req.Id})
}

// ImportGeometry imports geometry from a file format
func (s *GeometryServer) ImportGeometry(ctx context.Context, req *pb.ImportRequest) (*pb.ImportResponse, error) {
	// TODO: Implement import logic
	return nil, status.Errorf(codes.Unimplemented, "import not implemented")
}

// ExportGeometry exports geometry to a file format
func (s *GeometryServer) ExportGeometry(ctx context.Context, req *pb.ExportRequest) (*pb.ExportResponse, error) {
	// TODO: Implement export logic
	return nil, status.Errorf(codes.Unimplemented, "export not implemented")
}

// ConvertFormat converts geometry between formats
func (s *GeometryServer) ConvertFormat(ctx context.Context, req *pb.ConvertRequest) (*pb.ConvertResponse, error) {
	// TODO: Implement format conversion
	return nil, status.Errorf(codes.Unimplemented, "format conversion not implemented")
}

// ParseBIMFile parses a BIM file
func (s *GeometryServer) ParseBIMFile(req *pb.ParseBIMFileRequest, stream pb.GeometryService_ParseBIMFileServer) error {
	// TODO: Implement BIM parsing
	return status.Errorf(codes.Unimplemented, "BIM parsing not implemented")
}

// GetBIMMetadata retrieves BIM file metadata
func (s *GeometryServer) GetBIMMetadata(ctx context.Context, req *pb.GetBIMMetadataRequest) (*pb.BIMMetadata, error) {
	// TODO: Implement BIM metadata retrieval
	return nil, status.Errorf(codes.Unimplemented, "BIM metadata not implemented")
}

// ExtractBIMElements extracts elements from a BIM file
func (s *GeometryServer) ExtractBIMElements(ctx context.Context, req *pb.ExtractBIMElementsRequest) (*pb.ExtractBIMElementsResponse, error) {
	// TODO: Implement BIM element extraction
	return nil, status.Errorf(codes.Unimplemented, "BIM extraction not implemented")
}

// Helper methods

func (s *GeometryServer) protoToModel(pbGeom *pb.Geometry) (*models.Geometry, error) {
	g := &models.Geometry{
		ID:         pbGeom.Id,
		ElementID:  pbGeom.ElementId,
		DesignID:   pbGeom.DocumentId,
		ProjectID:  pbGeom.ProjectId,
		TenantID:   pbGeom.TenantId,
		Type:       models.GeometryType(pbGeom.Type.String()),
		Properties: models.JSONB(pbGeom.Properties.AsMap()),
		Metadata:   models.JSONB(pbGeom.Metadata.AsMap()),
		SRID:       int(pbGeom.Srid),
		Version:    int(pbGeom.Version),
		CreatedBy:  pbGeom.CreatedBy,
		UpdatedBy:  pbGeom.UpdatedBy,
	}
	
	// Convert geometry data
	if pbGeom.Point != nil {
		g.Geom2D = geom.NewPoint(geom.XYZ).MustSetCoords(geom.Coord{
			pbGeom.Point.X, pbGeom.Point.Y, pbGeom.Point.Z,
		})
	} else if pbGeom.Line != nil {
		g.Geom2D = geom.NewLineString(geom.XYZ).MustSetCoords([]geom.Coord{
			{pbGeom.Line.Start.X, pbGeom.Line.Start.Y, pbGeom.Line.Start.Z},
			{pbGeom.Line.End.X, pbGeom.Line.End.Y, pbGeom.Line.End.Z},
		})
	} else if pbGeom.Polygon != nil {
		rings := make([][]geom.Coord, 0)
		
		// Exterior ring
		exterior := make([]geom.Coord, 0, len(pbGeom.Polygon.ExteriorRing))
		for _, p := range pbGeom.Polygon.ExteriorRing {
			exterior = append(exterior, geom.Coord{p.X, p.Y, p.Z})
		}
		rings = append(rings, exterior)
		
		// Interior rings
		for _, ring := range pbGeom.Polygon.InteriorRings {
			interior := make([]geom.Coord, 0, len(ring.Points))
			for _, p := range ring.Points {
				interior = append(interior, geom.Coord{p.X, p.Y, p.Z})
			}
			rings = append(rings, interior)
		}
		
		g.Geom2D = geom.NewPolygon(geom.XYZ).MustSetCoords(rings)
	} else if pbGeom.Polyline != nil {
		coords := make([]geom.Coord, 0, len(pbGeom.Polyline.Points))
		for _, p := range pbGeom.Polyline.Points {
			coords = append(coords, geom.Coord{p.X, p.Y, p.Z})
		}
		g.Geom2D = geom.NewLineString(geom.XYZ).MustSetCoords(coords)
	}
	
	return g, nil
}

func (s *GeometryServer) modelToProto(g *models.Geometry) *pb.Geometry {
	pbGeom := &pb.Geometry{
		Id:         g.ID,
		ElementId:  g.ElementID,
		DocumentId: g.DesignID,
		ProjectId:  g.ProjectID,
		TenantId:   g.TenantID,
		Type:       pb.GeometryType(pb.GeometryType_value[string(g.Type)]),
		Area:       g.Area,
		Length:     g.Length,
		VertexCount: int32(g.VertexCount),
		Srid:       int32(g.SRID),
		Version:    int64(g.Version),
		CreatedBy:  g.CreatedBy,
		UpdatedBy:  g.UpdatedBy,
	}
	
	// Convert geometry data
	if g.Geom2D != nil {
		switch geom := g.Geom2D.(type) {
		case *geom.Point:
			coords := geom.Coords()
			pbGeom.Point = &pb.Point{
				X: coords.X(),
				Y: coords.Y(),
				Z: coords.Z(),
			}
		case *geom.LineString:
			coords := geom.Coords()
			if len(coords) >= 2 {
				pbGeom.Line = &pb.Line{
					Start: &pb.Point{X: coords[0].X(), Y: coords[0].Y(), Z: coords[0].Z()},
					End:   &pb.Point{X: coords[1].X(), Y: coords[1].Y(), Z: coords[1].Z()},
				}
			}
		case *geom.Polygon:
			coords := geom.Coords()
			if len(coords) > 0 && len(coords[0]) > 0 {
				exterior := make([]*pb.Point, 0, len(coords[0]))
				for _, c := range coords[0] {
					exterior = append(exterior, &pb.Point{X: c.X(), Y: c.Y(), Z: c.Z()})
				}
				pbGeom.Polygon = &pb.Polygon{
					ExteriorRing: exterior,
				}
			}
		}
	}
	
	// Convert bounding box
	if g.BBox != nil {
		pbGeom.Bbox = &pb.BoundingBox3D{
			MinX: g.BBox.MinX,
			MinY: g.BBox.MinY,
			MinZ: g.BBox.MinZ,
			MaxX: g.BBox.MaxX,
			MaxY: g.BBox.MaxY,
			MaxZ: g.BBox.MaxZ,
		}
	}
	
	return pbGeom
}

func floatPtr(f float64) *float64 {
	return &f
}

// StartGRPCServer starts the gRPC server
func StartGRPCServer(server *GeometryServer, cfg *config.Config) error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Server.GRPCPort))
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}
	
	s := grpc.NewServer()
	pb.RegisterGeometryServiceServer(s, server)
	reflection.Register(s)
	
	log.Printf("Starting Geometry gRPC server on port %d", cfg.Server.GRPCPort)
	return s.Serve(lis)
}

// StartHTTPServer starts the HTTP server
func StartHTTPServer(server *GeometryServer, cfg *config.Config) error {
	router := mux.NewRouter()
	
	// Health check
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})
	
	// Metrics
	router.Handle("/metrics", promhttp.Handler())
	
	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.HTTPPort),
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}
	
	log.Printf("Starting Geometry HTTP server on port %d", cfg.Server.HTTPPort)
	return httpServer.ListenAndServe()
}
