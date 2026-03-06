package server

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	pb "github.com/archplatform/collaboration-service/api/v1"
	"github.com/archplatform/collaboration-service/pkg/config"
	"github.com/archplatform/collaboration-service/pkg/errors"
	"github.com/archplatform/collaboration-service/pkg/models"
	"github.com/archplatform/collaboration-service/pkg/websocket"
	"github.com/archplatform/collaboration-service/pkg/yjs"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// CollaborationServer implements the CollaborationService gRPC server
type CollaborationServer struct {
	pb.UnimplementedCollaborationServiceServer
	
	db           *gorm.DB
	redis        *redis.Client
	yjsManager   *yjs.DocumentManager
	wsServer     *websocket.Server
	logger       *zap.Logger
	config       *config.Config
	sessions     map[string]*models.CollaborationSession
	sessionMu    sync.RWMutex
	eventBus     EventBus
}

// EventBus interface for event publishing
type EventBus interface {
	Publish(ctx context.Context, topic string, payload interface{}) error
	Subscribe(ctx context.Context, topic string) (<-chan interface{}, error)
}

// NATSEventBus implements EventBus using NATS
type NATSEventBus struct {
	// TODO: Add NATS connection
}

// Publish publishes an event
func (n *NATSEventBus) Publish(ctx context.Context, topic string, payload interface{}) error {
	// TODO: Implement NATS publish
	return nil
}

// Subscribe subscribes to a topic
func (n *NATSEventBus) Subscribe(ctx context.Context, topic string) (<-chan interface{}, error) {
	// TODO: Implement NATS subscribe
	return make(<-chan interface{}), nil
}

// NewCollaborationServer creates a new collaboration server
func NewCollaborationServer(cfg *config.Config, logger *zap.Logger) (*CollaborationServer, error) {
	// Connect to database
	db, err := connectDatabase(cfg.Database)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	
	// Connect to Redis
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.RedisAddr(),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
		PoolSize: cfg.Redis.PoolSize,
	})
	
	// Initialize Yjs manager
	yjsManager := yjs.NewDocumentManager()
	
	// Initialize WebSocket server
	wsServer := websocket.NewServer(logger, yjsManager)
	
	// Start WebSocket server goroutine
	go wsServer.Run()
	
	server := &CollaborationServer{
		db:         db,
		redis:      rdb,
		yjsManager: yjsManager,
		wsServer:   wsServer,
		logger:     logger,
		config:     cfg,
		sessions:   make(map[string]*models.CollaborationSession),
		eventBus:   &NATSEventBus{},
	}
	
	// Start cleanup task
	go server.cleanupTask()
	
	return server, nil
}

// connectDatabase connects to PostgreSQL database
func connectDatabase(cfg config.DatabaseConfig) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(cfg.DatabaseDSN()), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, err
	}
	
	// Configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	
	// Auto migrate
	if err := models.AutoMigrate(db); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}
	
	return db, nil
}

