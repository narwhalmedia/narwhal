package nats

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	domainevents "github.com/narwhalmedia/narwhal/internal/domain/events"
)

// SagaState represents the state of a saga
type SagaState string

const (
	SagaStatePending     SagaState = "pending"
	SagaStateRunning     SagaState = "running"
	SagaStateCompleted   SagaState = "completed"
	SagaStateFailed      SagaState = "failed"
	SagaStateCompensating SagaState = "compensating"
	SagaStateCompensated SagaState = "compensated"
)

// Saga represents a distributed transaction
type Saga struct {
	ID            string                 `json:"id"`
	Type          string                 `json:"type"`
	State         SagaState              `json:"state"`
	CurrentStep   int                    `json:"current_step"`
	Data          map[string]interface{} `json:"data"`
	StartedAt     time.Time              `json:"started_at"`
	CompletedAt   *time.Time             `json:"completed_at,omitempty"`
	Error         string                 `json:"error,omitempty"`
	CompensatedAt *time.Time             `json:"compensated_at,omitempty"`
}

// SagaStep represents a step in a saga
type SagaStep interface {
	Name() string
	Execute(ctx context.Context, saga *Saga) error
	Compensate(ctx context.Context, saga *Saga) error
}

// SagaDefinition defines a saga workflow
type SagaDefinition struct {
	Type  string
	Steps []SagaStep
}

// SagaOrchestrator manages saga execution
type SagaOrchestrator struct {
	client      *Client
	publisher   *Publisher
	logger      *zap.Logger
	definitions map[string]*SagaDefinition
	store       SagaStore
}

// SagaStore persists saga state
type SagaStore interface {
	Save(ctx context.Context, saga *Saga) error
	Get(ctx context.Context, id string) (*Saga, error)
	UpdateState(ctx context.Context, id string, state SagaState) error
}

// NewSagaOrchestrator creates a new saga orchestrator
func NewSagaOrchestrator(client *Client, publisher *Publisher, store SagaStore, logger *zap.Logger) *SagaOrchestrator {
	return &SagaOrchestrator{
		client:      client,
		publisher:   publisher,
		logger:      logger.Named("saga"),
		definitions: make(map[string]*SagaDefinition),
		store:       store,
	}
}

// RegisterSaga registers a saga definition
func (o *SagaOrchestrator) RegisterSaga(definition *SagaDefinition) {
	o.definitions[definition.Type] = definition
	o.logger.Info("registered saga",
		zap.String("type", definition.Type),
		zap.Int("steps", len(definition.Steps)),
	)
}

// StartSaga begins a new saga execution
func (o *SagaOrchestrator) StartSaga(ctx context.Context, sagaType string, data map[string]interface{}) (*Saga, error) {
	definition, ok := o.definitions[sagaType]
	if !ok {
		return nil, fmt.Errorf("unknown saga type: %s", sagaType)
	}

	saga := &Saga{
		ID:          uuid.New().String(),
		Type:        sagaType,
		State:       SagaStatePending,
		CurrentStep: 0,
		Data:        data,
		StartedAt:   time.Now(),
	}

	// Save initial state
	if err := o.store.Save(ctx, saga); err != nil {
		return nil, fmt.Errorf("failed to save saga: %w", err)
	}

	// Publish saga started event
	event := &SagaStartedEvent{
		BaseEvent: domainevents.NewBaseEvent(
			uuid.MustParse(saga.ID),
			"Saga",
			"SagaStarted",
			1,
		),
		SagaType: sagaType,
		Data:     data,
	}

	if err := o.publisher.PublishEvent(ctx, event); err != nil {
		o.logger.Error("failed to publish saga started event",
			zap.Error(err),
			zap.String("saga_id", saga.ID),
		)
	}

	// Start execution
	go o.executeSaga(context.Background(), saga, definition)

	return saga, nil
}

