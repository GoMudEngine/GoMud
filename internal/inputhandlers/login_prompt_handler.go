package inputhandlers

import (
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"unicode/utf8"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/connections"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/term"
	"github.com/GoMudEngine/GoMud/internal/users"
)

var (
	ErrInputRequired        = errors.New(`input required`)
	ErrInvalidResponse      = errors.New(`invalid response`)
	ErrPasswordsDidNotMatch = errors.New(`your passwords did not match`)
)

const promptHandlerStateKey = "PromptHandlerState" // Keep unexported if only used within this package

// CompletionFunc is called when all steps are successfully completed.
// It receives the accumulated results and the shared state.
// It should return true if the overall process succeeded and the handler can be removed,
// false otherwise (e.g., login failed, disconnect).
type CompletionFunc func(results map[string]string, sharedState map[string]any, clientInput *connections.ClientInput) bool

// PromptHandlerState holds the current state of the multi-step prompt process for a connection.
type PromptHandlerState struct {
	Steps            []*PromptStep
	CurrentStepIndex int
	Results          map[string]string
	OnComplete       CompletionFunc
	maskTemplate     string // Cached mask character template
}

// ValidationFunc defines a function type for validating user input for a step.
// It takes the raw input and the accumulated results so far.
// It returns the cleaned/validated value (or an empty string) and an error if validation fails.
type ValidationFunc func(input string, results map[string]string) (string, error)

// ConditionFunc defines a function type to determine if a step should be executed.
// It takes the accumulated results so far.
type ConditionFunc func(results map[string]string) bool

// DataFunc generates dynamic data for a prompt step based on prior results.
type DataFunc func(results map[string]string) map[string]any

// PromptStep defines a single step in a multi-step prompt process.
type PromptStep struct {
	ID             string         // Unique identifier for this step's result
	PromptTemplate string         // Template name for the prompt message
	GetDataFunc    DataFunc       // Function to generate template data dynamically (optional)
	MaskInput      bool           // Should the input be masked?
	MaskTemplate   string         // Template for the mask character (optional)
	Validator      ValidationFunc // Function to validate the input, if returns false, repeat prompt
	Condition      ConditionFunc  // Do this step unless Condition Function returns false
	FailureCount   int            // Can be added if needed, managed within PromptHandlerState
}

// AlwaysRun is a default ConditionFunc that always returns true.
func AlwaysRun(_ map[string]string) bool {
	return true
}

// DefaultValidator is a basic validator that accepts any non-empty input.
func DefaultValidator(input string, _ map[string]string) (string, error) {
	if len(input) == 0 {
		return "", ErrInputRequired // Example error message key
	}
	return input, nil
}

func ValidateNewEntry(input string, _ map[string]string) (string, error) {
	if strings.ToLower(input) == `new` {
		return `new`, nil
	}

	validation := configs.GetValidationConfig()

	if len(input) < int(validation.NameSizeMin) {
		return "", errors.New("try again.")
	}

	return input, nil
}

// Specific Validators
func ValidateUsername(input string, _ map[string]string) (string, error) {
	if err := users.ValidateName(input); err != nil {
		return "", err
	}

	return input, nil
}

func ValidatePassword(input string, _ map[string]string) (string, error) {
	if err := users.ValidatePassword(input); err != nil {
		return "", err
	}
	return input, nil
}

func ValidatePassword2(input string, results map[string]string) (string, error) {
	if results[`password-new`] != input {
		return "", ErrPasswordsDidNotMatch
	}
	return input, nil
}

func ValidateEmail(input string, results map[string]string) (string, error) {

	if input == `` {

		if configs.GetValidationConfig().EmailOnJoin == `required` {
			return ``, ErrInputRequired
		}

		return ``, nil
	}

	if _, err := mail.ParseAddress(input); err != nil {
		return ``, err
	}

	return input, nil
}

func ValidateYesNo(input string, _ map[string]string) (string, error) {

	cleanInput := strings.ToLower(input)
	if len(cleanInput) == 0 {
		return "n", nil // Default to 'no' if empty input
	}

	cleanInput = cleanInput[0:1]

	if cleanInput == `y` {
		return "y", nil
	}

	if cleanInput == `n` {
		return "n", nil
	}
	// Provide more specific feedback
	return "", ErrInvalidResponse
}

// ValidateCharacterName checks that a character name is valid for a new user.
// It replicates the checks from the tutorial's `start` command.
func ValidateCharacterName(input string, results map[string]string) (string, error) {
	if len(input) == 0 {
		return "", ErrInputRequired
	}

	// Can't match their account username
	if strings.EqualFold(input, results["username-new"]) {
		return "", errors.New("your character name cannot match your account username")
	}

	// Centralised validation: length, regex, banned names, mob names, existing character/user
	if err := users.ValidateActorName(input, users.ValidateActorOpts{}); err != nil {
		return "", err
	}

	return input, nil
}

