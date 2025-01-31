# drone-cucumber

## Building

Build the plugin binary:

```text
scripts/build.sh
```

Build the plugin image:

```text
docker build -t plugins/cucumber -f docker/Dockerfile .
```

## Testing

Execute the plugin from your current working directory:
## This plugin processes Cucumber JSON report files and logs the test results in the console. It supports various configurations for handling failed, skipped, pending, and undefined steps, as well as thresholds for failing the build based on the number or percentage of failures.
```
docker run --rm \
  -e PLUGIN_FILE_INCLUDE_PATTERN="**/*.json" \
  -e PLUGIN_FILE_EXCLUDE_PATTERN="" \
  -e PLUGIN_FAILED_AS_NOT_FAILING_STATUS=false \
  -e PLUGIN_FAILED_FEATURES_NUMBER=5 \
  -e PLUGIN_FAILED_FEATURES_PERCENTAGE=10.0 \
  -e PLUGIN_FAILED_SCENARIOS_NUMBER=10 \
  -e PLUGIN_FAILED_SCENARIOS_PERCENTAGE=20.0 \
  -e PLUGIN_FAILED_STEPS_NUMBER=20 \
  -e PLUGIN_FAILED_STEPS_PERCENTAGE=30.0 \
  -e PLUGIN_JSON_REPORT_DIRECTORY="./reports" \
  -e PLUGIN_MERGE_FEATURES_BY_ID=true \
  -e PLUGIN_PENDING_AS_NOT_FAILING_STATUS=false \
  -e PLUGIN_PENDING_STEPS_NUMBER=5 \
  -e PLUGIN_PENDING_STEPS_PERCENTAGE=10.0 \
  -e PLUGIN_SKIP_EMPTY_JSON_FILES=true \
  -e PLUGIN_SKIPPED_AS_NOT_FAILING_STATUS=false \
  -e PLUGIN_SKIPPED_STEPS_NUMBER=5 \
  -e PLUGIN_SKIPPED_STEPS_PERCENTAGE=10.0 \
  -e PLUGIN_SORTING_METHOD="NATURAL" \
  -e PLUGIN_STOP_BUILD_ON_FAILED_REPORT=true \
  -e PLUGIN_UNDEFINED_AS_NOT_FAILING_STATUS=false \
  -e PLUGIN_UNDEFINED_STEPS_NUMBER=5 \
  -e PLUGIN_UNDEFINED_STEPS_PERCENTAGE=10.0 \
  -e PLUGIN_LOG_LEVEL="info" \
  -v $(pwd):$(pwd) \
  plugins/cucumber
```
## Example Harness Step:
```
- step:
    identifier: cucumber-report-processing
    name: Cucumber Report Processing
    spec:
      image: plugins/cucumber
      settings:
        file_include_pattern: "**/*.json"
        file_exclude_pattern: ""
        failed_as_not_failing_status: false
        failed_features_number: 5
        failed_features_percentage: 10.0
        failed_scenarios_number: 10
        failed_scenarios_percentage: 20.0
        failed_steps_number: 20
        failed_steps_percentage: 30.0
        json_report_directory: "./reports"
        merge_features_by_id: true
        pending_as_not_failing_status: false
        pending_steps_number: 5
        pending_steps_percentage: 10.0
        skip_empty_json_files: true
        skipped_as_not_failing_status: false
        skipped_steps_number: 5
        skipped_steps_percentage: 10.0
        sorting_method: "NATURAL"
        stop_build_on_failed_report: true
        undefined_as_not_failing_status: false
        undefined_steps_number: 5
        undefined_steps_percentage: 10.0
        level: "info"
    timeout: ''
    type: Plugin
```

## Plugin Settings
- `PLUGIN_FILE_INCLUDE_PATTERN`
Description: The file name pattern to locate Cucumber JSON report files. Supports Ant-style patterns.
Example: **/*.json

- `PLUGIN_FILE_EXCLUDE_PATTERN`
Description: The file name pattern to exclude specific Cucumber JSON report files. Supports Ant-style patterns.
Example: **/excluded/*.json

- `PLUGIN_FAILED_AS_NOT_FAILING_STATUS`
Description: If true, failed steps will not be considered as failing status.
Example: false

- `PLUGIN_FAILED_FEATURES_NUMBER`
Description: Maximum number of failed features before the build is marked as FAILURE.
Example: 5

- `PLUGIN_FAILED_FEATURES_PERCENTAGE`
Description: Maximum percentage of failed features before the build is marked as FAILURE.
Example: 10.0

- `PLUGIN_FAILED_SCENARIOS_NUMBER`
Description: Maximum number of failed scenarios before the build is marked as FAILURE.
Example: 10

- `PLUGIN_FAILED_SCENARIOS_PERCENTAGE`
Description: Maximum percentage of failed scenarios before the build is marked as FAILURE.
Example: 20.0

- `PLUGIN_FAILED_STEPS_NUMBER`
Description: Maximum number of failed steps before the build is marked as FAILURE.
Example: 20

- `PLUGIN_FAILED_STEPS_PERCENTAGE`
Description: Maximum percentage of failed steps before the build is marked as FAILURE.
Example: 30.0

- `PLUGIN_JSON_REPORT_DIRECTORY`
Description: The directory where Cucumber JSON reports are located.
Example: ./reports

- `PLUGIN_MERGE_FEATURES_BY_ID`
Description: If true, features with the same ID will be merged into a single feature.
Example: true

- `PLUGIN_PENDING_AS_NOT_FAILING_STATUS`
Description: If true, pending steps will not be considered as failing status.
Example: false

- `PLUGIN_PENDING_STEPS_NUMBER`
Description: Maximum number of pending steps before the build is marked as FAILURE.
Example: 5

- `PLUGIN_PENDING_STEPS_PERCENTAGE`
Description: Maximum percentage of pending steps before the build is marked as FAILURE.
Example: 10.0

- `PLUGIN_SKIP_EMPTY_JSON_FILES`
Description: If true, empty JSON files will be skipped during processing.
Example: true

- `PLUGIN_SKIPPED_AS_NOT_FAILING_STATUS`
Description: If true, skipped steps will not be considered as failing status.
Example: false

- `PLUGIN_SKIPPED_STEPS_NUMBER`
Description: Maximum number of skipped steps before the build is marked as FAILURE.
Example: 5

- `PLUGIN_SKIPPED_STEPS_PERCENTAGE`
Description: Maximum percentage of skipped steps before the build is marked as FAILURE.
Example: 10.0

- `PLUGIN_SORTING_METHOD`
Description: Specifies the method for sorting features. Can be NATURAL or ALPHABETICAL.
Example: NATURAL

- `PLUGIN_STOP_BUILD_ON_FAILED_REPORT`
Description: If true, the build will be stopped if any report processing fails.
Example: true

- `PLUGIN_UNDEFINED_AS_NOT_FAILING_STATUS`
Description: If true, undefined steps will not be considered as failing status.
Example: false

- `PLUGIN_UNDEFINED_STEPS_NUMBER`
Description: Maximum number of undefined steps before the build is marked as FAILURE.
Example: 5

- `PLUGIN_UNDEFINED_STEPS_PERCENTAGE`
Description: Maximum percentage of undefined steps before the build is marked as FAILURE.
Example: 10.0

- `PLUGIN_LOG_LEVEL`
Description: Defines the plugin log level. Set this to debug to see detailed logs.
Example: info
	