// executeSaga executes saga steps
func (o *SagaOrchestrator) executeSaga(ctx context.Context, saga *Saga, definition *SagaDefinition) {
	// Update state to running
	saga.State = SagaStateRunning
	if err := o.store.UpdateState(ctx, saga.ID, SagaStateRunning); err != nil {
		o.logger.Error("failed to update saga state",
			zap.Error(err),
			zap.String("saga_id", saga.ID),
		)
		return
	}

	// Execute steps
	for i, step := range definition.Steps {
		saga.CurrentStep = i

		o.logger.Info("executing saga step",
			zap.String("saga_id", saga.ID),
			zap.String("step", step.Name()),
			zap.Int("step_number", i+1),
		)

		// Execute step
		if err := step.Execute(ctx, saga); err != nil {
			o.logger.Error("saga step failed",
				zap.Error(err),
				zap.String("saga_id", saga.ID),
				zap.String("step", step.Name()),
			)

			// Start compensation
			o.compensateSaga(ctx, saga, definition, i, err)
			return
		}

		// Save progress
		if err := o.store.Save(ctx, saga); err != nil {
			o.logger.Error("failed to save saga progress",
				zap.Error(err),
				zap.String("saga_id", saga.ID),
			)
		}

		// Publish step completed event
		o.publishStepEvent(ctx, saga, step.Name(), "completed")
	}

	// All steps completed successfully
	now := time.Now()
	saga.State = SagaStateCompleted
	saga.CompletedAt = &now

	if err := o.store.Save(ctx, saga); err != nil {
		o.logger.Error("failed to save completed saga",
			zap.Error(err),
			zap.String("saga_id", saga.ID),
		)
	}

	// Publish saga completed event
	o.publishSagaEvent(ctx, saga, "SagaCompleted")

	o.logger.Info("saga completed successfully",
		zap.String("saga_id", saga.ID),
		zap.Duration("duration", time.Since(saga.StartedAt)),
	)
}

// compensateSaga executes compensation steps
func (o *SagaOrchestrator) compensateSaga(ctx context.Context, saga *Saga, definition *SagaDefinition, failedStep int, originalErr error) {
	saga.State = SagaStateCompensating
	saga.Error = originalErr.Error()

	if err := o.store.Save(ctx, saga); err != nil {
		o.logger.Error("failed to save compensating saga",
			zap.Error(err),
			zap.String("saga_id", saga.ID),
		)
	}

	// Compensate in reverse order
	for i := failedStep - 1; i >= 0; i-- {
		step := definition.Steps[i]

		o.logger.Info("compensating saga step",
			zap.String("saga_id", saga.ID),
			zap.String("step", step.Name()),
			zap.Int("step_number", i+1),
		)

		if err := step.Compensate(ctx, saga); err != nil {
			o.logger.Error("compensation failed",
				zap.Error(err),
				zap.String("saga_id", saga.ID),
				zap.String("step", step.Name()),
			)
			// Continue compensating other steps
		}

		// Publish compensation event
		o.publishStepEvent(ctx, saga, step.Name(), "compensated")
	}

	// Update final state
	now := time.Now()
	saga.State = SagaStateCompensated
	saga.CompensatedAt = &now

	if err := o.store.Save(ctx, saga); err != nil {
		o.logger.Error("failed to save compensated saga",
			zap.Error(err),
			zap.String("saga_id", saga.ID),
		)
	}

	// Publish saga compensated event
	o.publishSagaEvent(ctx, saga, "SagaCompensated")

	o.logger.Info("saga compensated",
		zap.String("saga_id", saga.ID),
		zap.String("error", originalErr.Error()),
	)
}

// publishStepEvent publishes a saga step event
func (o *SagaOrchestrator) publishStepEvent(ctx context.Context, saga *Saga, stepName, status string) {
	event := &SagaStepEvent{
		BaseEvent: domainevents.NewBaseEvent(
			uuid.MustParse(saga.ID),
			"Saga",
			"SagaStep"+status,
			1,
		),
		SagaType: saga.Type,
		StepName: stepName,
		Status:   status,
	}

	if err := o.publisher.PublishEvent(ctx, event); err != nil {
		o.logger.Error("failed to publish saga step event",
			zap.Error(err),
			zap.String("saga_id", saga.ID),
			zap.String("step", stepName),
		)
	}
}

// publishSagaEvent publishes a saga lifecycle event
func (o *SagaOrchestrator) publishSagaEvent(ctx context.Context, saga *Saga, eventType string) {
	event := &SagaLifecycleEvent{
		BaseEvent: domainevents.NewBaseEvent(
			uuid.MustParse(saga.ID),
			"Saga",
			eventType,
			1,
		),
		SagaType: saga.Type,
		State:    string(saga.State),
		Error:    saga.Error,
	}

	if err := o.publisher.PublishEvent(ctx, event); err != nil {
		o.logger.Error("failed to publish saga event",
			zap.Error(err),
			zap.String("saga_id", saga.ID),
			zap.String("event_type", eventType),
		)
	}
}

// Saga Events

// SagaStartedEvent is published when a saga starts
type SagaStartedEvent struct {
	domainevents.BaseEvent
	SagaType string                 `json:"saga_type"`
	Data     map[string]interface{} `json:"data"`
}

// SagaStepEvent is published for saga step transitions
type SagaStepEvent struct {
	domainevents.BaseEvent
	SagaType string `json:"saga_type"`
	StepName string `json:"step_name"`
	Status   string `json:"status"`
}

// SagaLifecycleEvent is published for saga lifecycle changes
type SagaLifecycleEvent struct {
	domainevents.BaseEvent
	SagaType string `json:"saga_type"`
	State    string `json:"state"`
	Error    string `json:"error,omitempty"`
}