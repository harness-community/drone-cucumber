package plugin

// Feature represents a single feature in the Cucumber JSON report.
type Feature struct {
	ID          string    `json:"id"`
	URI         string    `json:"uri"`
	Keyword     string    `json:"keyword"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Line        int       `json:"line"`
	Elements    []Element `json:"elements"`
}

// Element represents a scenario or scenario outline in the Cucumber JSON report.
type Element struct {
	ID          string `json:"id"`
	Keyword     string `json:"keyword"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Line        int    `json:"line"`
	Type        string `json:"type"`
	Steps       []Step `json:"steps"`
}

// Step represents a single step in a scenario.
type Step struct {
	Keyword string `json:"keyword"`
	Name    string `json:"name"`
	Line    int    `json:"line"`
	Result  Result `json:"result"`
}

// Result represents the result of a step execution.
type Result struct {
	Status       string `json:"status"`
	Duration     int64  `json:"duration"`
	ErrorMessage string `json:"error_message,omitempty"`
}

// Results represents the aggregated results of the Cucumber report.
type Results struct {
	FeatureCount         int                 // Total number of features
	ScenarioCount        int                 // Total number of scenarios
	StepCount            int                 // Total number of steps
	PassedTests          int                 // Number of passed steps
	FailedTests          int                 // Number of failed steps
	SkippedTests         int                 // Number of skipped steps
	PendingTests         int                 // Number of pending steps
	UndefinedTests       int                 // Number of undefined steps
	DurationMS           float64             // Total duration in milliseconds
	FailedSteps          []FailedStepDetails // Details of failed steps
	TotalFailedFeatures  int                 // Total number of failed features
	TotalPassedFeatures  int                 // Total number of passed features
	TotalFailedScenarios int                 // Total number of failed scenarios
	TotalPassedScenarios int                 // Total number of passed scenarios
	TotalFailedSteps     int                 // Total number of failed steps
	TotalPassedSteps     int                 // Total number of passed steps
}

// FailedStepDetails represents details of a failed step.
type FailedStepDetails struct {
	Feature      string
	Scenario     string
	Step         string
	ErrorMessage string
}
