package yjs

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewManager(t *testing.T) {
	manager := NewManager()
	assert.NotNil(t, manager)
	assert.NotNil(t, manager.documents)
	assert.NotNil(t, manager.clients)
}

func TestManager_CreateDocument(t *testing.T) {
	manager := NewManager()
	
	docID := uuid.New().String()
	projectID := uuid.New().String()
	creatorID := uuid.New().String()
	
	doc, err := manager.CreateDocument(docID, projectID, creatorID)
	
	require.NoError(t, err)
	assert.NotNil(t, doc)
	assert.Equal(t, docID, doc.ID)
	assert.Equal(t, projectID, doc.ProjectID)
	assert.Equal(t, creatorID, doc.CreatorID)
	assert.Equal(t, DocumentTypeDrawing, doc.Type)
	assert.False(t, doc.IsLocked)
	assert.WithinDuration(t, time.Now(), doc.CreatedAt, time.Second)
}

func TestManager_CreateDocument_DuplicateID(t *testing.T) {
	manager := NewManager()
	
	docID := uuid.New().String()
	projectID := uuid.New().String()
	creatorID := uuid.New().String()
	
	// First creation should succeed
	_, err := manager.CreateDocument(docID, projectID, creatorID)
	require.NoError(t, err)
	
	// Second creation with same ID should fail
	_, err = manager.CreateDocument(docID, projectID, creatorID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestManager_GetDocument(t *testing.T) {
	manager := NewManager()
	
	docID := uuid.New().String()
	projectID := uuid.New().String()
	creatorID := uuid.New().String()
	
	// Create document
	_, err := manager.CreateDocument(docID, projectID, creatorID)
	require.NoError(t, err)
	
	// Get existing document
	doc, err := manager.GetDocument(docID)
	require.NoError(t, err)
	assert.NotNil(t, doc)
	assert.Equal(t, docID, doc.ID)
	
	// Get non-existent document
	_, err = manager.GetDocument(uuid.New().String())
	assert.Error(t, err)
}

func TestManager_LockUnlockDocument(t *testing.T) {
	manager := NewManager()
	
	docID := uuid.New().String()
	projectID := uuid.New().String()
	creatorID := uuid.New().String()
	
	_, err := manager.CreateDocument(docID, projectID, creatorID)
	require.NoError(t, err)
	
	// Lock document
	err = manager.LockDocument(docID, creatorID)
	require.NoError(t, err)
	
	doc, _ := manager.GetDocument(docID)
	assert.True(t, doc.IsLocked)
	assert.Equal(t, creatorID, doc.LockedBy)
	
	// Unlock document
	err = manager.UnlockDocument(docID, creatorID)
	require.NoError(t, err)
	
	doc, _ = manager.GetDocument(docID)
	assert.False(t, doc.IsLocked)
	assert.Empty(t, doc.LockedBy)
}

func TestManager_LockDocument_WrongUser(t *testing.T) {
	manager := NewManager()
	
	docID := uuid.New().String()
	projectID := uuid.New().String()
	creatorID := uuid.New().String()
	otherUserID := uuid.New().String()
	
	_, err := manager.CreateDocument(docID, projectID, creatorID)
	require.NoError(t, err)
	
	// Lock by creator
	err = manager.LockDocument(docID, creatorID)
	require.NoError(t, err)
	
	// Try to unlock by other user
	err = manager.UnlockDocument(docID, otherUserID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not authorized")
}

func TestManager_ApplyUpdate(t *testing.T) {
	manager := NewManager()
	
	docID := uuid.New().String()
	projectID := uuid.New().String()
	creatorID := uuid.New().String()
	
	doc, err := manager.CreateDocument(docID, projectID, creatorID)
	require.NoError(t, err)
	
	update := &DocumentUpdate{
		Type:      UpdateTypeStateVector,
		Data:      []byte("test data"),
		Timestamp: time.Now(),
		UserID:    creatorID,
	}
	
	err = manager.ApplyUpdate(docID, update)
	require.NoError(t, err)
	
	// Check that version was incremented
	updatedDoc, _ := manager.GetDocument(docID)
	assert.Equal(t, doc.Version+1, updatedDoc.Version)
	assert.WithinDuration(t, time.Now(), updatedDoc.UpdatedAt, time.Second)
}

func TestManager_GetDocumentHistory(t *testing.T) {
	manager := NewManager()
	
	docID := uuid.New().String()
	projectID := uuid.New().String()
	creatorID := uuid.New().String()
	
	_, err := manager.CreateDocument(docID, projectID, creatorID)
	require.NoError(t, err)
	
	// Add some updates
	for i := 0; i < 5; i++ {
		update := &DocumentUpdate{
			Type:      UpdateTypeInsert,
			Data:      []byte("update data"),
			Timestamp: time.Now(),
			UserID:    creatorID,
		}
		err := manager.ApplyUpdate(docID, update)
		require.NoError(t, err)
	}
	
	// Get history
	history, err := manager.GetDocumentHistory(docID, 0, 10)
	require.NoError(t, err)
	assert.NotEmpty(t, history)
}

func TestManager_ListDocumentsByProject(t *testing.T) {
	manager := NewManager()
	
	projectID := uuid.New().String()
	creatorID := uuid.New().String()
	
	// Create multiple documents in the same project
	for i := 0; i < 3; i++ {
		_, err := manager.CreateDocument(uuid.New().String(), projectID, creatorID)
		require.NoError(t, err)
	}
	
	// Create document in different project
	otherProjectID := uuid.New().String()
	_, err := manager.CreateDocument(uuid.New().String(), otherProjectID, creatorID)
	require.NoError(t, err)
	
	// List documents
	docs, err := manager.ListDocumentsByProject(projectID)
	require.NoError(t, err)
	assert.Len(t, docs, 3)
}

func TestManager_DeleteDocument(t *testing.T) {
	manager := NewManager()
	
	docID := uuid.New().String()
	projectID := uuid.New().String()
	creatorID := uuid.New().String()
	
	_, err := manager.CreateDocument(docID, projectID, creatorID)
	require.NoError(t, err)
	
	// Delete document
	err = manager.DeleteDocument(docID)
	require.NoError(t, err)
	
	// Verify document is deleted
	_, err = manager.GetDocument(docID)
	assert.Error(t, err)
}