// CreatePromptHandler creates a generic input handler for multi-step prompts.
func CreatePromptHandler(steps []*PromptStep, onComplete CompletionFunc) connections.InputHandler {

	// Return the actual handler function closure
	return func(clientInput *connections.ClientInput, sharedState map[string]any) bool {

		var state *PromptHandlerState
		stateVal, ok := sharedState[promptHandlerStateKey]

		if !ok {

			state = &PromptHandlerState{
				Steps:            steps,
				CurrentStepIndex: -1, // Will be incremented by advanceAndSendPrompt
				Results:          make(map[string]string),
				OnComplete:       onComplete,
			}
			sharedState[promptHandlerStateKey] = state

			// Find and send the first applicable prompt
			if advanceAndSendPrompt(state, clientInput) {
				// Edge case: No steps were applicable, sequence completes immediately
				mudlog.Warn("Prompt sequence completed immediately (no applicable steps)", "connectionId", clientInput.ConnectionId)
				success := state.OnComplete(state.Results, sharedState, clientInput)
				delete(sharedState, promptHandlerStateKey) // Clean up state
				return success
			}
			return false // Waiting for input for the first prompt
		}

		state = stateVal.(*PromptHandlerState) // We assume it's the correct type

		// Check if index is valid (should always be, unless state is corrupted)
		if state.CurrentStepIndex < 0 || state.CurrentStepIndex >= len(state.Steps) {

			mudlog.Error("Invalid prompt state: CurrentStepIndex out of bounds", "index", state.CurrentStepIndex, "numSteps", len(state.Steps), "connectionId", clientInput.ConnectionId)
			// Clean up corrupted state
			delete(sharedState, promptHandlerStateKey)
			// Disconnect likely safest
			connections.Remove(clientInput.ConnectionId)
			// Indicate failure/disconnect
			return false
		}

		currentStep := state.Steps[state.CurrentStepIndex]

		// Input Buffering and Echo
		if !clientInput.EnterPressed {

			if clientInput.BSPressed && len(clientInput.Buffer) > 0 {

				// Handle Backspace - properly handle UTF-8 multi-byte characters
				bufferStr := string(clientInput.Buffer)
				if len(bufferStr) > 0 {
					// Find the start of the last rune
					_, size := utf8.DecodeLastRune(clientInput.Buffer)
					if size > 0 {
						clientInput.Buffer = clientInput.Buffer[:len(clientInput.Buffer)-size]
					}
				}
				//connections.SendTo([]byte{term.ASCII_BACKSPACE, term.ASCII_SPACE, term.ASCII_BACKSPACE}, clientInput.ConnectionId)

			} else if !clientInput.BSPressed && len(clientInput.DataIn) > 0 && clientInput.DataIn[0] >= 32 {

				// Handle printable characters
				//clientInput.Buffer = append(clientInput.Buffer, clientInput.DataIn...)

				// Echo or Mask
				if currentStep.MaskInput {

					// Cache the mask template string if needed
					if state.maskTemplate == "" && currentStep.MaskTemplate != "" {

						if maskStr, err := templates.Process(currentStep.MaskTemplate, nil); err != nil {
							mudlog.Error("Mask template error", "template", currentStep.MaskTemplate, "error", err)
							state.maskTemplate = "*" // Fallback mask
						} else {
							state.maskTemplate = templates.AnsiParse(maskStr)
						}

					} else if state.maskTemplate == "" {
						state.maskTemplate = "*" // Default fallback if no template specified
					}

					// Send mask character(s)
					for i := 0; i < len(clientInput.DataIn); i++ {
						connections.SendTo([]byte(state.maskTemplate), clientInput.ConnectionId)
					}

				} else {
					// Echo input directly
					connections.SendTo(clientInput.DataIn, clientInput.ConnectionId)
				}

			}

			// Non-Enter input processed, wait for more
			return false
		}

		if connections.IsWebsocket(clientInput.ConnectionId) {
			connections.SendTo(clientInput.Buffer, clientInput.ConnectionId) // Echo newline
		}

		// Enter Pressed: Process Input
		connections.SendTo(term.CRLF, clientInput.ConnectionId) // Echo newline
		submittedInput := strings.TrimSpace(string(clientInput.Buffer))
		clientInput.Buffer = clientInput.Buffer[:0] // Clear buffer for next input
		state.maskTemplate = ""                     // Clear cached mask template

		// Validation

		validatedValue, err := currentStep.Validator(submittedInput, state.Results)
		if err != nil {

			/////////////////////////////////////////////////////////////////
			// Validation failed, send error message and re-prompt
			/////////////////////////////////////////////////////////////////

			errMsg := err.Error() // Use the error message directly (could be from language.T)
			connections.SendTo([]byte(errMsg), clientInput.ConnectionId)
			connections.SendTo(term.CRLF, clientInput.ConnectionId)

			if currentStep.ID == `password-new-verify` {
				state.CurrentStepIndex -= 1
				currentStep = state.Steps[state.CurrentStepIndex]
			}

			// Increment failure counter in case we want to disconnect the user
			currentStep.FailureCount++
			if currentStep.FailureCount >= 3 {
				connections.SendTo([]byte(term.CRLFStr+`Too many mistakes.`+term.CRLFStr+term.CRLFStr), clientInput.ConnectionId)
				connections.Remove(clientInput.ConnectionId)
				state.CurrentStepIndex += 99
				return false
			}

			// Resend current prompt
			sendPrompt(currentStep, clientInput, state.Results)

			// Waiting for input again
			return false
		}

		/////////////////////////////////////////////////////////////////
		// Validation success
		/////////////////////////////////////////////////////////////////

		state.Results[currentStep.ID] = validatedValue

		if currentStep.MaskInput {
			mudlog.Debug("Prompt Step Success", "step", currentStep.ID, "value", "***REDACTED***", "connectionId", clientInput.ConnectionId)
		} else {
			mudlog.Debug("Prompt Step Success", "step", currentStep.ID, "value", validatedValue, "connectionId", clientInput.ConnectionId)
		}

		// Advance to Next Step or Complete
		if advanceAndSendPrompt(state, clientInput) {

			// Sequence complete
			mudlog.Debug("Prompt sequence completed", "connectionId", clientInput.ConnectionId)

			// Call the final completion function
			success := state.OnComplete(state.Results, sharedState, clientInput)

			// Clean up state
			delete(sharedState, promptHandlerStateKey)

			// Return completion result (true = handler done, false = e.g., login failed/disconnect)
			return success
		}

		// Moved to the next prompt, waiting for input
		return false
	}
}