// CreateSession creates a new collaboration session
func (s *CollaborationServer) CreateSession(ctx context.Context, req *pb.CreateSessionRequest) (*pb.CreateSessionResponse, error) {
	s.logger.Info("CreateSession called",
		zap.String("document_id", req.DocumentId),
		zap.String("user_id", req.UserId),
	)
	
	// Check if active session already exists for this document
	existingSession, err := s.findActiveSession(ctx, req.DocumentId)
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, status.Errorf(codes.Internal, "failed to check existing session: %v", err)
	}
	
	if existingSession != nil {
		// Return existing session
		token, err := s.generateSessionToken(existingSession.ID, req.UserId)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to generate token: %v", err)
		}
		
		return &pb.CreateSessionResponse{
			SessionId:    existingSession.ID,
			WebsocketUrl: s.generateWebSocketURL(existingSession.ID),
			Token:        token,
			ExpiresAt:    timestamppb.New(*existingSession.ExpiresAt),
		}, nil
	}
	
	// Create new session
	expiresAt := time.Now().Add(s.config.Session.TTL)
	session := &models.CollaborationSession{
		ID:          uuid.New().String(),
		DocumentID:  req.DocumentId,
		TenantID:    req.TenantId,
		SessionType: models.SessionType(req.SessionType.String()),
		Status:      models.SessionStatusActive,
		CreatedBy:   req.UserId,
		ExpiresAt:   &expiresAt,
		Metadata:    models.JSONB(req.Metadata),
		ServerClock: 0,
	}
	
	// Save to database
	if err := s.db.WithContext(ctx).Create(session).Error; err != nil {
		s.logger.Error("Failed to create session", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to create session: %v", err)
	}
	
	// Cache in memory
	s.sessionMu.Lock()
	s.sessions[session.ID] = session
	s.sessionMu.Unlock()
	
	// Initialize Yjs document
	s.yjsManager.GetOrCreateDocument(req.DocumentId, session.ID)
	
	// Generate token
	token, err := s.generateSessionToken(session.ID, req.UserId)
	if err != nil {
		s.logger.Error("Failed to generate token", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to generate token: %v", err)
	}
	
	// Publish event
	s.eventBus.Publish(ctx, "collaboration.session.created", map[string]interface{}{
		"session_id":  session.ID,
		"document_id": session.DocumentID,
		"user_id":     req.UserId,
	})
	
	return &pb.CreateSessionResponse{
		SessionId:    session.ID,
		WebsocketUrl: s.generateWebSocketURL(session.ID),
		Token:        token,
		ExpiresAt:    timestamppb.New(expiresAt),
	}, nil
}

// JoinSession handles a user joining a session
func (s *CollaborationServer) JoinSession(req *pb.JoinSessionRequest, stream pb.CollaborationService_JoinSessionServer) error {
	ctx := stream.Context()
	
	// Validate session
	session, err := s.getSession(ctx, req.SessionId)
	if err != nil {
		return status.Errorf(codes.NotFound, "session not found: %v", err)
	}
	
	if session.Status != models.SessionStatusActive {
		return status.Errorf(codes.FailedPrecondition, "session is not active")
	}
	
	// Check session capacity
	participantCount, err := s.getParticipantCount(ctx, req.SessionId)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to get participant count: %v", err)
	}
	
	if participantCount >= s.config.Session.MaxParticipants {
		return status.Errorf(codes.ResourceExhausted, "session is full")
	}
	
	// Create or update participant
	participant := &models.SessionParticipant{
		ID:              uuid.New().String(),
		SessionID:       session.ID,
		UserID:          req.UserId,
		UserName:        req.UserName,
		UserAvatar:      req.UserAvatar,
		PermissionLevel: s.getDefaultPermission(session.ID, req.UserId),
		ClientType:      req.ClientInfo.ClientType,
		ClientVersion:   req.ClientInfo.Version,
		ClientPlatform:  req.ClientInfo.Platform,
		IsActive:        true,
	}
	
	// Upsert participant
	if err := s.db.WithContext(ctx).
		Where("session_id = ? AND user_id = ?", session.ID, req.UserId).
		Assign(participant).
		FirstOrCreate(participant).Error; err != nil {
		return status.Errorf(codes.Internal, "failed to create participant: %v", err)
	}
	
	// Cache participant in Redis
	s.cacheParticipant(ctx, session.ID, participant)
	
	// Increment server clock
	serverClock := s.incrementServerClock(ctx, session.ID)
	
	// Send initial state
	initialState, err := s.yjsManager.GetState(session.DocumentID)
	if err != nil {
		s.logger.Error("Failed to get initial state", zap.Error(err))
	}
	
	// Send user joined event
	if err := stream.Send(&pb.CollaborationEvent{
		Event: &pb.CollaborationEvent_UserJoined{
			UserJoined: &pb.UserJoinedEvent{
				User: &pb.UserInfo{
					UserId:          participant.UserID,
					UserName:        participant.UserName,
					UserAvatar:      participant.UserAvatar,
					PermissionLevel: pb.PermissionLevel_EDITOR,
				},
				JoinedAt: timestamppb.Now(),
			},
		},
	}); err != nil {
		return status.Errorf(codes.Internal, "failed to send event: %v", err)
	}
	
	s.logger.Info("User joined session",
		zap.String("session_id", session.ID),
		zap.String("user_id", req.UserId),
		zap.Int64("server_clock", serverClock),
	)
	
	// TODO: Implement event streaming
	// For now, just keep the stream open
	<-ctx.Done()
	
	// Handle user leave
	s.handleUserLeave(ctx, session.ID, req.UserId)
	
	return nil
}

