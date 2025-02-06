package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"
)

// Constants for Sorting Method
const (
	SortingMethodNatural      = "NATURAL"
	SortingMethodAlphabetical = "ALPHABETICAL"
)

// Args represents the plugin's configurable arguments.
type Args struct {
	FileIncludePattern          string  `envconfig:"PLUGIN_FILE_INCLUDE_PATTERN"`
	FileExcludePattern          string  `envconfig:"PLUGIN_FILE_EXCLUDE_PATTERN"`
	FailedAsNotFailingStatus    bool    `envconfig:"PLUGIN_FAILED_AS_NOT_FAILING_STATUS"`
	FailedFeaturesNumber        int     `envconfig:"PLUGIN_FAILED_FEATURES_NUMBER"`
	FailedFeaturesPercentage    float64 `envconfig:"PLUGIN_FAILED_FEATURES_PERCENTAGE"`
	FailedScenariosNumber       int     `envconfig:"PLUGIN_FAILED_SCENARIOS_NUMBER"`
	FailedScenariosPercentage   float64 `envconfig:"PLUGIN_FAILED_SCENARIOS_PERCENTAGE"`
	FailedStepsNumber           int     `envconfig:"PLUGIN_FAILED_STEPS_NUMBER"`
	FailedStepsPercentage       float64 `envconfig:"PLUGIN_FAILED_STEPS_PERCENTAGE"`
	JSONReportDirectory         string  `envconfig:"PLUGIN_JSON_REPORT_DIRECTORY"`
	MergeFeaturesById           bool    `envconfig:"PLUGIN_MERGE_FEATURES_BY_ID"`
	PendingAsNotFailingStatus   bool    `envconfig:"PLUGIN_PENDING_AS_NOT_FAILING_STATUS"`
	PendingStepsNumber          int     `envconfig:"PLUGIN_PENDING_STEPS_NUMBER"`
	PendingStepsPercentage      float64 `envconfig:"PLUGIN_PENDING_STEPS_PERCENTAGE"`
	SkipEmptyJSONFiles          bool    `envconfig:"PLUGIN_SKIP_EMPTY_JSON_FILES"`
	SkippedAsNotFailingStatus   bool    `envconfig:"PLUGIN_SKIPPED_AS_NOT_FAILING_STATUS"`
	SkippedStepsNumber          int     `envconfig:"PLUGIN_SKIPPED_STEPS_NUMBER"`
	SkippedStepsPercentage      float64 `envconfig:"PLUGIN_SKIPPED_STEPS_PERCENTAGE"`
	SortingMethod               string  `envconfig:"PLUGIN_SORTING_METHOD"`
	StopBuildOnFailedReport     bool    `envconfig:"PLUGIN_STOP_BUILD_ON_FAILED_REPORT"`
	UndefinedAsNotFailingStatus bool    `envconfig:"PLUGIN_UNDEFINED_AS_NOT_FAILING_STATUS"`
	UndefinedStepsNumber        int     `envconfig:"PLUGIN_UNDEFINED_STEPS_NUMBER"`
	UndefinedStepsPercentage    float64 `envconfig:"PLUGIN_UNDEFINED_STEPS_PERCENTAGE"`
	Level                       string  `envconfig:"PLUGIN_LOG_LEVEL"`
}

// ValidateInputs ensures the user inputs meet the plugin requirements.
func ValidateInputs(args Args) error {
	if args.FileIncludePattern == "" {
		args.FileIncludePattern = "**/*.json" // Default pattern
	}

	if args.FailedFeaturesNumber < 0 || args.FailedScenariosNumber < 0 || args.FailedStepsNumber < 0 ||
		args.PendingStepsNumber < 0 || args.SkippedStepsNumber < 0 || args.UndefinedStepsNumber < 0 {
		return errors.New("threshold values must be non-negative. Check the configured values")
	}

	// Set default SortingMethod to NATURAL if not provided
	if args.SortingMethod == "" {
		args.SortingMethod = SortingMethodNatural
	}

	// Validate SortingMethod input
	if args.SortingMethod != SortingMethodNatural && args.SortingMethod != SortingMethodAlphabetical {
		return fmt.Errorf("invalid SortingMethod value. It must be '%s' or '%s'", SortingMethodNatural, SortingMethodAlphabetical)
	}

	return nil
}

