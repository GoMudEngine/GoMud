package configs

import "golang.org/x/text/language"

type Translation struct {
	DefaultLanguage ConfigString      `yaml:"DefaultLanguage"` // Specify the default game language (fallback)
	Language        ConfigString      `yaml:"Language"`        // Specify the game language
	LanguagePaths   ConfigSliceString `yaml:"LanguagePaths"`   // Specify the game language file paths
}

func (t *Translation) Validate() {

	// Ignore LanguagePaths

	dl := language.Make(t.DefaultLanguage.String())
	if dl.IsRoot() {
		t.Language = `en` // default
	}

	l := language.Make(t.Language.String())
	if l.IsRoot() {
		t.Language = `en` // default
	}

}

func GetTranslationConfig() Translation {
	ensureConfigValidated()

	configDataLock.RLock()
	defer configDataLock.RUnlock()
	return configData.Translation
}
