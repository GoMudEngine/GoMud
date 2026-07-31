# GoMud Prompt System Context

## Overview

The GoMud prompt system provides interactive user input/response handling for complex multi-step operations. It enables commands to ask questions, collect responses, maintain state across interactions, and validate user input. The system integrates seamlessly with the main input processing loop in `world.go` and supports both simple question/answer flows and complex multi-step workflows.

## Architecture

The prompt system consists of two main components:

### 1. Core Prompt System (`internal/prompt/`)

**Key Types:**
- `Question` - Individual question with prompt text, options, and response validation
- `Prompt` - Container for multiple questions with state management and recall functionality

**Integration Points:**
- `UserRecord.StartPrompt()` - Initiates or retrieves existing prompt sessions
- `UserRecord.ClearPrompt()` - Cleans up completed or aborted prompt sessions
- `UserRecord.GetCommandPrompt()` - Integrates with main command prompt display
- `world.go` input loop - Processes prompt responses before regular commands

### 2. Advanced Login Prompt Handler (`internal/inputhandlers/`)

**Features:**
- Multi-step prompt sequences with conditional execution
- Template-based prompt rendering with dynamic data
- Validation and error handling
- State persistence across prompt steps

## Core Components

### Question Structure
```go
type Question struct {
    Prompt      string   // Question text displayed to user
    Options     []string // Valid response options (empty = any input)
    Response    string   // User's response (populated after input)
    Done        bool     // Whether question has been answered
    // Additional validation and state fields
}
```

### Prompt Management
```go
type Prompt struct {
    Command     string              // Command that initiated the prompt
    Questions   []*Question         // Sequence of questions
    Recall      map[string]string   // Persistent data storage
    // State management fields
}
```

## Key Features

### 1. **Session Management**
- Prompts persist across user inputs until completed or cleared
- Automatic session cleanup on completion or abort
- State isolation between different users

### 2. **Question Flow Control**
- Sequential question processing with automatic advancement
- Optional questions based on previous responses
- Branching logic for complex workflows

### 3. **Input Validation**
- Predefined option lists for constrained choices
- Custom validation logic in command handlers
- Automatic retry on invalid input

### 4. **State Persistence**
- `Recall()` mechanism for storing data across questions
- Template integration for dynamic content generation
- Cross-question data sharing

### 5. **Integration with Command System**
- Seamless integration with existing command handlers
- Priority processing over regular commands
- Automatic prompt display in command prompt

## Usage Patterns

### 1. Simple Question/Answer Flow

**Example: Password Change**
```go
func Password(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
    cmdPrompt, _ := user.StartPrompt(`password`, rest)
    
    question := cmdPrompt.Ask(`What is your current password?`, []string{})
    if !question.Done {
        return true, nil
    }
    
    if !user.PasswordMatches(question.Response) {
        user.SendText(`Sorry, your password was incorrect.`)
        user.ClearPrompt()
        return true, nil
    }
    
    // Continue with new password questions...
}
```

### 2. Multi-Step Creation Workflow

**Example: Mob Creation**
```go
func mob_Create(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
    cmdPrompt, isNew := user.StartPrompt(`mob create`, rest)
    
    if isNew {
        user.SendText(`Starting mob creation...`)
    }
    
    // Step 1: Get mob name
    question := cmdPrompt.Ask(`What name do you want to give this mob?`, []string{})
    if !question.Done {
        return true, nil
    }
    if question.Response == `` {
        user.SendText("Aborting...")
        user.ClearPrompt()
        return true, nil
    }
    cmdPrompt.Remember(`name`, question.Response)
    
    // Step 2: Get description
    question = cmdPrompt.Ask(`Describe this mob:`, []string{})
    if !question.Done {
        return true, nil
    }
    // Continue with additional steps...
    
    // Final confirmation
    question = cmdPrompt.Ask(`Create this mob? (y/n)`, []string{`y`, `n`})
    if !question.Done {
        return true, nil
    }
    
    user.ClearPrompt()
    
    if question.Response != `y` {
        user.SendText("Aborting...")
        return true, nil
    }
    
    // Create the mob using collected data
    return true, nil
}
```

