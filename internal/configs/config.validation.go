package configs

import "regexp"

const minimumPasswordLength = 4
const maximumPasswordLength = 16

type Validation struct {
	NameSizeMin      ConfigInt         `yaml:"NameSizeMin"`
	NameSizeMax      ConfigInt         `yaml:"NameSizeMax"`
	PasswordSizeMin  ConfigInt         `yaml:"PasswordSizeMin"`
	PasswordSizeMax  ConfigInt         `yaml:"PasswordSizeMax"`
	NameRejectRegex  ConfigString      `yaml:"NameRejectRegex"`
	NameRejectReason ConfigString      `yaml:"NameRejectReason"`
	EmailOnJoin      ConfigString      `yaml:"EmailOnJoin"`
	BannedNames      ConfigSliceString `yaml:"BannedNames"` // List of names that are not allowed to be used
}

func (v *Validation) Validate() {

	// Ignore BannedNames

	if v.NameRejectRegex != `` {
		if _, err := regexp.Compile(string(v.NameRejectRegex)); err != nil {
			v.NameRejectRegex = `^[a-zA-Z0-9_]+$`
			v.NameRejectReason = `Must only contain Alpha-numeric and underscores.`
		}
	}

	if v.NameSizeMin < 1 {
		v.NameSizeMin = 1
	}
	if v.NameSizeMax > 80 {
		v.NameSizeMax = 80
	}

	if v.PasswordSizeMin < 1 {
		v.PasswordSizeMin = 1
	}
	if v.PasswordSizeMax < v.PasswordSizeMin {
		v.PasswordSizeMax = v.PasswordSizeMin
	}

	if v.EmailOnJoin != `required` && v.EmailOnJoin != `none` {
		v.EmailOnJoin = `optional`
	}

}

func GetValidationConfig() Validation {
	ensureConfigValidated()

	configDataLock.RLock()
	defer configDataLock.RUnlock()
	return configData.Validation
}