// Exec handles Cucumber JSON report processing and logs details.
func Exec(ctx context.Context, args Args) error {
	files, err := locateFiles(args.JSONReportDirectory, args.FileIncludePattern, args.FileExcludePattern)
	if err != nil {
		logger := logrus.WithError(err)
		logger.Error("Error locating files")
		return errors.New("failed to locate files: " + err.Error())
	}

	if len(files) == 0 {
		return errors.New("no Cucumber JSON report files found. Check the report file pattern")
	}

	var (
		resultsChan = make(chan Results, len(files))
		errorsChan  = make(chan error, len(files))
	)

	var wg sync.WaitGroup
	maxWorkers := 5 // Adjust this based on system capacity
	sem := make(chan struct{}, maxWorkers)

	for _, file := range files {
		wg.Add(1)
		sem <- struct{}{}
		go func(f string) {

			defer wg.Done()
			defer func() { <-sem }()
			res, err := processFile(f, args.SkipEmptyJSONFiles, args)
			if err != nil {
				errorsChan <- fmt.Errorf("failed to process file %s: %w", f, err)
				return
			}
			resultsChan <- res
		}(file)
	}
	wg.Wait()

	var aggregatedResults Results
	var skippedFiles []string

	var mu sync.Mutex
	for i := 0; i < len(files); i++ {
		select {
		case res := <-resultsChan:
			mu.Lock()
			aggregatedResults.FeatureCount += res.FeatureCount
			aggregatedResults.ScenarioCount += res.ScenarioCount
			aggregatedResults.StepCount += res.StepCount
			aggregatedResults.PassedTests += res.PassedTests
			aggregatedResults.FailedTests += res.FailedTests
			aggregatedResults.SkippedTests += res.SkippedTests
			aggregatedResults.PendingTests += res.PendingTests
			aggregatedResults.UndefinedTests += res.UndefinedTests
			aggregatedResults.DurationMS += res.DurationMS
			aggregatedResults.FailedSteps = append(aggregatedResults.FailedSteps, res.FailedSteps...)
			aggregatedResults.TotalFailedFeatures += res.TotalFailedFeatures
			aggregatedResults.TotalPassedFeatures += res.TotalPassedFeatures
			aggregatedResults.TotalFailedScenarios += res.TotalFailedScenarios
			aggregatedResults.TotalPassedScenarios += res.TotalPassedScenarios
			aggregatedResults.TotalFailedSteps += res.TotalFailedSteps
			aggregatedResults.TotalPassedSteps += res.TotalPassedSteps
			mu.Unlock()
		case err := <-errorsChan:
			logrus.Warn(err)
			if e, ok := err.(*os.PathError); ok {
				skippedFiles = append(skippedFiles, e.Path)
			}
		}
	}

	// Log skipped files
	if len(skippedFiles) > 0 {
		logrus.Warnf("Skipped %d files due to errors: %v", len(skippedFiles), skippedFiles)
	}

	// Log aggregated results
	logAggregatedResults(aggregatedResults)

	// Write stats to file
	writeTestStats(aggregatedResults, logrus.New())

	// Check if the build should be stopped due to failed tests
	if args.StopBuildOnFailedReport && aggregatedResults.FailedTests > 0 {
		logrus.Errorf("Build failed due to failed tests. Total failed tests: %d", aggregatedResults.FailedTests)
		return fmt.Errorf("build failed due to failed tests. Total failed tests: %d", aggregatedResults.FailedTests)
	}

	// Validate thresholds at the aggregate level
	if err := validateThresholds(aggregatedResults, args); err != nil {
		logger := logrus.WithFields(logrus.Fields{
			"Feature Count":  aggregatedResults.FeatureCount,
			"Scenario Count": aggregatedResults.ScenarioCount,
			"Step Count":     aggregatedResults.StepCount,
			"Failed":         aggregatedResults.FailedTests,
			"Skipped":        aggregatedResults.SkippedTests,
			"Pending":        aggregatedResults.PendingTests,
			"Undefined":      aggregatedResults.UndefinedTests,
		})
		logger.Error(err.Error())
		return err
	}

	return nil
}