// LeaveSession handles a user leaving a session
func (s *CollaborationServer) LeaveSession(ctx context.Context, req *pb.LeaveSessionRequest) (*emptypb.Empty, error) {
	if err := s.handleUserLeave(ctx, req.SessionId, req.UserId); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

// GetSessionInfo returns information about a session
func (s *CollaborationServer) GetSessionInfo(ctx context.Context, req *pb.GetSessionInfoRequest) (*pb.SessionInfo, error) {
	session, err := s.getSession(ctx, req.SessionId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "session not found: %v", err)
	}
	
	// Get active users
	users, err := s.getActiveUsers(ctx, req.SessionId)
	if err != nil {
		s.logger.Error("Failed to get active users", zap.Error(err))
	}
	
	return &pb.SessionInfo{
		SessionId:   session.ID,
		DocumentId:  session.DocumentID,
		SessionType: pb.SessionType(pb.SessionType_value[string(session.SessionType)]),
		Status:      pb.SessionStatus(pb.SessionStatus_value[string(session.Status)]),
		ActiveUsers: users,
		UserCount:   int32(len(users)),
		CreatedAt:   timestamppb.New(session.CreatedAt),
		ExpiresAt:   timestamppb.New(*session.ExpiresAt),
		CreatedBy:   session.CreatedBy,
		ServerClock: session.ServerClock,
	}, nil
}

// ListActiveSessions lists active sessions
func (s *CollaborationServer) ListActiveSessions(ctx context.Context, req *pb.ListActiveSessionsRequest) (*pb.ListActiveSessionsResponse, error) {
	var sessions []models.CollaborationSession
	
	query := s.db.WithContext(ctx).
		Where("status = ?", models.SessionStatusActive).
		Where("expires_at > ?", time.Now())
	
	if req.DocumentId != "" {
		query = query.Where("document_id = ?", req.DocumentId)
	}
	if req.TenantId != "" {
		query = query.Where("tenant_id = ?", req.TenantId)
	}
	
	if err := query.Find(&sessions).Error; err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list sessions: %v", err)
	}
	
	var pbSessions []*pb.SessionInfo
	for _, session := range sessions {
		pbSessions = append(pbSessions, &pb.SessionInfo{
			SessionId:   session.ID,
			DocumentId:  session.DocumentID,
			SessionType: pb.SessionType(pb.SessionType_value[string(session.SessionType)]),
			Status:      pb.SessionStatus(pb.SessionStatus_value[string(session.Status)]),
			CreatedAt:   timestamppb.New(session.CreatedAt),
			ExpiresAt:   timestamppb.New(*session.ExpiresAt),
			CreatedBy:   session.CreatedBy,
		})
	}
	
	return &pb.ListActiveSessionsResponse{
		Sessions: pbSessions,
	}, nil
}

// CloseSession closes a collaboration session
func (s *CollaborationServer) CloseSession(ctx context.Context, req *pb.CloseSessionRequest) (*emptypb.Empty, error) {
	if err := s.db.WithContext(ctx).
		Model(&models.CollaborationSession{}).
		Where("id = ?", req.SessionId).
		Update("status", models.SessionStatusClosed).Error; err != nil {
		return nil, status.Errorf(codes.Internal, "failed to close session: %v", err)
	}
	
	// Remove from cache
	s.sessionMu.Lock()
	delete(s.sessions, req.SessionId)
	s.sessionMu.Unlock()
	
	// Publish event
	s.eventBus.Publish(ctx, "collaboration.session.closed", map[string]interface{}{
		"session_id": req.SessionId,
		"reason":     req.Reason,
	})
	
	return &emptypb.Empty{}, nil
}

