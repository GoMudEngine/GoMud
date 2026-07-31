# Command Line Flags System Context

## Overview

The `internal/flags` package provides command-line argument processing for the GoMud game engine. It handles utility flags for version display and port availability searching, supporting server deployment and troubleshooting scenarios.

## Key Components

### Core Files
- **flags.go**: Command-line flag processing and utility functions

### Key Functions

#### Main Flag Handler
- **HandleFlags(serverVersion string)**: Primary flag processing function
  - Parses command-line arguments using Go's flag package
  - Handles version display and port search functionality
  - Exits the program after processing utility flags

#### Utility Functions
- **doPortSearch(portRangeStr string)**: Port availability scanning utility
  - Parses port range specification (format: "start-end")
  - Searches for first 10 available ports in specified range
  - Validates port range parameters and provides error feedback
  - Logs search progress and results

- **isPortInUse(port int) bool**: Port availability checker
  - Attempts to bind to specified port using TCP listener
  - Returns true if port is in use (bind fails)
  - Returns false if port is available (bind succeeds)
  - Properly closes test connections to avoid resource leaks

## Supported Flags

### Version Flag
- **Flag**: `-version`
- **Type**: Boolean
- **Purpose**: Display current binary version information
- **Behavior**: Prints version string and exits with code 0

### Port Search Flag
- **Flag**: `-port-search`
- **Type**: String
- **Format**: `"start-end"` (e.g., "30000-40000")
- **Purpose**: Find available ports in specified range
- **Behavior**: Searches for first 10 open ports and exits with code 0

## Dependencies

### Internal Dependencies
- `internal/mudlog`: For logging search results and error messages

### External Dependencies
- Standard library: `flag`, `fmt`, `net`, `os`, `strconv`, `strings`

## Usage Patterns

### Version Display
```bash
./gomud -version
# Output: v1.2.3
```

### Port Range Searching
```bash
./gomud -port-search=30000-40000
# Searches for 10 available ports between 30000 and 40000
```

### Integration in Main
```go
func main() {
    flags.HandleFlags("v1.2.3")
    // Continue with normal server startup if no utility flags were used
}
```

## Port Search Functionality

### Range Specification
- **Format**: "startPort-endPort"
- **Validation**: Ensures start < end and both are valid integers
- **Error Handling**: Logs errors for invalid range specifications

### Search Algorithm
- **Sequential Scanning**: Tests ports in order from start to end
- **Limit**: Stops after finding 10 available ports
- **Efficiency**: Uses TCP bind test for availability checking

### Output Format
- **Progress Logging**: Reports search parameters and progress
- **Found Ports**: Lists each available port as discovered
- **Summary**: Reports total number of available ports found

## Error Handling

### Port Range Validation
- **Invalid Format**: Handles missing or malformed range separators
- **Invalid Numbers**: Gracefully handles non-numeric port specifications
- **Logic Validation**: Ensures start port is less than end port
- **Zero Values**: Prevents invalid port numbers (0)

### Network Errors
- **Bind Failures**: Interprets network errors as port unavailability
- **Resource Management**: Properly closes test connections
- **Exception Safety**: Handles network-related exceptions gracefully

## Logging Integration

### Search Logging
- **Parameters**: Logs search range and criteria
- **Progress**: Reports each available port as found
- **Results**: Summarizes total ports discovered
- **Errors**: Logs validation and processing errors

### Log Levels
- **Info**: Normal operation and results
- **Error**: Validation failures and error conditions

## Integration Points

### Server Startup
- **Early Processing**: Flags processed before main server initialization
- **Exit Behavior**: Utility flags exit program after completion
- **Version Integration**: Uses passed version string for display

### Deployment Support
- **Port Discovery**: Helps identify available ports for server deployment
- **Version Verification**: Confirms deployed binary version
- **Troubleshooting**: Assists with network configuration issues

## Performance Considerations

### Port Scanning
- **Sequential Testing**: Tests one port at a time
- **Early Termination**: Stops after finding required number of ports
- **Resource Cleanup**: Closes test connections immediately
- **Network Efficiency**: Minimal network resource usage

### Memory Usage
- **Minimal Footprint**: Uses standard library functions efficiently
- **No Persistent State**: Utility functions don't retain data
- **Clean Exit**: Proper resource cleanup before program termination

## Deployment Integration

### Container Deployment
- **Port Discovery**: Useful for dynamic port allocation
- **Version Tracking**: Helps track deployed versions
- **Health Checks**: Can be used in deployment health verification

### Automation Support
- **Scripting**: Flags support automated deployment scripts
- **CI/CD Integration**: Version checking in build pipelines
- **Configuration**: Port discovery for configuration generation