// locateFiles identifies files matching the given pattern and checks read permissions.
func locateFiles(directory, includePattern, excludePattern string) ([]string, error) {
	matches, err := filepath.Glob(filepath.Join(directory, includePattern))
	if err != nil {
		logger := logrus.WithError(err).WithField("Pattern", includePattern)
		logger.Error("Error occurred while searching for files")
		return nil, errors.New("failed to search for files: " + err.Error())
	}

	logrus.Infof("Found %d files matching the pattern: %s", len(matches), includePattern)

	if len(matches) == 0 {
		return nil, errors.New("no files found matching the report filename pattern")
	}

	validFiles := []string{}
	for _, file := range matches {
		if fileInfo, err := os.Stat(file); err == nil {
			if fileInfo.Mode().Perm()&(1<<(uint(7))) != 0 {
				validFiles = append(validFiles, file)
			} else {
				logrus.Warnf("File found but not readable: %s", file)
			}
		} else {
			logrus.Warnf("Error accessing file: %s. Error: %v", file, err)
		}
	}

	logrus.Infof("Number of readable files: %d", len(validFiles))

	if len(validFiles) == 0 {
		return nil, errors.New("no readable files found matching the report filename pattern")
	}

	return validFiles, nil
}

// processFile reads a Cucumber JSON report and computes statistics.
func processFile(filename string, skipEmptyFiles bool, args Args) (Results, error) {
	logrus.Infof("Processing file: %s", filename)

	fileContent, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			logrus.Errorf("File not found: %s", filename)
			return Results{}, fmt.Errorf("file not found: %s", filename)
		}
		if os.IsPermission(err) {
			logrus.Errorf("Permission denied for file: %s", filename)
			return Results{}, fmt.Errorf("permission denied for file: %s", filename)
		}
		logrus.Errorf("Error opening file: %s. Error: %v", filename, err)
		return Results{}, fmt.Errorf("error opening file: %s. Error: %v", filename, err)
	}

	if skipEmptyFiles && len(fileContent) == 0 {
		logrus.Infof("Skipping empty file: %s", filename)
		return Results{}, nil
	}

	var features []Feature
	if err := json.Unmarshal(fileContent, &features); err != nil {
		logrus.WithError(err).WithField("File", filename).Error("Failed to parse Cucumber JSON")
		return Results{}, fmt.Errorf("failed to parse Cucumber JSON for file: %s. Error: %v", filename, err)
	}

	// Merge features by ID if required
	if args.MergeFeaturesById {
		features = mergeFeaturesById(features)
	}

	// Sort features if required
	if args.SortingMethod == SortingMethodAlphabetical {
		sortFeaturesAlphabetically(features)
	}

	return computeStats(features, args), nil
}

// mergeFeaturesById merges features with the same ID into a single feature.
func mergeFeaturesById(features []Feature) []Feature {
	featureMap := make(map[string]Feature)
	for _, feature := range features {
		if existingFeature, ok := featureMap[feature.ID]; ok {
			// Merge scenarios
			existingFeature.Elements = append(existingFeature.Elements, feature.Elements...)
			featureMap[feature.ID] = existingFeature
		} else {
			featureMap[feature.ID] = feature
		}
	}

	mergedFeatures := make([]Feature, 0, len(featureMap))
	for _, feature := range featureMap {
		mergedFeatures = append(mergedFeatures, feature)
	}

	return mergedFeatures
}

// sortFeaturesAlphabetically sorts features by name in alphabetical order.
func sortFeaturesAlphabetically(features []Feature) {
	sort.Slice(features, func(i, j int) bool {
		return strings.ToLower(features[i].Name) < strings.ToLower(features[j].Name)
	})
}

// computeStats computes statistics from the parsed Cucumber JSON report.
func computeStats(features []Feature, args Args) Results {
	results := Results{}

	for _, feature := range features {
		results.FeatureCount++
		featureFailed := false

		for _, element := range feature.Elements {
			results.ScenarioCount++
			scenarioFailed := false

			for _, step := range element.Steps {
				results.StepCount++
				switch step.Result.Status {
				case "passed":
					results.PassedTests++
					results.TotalPassedSteps++
				case "failed":
					if !args.FailedAsNotFailingStatus {
						results.FailedTests++
						results.TotalFailedSteps++
						scenarioFailed = true
						featureFailed = true
						results.FailedSteps = append(results.FailedSteps, FailedStepDetails{
							Feature:      feature.Name,
							Scenario:     element.Name,
							Step:         step.Name,
							ErrorMessage: step.Result.ErrorMessage,
						})
					}
				case "skipped":
					if !args.SkippedAsNotFailingStatus {
						results.SkippedTests++
					}
				case "pending":
					if !args.PendingAsNotFailingStatus {
						results.PendingTests++
					}
				case "undefined":
					if !args.UndefinedAsNotFailingStatus {
						results.UndefinedTests++
					}
				}
				results.DurationMS += float64(step.Result.Duration) / 1e6 // Convert nanoseconds to milliseconds
			}

			if scenarioFailed {
				results.TotalFailedScenarios++
			} else {
				results.TotalPassedScenarios++
			}
		}

		if featureFailed {
			results.TotalFailedFeatures++
		} else {
			results.TotalPassedFeatures++
		}
	}

	return results
}