// SyncOperations handles bidirectional operation sync
func (s *CollaborationServer) SyncOperations(stream pb.CollaborationService_SyncOperationsServer) error {
	ctx := stream.Context()
	
	// Receive operations
	go func() {
		for {
			batch, err := stream.Recv()
			if err != nil {
				s.logger.Error("Failed to receive operation", zap.Error(err))
				return
			}
			
			if err := s.processOperationBatch(ctx, batch); err != nil {
				s.logger.Error("Failed to process operation batch", zap.Error(err))
				// Send error ack
				for _, op := range batch.Operations {
					stream.Send(&pb.OperationAck{
						OperationId:  op.OperationId,
						Status:       pb.AckStatus_ACK_STATUS_REJECTED,
						ErrorMessage: err.Error(),
					})
				}
				continue
			}
			
			// Send success ack
			for _, op := range batch.Operations {
				stream.Send(&pb.OperationAck{
					OperationId: op.OperationId,
					Status:      pb.AckStatus_ACK_STATUS_SUCCESS,
					ServerClock: batch.ServerClock,
				})
			}
		}
	}()
	
	// Wait for context cancellation
	<-ctx.Done()
	return ctx.Err()
}

// GetMissingOperations retrieves missing operations
func (s *CollaborationServer) GetMissingOperations(ctx context.Context, req *pb.GetMissingOperationsRequest) (*pb.OperationBatch, error) {
	var operations []models.OperationLog
	
	if err := s.db.WithContext(ctx).
		Where("session_id = ?", req.SessionId).
		Where("server_clock > ? AND server_clock <= ?", req.FromClock, req.ToClock).
		Order("server_clock ASC").
		Find(&operations).Error; err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get operations: %v", err)
	}
	
	var pbOperations []*pb.Operation
	for _, op := range operations {
		pbOperations = append(pbOperations, &pb.Operation{
			OperationId: op.OperationID,
			Type:        pb.OperationType(pb.OperationType_value[string(op.OperationType)]),
			TargetId:    *op.TargetID,
			Data:        op.OperationData,
			Timestamp:   timestamppb.New(op.CreatedAt),
		})
	}
	
	return &pb.OperationBatch{
		SessionId:  req.SessionId,
		Operations: pbOperations,
	}, nil
}

// UpdateCursor updates user cursor position
func (s *CollaborationServer) UpdateCursor(ctx context.Context, req *pb.UpdateCursorRequest) (*emptypb.Empty, error) {
	// Update in database
	if err := s.db.WithContext(ctx).
		Model(&models.SessionParticipant{}).
		Where("session_id = ? AND user_id = ?", req.SessionId, req.UserId).
		Update("cursor_position", models.JSONB{
			"element_id": req.Position.ElementId,
			"x":          req.Position.X,
			"y":          req.Position.Y,
			"z":          req.Position.Z,
		}).Error; err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update cursor: %v", err)
	}
	
	return &emptypb.Empty{}, nil
}

// UpdateSelection updates user selection
func (s *CollaborationServer) UpdateSelection(ctx context.Context, req *pb.UpdateSelectionRequest) (*emptypb.Empty, error) {
	// Update in database
	if err := s.db.WithContext(ctx).
		Model(&models.SessionParticipant{}).
		Where("session_id = ? AND user_id = ?", req.SessionId, req.UserId).
		Update("selection_range", models.JSONB{
			"element_ids": req.Selection.ElementIds,
			"type":        req.Selection.Type.String(),
		}).Error; err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update selection: %v", err)
	}
	
	return &emptypb.Empty{}, nil
}

