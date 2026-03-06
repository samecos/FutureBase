package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SessionType represents the type of collaboration session
type SessionType string

const (
	SessionTypeDesign      SessionType = "design"
	SessionTypeReview      SessionType = "review"
	SessionTypePresentation SessionType = "presentation"
)

// SessionStatus represents the status of a collaboration session
type SessionStatus string

const (
	SessionStatusActive   SessionStatus = "active"
	SessionStatusPaused   SessionStatus = "paused"
	SessionStatusClosing  SessionStatus = "closing"
	SessionStatusClosed   SessionStatus = "closed"
)

// PermissionLevel represents user permission level in a session
type PermissionLevel string

const (
	PermissionLevelViewer    PermissionLevel = "viewer"
	PermissionLevelCommenter PermissionLevel = "commenter"
	PermissionLevelEditor    PermissionLevel = "editor"
	PermissionLevelAdmin     PermissionLevel = "admin"
	PermissionLevelOwner     PermissionLevel = "owner"
)

// OperationType represents the type of operation
type OperationType string

const (
	OperationTypeInsert          OperationType = "insert"
	OperationTypeUpdate          OperationType = "update"
	OperationTypeDelete          OperationType = "delete"
	OperationTypeTransform       OperationType = "transform"
	OperationTypePropertyChange  OperationType = "property_change"
	OperationTypeGeometryChange  OperationType = "geometry_change"
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

// CollaborationSession represents a collaboration session
type CollaborationSession struct {
	ID          string          `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	DocumentID  string          `gorm:"type:uuid;not null;index" json:"document_id"`
	TenantID    string          `gorm:"type:uuid;not null;index" json:"tenant_id"`
	SessionType SessionType     `gorm:"type:varchar(32);not null;default:'design'" json:"session_type"`
	Status      SessionStatus   `gorm:"type:varchar(32);not null;default:'active'" json:"status"`
	CreatedBy   string          `gorm:"type:uuid;not null" json:"created_by"`
	CreatedAt   time.Time       `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt   time.Time       `gorm:"not null;default:now()" json:"updated_at"`
	ExpiresAt   *time.Time      `json:"expires_at"`
	Metadata    JSONB           `gorm:"type:jsonb;default:'{}'" json:"metadata"`
	YjsState    []byte          `gorm:"type:bytea" json:"yjs_state"`
	ServerClock int64           `gorm:"default:0" json:"server_clock"`
	
	// Relationships
	Participants []SessionParticipant `gorm:"foreignKey:SessionID" json:"participants,omitempty"`
	Operations   []OperationLog       `gorm:"foreignKey:SessionID" json:"operations,omitempty"`
}

// TableName returns the table name
func (CollaborationSession) TableName() string {
	return "collaboration_sessions"
}

// BeforeCreate hook
func (s *CollaborationSession) BeforeCreate(tx *gorm.DB) error {
	if s.ID == "" {
		s.ID = uuid.New().String()
	}
	return nil
}

// SessionParticipant represents a participant in a session
type SessionParticipant struct {
	ID              string          `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	SessionID       string          `gorm:"type:uuid;not null;index:idx_session_user,unique" json:"session_id"`
	UserID          string          `gorm:"type:uuid;not null;index:idx_session_user,unique" json:"user_id"`
	UserName        string          `gorm:"type:varchar(255)" json:"user_name"`
	UserAvatar      string          `gorm:"type:varchar(500)" json:"user_avatar"`
	PermissionLevel PermissionLevel `gorm:"type:varchar(32);not null;default:'viewer'" json:"permission_level"`
	ClientType      string          `gorm:"type:varchar(32)" json:"client_type"`
	ClientVersion   string          `gorm:"type:varchar(32)" json:"client_version"`
	ClientPlatform  string          `gorm:"type:varchar(32)" json:"client_platform"`
	CursorPosition  JSONB           `gorm:"type:jsonb" json:"cursor_position"`
	SelectionRange  JSONB           `gorm:"type:jsonb" json:"selection_range"`
	JoinedAt        time.Time       `gorm:"not null;default:now()" json:"joined_at"`
	LastActivityAt  time.Time       `gorm:"not null;default:now()" json:"last_activity_at"`
	IsActive        bool            `gorm:"default:true;index" json:"is_active"`
	
	// Relationships
	Session CollaborationSession `gorm:"foreignKey:SessionID" json:"-"`
}

// TableName returns the table name
func (SessionParticipant) TableName() string {
	return "session_participants"
}

// BeforeCreate hook
func (p *SessionParticipant) BeforeCreate(tx *gorm.DB) error {
	if p.ID == "" {
		p.ID = uuid.New().String()
	}
	return nil
}

// OperationLog represents an operation in the operation log
type OperationLog struct {
	ID            string        `gorm:"primaryKey" json:"id"`
	SessionID     string        `gorm:"type:uuid;not null;index" json:"session_id"`
	OperationID   string        `gorm:"type:uuid;not null;default:gen_random_uuid()" json:"operation_id"`
	UserID        string        `gorm:"type:uuid;not null;index" json:"user_id"`
	ClientClock   int64         `gorm:"not null" json:"client_clock"`
	ServerClock   int64         `gorm:"not null;index" json:"server_clock"`
	OperationType OperationType `gorm:"type:varchar(32);not null" json:"operation_type"`
	TargetID      *string       `gorm:"type:uuid;index" json:"target_id"`
	OperationData JSONB         `gorm:"type:jsonb;not null" json:"operation_data"`
	YjsUpdate     []byte        `gorm:"type:bytea" json:"yjs_update"`
	Metadata      JSONB         `gorm:"type:jsonb;default:'{}'" json:"metadata"`
	IsUndone      bool          `gorm:"default:false;index" json:"is_undone"`
	UndoneAt      *time.Time    `json:"undone_at"`
	UndoneBy      *string       `gorm:"type:uuid" json:"undone_by"`
	CreatedAt     time.Time     `gorm:"not null;default:now()" json:"created_at"`
	
	// Relationships
	Session CollaborationSession `gorm:"foreignKey:SessionID" json:"-"`
}

// TableName returns the table name
func (OperationLog) TableName() string {
	return "operation_logs"
}

// SessionPermission represents permission settings for a session
type SessionPermission struct {
	ID              string          `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	SessionID       string          `gorm:"type:uuid;not null;index" json:"session_id"`
	UserID          string          `gorm:"type:uuid;not null" json:"user_id"`
	PermissionLevel PermissionLevel `gorm:"type:varchar(32);not null" json:"permission_level"`
	GrantedBy       string          `gorm:"type:uuid;not null" json:"granted_by"`
	GrantedAt       time.Time       `gorm:"not null;default:now()" json:"granted_at"`
	RevokedAt       *time.Time      `json:"revoked_at"`
	RevokedBy       *string         `gorm:"type:uuid" json:"revoked_by"`
	IsActive        bool            `gorm:"default:true" json:"is_active"`
	
	// Relationships
	Session CollaborationSession `gorm:"foreignKey:SessionID" json:"-"`
}

// TableName returns the table name
func (SessionPermission) TableName() string {
	return "session_permissions"
}

// OfflineOperation represents an offline operation queue
type OfflineOperation struct {
	ID            string    `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	SessionID     string    `gorm:"type:uuid;not null;index" json:"session_id"`
	UserID        string    `gorm:"type:uuid;not null;index" json:"user_id"`
	ClientClock   int64     `gorm:"not null" json:"client_clock"`
	OperationData JSONB     `gorm:"type:jsonb;not null" json:"operation_data"`
	YjsUpdate     []byte    `gorm:"type:bytea" json:"yjs_update"`
	CreatedAt     time.Time `gorm:"not null;default:now()" json:"created_at"`
	SyncedAt      *time.Time `json:"synced_at"`
	RetryCount    int       `gorm:"default:0" json:"retry_count"`
	ErrorMessage  string    `gorm:"type:text" json:"error_message"`
}

// TableName returns the table name
func (OfflineOperation) TableName() string {
	return "offline_operations"
}

// DocumentSnapshot represents a snapshot of a document state
type DocumentSnapshot struct {
	ID           string    `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	SessionID    string    `gorm:"type:uuid;not null;index" json:"session_id"`
	DocumentID   string    `gorm:"type:uuid;not null;index" json:"document_id"`
	TenantID     string    `gorm:"type:uuid;not null" json:"tenant_id"`
	CreatedBy    string    `gorm:"type:uuid;not null" json:"created_by"`
	ServerClock  int64     `gorm:"not null" json:"server_clock"`
	Description  string    `gorm:"type:varchar(500)" json:"description"`
	YjsState     []byte    `gorm:"type:bytea" json:"yjs_state"`
	StateSize    int64     `json:"state_size"`
	CreatedAt    time.Time `gorm:"not null;default:now()" json:"created_at"`
}

// TableName returns the table name
func (DocumentSnapshot) TableName() string {
	return "document_snapshots"
}

// AutoMigrate performs database migration
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&CollaborationSession{},
		&SessionParticipant{},
		&OperationLog{},
		&SessionPermission{},
		&OfflineOperation{},
		&DocumentSnapshot{},
	)
}