// logAggregatedResults logs the aggregated results in a structured and informative way.
func logAggregatedResults(results Results) {
	logrus.Infof("\n===============================================\n")
	logrus.Infof("Cucumber Test Report Summary\n")
	logrus.Infof("===============================================\n")
	logrus.Infof("📁 Total Features: %d\n", results.FeatureCount)
	logrus.Infof("📄 Total Scenarios: %d\n", results.ScenarioCount)
	logrus.Infof("🔍 Total Steps: %d\n", results.StepCount)
	logrus.Infof("❌ Total Failed Features: %d\n", results.TotalFailedFeatures)
	logrus.Infof("❌ Total Failed Scenarios: %d\n", results.TotalFailedScenarios)
	logrus.Infof("❌ Total Failed Steps: %d\n", results.TotalFailedSteps)
	logrus.Infof("✅ Total Passed Features: %d\n", results.TotalPassedFeatures)
	logrus.Infof("✅ Total Passed Scenarios: %d\n", results.TotalPassedScenarios)
	logrus.Infof("✅ Total Passed Steps: %d\n", results.TotalPassedSteps)
	logrus.Infof("✅ Total Passed Tests: %d\n", results.PassedTests)
	logrus.Infof("❌ Total Failed Tests: %d\n", results.FailedTests)
	logrus.Infof("⏸️ Total Skipped Tests: %d\n", results.SkippedTests)
	logrus.Infof("🔄 Total Pending Tests: %d\n", results.PendingTests)
	logrus.Infof("❓ Total Undefined Tests: %d\n", results.UndefinedTests)
	logrus.Infof("⏱️ Total Duration: %.2f ms\n", results.DurationMS)
	logrus.Infof("===============================================\n")

	// Log failed step details
	if len(results.FailedSteps) > 0 {
		logrus.Infof("Failed Step Details:\n")
		logrus.Infof("-----------------------------------------------\n")
		for i, step := range results.FailedSteps {
			logrus.Infof("%d. Feature: %s\n", i+1, step.Feature)
			logrus.Infof("   Scenario: %s\n", step.Scenario)
			logrus.Infof("   Step: %s\n", step.Step)
			logrus.Infof("   Error: %s\n", step.ErrorMessage)
			logrus.Infof("-----------------------------------------------\n")
		}
	}
}