// GetDocumentState returns the current document state
func (s *CollaborationServer) GetDocumentState(ctx context.Context, req *pb.GetDocumentStateRequest) (*pb.DocumentState, error) {
	session, err := s.getSession(ctx, req.SessionId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "session not found: %v", err)
	}
	
	state, err := s.yjsManager.GetState(session.DocumentID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get state: %v", err)
	}
	
	return &pb.DocumentState{
		SessionId:   session.ID,
		DocumentId:  session.DocumentID,
		YjsState:    state,
		ServerClock: session.ServerClock,
		Timestamp:   timestamppb.Now(),
	}, nil
}

// Helper methods

func (s *CollaborationServer) getSession(ctx context.Context, sessionID string) (*models.CollaborationSession, error) {
	// Check memory cache
	s.sessionMu.RLock()
	if session, ok := s.sessions[sessionID]; ok {
		s.sessionMu.RUnlock()
		return session, nil
	}
	s.sessionMu.RUnlock()
	
	// Query database
	var session models.CollaborationSession
	if err := s.db.WithContext(ctx).First(&session, "id = ?", sessionID).Error; err != nil {
		return nil, err
	}
	
	// Cache in memory
	s.sessionMu.Lock()
	s.sessions[sessionID] = &session
	s.sessionMu.Unlock()
	
	return &session, nil
}

func (s *CollaborationServer) findActiveSession(ctx context.Context, documentID string) (*models.CollaborationSession, error) {
	var session models.CollaborationSession
	if err := s.db.WithContext(ctx).
		Where("document_id = ? AND status = ? AND expires_at > ?",
			documentID, models.SessionStatusActive, time.Now()).
		First(&session).Error; err != nil {
		return nil, err
	}
	return &session, nil
}

