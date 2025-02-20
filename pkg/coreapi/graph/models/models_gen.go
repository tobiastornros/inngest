// Code generated by github.com/99designs/gqlgen, DO NOT EDIT.

package models

import (
	"fmt"
	"io"
	"strconv"
	"time"
)

type FunctionRunEvent interface {
	IsFunctionRunEvent()
}

type ActionVersionQuery struct {
	Dsn          string `json:"dsn"`
	VersionMajor *int   `json:"versionMajor"`
	VersionMinor *int   `json:"versionMinor"`
}

type Config struct {
	Execution *ExecutionConfig `json:"execution"`
}

type CreateActionVersionInput struct {
	Config string `json:"config"`
}

type DeployFunctionInput struct {
	Env    *Environment `json:"env"`
	Config string       `json:"config"`
	Live   *bool        `json:"live"`
}

type Event struct {
	ID           string         `json:"id"`
	Workspace    *Workspace     `json:"workspace"`
	Name         *string        `json:"name"`
	CreatedAt    *time.Time     `json:"createdAt"`
	Payload      *string        `json:"payload"`
	Schema       *string        `json:"schema"`
	Status       *EventStatus   `json:"status"`
	PendingRuns  *int           `json:"pendingRuns"`
	TotalRuns    *int           `json:"totalRuns"`
	Raw          *string        `json:"raw"`
	FunctionRuns []*FunctionRun `json:"functionRuns"`
}

type EventQuery struct {
	WorkspaceID string `json:"workspaceId"`
	EventID     string `json:"eventId"`
}

type EventsQuery struct {
	WorkspaceID string  `json:"workspaceId"`
	LastEventID *string `json:"lastEventId"`
}

type ExecutionConfig struct {
	Drivers *ExecutionDriversConfig `json:"drivers"`
}

type ExecutionDockerDriverConfig struct {
	Registry  *string `json:"registry"`
	Namespace *string `json:"namespace"`
}

type ExecutionDriversConfig struct {
	Docker *ExecutionDockerDriverConfig `json:"docker"`
}

type FunctionEvent struct {
	Workspace   *Workspace         `json:"workspace"`
	FunctionRun *FunctionRun       `json:"functionRun"`
	Type        *FunctionEventType `json:"type"`
	Output      *string            `json:"output"`
	CreatedAt   *time.Time         `json:"createdAt"`
}

func (FunctionEvent) IsFunctionRunEvent() {}

type FunctionRun struct {
	ID           string             `json:"id"`
	Name         *string            `json:"name"`
	Workspace    *Workspace         `json:"workspace"`
	Status       *FunctionRunStatus `json:"status"`
	WaitingFor   *StepEventWait     `json:"waitingFor"`
	PendingSteps *int               `json:"pendingSteps"`
	StartedAt    *time.Time         `json:"startedAt"`
	Timeline     []FunctionRunEvent `json:"timeline"`
	Event        *Event             `json:"event"`
}

type FunctionRunQuery struct {
	WorkspaceID   string `json:"workspaceId"`
	FunctionRunID string `json:"functionRunId"`
}

type FunctionRunsQuery struct {
	WorkspaceID string `json:"workspaceId"`
}

type StepEvent struct {
	Workspace   *Workspace     `json:"workspace"`
	FunctionRun *FunctionRun   `json:"functionRun"`
	StepID      *string        `json:"stepID"`
	Name        *string        `json:"name"`
	Type        *StepEventType `json:"type"`
	Output      *string        `json:"output"`
	CreatedAt   *time.Time     `json:"createdAt"`
	WaitingFor  *StepEventWait `json:"waitingFor"`
}

func (StepEvent) IsFunctionRunEvent() {}

type StepEventWait struct {
	EventName  *string   `json:"eventName"`
	Expression *string   `json:"expression"`
	ExpiryTime time.Time `json:"expiryTime"`
}

type UpdateActionVersionInput struct {
	Dsn          string `json:"dsn"`
	VersionMajor int    `json:"versionMajor"`
	VersionMinor int    `json:"versionMinor"`
	Enabled      *bool  `json:"enabled"`
}

type Workspace struct {
	ID string `json:"id"`
}

type EventStatus string

const (
	EventStatusRunning         EventStatus = "RUNNING"
	EventStatusCompleted       EventStatus = "COMPLETED"
	EventStatusPaused          EventStatus = "PAUSED"
	EventStatusFailed          EventStatus = "FAILED"
	EventStatusPartiallyFailed EventStatus = "PARTIALLY_FAILED"
	EventStatusNoFunctions     EventStatus = "NO_FUNCTIONS"
)

