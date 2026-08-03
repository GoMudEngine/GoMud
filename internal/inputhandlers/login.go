package inputhandlers

import (
	// ... other imports

	"fmt"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/connections"
	"github.com/GoMudEngine/GoMud/internal/language"
	"github.com/GoMudEngine/GoMud/internal/moderation"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/species"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/term"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// FinalizeLoginOrCreate is called after all prompts are successfully answered.
func FinalizeLoginOrCreate(results map[string]string, sharedState map[string]any, clientInput *connections.ClientInput) bool {

	// IP ban: reject (and block re-registration) before any account work. Skip
	// local/loopback connections so an admin can never lock themselves out.
	//
	// ClientIP(), not RemoteAddr(): for a websocket the socket peer is the TCP
	// peer of the HTTP upgrade, which behind a reverse proxy is the proxy. That
	// reported 127.0.0.1 for every /webclient player, so IsLocal() was true and
	// this whole block was skipped — a banned player only had to switch from
	// telnet to the web client. Telnet is unaffected: nothing sets clientIP on
	// a telnet connection, so ClientIP() is still the socket peer there.
	if connDetails := connections.Get(clientInput.ConnectionId); connDetails != nil && !connDetails.IsLocal() {
		host := connDetails.ClientIP()
		if reason, banned := moderation.IsIPBanned(host); banned {
			connections.SendTo([]byte("Your connection has been banned. Reason: "+reason), clientInput.ConnectionId)
			connections.SendTo(term.CRLF, clientInput.ConnectionId)
			connections.Remove(clientInput.ConnectionId)
			return false
		}
	}

	username := results["username"]
	password := results["password"]

	if username != `new` {
		userExists := users.Exists(username)

		if userExists {

			if results["kickuser"] == "y" {

				connDetails := connections.Get(clientInput.ConnectionId)

				// Disconnect/kick the user currently connected
				userid := users.FindUserId(results["username"])
				user := users.GetByUserId(userid)

				existingConnectionId := user.ConnectionId()

				// Send a goodbye message to the currently connected user
				tplTxt, _ := templates.Process("goodbye", nil)
				connections.SendTo([]byte(templates.AnsiParse(tplTxt)), existingConnectionId)

				users.SetZombieUser(userid)
				connections.Kick(existingConnectionId, fmt.Sprintf(`Duplicate login (ip: %s)`, connDetails.RemoteAddr()))

			}

			// Existing User Login Logic (No changes needed)
			tmpUser, err := users.LoadUser(username)
			if err != nil {
				mudlog.Error("Failed to load existing user during login", "username", username, "error", err)
				connections.SendTo([]byte(language.T("Error.LoginFailedGeneric")), clientInput.ConnectionId)
				connections.SendTo(term.CRLF, clientInput.ConnectionId)
				connections.Remove(clientInput.ConnectionId)
				return false // Indicate failure, connection removed
			}

			if !tmpUser.PasswordMatches(password) {
				connections.SendTo([]byte(`Nope. Bye!`), clientInput.ConnectionId)
				connections.SendTo(term.CRLF, clientInput.ConnectionId)
				connections.Remove(clientInput.ConnectionId)
				return false // Indicate failure, connection removed
			}

			// Account ban: reject a banned account after the password is verified
			// (so it can't be used to probe which accounts exist) and before the
			// character enters the world.
			if reason, banned := moderation.IsAccountBanned(username); banned {
				connections.SendTo([]byte("This account has been banned. Reason: "+reason), clientInput.ConnectionId)
				connections.SendTo(term.CRLF, clientInput.ConnectionId)
				connections.Remove(clientInput.ConnectionId)
				return false
			}

			loggedInUser, msg, err := users.LoginUser(tmpUser, clientInput.ConnectionId)
			if err != nil {
				connections.SendTo([]byte(msg), clientInput.ConnectionId)
				connections.SendTo(term.CRLF, clientInput.ConnectionId)
				connections.Remove(clientInput.ConnectionId)
				return false // Indicate failure, connection removed
			}

			sharedState["UserObject"] = loggedInUser // For main loop

			if len(msg) > 0 {
				connections.SendTo([]byte(msg), clientInput.ConnectionId)
				connections.SendTo(term.CRLF, clientInput.ConnectionId)
			}
			mudlog.Info("User logged in", "username", username, "connectionId", clientInput.ConnectionId)
			return true // Indicate success, handler can be removed

		} else {
			connections.SendTo([]byte(`Invalid login.`), clientInput.ConnectionId)
			connections.SendTo(term.CRLF, clientInput.ConnectionId)
			connections.Remove(clientInput.ConnectionId)
			return false // Indicate failure, connection removed
		}
	} else {
		/*
			username-new
			password-new
			password-new-verify
			email-new
			screen-reader-new y/n
			confirm_create y/n
		*/

		confirmCreate, exists := results["confirm_create"] // Assumes step ID "confirm_create"
		if !exists || confirmCreate != "y" {
			connections.SendTo([]byte(`Okay, bye!`), clientInput.ConnectionId) // Use language key
			connections.SendTo(term.CRLF, clientInput.ConnectionId)
			connections.Remove(clientInput.ConnectionId)
			return false // Indicate failure, connection removed
		}

		username := results["username-new"]
		password := results["password-new"]

		if users.Exists(results["username-new"]) {
			connections.SendTo([]byte(`I'm sorry, that user already exists!`), clientInput.ConnectionId) // Use language key
			connections.SendTo(term.CRLF, clientInput.ConnectionId)
			connections.Remove(clientInput.ConnectionId)
			return false
		}

		newUser := users.NewUserRecord(0, clientInput.ConnectionId)
		newUser.EmailAddress = results["email-new"]
		newUser.ScreenReader = results["screen-reader-new"] == `y`

		// Error handling for SetUsername/SetPassword might be redundant if validation passed, but good practice
		if err := newUser.SetUsername(username); err != nil {
			mudlog.Error("Internal error setting username post-validation", "username", username, "error", err)
			connections.SendTo([]byte(language.T("Error.UserCreationFailed")), clientInput.ConnectionId) // Generic creation error
			connections.SendTo(term.CRLF, clientInput.ConnectionId)
			connections.Remove(clientInput.ConnectionId)
			return false
		}
		if err := newUser.SetPassword(password); err != nil {
			mudlog.Error("Internal error setting password post-validation", "username", username, "error", err)
			connections.SendTo([]byte(language.T("Error.UserCreationFailed")), clientInput.ConnectionId) // Generic creation error
			connections.SendTo(term.CRLF, clientInput.ConnectionId)
			connections.Remove(clientInput.ConnectionId)
			return false
		}

		// Character name is the same as the username
		if err := newUser.SetCharacterName(username); err != nil {
			mudlog.Error("Internal error setting character name from username", "name", username, "error", err)
			connections.SendTo([]byte(language.T("Error.UserCreationFailed")), clientInput.ConnectionId)
			connections.SendTo(term.CRLF, clientInput.ConnectionId)
			connections.Remove(clientInput.ConnectionId)
			return false
		}

		// All players are human in Delusions of Grandeur
		if humanSpecies, ok := species.FindSpecies("human"); ok {
			newUser.Character.SpeciesId = humanSpecies.Id()
			newUser.Character.Validate()
			// Wire intrinsic mutations for the species (no-op for humans today;
			// future-proofs non-human playable species).
			sp := species.GetSpecies(humanSpecies.Id())
			newUser.Character.ApplyIntrinsicMutations(sp)
		}

		if err := users.CreateUser(newUser); err != nil {
			mudlog.Error("Could not create user", "username", username, "error", err)
			// Try to give specific feedback if possible, otherwise generic
			connections.SendTo([]byte(err.Error()), clientInput.ConnectionId)
			connections.SendTo(term.CRLF, clientInput.ConnectionId)
			connections.Remove(clientInput.ConnectionId)
			return false // Indicate failure, connection removed
		}

		sharedState["UserObject"] = newUser // For main loop

		mudlog.Info("New user created", "username", username, "connectionId", clientInput.ConnectionId)

		return true // Indicate success, handler can be removed
	}
}

func GetLoginPromptHandler() connections.InputHandler {

	// Define the steps for the login process
	loginSteps := []*PromptStep{
		{
			ID:             "username",
			PromptTemplate: "login/username.prompt",
			MaskInput:      false,
			Validator:      ValidateNewEntry,
			// Plaintext MSSP: crawlers may send "mssp-request" instead of a
			// name — answer with the status block and close.
			Intercept: MSSPTextRequestIntercept,
		},
		//////////////////////////////////////////////////
		// If NOT a new user signup (Just a login)
		//////////////////////////////////////////////////
		{
			ID:             "password",
			PromptTemplate: "login/password.prompt",
			MaskInput:      true,
			MaskTemplate:   "login/password.mask", // Optional: specify if different from "*"
			Validator:      ValidatePassword,
			Condition:      func(results map[string]string) bool { return results["username"] != `new` }, // Only run if username was not "new"
		},
		{
			ID:             "kickuser",
			PromptTemplate: "generic/prompt.yn",
			GetDataFunc: func(results map[string]string) map[string]any {
				// Dynamically generate the data for the generic y/n prompt
				return map[string]any{
					"prompt":  "User is already connected. Kick them?",
					"options": []string{"y", "n"},
					"default": "n", // Default shown in the prompt, actual default on empty input handled by validator
				}
			},
			MaskInput: false,
			Validator: ValidateYesNo,
			Condition: func(results map[string]string) bool {
				if results["username"] == `new` {
					return false
				}

				userid := users.FindUserId(results["username"])

				user := users.GetByUserId(userid)

				return user != nil && user.PasswordMatches(results["password"])
			}, // Only run if username was not "new", password matches, and user is currently online.
		},
		//////////////////////////////////////////////////
		// End If NOT a new user signup (Just a login)
		//////////////////////////////////////////////////
		//////////////////////////////////////////////////
		// If a new user signup
		//////////////////////////////////////////////////
		{
			ID:             "username-new",
			PromptTemplate: "login/username-new.prompt",
			MaskInput:      false,
			Validator:      ValidateUsername,
			Condition:      func(results map[string]string) bool { return results["username"] == `new` }, // Only run if username was "new"
		},
		{
			ID:             "password-new",
			PromptTemplate: "login/password-new.prompt",
			MaskInput:      true,
			MaskTemplate:   "login/password.mask", // Optional: specify if different from "*"
			Validator:      ValidatePassword,
			Condition:      func(results map[string]string) bool { return results["username"] == `new` }, // Only run if username was "new"
		},
		{
			ID:             "password-new-verify",
			PromptTemplate: "login/password-new-verify.prompt",
			MaskInput:      true,
			MaskTemplate:   "login/password.mask", // Optional: specify if different from "*"
			Validator:      ValidatePassword2,
			Condition:      func(results map[string]string) bool { return results["username"] == `new` }, // Only run if username was "new"
		},
		{
			ID:             "email-new",
			PromptTemplate: "login/email-new.prompt",
			GetDataFunc: func(results map[string]string) map[string]any {
				// Dynamically generate the data for the generic y/n prompt
				return map[string]any{
					"emailIsOptional": configs.GetValidationConfig().EmailOnJoin != `required`,
				}
			},
			MaskInput: false,
			Validator: ValidateEmail,
			Condition: func(results map[string]string) bool {
				return results["username"] == `new` && configs.GetValidationConfig().EmailOnJoin != `none` // Only run if username was "new" and email is enabled
			},
		},
		{
			ID:             "screen-reader-new",
			PromptTemplate: "generic/prompt.yn",
			GetDataFunc: func(results map[string]string) map[string]any {
				// Dynamically generate the data for the generic y/n prompt
				return map[string]any{
					"prompt":  "Are you using a screen reader?",
					"options": []string{"y", "n"},
					"default": "n", // Default shown in the prompt, actual default on empty input handled by validator
				}
			},
			MaskInput: false,
			Validator: ValidateYesNo,
			Condition: func(results map[string]string) bool { return results["username"] == `new` }, // Only run if username was "new"
		},
		{
			ID:             "confirm_create",
			PromptTemplate: "generic/prompt.yn", // Use the generic yes/no template
			GetDataFunc: func(results map[string]string) map[string]any {
				// Dynamically generate the data for the generic y/n prompt
				return map[string]any{
					"prompt": language.T("Login.CreateUser", map[any]any{ // Use language.T for the prompt text
						"Username": results["username-new"], // Inject username
					}),
					"options": []string{"y", "n"},
					"default": "n", // Default shown in the prompt, actual default on empty input handled by validator
				}
			},
			MaskInput: false,
			Validator: ValidateYesNo,
			Condition: func(results map[string]string) bool { return results["username"] == `new` }, // Only run if username was "new"
		},
		//////////////////////////////////////////////////
		// End If a new user signup
		//////////////////////////////////////////////////
	}

	// Create and return the handler using the generic factory function
	return CreatePromptHandler(loginSteps, FinalizeLoginOrCreate)
}