### 3. Menu-Driven Selection

**Example: Character Management**
```go
func Character(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
    menuOptions := []string{`new`, `delete`, `switch`, `hire`, `dismiss`, `quit`}
    
    cmdPrompt, isNew := user.StartPrompt(`character`, rest)
    
    question := cmdPrompt.Ask(`Choose an option:`, menuOptions, `new`)
    if !question.Done {
        return true, nil
    }
    
    if question.Response == `quit` {
        user.ClearPrompt()
        return true, nil
    }
    
    // Handle selected option...
}
```

### 4. Conditional Branching

**Example: Room Container Editing**
```go
func room_EditContainers(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
    cmdPrompt, _ := user.StartPrompt(`room edit containers`, rest)
    
    question := cmdPrompt.Ask(`Choose one:`, []string{`new`}, `new`)
    if !question.Done {
        return true, nil
    }
    
    if question.Response == `new` {
        // Branch to new container creation
        question = cmdPrompt.Ask(`Container name:`, []string{})
        // Handle new container...
    } else {
        // Branch to existing container editing
        // Handle container selection and editing...
    }
}
```

## Integration with Input Loop

The prompt system integrates with the main input processing in `world.go`:

```go
// In world.go input processing
if user.GetPrompt() != nil {
    if activeQuestion := user.GetPrompt().GetNextQuestion(); activeQuestion != nil {
        // Process user input as prompt response
        activeQuestion.SetResponse(string(clientInput.Buffer))
        
        // Execute command handler to process the response
        handled, err := usercommands.Process(user.GetPrompt().Command, user, room)
        
        if err != nil {
            // Handle error
        }
    } else {
        // If prompt exists but no pending questions, clear it
        user.ClearPrompt()
    }
}
```

## Default Conditions and Parameters

### Question Options
- **Empty options list**: Accepts any text input
- **Predefined options**: Restricts input to specific choices
- **Default value**: Used when user provides empty response

### Prompt Behavior
- **Automatic retry**: Invalid responses re-display the question
- **Case sensitivity**: Options are case-sensitive by default
- **Whitespace handling**: Leading/trailing whitespace is trimmed
- **Empty response handling**: Can be treated as abort or use default value

### State Management
- **Automatic cleanup**: Prompts are cleared on completion or explicit abort
- **Session persistence**: Prompts survive server restarts (if user reconnects)
- **Memory recall**: Previous answers can be retrieved using `Recall()`

## Error Handling and Validation

### Input Validation
```go
// Validate required input
if question.Response == `` {
    user.SendText("This field is required.")
    user.ClearPrompt()
    return true, nil
}

// Validate against business rules
if !isValidMobName(question.Response) {
    user.SendText("Invalid mob name format.")
    user.ClearPrompt()
    return true, nil
}
```

### Graceful Abort Handling
```go
// Allow user to quit at any step
if question.Response == `quit` || question.Response == `` {
    user.SendText("Operation cancelled.")
    user.ClearPrompt()
    return true, nil
}
```

## Testing and Examples

The prompt system includes comprehensive test coverage in `prompt_test.go` demonstrating:
- Basic question/answer cycles
- Multi-step prompt flows
- State management and recall
- Error conditions and edge cases
- Integration with user session management

## Dependencies

- `internal/users` - User session management and prompt storage
- `internal/connections` - Client communication for prompt display
- `internal/templates` - Template processing for dynamic prompts
- `internal/term` - Terminal control codes for prompt formatting
- `world.go` - Main input loop integration

## Performance Considerations

- Prompts are stored in memory per user session
- Automatic cleanup prevents memory leaks
- Minimal overhead when no prompts are active
- Efficient string processing for large option lists

This prompt system enables GoMud to provide sophisticated interactive experiences while maintaining clean separation between game logic and user interface concerns.
