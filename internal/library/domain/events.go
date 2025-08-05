package domain

import (
	"time"

	"github.com/google/uuid"
	"github.com/narwhalmedia/narwhal/pkg/models"
)

// LibraryCreatedEvent is published when a library is created
type LibraryCreatedEvent struct {
	Library   *Library
	timestamp int64
}

func NewLibraryCreatedEvent(library *Library) *LibraryCreatedEvent {
	return &LibraryCreatedEvent{
		Library:   library,
		timestamp: time.Now().Unix(),
	}
}

func (e *LibraryCreatedEvent) EventType() string {
	return "library.created"
}

func (e *LibraryCreatedEvent) Timestamp() int64 {
	return e.timestamp
}

func (e *LibraryCreatedEvent) AggregateID() string {
	return e.Library.ID.String()
}

// LibraryUpdatedEvent is published when a library is updated
type LibraryUpdatedEvent struct {
	Library   *Library
	timestamp int64
}

func NewLibraryUpdatedEvent(library *Library) *LibraryUpdatedEvent {
	return &LibraryUpdatedEvent{
		Library:   library,
		timestamp: time.Now().Unix(),
	}
}

func (e *LibraryUpdatedEvent) EventType() string {
	return "library.updated"
}

func (e *LibraryUpdatedEvent) Timestamp() int64 {
	return e.timestamp
}

func (e *LibraryUpdatedEvent) AggregateID() string {
	return e.Library.ID.String()
}

// LibraryDeletedEvent is published when a library is deleted
type LibraryDeletedEvent struct {
	LibraryID uuid.UUID
	timestamp int64
}

func NewLibraryDeletedEvent(libraryID uuid.UUID) *LibraryDeletedEvent {
	return &LibraryDeletedEvent{
		LibraryID: libraryID,
		timestamp: time.Now().Unix(),
	}
}

func (e *LibraryDeletedEvent) EventType() string {
	return "library.deleted"
}

func (e *LibraryDeletedEvent) Timestamp() int64 {
	return e.timestamp
}

func (e *LibraryDeletedEvent) AggregateID() string {
	return e.LibraryID.String()
}

// LibraryScanCompletedEvent is published when a library scan is completed
type LibraryScanCompletedEvent struct {
	Library      *Library
	NewFiles     int
	UpdatedFiles int
	timestamp    int64
}

func NewLibraryScanCompletedEvent(library *Library, newFiles, updatedFiles int) *LibraryScanCompletedEvent {
	return &LibraryScanCompletedEvent{
		Library:      library,
		NewFiles:     newFiles,
		UpdatedFiles: updatedFiles,
		timestamp:    time.Now().Unix(),
	}
}

func (e *LibraryScanCompletedEvent) EventType() string {
	return "library.scan.completed"
}

func (e *LibraryScanCompletedEvent) Timestamp() int64 {
	return e.timestamp
}

func (e *LibraryScanCompletedEvent) AggregateID() string {
	return e.Library.ID.String()
}

// MediaAddedEvent is published when a media item is added
type MediaAddedEvent struct {
	Media     *models.Media
	timestamp int64
}

func NewMediaAddedEvent(media *models.Media) *MediaAddedEvent {
	return &MediaAddedEvent{
		Media:     media,
		timestamp: time.Now().Unix(),
	}
}

func (e *MediaAddedEvent) EventType() string {
	return "media.added"
}

func (e *MediaAddedEvent) Timestamp() int64 {
	return e.timestamp
}

func (e *MediaAddedEvent) AggregateID() string {
	return e.Media.ID.String()
}

// MediaUpdatedEvent is published when a media item is updated
type MediaUpdatedEvent struct {
	Media     *models.Media
	timestamp int64
}

func NewMediaUpdatedEvent(media *models.Media) *MediaUpdatedEvent {
	return &MediaUpdatedEvent{
		Media:     media,
		timestamp: time.Now().Unix(),
	}
}

func (e *MediaUpdatedEvent) EventType() string {
	return "media.updated"
}

func (e *MediaUpdatedEvent) Timestamp() int64 {
	return e.timestamp
}

func (e *MediaUpdatedEvent) AggregateID() string {
	return e.Media.ID.String()
}

// MediaDeletedEvent is published when a media item is deleted
type MediaDeletedEvent struct {
	MediaID   string
	timestamp int64
}

func NewMediaDeletedEvent(mediaID string) *MediaDeletedEvent {
	return &MediaDeletedEvent{
		MediaID:   mediaID,
		timestamp: time.Now().Unix(),
	}
}

func (e *MediaDeletedEvent) EventType() string {
	return "media.deleted"
}

func (e *MediaDeletedEvent) Timestamp() int64 {
	return e.timestamp
}

func (e *MediaDeletedEvent) AggregateID() string {
	return e.MediaID
}
