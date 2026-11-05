package historydb

import (
	"database/sql"
	"time"
)

type Store struct{ db *sql.DB }

type Run struct {
	ID                int64
	Collection        string
	RequestName       string
	TargetDomain      string
	URL               string
	Method            string
	StatusCode        int
	ResponseTime      int64
	ResponseBody      string
	ExecutedAt        time.Time
	CapturedVariables []CapturedVariable
}

type CapturedVariable struct {
	DomainScope string
	Name        string
	Value       string
}

const (
	maxCapturedVariables = 1000
	maxVariableDepth     = 32
)