// validateThresholds validates test report thresholds based on aggregate results.
func validateThresholds(results Results, args Args) error {
	logrus.Infof("Threshold Validation:\n")
	logrus.Infof("-----------------------------------------------\n")

	// Validate absolute thresholds
	if args.FailedFeaturesNumber > 0 {
		if results.FailedTests > args.FailedFeaturesNumber {
			logrus.Infof("Failed Features: %d (Threshold: %d) ❌\n", results.FailedTests, args.FailedFeaturesNumber)
			return fmt.Errorf("failed features count (%d) exceeds the threshold (%d)", results.FailedTests, args.FailedFeaturesNumber)
		}
		logrus.Infof("Failed Features: %d (Threshold: %d) ✅\n", results.FailedTests, args.FailedFeaturesNumber)
	}

	if args.FailedScenariosNumber > 0 {
		if results.FailedTests > args.FailedScenariosNumber {
			logrus.Infof("Failed Scenarios: %d (Threshold: %d) ❌\n", results.FailedTests, args.FailedScenariosNumber)
			return fmt.Errorf("failed scenarios count (%d) exceeds the threshold (%d)", results.FailedTests, args.FailedScenariosNumber)
		}
		logrus.Infof("Failed Scenarios: %d (Threshold: %d) ✅\n", results.FailedTests, args.FailedScenariosNumber)
	}

	if args.FailedStepsNumber > 0 {
		if results.FailedTests > args.FailedStepsNumber {
			logrus.Infof("Failed Steps: %d (Threshold: %d) ❌\n", results.FailedTests, args.FailedStepsNumber)
			return fmt.Errorf("failed steps count (%d) exceeds the threshold (%d)", results.FailedTests, args.FailedStepsNumber)
		}
		logrus.Infof("Failed Steps: %d (Threshold: %d) ✅\n", results.FailedTests, args.FailedStepsNumber)
	}

	// Validate percentage thresholds
	if args.FailedFeaturesPercentage > 0 {
		failureRate := float64(results.FailedTests) / float64(results.FeatureCount) * 100
		if failureRate > args.FailedFeaturesPercentage {
			logrus.Infof("Failed Features Percentage: %.2f%% (Threshold: %.2f%%) ❌\n", failureRate, args.FailedFeaturesPercentage)
			return fmt.Errorf("failed features percentage (%.2f%%) exceeds the threshold (%.2f%%)", failureRate, args.FailedFeaturesPercentage)
		}
		logrus.Infof("Failed Features Percentage: %.2f%% (Threshold: %.2f%%) ✅\n", failureRate, args.FailedFeaturesPercentage)
	}

	if args.FailedScenariosPercentage > 0 {
		failureRate := float64(results.FailedTests) / float64(results.ScenarioCount) * 100
		if failureRate > args.FailedScenariosPercentage {
			logrus.Infof("Failed Scenarios Percentage: %.2f%% (Threshold: %.2f%%) ❌\n", failureRate, args.FailedScenariosPercentage)
			return fmt.Errorf("failed scenarios percentage (%.2f%%) exceeds the threshold (%.2f%%)", failureRate, args.FailedScenariosPercentage)
		}
		logrus.Infof("Failed Scenarios Percentage: %.2f%% (Threshold: %.2f%%) ✅\n", failureRate, args.FailedScenariosPercentage)
	}

	if args.FailedStepsPercentage > 0 {
		failureRate := float64(results.FailedTests) / float64(results.StepCount) * 100
		if failureRate > args.FailedStepsPercentage {
			logrus.Infof("Failed Steps Percentage: %.2f%% (Threshold: %.2f%%) ❌\n", failureRate, args.FailedStepsPercentage)
			return fmt.Errorf("failed steps percentage (%.2f%%) exceeds the threshold (%.2f%%)", failureRate, args.FailedStepsPercentage)
		}
		logrus.Infof("Failed Steps Percentage: %.2f%% (Threshold: %.2f%%) ✅\n", failureRate, args.FailedStepsPercentage)
	}

	// Validate pending steps thresholds
	if args.PendingStepsNumber > 0 {
		if results.PendingTests > args.PendingStepsNumber {
			logrus.Infof("Pending Steps: %d (Threshold: %d) ❌\n", results.PendingTests, args.PendingStepsNumber)
			return fmt.Errorf("pending steps count (%d) exceeds the threshold (%d)", results.PendingTests, args.PendingStepsNumber)
		}
		logrus.Infof("Pending Steps: %d (Threshold: %d) ✅\n", results.PendingTests, args.PendingStepsNumber)
	}

	if args.PendingStepsPercentage > 0 {
		pendingRate := float64(results.PendingTests) / float64(results.StepCount) * 100
		if pendingRate > args.PendingStepsPercentage {
			logrus.Infof("Pending Steps Percentage: %.2f%% (Threshold: %.2f%%) ❌\n", pendingRate, args.PendingStepsPercentage)
			return fmt.Errorf("pending steps percentage (%.2f%%) exceeds the threshold (%.2f%%)", pendingRate, args.PendingStepsPercentage)
		}
		logrus.Infof("Pending Steps Percentage: %.2f%% (Threshold: %.2f%%) ✅\n", pendingRate, args.PendingStepsPercentage)
	}

	// Validate skipped steps thresholds
	if args.SkippedStepsNumber > 0 {
		if results.SkippedTests > args.SkippedStepsNumber {
			logrus.Infof("Skipped Steps: %d (Threshold: %d) ❌\n", results.SkippedTests, args.SkippedStepsNumber)
			return fmt.Errorf("skipped steps count (%d) exceeds the threshold (%d)", results.SkippedTests, args.SkippedStepsNumber)
		}
		logrus.Infof("Skipped Steps: %d (Threshold: %d) ✅\n", results.SkippedTests, args.SkippedStepsNumber)
	}

	if args.SkippedStepsPercentage > 0 {
		skipRate := float64(results.SkippedTests) / float64(results.StepCount) * 100
		if skipRate > args.SkippedStepsPercentage {
			logrus.Infof("Skipped Steps Percentage: %.2f%% (Threshold: %.2f%%) ❌\n", skipRate, args.SkippedStepsPercentage)
			return fmt.Errorf("skipped steps percentage (%.2f%%) exceeds the threshold (%.2f%%)", skipRate, args.SkippedStepsPercentage)
		}
		logrus.Infof("Skipped Steps Percentage: %.2f%% (Threshold: %.2f%%) ✅\n", skipRate, args.SkippedStepsPercentage)
	}

	// Validate undefined steps thresholds
	if args.UndefinedStepsNumber > 0 {
		if results.UndefinedTests > args.UndefinedStepsNumber {
			logrus.Infof("Undefined Steps: %d (Threshold: %d) ❌\n", results.UndefinedTests, args.UndefinedStepsNumber)
			return fmt.Errorf("undefined steps count (%d) exceeds the threshold (%d)", results.UndefinedTests, args.UndefinedStepsNumber)
		}
		logrus.Infof("Undefined Steps: %d (Threshold: %d) ✅\n", results.UndefinedTests, args.UndefinedStepsNumber)
	}

	if args.UndefinedStepsPercentage > 0 {
		undefinedRate := float64(results.UndefinedTests) / float64(results.StepCount) * 100
		if undefinedRate > args.UndefinedStepsPercentage {
			logrus.Infof("Undefined Steps Percentage: %.2f%% (Threshold: %.2f%%) ❌\n", undefinedRate, args.UndefinedStepsPercentage)
			return fmt.Errorf("undefined steps percentage (%.2f%%) exceeds the threshold (%.2f%%)", undefinedRate, args.UndefinedStepsPercentage)
		}
		logrus.Infof("Undefined Steps Percentage: %.2f%% (Threshold: %.2f%%) ✅\n", undefinedRate, args.UndefinedStepsPercentage)
	}

	logrus.Infof("===============================================")
	return nil
}

