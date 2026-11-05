package model


type SharedState struct {
	TabActive        int
	Widht            int
	Height           int
	SessionVariables map[string]string
	GlobalScope      string
	ActiveEnv        string            // name of the currently active environment
	EnvVars          map[string]string // variables loaded from the active environment
}

func NewSharedState() *SharedState {
	return &SharedState{
		TabActive:        0,
		SessionVariables: make(map[string]string),
	}
}

