package plugin

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// TestValidateInputs validates input arguments for correctness
func TestValidateInputs(t *testing.T) {
	tests := []struct {
		name      string
		args      Args
		expectErr bool
		errMsg    string
	}{
		{
			name: "Valid Inputs",
			args: Args{
				JSONReportDirectory:   "./testdata",
				FileIncludePattern:    "*.json",
				SortingMethod:         SortingMethodNatural,
				FailedFeaturesNumber:  2,
				FailedScenariosNumber: 3,
				FailedStepsNumber:     5,
			},
			expectErr: false,
		},
		{
			name: "Invalid Sorting Method",
			args: Args{
				SortingMethod: "INVALID",
			},
			expectErr: true,
			errMsg:    "invalid SortingMethod value",
		},
		{
			name: "Negative Thresholds",
			args: Args{
				FailedFeaturesNumber: -1,
			},
			expectErr: true,
			errMsg:    "threshold values must be non-negative",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateInputs(tc.args)
			if tc.expectErr {
				if err == nil || !strings.Contains(err.Error(), tc.errMsg) {
					t.Errorf("Expected error '%s', but got %v", tc.errMsg, err)
				}
			} else if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

// TestLocateFiles tests locating files with include/exclude patterns
func TestLocateFiles(t *testing.T) {

	tests := []struct {
		name           string
		directory      string
		includePattern string
		expectedErr    bool
		errMsg         string
	}{
		{
			name:           "Valid Directory and Pattern",
			directory:      "../testdata",
			includePattern: "*.json",
			expectedErr:    false,
		},
		{
			name:           "No Files Found",
			directory:      "./testdata",
			includePattern: "*.invalid",
			expectedErr:    true,
			errMsg:         "no files found matching",
		},
		{
			name:           "Invalid Directory",
			directory:      "./invalid",
			includePattern: "*.json",
			expectedErr:    true,
			errMsg:         "no files found matching the report filename pattern",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Convert relative path to absolute to avoid working directory issues
			absPath, err := filepath.Abs(tc.directory)
			if err != nil {
				t.Fatalf("Failed to get absolute path: %v", err)
			}
			t.Logf("Checking directory: %s", absPath)

			// Print directory contents for debugging
			entries, _ := os.ReadDir(tc.directory)
			t.Logf("Directory contents of %s:", tc.directory)
			for _, entry := range entries {
				t.Logf(" - %s", entry.Name())
			}

			// Run locateFiles function
			files, err := locateFiles(tc.directory, tc.includePattern, "")
			t.Logf("Files found: %v", files)

			// Expected error handling
			if tc.expectedErr {
				if err == nil || !strings.Contains(err.Error(), tc.errMsg) {
					t.Errorf("Expected error '%s', but got %v", tc.errMsg, err)
				}
			} else if err != nil {
				t.Errorf("Unexpected error: %v", err)
			} else if len(files) == 0 {
				t.Errorf("Expected JSON files to be found, but none were returned")
			}
		})
	}
}

// TestProcessFile tests file processing and JSON parsing
func TestProcessFile(t *testing.T) {
	tests := []struct {
		name      string
		filePath  string
		skipEmpty bool
		expectErr bool
		errMsg    string
		expected  Results
	}{
		{
			name:      "Valid Cucumber JSON Report",
			filePath:  "../testdata/cucumber_report.json",
			skipEmpty: false,
			expectErr: false,
			expected: Results{
				FeatureCount:         2,
				ScenarioCount:        4,
				StepCount:            12,
				PassedTests:          7,
				FailedTests:          3,
				SkippedTests:         2,
				TotalFailedFeatures:  2,
				TotalPassedFeatures:  0,
				TotalFailedScenarios: 3,
				TotalPassedScenarios: 1,
				DurationMS:           26587.899999999998,
				TotalPassedSteps:     7,
				TotalFailedSteps:     3,
				FailedSteps: []FailedStepDetails{
					{
						Feature:      "Browserstack test",
						Scenario:     "Can add the product in cart",
						Step:         "I click on orders",
						ErrorMessage: "Orders page did not load.",
					},
					{
						Feature:      "Browserstack test",
						Scenario:     "Search Wikipedia",
						Step:         "I should see BrowserStack page",
						ErrorMessage: "Expected page not found.",
					},
					{
						Feature:      "Payment Gateway",
						Scenario:     "Failed payment",
						Step:         "I enter invalid payment details",
						ErrorMessage: "Payment details are invalid.",
					},
				},
			},
		},
		{
			name:      "Empty File",
			filePath:  "../testdata/empty.json",
			skipEmpty: true,
			expectErr: false,
			expected:  Results{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := processFile(tc.filePath, tc.skipEmpty, Args{})
			if tc.expectErr {
				if err == nil || !strings.Contains(err.Error(), tc.errMsg) {
					t.Errorf("Expected error '%s', but got %v", tc.errMsg, err)
				}
			} else if err != nil {
				t.Errorf("Unexpected error: %v", err)
			} else if diff := cmp.Diff(tc.expected, result); diff != "" {
				t.Errorf("Results mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestExec tests overall report execution process
func TestExec(t *testing.T) {
	tests := []struct {
		name      string
		args      Args
		expectErr bool
		errMsg    string
	}{
		{
			name: "Valid Execution",
			args: Args{
				JSONReportDirectory: "../testdata",
				FileIncludePattern:  "*.json",
				SortingMethod:       SortingMethodNatural,
			},
			expectErr: false,
		},
		{
			name: "No JSON Reports Found",
			args: Args{
				JSONReportDirectory: "../testdata",
				FileIncludePattern:  "*.invalid",
			},
			expectErr: true,
			errMsg:    "failed to locate files: no files found matching the report filename pattern",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := Exec(context.Background(), tc.args)
			if tc.expectErr {
				if err == nil || !strings.Contains(err.Error(), tc.errMsg) {
					t.Errorf("Expected error '%s', but got %v", tc.errMsg, err)
				}
			} else if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

// TestValidateThresholds tests the threshold validation logic
func TestValidateThresholds(t *testing.T) {
	tests := []struct {
		name      string
		results   Results
		args      Args
		expectErr bool
		errMsg    string
	}{
		{
			name: "Passes All Thresholds",
			results: Results{
				FeatureCount: 10,
				FailedTests:  1,
				PassedTests:  9,
			},
			args: Args{
				FailedFeaturesNumber:  2,
				FailedScenariosNumber: 3,
				FailedStepsNumber:     5,
			},
			expectErr: false,
		},
		{
			name: "Failed Features Exceed Threshold",
			results: Results{
				FeatureCount: 10,
				FailedTests:  5,
			},
			args: Args{
				FailedFeaturesNumber: 4,
			},
			expectErr: true,
			errMsg:    "failed features count (5) exceeds the threshold (4)",
		},
		{
			name: "Failed Steps Percentage Exceeds",
			results: Results{
				StepCount:   100,
				FailedTests: 21,
			},
			args: Args{
				FailedStepsPercentage: 20.0,
			},
			expectErr: true,
			errMsg:    "failed steps percentage (21.00%) exceeds the threshold (20.00%)",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateThresholds(tc.results, tc.args)
			if tc.expectErr {
				if err == nil || !strings.Contains(err.Error(), tc.errMsg) {
					t.Errorf("Expected error '%s', but got %v", tc.errMsg, err)
				}
			} else if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}