// sendPrompt sends the prompt for the given step to the client.
func sendPrompt(step *PromptStep, clientInput *connections.ClientInput, results map[string]string) {

	templateName := step.PromptTemplate
	var templateData map[string]any

	// Generate dynamic data if GetDataFunc is provided
	if step.GetDataFunc != nil {
		templateData = step.GetDataFunc(results)
	} else {
		// Fallback to empty map if no dynamic data needed/provided
		templateData = make(map[string]any)
	}

	// Process the template
	promptTxt, err := templates.Process(templateName, templateData)
	if err != nil {
		mudlog.Error("Prompt template error", "template", templateName, "error", err)
		promptTxt = fmt.Sprintf("Error generating prompt '%s'", step.ID) // Fallback
	}

	parsedPrompt := templates.AnsiParse(promptTxt)

	// Send TEXTMASK FIRST (synchronously) so the websocket input field
	// switches to type="password" BEFORE the prompt arrives. Previously
	// TEXTMASK was queued via events.AddToQueue, which dispatched
	// asynchronously after the prompt — leaving a race window where the
	// user could type into a plaintext input field and see their password
	// echoed on screen.
	if connections.IsWebsocket(clientInput.ConnectionId) {
		maskCmd := "TEXTMASK:false"
		if step.MaskInput {
			maskCmd = "TEXTMASK:true"
		}
		connections.SendTo([]byte(maskCmd), clientInput.ConnectionId)
	}

	connections.SendTo([]byte(parsedPrompt), clientInput.ConnectionId)
}

// advanceAndSendPrompt finds the next valid step and sends its prompt.
// Returns true if the sequence is complete, false otherwise.
func advanceAndSendPrompt(state *PromptHandlerState, clientInput *connections.ClientInput) bool {

	state.CurrentStepIndex++

	for state.CurrentStepIndex < len(state.Steps) {

		step := state.Steps[state.CurrentStepIndex]

		// Determine condition (default to AlwaysRun)
		condition := step.Condition
		if condition == nil {
			condition = AlwaysRun
		}

		// Check condition against current results
		if condition(state.Results) {
			// Condition met, send this prompt
			sendPrompt(step, clientInput, state.Results)
			return false // Not complete, waiting for input for this step
		}

		// Condition not met, skip this step
		mudlog.Debug("Skipping prompt step", "step", step.ID, "connectionId", clientInput.ConnectionId)

		state.CurrentStepIndex++
	}

	// No more steps left
	return true // Sequence is complete
}
