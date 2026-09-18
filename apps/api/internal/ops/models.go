package ops

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Investigation struct {
	ID                   uuid.UUID
	IncidentID           uuid.UUID
	IncidentReference    string
	Status               string
	RequestedByUserID    *uuid.UUID
	ModelName            *string
	ModelVersion         *string
	RootCauseHypothesis  *string
	ReasoningSummary     *string
	Confidence           *float64
	RiskLevel            *string
	ToolUsage            json.RawMessage
	ErrorMessage         *string
	RequestedAt          time.Time
	StartedAt            *time.Time
	CompletedAt          *time.Time
	Evidence             []Evidence
}

type Evidence struct {
	ID         uuid.UUID
	ToolName   string
	SourceType string
	Summary    string
	ArtifactURI *string
	SourceRef  *uuid.UUID
	Metadata   json.RawMessage
	CapturedAt time.Time
}

type Recommendation struct {
	ID                    uuid.UUID
	IncidentID            uuid.UUID
	InvestigationID       *uuid.UUID
	ActionType            string
	Title                 string
	Rationale             string
	TargetServiceID       uuid.UUID
	Parameters            json.RawMessage
	Confidence            float64
	RiskLevel             string
	RequiredApprovalRole  string
	Status                string
	CreatedAt             time.Time
}

type Remediation struct {
	ID                  uuid.UUID
	IncidentID          uuid.UUID
	IncidentReference   string
	RecommendationID    uuid.UUID
	ServiceID           uuid.UUID
	ServiceSlug         string
	Status              string
	ActionType          string
	Parameters          json.RawMessage
	RequestedByUserID   *uuid.UUID
	AttemptNumber       int
	ResultSummary       *string
	VerificationStatus  string
	VerificationDetails json.RawMessage
	ErrorMessage        *string
	StartedAt           *time.Time
	CompletedAt         *time.Time
	CreatedAt           time.Time
	Approval            *Approval
}

type Approval struct {
	ID          uuid.UUID
	Decision    string
	ActorUserID *uuid.UUID
	Comment     *string
	DecidedAt   *time.Time
}

type Alert struct {
	ID          uuid.UUID
	ServiceID   uuid.UUID
	ServiceSlug string
	DetectorID  string
	Severity    string
	Status      string
	Title       string
	Summary     string
	StartedAt   time.Time
	Labels      json.RawMessage
}

type Deployment struct {
	ID          uuid.UUID
	ServiceID   uuid.UUID
	ServiceSlug string
	Environment string
	Version     string
	GitSHA      *string
	Status      string
	StartedAt   time.Time
	CompletedAt *time.Time
}

type ListFilter struct {
	IncidentID string
	Status     string
	ServiceID  string
	Limit      int
	Cursor     string
}