// writeTestStats writes the test statistics to a file.
func writeTestStats(results Results, log *logrus.Logger) {
	// Calculate failure rate and skipped rate
	failureRate := 0.0
	if results.StepCount > 0 {
		failureRate = float64(results.FailedTests) / float64(results.StepCount) * 100
	}

	skippedRate := 0.0
	if results.StepCount > 0 {
		skippedRate = float64(results.SkippedTests) / float64(results.StepCount) * 100
	}

	// Prepare stats map
	statsMap := map[string]string{
		"FAILED_FEATURES":  strconv.Itoa(results.TotalFailedFeatures),
		"FAILED_SCENARIOS": strconv.Itoa(results.TotalFailedScenarios),
		"FAILED_STEPS":     strconv.Itoa(results.TotalFailedSteps),
		"PASSED_FEATURES":  strconv.Itoa(results.TotalPassedFeatures),
		"PASSED_SCENARIOS": strconv.Itoa(results.TotalPassedScenarios),
		"PASSED_STEPS":     strconv.Itoa(results.TotalPassedSteps),
		"SKIPPED_STEPS":    strconv.Itoa(results.SkippedTests),
		"PENDING_STEPS":    strconv.Itoa(results.PendingTests),
		"UNDEFINED_STEPS":  strconv.Itoa(results.UndefinedTests),
		"TOTAL_FEATURES":   strconv.Itoa(results.FeatureCount),
		"TOTAL_SCENARIOS":  strconv.Itoa(results.ScenarioCount),
		"TOTAL_STEPS":      strconv.Itoa(results.StepCount),
		"FAILURE_RATE":     fmt.Sprintf("%.2f", failureRate),
		"SKIPPED_RATE":     fmt.Sprintf("%.2f", skippedRate),
	}

	// Write stats to file
	for key, value := range statsMap {
		if err := WriteEnvToFile(key, value, log); err != nil {
			log.Errorf("Error writing %s: %s", key, err)
		}
	}
}

// WriteEnvToFile writes a key-value pair to the output file.
func WriteEnvToFile(key, value string, log *logrus.Logger) error {
	outputFile, err := os.OpenFile(os.Getenv("DRONE_OUTPUT"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Errorf("Failed to open output file: %v", err)
		return err
	}
	defer outputFile.Close()
	
	_, err = outputFile.WriteString(key + "=" + value + "\n")
	if err != nil {
		log.Errorf("Failed to write to env: %v", err)
		return err
	}
	return nil
}