var AllEventStatus = []EventStatus{
	EventStatusRunning,
	EventStatusCompleted,
	EventStatusPaused,
	EventStatusFailed,
	EventStatusPartiallyFailed,
	EventStatusNoFunctions,
}

func (e EventStatus) IsValid() bool {
	switch e {
	case EventStatusRunning, EventStatusCompleted, EventStatusPaused, EventStatusFailed, EventStatusPartiallyFailed, EventStatusNoFunctions:
		return true
	}
	return false
}

func (e EventStatus) String() string {
	return string(e)
}

func (e *EventStatus) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}

	*e = EventStatus(str)
	if !e.IsValid() {
		return fmt.Errorf("%s is not a valid EventStatus", str)
	}
	return nil
}

func (e EventStatus) MarshalGQL(w io.Writer) {
	fmt.Fprint(w, strconv.Quote(e.String()))
}

type FunctionEventType string

const (
	FunctionEventTypeStarted   FunctionEventType = "STARTED"
	FunctionEventTypeCompleted FunctionEventType = "COMPLETED"
	FunctionEventTypeFailed    FunctionEventType = "FAILED"
	FunctionEventTypeCancelled FunctionEventType = "CANCELLED"
)

var AllFunctionEventType = []FunctionEventType{
	FunctionEventTypeStarted,
	FunctionEventTypeCompleted,
	FunctionEventTypeFailed,
	FunctionEventTypeCancelled,
}

func (e FunctionEventType) IsValid() bool {
	switch e {
	case FunctionEventTypeStarted, FunctionEventTypeCompleted, FunctionEventTypeFailed, FunctionEventTypeCancelled:
		return true
	}
	return false
}

func (e FunctionEventType) String() string {
	return string(e)
}

func (e *FunctionEventType) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}

	*e = FunctionEventType(str)
	if !e.IsValid() {
		return fmt.Errorf("%s is not a valid FunctionEventType", str)
	}
	return nil
}

func (e FunctionEventType) MarshalGQL(w io.Writer) {
	fmt.Fprint(w, strconv.Quote(e.String()))
}

type FunctionRunStatus string

const (
	FunctionRunStatusCompleted FunctionRunStatus = "COMPLETED"
	FunctionRunStatusFailed    FunctionRunStatus = "FAILED"
	FunctionRunStatusCancelled FunctionRunStatus = "CANCELLED"
	FunctionRunStatusRunning   FunctionRunStatus = "RUNNING"
)

var AllFunctionRunStatus = []FunctionRunStatus{
	FunctionRunStatusCompleted,
	FunctionRunStatusFailed,
	FunctionRunStatusCancelled,
	FunctionRunStatusRunning,
}

func (e FunctionRunStatus) IsValid() bool {
	switch e {
	case FunctionRunStatusCompleted, FunctionRunStatusFailed, FunctionRunStatusCancelled, FunctionRunStatusRunning:
		return true
	}
	return false
}

func (e FunctionRunStatus) String() string {
	return string(e)
}

func (e *FunctionRunStatus) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}

	*e = FunctionRunStatus(str)
	if !e.IsValid() {
		return fmt.Errorf("%s is not a valid FunctionRunStatus", str)
	}
	return nil
}

func (e FunctionRunStatus) MarshalGQL(w io.Writer) {
	fmt.Fprint(w, strconv.Quote(e.String()))
}

type StepEventType string

const (
	StepEventTypeScheduled StepEventType = "SCHEDULED"
	StepEventTypeStarted   StepEventType = "STARTED"
	StepEventTypeCompleted StepEventType = "COMPLETED"
	StepEventTypeErrored   StepEventType = "ERRORED"
	StepEventTypeFailed    StepEventType = "FAILED"
	StepEventTypeWaiting   StepEventType = "WAITING"
)

var AllStepEventType = []StepEventType{
	StepEventTypeScheduled,
	StepEventTypeStarted,
	StepEventTypeCompleted,
	StepEventTypeErrored,
	StepEventTypeFailed,
	StepEventTypeWaiting,
}

func (e StepEventType) IsValid() bool {
	switch e {
	case StepEventTypeScheduled, StepEventTypeStarted, StepEventTypeCompleted, StepEventTypeErrored, StepEventTypeFailed, StepEventTypeWaiting:
		return true
	}
	return false
}

func (e StepEventType) String() string {
	return string(e)
}

func (e *StepEventType) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}

	*e = StepEventType(str)
	if !e.IsValid() {
		return fmt.Errorf("%s is not a valid StepEventType", str)
	}
	return nil
}

func (e StepEventType) MarshalGQL(w io.Writer) {
	fmt.Fprint(w, strconv.Quote(e.String()))
}
