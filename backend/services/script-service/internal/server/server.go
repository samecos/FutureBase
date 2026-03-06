package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/archplatform/script-service/internal/config"
	"github.com/archplatform/script-service/internal/engine"
	"github.com/archplatform/script-service/internal/models"
	"github.com/archplatform/script-service/internal/storage"
	pb "github.com/archplatform/script-service/proto"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ScriptServer implements the gRPC script service
type ScriptServer struct {
	pb.UnimplementedScriptServiceServer
	storage *storage.PostgresStorage
	engine  *engine.Engine
	config  *config.Config
}

// NewScriptServer creates a new script server
func NewScriptServer(storage *storage.PostgresStorage, eng *engine.Engine, cfg *config.Config) *ScriptServer {
	return &ScriptServer{
		storage: storage,
		engine:  eng,
		config:  cfg,
	}
}

// Start starts the gRPC server
func (s *ScriptServer) Start(port int) error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterScriptServiceServer(grpcServer, s)

	log.Printf("gRPC server listening on port %d", port)
	return grpcServer.Serve(lis)
}

// CreateScript creates a new script
func (s *ScriptServer) CreateScript(ctx context.Context, req *pb.CreateScriptRequest) (*pb.Script, error) {
	// Validate code syntax
	if err := s.engine.Validate(req.Code); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid script code: %v", err)
	}

	script := &models.Script{
		ID:             uuid.New(),
		TenantID:       uuid.MustParse(req.TenantId),
		Name:           req.Name,
		Description:    req.Description,
		Code:           req.Code,
		Language:       req.Language,
		Version:        1,
		Status:         models.ScriptStatusDraft,
		InputSchema:    req.InputSchema,
		OutputSchema:   req.OutputSchema,
		Tags:           req.Tags,
		Dependencies:   req.Dependencies,
		TimeoutSeconds: int(req.TimeoutSeconds),
		MaxMemoryMB:    int(req.MaxMemoryMb),
		CreatedBy:      uuid.MustParse(req.CreatedBy),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if req.ProjectId != "" {
		pid := uuid.MustParse(req.ProjectId)
		script.ProjectID = &pid
	}

	if err := s.storage.CreateScript(ctx, script); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create script: %v", err)
	}

	return s.mapScriptToProto(script), nil
}

// GetScript retrieves a script by ID
func (s *ScriptServer) GetScript(ctx context.Context, req *pb.GetScriptRequest) (*pb.Script, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid script ID")
	}

	script, err := s.storage.GetScript(ctx, id)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "script not found")
	}

	return s.mapScriptToProto(script), nil
}

// ListScripts lists scripts for a tenant
func (s *ScriptServer) ListScripts(ctx context.Context, req *pb.ListScriptsRequest) (*pb.ListScriptsResponse, error) {
	tenantID, err := uuid.Parse(req.TenantId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant ID")
	}

	scripts, err := s.storage.GetScriptsByTenant(ctx, tenantID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list scripts: %v", err)
	}

	response := &pb.ListScriptsResponse{
		Scripts: make([]*pb.Script, len(scripts)),
	}
	for i, script := range scripts {
		response.Scripts[i] = s.mapScriptToProto(script)
	}

	return response, nil
}

// ExecuteScript executes a script
func (s *ScriptServer) ExecuteScript(ctx context.Context, req *pb.ExecuteScriptRequest) (*pb.Execution, error) {
	scriptID, err := uuid.Parse(req.ScriptId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid script ID")
	}

	script, err := s.storage.GetScript(ctx, scriptID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "script not found")
	}

	// Parse input
	var input map[string]any
	if req.Input != "" {
		if err := json.Unmarshal([]byte(req.Input), &input); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid input JSON")
		}
	}

	// Create execution record
	execution := &models.ScriptExecution{
		ID:        uuid.New(),
		ScriptID:  scriptID,
		TenantID:  script.TenantID,
		Version:   script.Version,
		Status:    models.ExecutionStatusRunning,
		Input:     req.Input,
		StartedAt: time.Now(),
		CreatedBy: uuid.MustParse(req.UserId),
	}

	if err := s.storage.CreateExecution(ctx, execution); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create execution record")
	}

	// Execute script
	execCtx, cancel := context.WithTimeout(ctx, time.Duration(script.TimeoutSeconds)*time.Second)
	defer cancel()

	result, err := s.engine.Execute(execCtx, script, input)
	
	now := time.Now()
	execution.CompletedAt = &now
	execution.ExecutionTime = result.ExecutionTime
	execution.MemoryUsage = result.MemoryUsage
	execution.Logs = result.Logs

	if err != nil {
		execution.Status = models.ExecutionStatusFailed
		execution.Error = err.Error()
	} else if result.Error != "" {
		execution.Status = models.ExecutionStatusFailed
		execution.Error = result.Error
	} else {
		execution.Status = models.ExecutionStatusCompleted
		execution.Output = result.Output
	}

	if updateErr := s.storage.UpdateExecution(ctx, execution); updateErr != nil {
		log.Printf("Failed to update execution: %v", updateErr)
	}

	return s.mapExecutionToProto(execution), nil
}

// GetExecution retrieves an execution result
func (s *ScriptServer) GetExecution(ctx context.Context, req *pb.GetExecutionRequest) (*pb.Execution, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid execution ID")
	}

	execution, err := s.storage.GetExecution(ctx, id)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "execution not found")
	}

	return s.mapExecutionToProto(execution), nil
}

// Helper methods
func (s *ScriptServer) mapScriptToProto(script *models.Script) *pb.Script {
	pbScript := &pb.Script{
		Id:             script.ID.String(),
		TenantId:       script.TenantID.String(),
		Name:           script.Name,
		Description:    script.Description,
		Code:           script.Code,
		Language:       script.Language,
		Version:        int32(script.Version),
		Status:         string(script.Status),
		Tags:           script.Tags,
		InputSchema:    script.InputSchema,
		OutputSchema:   script.OutputSchema,
		Dependencies:   script.Dependencies,
		TimeoutSeconds: int32(script.TimeoutSeconds),
		MaxMemoryMb:    int32(script.MaxMemoryMB),
		CreatedBy:      script.CreatedBy.String(),
		CreatedAt:      script.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      script.UpdatedAt.Format(time.RFC3339),
	}
	
	if script.ProjectID != nil {
		pbScript.ProjectId = script.ProjectID.String()
	}
	
	return pbScript
}

func (s *ScriptServer) mapExecutionToProto(exec *models.ScriptExecution) *pb.Execution {
	pbExec := &pb.Execution{
		Id:            exec.ID.String(),
		ScriptId:      exec.ScriptID.String(),
		Status:        string(exec.Status),
		Input:         exec.Input,
		Output:        exec.Output,
		Error:         exec.Error,
		Logs:          exec.Logs,
		ExecutionTime: int32(exec.ExecutionTime),
		MemoryUsage:   exec.MemoryUsage,
		StartedAt:     exec.StartedAt.Format(time.RFC3339),
		CreatedBy:     exec.CreatedBy.String(),
		CacheHit:      exec.CacheHit,
		WorkflowId:    exec.WorkflowID,
	}
	
	if exec.CompletedAt != nil {
		pbExec.CompletedAt = exec.CompletedAt.Format(time.RFC3339)
	}
	
	return pbExec
}
