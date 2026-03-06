package yjs

import (
	"fmt"
	"sync"
	"time"
)

// DocumentManager manages Yjs documents
type DocumentManager struct {
	documents map[string]*Document
	mu        sync.RWMutex
}

// Document represents a Yjs document
type Document struct {
	ID             string
	SessionID      string
	Updates        [][]byte
	State          []byte
	StateVector    []byte
	LastActivity   time.Time
	mu             sync.RWMutex
}

// NewDocumentManager creates a new document manager
func NewDocumentManager() *DocumentManager {
	dm := &DocumentManager{
		documents: make(map[string]*Document),
	}
	
	// Start cleanup goroutine
	go dm.cleanupTask()
	
	return dm
}

// GetOrCreateDocument gets or creates a document
func (dm *DocumentManager) GetOrCreateDocument(docID string, sessionID string) *Document {
	dm.mu.Lock()
	defer dm.mu.Unlock()
	
	if doc, ok := dm.documents[docID]; ok {
		doc.LastActivity = time.Now()
		return doc
	}
	
	doc := &Document{
		ID:           docID,
		SessionID:    sessionID,
		Updates:      make([][]byte, 0),
		LastActivity: time.Now(),
	}
	
	dm.documents[docID] = doc
	return doc
}

// GetDocument gets a document by ID
func (dm *DocumentManager) GetDocument(docID string) (*Document, bool) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()
	
	doc, ok := dm.documents[docID]
	return doc, ok
}

// RemoveDocument removes a document
func (dm *DocumentManager) RemoveDocument(docID string) {
	dm.mu.Lock()
	defer dm.mu.Unlock()
	
	delete(dm.documents, docID)
}

// ApplyUpdate applies an update to a document
func (dm *DocumentManager) ApplyUpdate(docID string, update []byte) error {
	doc := dm.GetOrCreateDocument(docID, "")
	
	doc.mu.Lock()
	defer doc.mu.Unlock()
	
	// Store the update
	doc.Updates = append(doc.Updates, update)
	doc.LastActivity = time.Now()
	
	// In a real implementation, you would use a Yjs library
	// to merge updates and compute the new state
	// For now, we'll just append the update
	if doc.State == nil {
		doc.State = update
	} else {
		// This is a simplified merge - real implementation would use Yjs
		doc.State = append(doc.State, update...)
	}
	
	return nil
}

// GetState returns the current document state
func (dm *DocumentManager) GetState(docID string) ([]byte, error) {
	doc, ok := dm.GetDocument(docID)
	if !ok {
		return nil, fmt.Errorf("document not found: %s", docID)
	}
	
	doc.mu.RLock()
	defer doc.mu.RUnlock()
	
	return doc.State, nil
}

// GetStateVector returns the state vector for a document
func (dm *DocumentManager) GetStateVector(docID string) ([]byte, error) {
	doc, ok := dm.GetDocument(docID)
	if !ok {
		return nil, fmt.Errorf("document not found: %s", docID)
	}
	
	doc.mu.RLock()
	defer doc.mu.RUnlock()
	
	// In a real implementation, this would compute the state vector
	// For now, return a simple placeholder
	if doc.StateVector == nil {
		return []byte{}, nil
	}
	return doc.StateVector, nil
}

// ComputeDiff computes the diff from a state vector
func (dm *DocumentManager) ComputeDiff(docID string, stateVector []byte) ([]byte, error) {
	doc, ok := dm.GetDocument(docID)
	if !ok {
		return nil, fmt.Errorf("document not found: %s", docID)
	}
	
	doc.mu.RLock()
	defer doc.mu.RUnlock()
	
	// In a real implementation, this would compute the diff
	// based on the provided state vector
	// For now, return the full state
	return doc.State, nil
}

// MergeUpdates merges multiple updates into one
func (dm *DocumentManager) MergeUpdates(updates [][]byte) ([]byte, error) {
	if len(updates) == 0 {
		return []byte{}, nil
	}
	
	// In a real implementation, this would use Yjs to merge updates
	// For now, just concatenate them
	var result []byte
	for _, update := range updates {
		result = append(result, update...)
	}
	return result, nil
}

// GetDocumentStats returns statistics for a document
func (dm *DocumentManager) GetDocumentStats(docID string) (map[string]interface{}, error) {
	doc, ok := dm.GetDocument(docID)
	if !ok {
		return nil, fmt.Errorf("document not found: %s", docID)
	}
	
	doc.mu.RLock()
	defer doc.mu.RUnlock()
	
	return map[string]interface{}{
		"update_count":   len(doc.Updates),
		"state_size":     len(doc.State),
		"last_activity":  doc.LastActivity,
	}, nil
}

// cleanupTask periodically cleans up inactive documents
func (dm *DocumentManager) cleanupTask() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	
	for range ticker.C {
		dm.cleanupInactiveDocuments()
	}
}

// cleanupInactiveDocuments removes documents inactive for more than 24 hours
func (dm *DocumentManager) cleanupInactiveDocuments() {
	dm.mu.Lock()
	defer dm.mu.Unlock()
	
	cutoff := time.Now().Add(-24 * time.Hour)
	for id, doc := range dm.documents {
		if doc.LastActivity.Before(cutoff) {
			delete(dm.documents, id)
		}
	}
}

// GetActiveDocumentCount returns the number of active documents
func (dm *DocumentManager) GetActiveDocumentCount() int {
	dm.mu.RLock()
	defer dm.mu.RUnlock()
	
	return len(dm.documents)
}