func (s *CollaborationServer) getParticipantCount(ctx context.Context, sessionID string) (int64, error) {
	var count int64
	if err := s.db.WithContext(ctx).
		Model(&models.SessionParticipant{}).
		Where("session_id = ? AND is_active = ?", sessionID, true).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (s *CollaborationServer) getActiveUsers(ctx context.Context, sessionID string) ([]*pb.UserInfo, error) {
	var participants []models.SessionParticipant
	if err := s.db.WithContext(ctx).
		Where("session_id = ? AND is_active = ?", sessionID, true).
		Find(&participants).Error; err != nil {
		return nil, err
	}
	
	var users []*pb.UserInfo
	for _, p := range participants {
		users = append(users, &pb.UserInfo{
			UserId:          p.UserID,
			UserName:        p.UserName,
			UserAvatar:      p.UserAvatar,
			PermissionLevel: pb.PermissionLevel(pb.PermissionLevel_value[string(p.PermissionLevel)]),
			JoinedAt:        timestamppb.New(p.JoinedAt),
		})
	}
	
	return users, nil
}

func (s *CollaborationServer) handleUserLeave(ctx context.Context, sessionID, userID string) error {
	// Update participant status
	if err := s.db.WithContext(ctx).
		Model(&models.SessionParticipant{}).
		Where("session_id = ? AND user_id = ?", sessionID, userID).
		Update("is_active", false).Error; err != nil {
		return err
	}
	
	// Remove from Redis
	s.redis.HDel(ctx, fmt.Sprintf("session:%s:participants", sessionID), userID)
	
	s.logger.Info("User left session",
		zap.String("session_id", sessionID),
		zap.String("user_id", userID),
	)
	
	return nil
}

func (s *CollaborationServer) processOperationBatch(ctx context.Context, batch *pb.OperationBatch) error {
	// Increment server clock
	serverClock := s.incrementServerClock(ctx, batch.SessionId)
	
	// Persist operations
	for _, op := range batch.Operations {
		dbOp := &models.OperationLog{
			SessionID:     batch.SessionId,
			OperationID:   op.OperationId,
			UserID:        batch.UserId,
			ClientClock:   batch.ClientClock,
			ServerClock:   serverClock,
			OperationType: models.OperationType(op.Type.String()),
			TargetID:      &op.TargetId,
			OperationData: models.JSONB(op.Data),
			YjsUpdate:     batch.YjsUpdate,
		}
		
		if err := s.db.WithContext(ctx).Create(dbOp).Error; err != nil {
			return err
		}
	}
	
	// Apply to Yjs document
	if len(batch.YjsUpdate) > 0 {
		session, err := s.getSession(ctx, batch.SessionId)
		if err != nil {
			return err
		}
		
		if err := s.yjsManager.ApplyUpdate(session.DocumentID, batch.YjsUpdate); err != nil {
			s.logger.Error("Failed to apply Yjs update", zap.Error(err))
		}
	}
	
	return nil
}

func (s *CollaborationServer) incrementServerClock(ctx context.Context, sessionID string) int64 {
	// Try Redis first
	clock, err := s.redis.Incr(ctx, fmt.Sprintf("session:%s:clock", sessionID)).Result()
	if err == nil {
		return clock
	}
	
	// Fallback to database
	s.db.Exec("UPDATE collaboration_sessions SET server_clock = server_clock + 1 WHERE id = ?", sessionID)
	
	var session models.CollaborationSession
	s.db.First(&session, "id = ?", sessionID)
	
	return session.ServerClock
}

func (s *CollaborationServer) generateSessionToken(sessionID, userID string) (string, error) {
	// TODO: Implement JWT token generation
	return fmt.Sprintf("token_%s_%s_%d", sessionID, userID, time.Now().Unix()), nil
}

func (s *CollaborationServer) generateWebSocketURL(sessionID string) string {
	return fmt.Sprintf("ws://localhost:%d/ws?session=%s", s.config.Server.WebSocketPort, sessionID)
}

func (s *CollaborationServer) getDefaultPermission(sessionID, userID string) models.PermissionLevel {
	// TODO: Check if user is creator or has specific role
	return models.PermissionLevelEditor
}

func (s *CollaborationServer) cacheParticipant(ctx context.Context, sessionID string, participant *models.SessionParticipant) {
	// Cache in Redis
	key := fmt.Sprintf("session:%s:participants", sessionID)
	data, _ := json.Marshal(participant)
	s.redis.HSet(ctx, key, participant.UserID, data)
	s.redis.Expire(ctx, key, 24*time.Hour)
}

func (s *CollaborationServer) cleanupTask() {
	ticker := time.NewTicker(s.config.Session.CleanupInterval)
	defer ticker.Stop()
	
	for range ticker.C {
		ctx := context.Background()
		
		// Close expired sessions
		s.db.WithContext(ctx).
			Model(&models.CollaborationSession{}).
			Where("expires_at < ? AND status = ?", time.Now(), models.SessionStatusActive).
			Update("status", models.SessionStatusClosed)
		
		// Deactivate inactive participants
		s.db.WithContext(ctx).
			Model(&models.SessionParticipant{}).
			Where("last_activity_at < ? AND is_active = ?",
				time.Now().Add(-10*time.Minute), true).
			Update("is_active", false)
	}
}

// StartGRPCServer starts the gRPC server
func StartGRPCServer(server *CollaborationServer, cfg *config.Config) error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Server.GRPCPort))
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}
	
	s := grpc.NewServer()
	pb.RegisterCollaborationServiceServer(s, server)
	reflection.Register(s)
	
	log.Printf("Starting gRPC server on port %d", cfg.Server.GRPCPort)
	return s.Serve(lis)
}

// StartHTTPServer starts the HTTP server (including WebSocket)
func StartHTTPServer(server *CollaborationServer, cfg *config.Config) error {
	router := mux.NewRouter()
	
	// WebSocket endpoint
	router.HandleFunc("/ws", server.wsServer.HandleWebSocket)
	
	// Health check
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
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
	
	log.Printf("Starting HTTP server on port %d", cfg.Server.HTTPPort)
	return httpServer.ListenAndServe()
}
