package configs

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
	"sync"

	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/util"
	"gopkg.in/yaml.v2"
)

const (
	PVPEnabled  = `enabled`
	PVPDisabled = `disabled`
	PVPOff      = `off`
	PVPLimited  = `limited`
)

var (
	configData Config         = Config{}
	overrides  map[string]any = make(map[string]any)

	keyLookups  map[string]string = map[string]string{}
	typeLookups map[string]string = map[string]string{}

	configDataLock       sync.RWMutex
	ErrInvalidConfigName = errors.New("invalid config name")
	ErrLockedConfig      = errors.New("config name is locked")
)

type Config struct {
	// Start config subsections
	Server       Server       `yaml:"Server"`
	Memory       Memory       `yaml:"Memory"`
	LootGoblin   LootGoblin   `yaml:"LootGoblin"`
	Timing       Timing       `yaml:"Timing"`
	FilePaths    FilePaths    `yaml:"FilePaths"`
	GamePlay     GamePlay     `yaml:"GamePlay"`
	Integrations Integrations `yaml:"Integrations"`
	TextFormats  TextFormats  `yaml:"TextFormats"`
	Translation  Translation  `yaml:"Translation"`
	Network      Network      `yaml:"Network"`
	SpecialRooms SpecialRooms `yaml:"SpecialRooms"`
	Validation   Validation   `yaml:"Validation"`
	Roles        Roles        `yaml:"Roles"`
	// Plugins is a special case
	Modules Modules `yaml:"Modules"`

	Balance Balance `yaml:"Balance"`

	LLM LLM `yaml:"LLM"`

	Analytics Analytics `yaml:"Analytics"`

	Logging Logging `yaml:"Logging"`

	Playtest Playtest `yaml:"Playtest"`

	// End config subsections

	seedInt int64 `yaml:"-"`

	validated bool
}

func AddOverlayOverrides(dotMap map[string]any) error {
	// Write lock: this mutates configData (via OverlayOverrides) and the
	// keyLookups/typeLookups/overrides maps. It previously held only RLock,
	// racing with concurrent getters/readers. No config getter calls this, so
	// upgrading to the write lock cannot deadlock.
	configDataLock.Lock()
	defer configDataLock.Unlock()

	for k, v := range dotMap {

		if strings.Index(k, `.`) != -1 {

			parts := strings.Split(k, `.`)

			for i := len(parts) - 1; i >= 0; i-- {
				tmpKey := strings.Join(parts[i:], `.`)
				keyLookups[strings.ToLower(tmpKey)] = k

				tmpKey = strings.Join(parts[i:], ``)
				keyLookups[strings.ToLower(tmpKey)] = k

			}

		} else {
			keyLookups[strings.ToLower(k)] = k
		}

		typeLookups[k] = reflect.TypeOf(v).String()

		overrides[k] = v
	}

	return configData.OverlayOverrides(dotMap)
}

// OverlayDotMap overlays values from a dot-syntax map onto the Config.
func (c *Config) OverlayOverrides(dotMap map[string]any) error {
	// First unflatten the dot map into a nested map.
	nestedMap := unflattenMap(dotMap)

	// Marshal the nested map into YAML bytes.
	b, err := yaml.Marshal(nestedMap)
	if err != nil {
		return err
	}

	// Unmarshal the YAML bytes into the existing Config struct.
	return yaml.Unmarshal(b, c)
}

func (c *Config) DotPaths() map[string]any {
	result := make(map[string]any)
	// Get the underlying value of c (we assume c is a pointer).
	v := reflect.ValueOf(c).Elem()
	c.buildDotPaths(v, "", result)
	return result
}

func (c *Config) buildDotPaths(v reflect.Value, prefix string, result map[string]any) {
	// If the value is a pointer, dereference it.
	if v.Kind() == reflect.Interface || v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return
		}
		c.buildDotPaths(v.Elem(), prefix, result)
		return
	}

	switch v.Kind() {
	case reflect.Struct:
		t := v.Type()
		for i := 0; i < v.NumField(); i++ {
			field := t.Field(i)
			// Skip unexported fields.
			if field.PkgPath != "" {
				continue
			}
			fieldVal := v.Field(i)
			// Determine the key name using the YAML tag if available.
			key := field.Name
			if yamlTag, ok := field.Tag.Lookup("yaml"); ok {
				if tagName := strings.Split(yamlTag, ",")[0]; tagName != "" {
					key = tagName
				}
			}
			// Construct the new prefix.
			newPrefix := key
			if prefix != "" {
				newPrefix = prefix + "." + key
			}
			// Recursively build paths.
			c.buildDotPaths(fieldVal, newPrefix, result)
		}
	case reflect.Map:
		// If the map is nil, store nil for the current prefix.
		if v.IsNil() {
			result[prefix] = make(map[string]any)
			return
		}
		// Iterate over each key in the map.
		for _, key := range v.MapKeys() {
			// Convert the key to string (works for string keys; for others, fmt.Sprintf is used).
			keyStr := fmt.Sprintf("%v", key.Interface())
			newPrefix := keyStr
			if prefix != "" {
				newPrefix = prefix + "." + keyStr
			}
			c.buildDotPaths(v.MapIndex(key), newPrefix, result)
		}
	default:
		// For non-struct fields, store the value using the accumulated prefix.
		result[prefix] = v.Interface()
	}
}

func GetOverrides() map[string]any {
	return overrides
}

func (c *Config) SetOverrides(newOverrides map[string]any) error {

	overrides = newOverrides
	c.OverlayOverrides(overrides)

	return nil
}

// Ensures certain ranges and defaults are observed
func (c *Config) Validate() {

	c.Server.Validate()
	c.Memory.Validate()
	c.LootGoblin.Validate()
	c.Timing.Validate()
	c.FilePaths.Validate()
	c.GamePlay.Validate()
	c.Integrations.Validate()
	c.TextFormats.Validate()
	c.Translation.Validate()
	c.Network.Validate()
	c.SpecialRooms.Validate()
	c.Validation.Validate()
	c.Modules.Validate()
	c.Roles.Validate()
	c.Balance.Validate()
	c.LLM.Validate()
	c.Analytics.Validate()
	c.Playtest.Validate()

	// nothing to do with LootGoblinIncludeRecentRooms

	// Nothing to do with Locked

	c.seedInt = 0
	for i, num := range util.Md5Bytes([]byte(string(c.Server.Seed))) {
		c.seedInt += int64(num) << i
	}

	c.validated = true
}

// ensureConfigValidated runs configData.Validate() exactly once before the
// first config read, under the WRITE lock. Validate() mutates configData
// (clamps fields, computes seedInt, sets validated=true). The getters used to
// run it while holding only configDataLock.RLock(), so two goroutines hitting
// any getter concurrently on first use both observed validated==false and both
// mutated configData — a data race the -race detector flags (and a real
// startup race, since many goroutines read config before it is validated).
// Double-checked locking keeps the hot path (already validated) on the cheap
// read lock. Safe from deadlock: Validate() and the subsection validators only
// touch their own fields and never call a config getter.
func ensureConfigValidated() {
	configDataLock.RLock()
	done := configData.validated
	configDataLock.RUnlock()
	if done {
		return
	}

	configDataLock.Lock()
	defer configDataLock.Unlock()
	if !configData.validated {
		configData.Validate()
	}
}

func (c *Config) setEnvAssignments(clear bool) {

	// We use reflect.Indirect to handle if cfg is a pointer or not
	v := reflect.ValueOf(c).Elem()

	// We'll need the struct type as well (to get field names).
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		fieldVal := v.Field(i)
		fieldType := t.Field(i)

		if fieldVal.Type().Kind() != reflect.String {
			continue
		}

		if envName := fieldType.Tag.Get(`env`); envName != `` {
			if fieldVal.CanSet() {
				if envVal := os.Getenv(envName); envVal != `` {

					if clear {
						envVal = ``
					}

					fieldVal.Set(reflect.ValueOf(ConfigSecret(envVal)))

				}
			}
		}

	}
}

func (c Config) IsBannedName(name string) (string, bool) {

	name = strings.ToLower(strings.TrimSpace(name))

	for _, bannedName := range c.Validation.BannedNames {
		if util.StringWildcardMatch(name, strings.ToLower(bannedName)) {
			return bannedName, true
		}
	}

	return "", false
}

func (c Config) SeedInt() int64 {
	return c.seedInt
}

func (c Config) AllConfigData(excludeStrings ...string) map[string]any {

	finalOutput := make(map[string]any)

	for name, value := range c.DotPaths() {

		if len(excludeStrings) > 0 {
			testName := strings.ToLower(name)
			skip := false
			for _, s := range excludeStrings {
				if util.StringWildcardMatch(testName, s) {
					skip = true
					break
				}
			}
			if skip {
				continue
			}
		}

		finalOutput[name] = value
	}
	return finalOutput
}

func SetVal(propertyPath string, newVal string) error {

	propertyPath, propertyType := FindFullPath(propertyPath)
	if propertyType == `` {
		return errors.New(`invalid property name: ` + propertyPath)
	}

	quickMap := make(map[string]any)
	quickMap[propertyPath] = StringToConfigValue(newVal, propertyType)

	flatOverrides := Flatten(overrides)
	flatQuickmap := Flatten(quickMap)

	for k, v := range flatQuickmap {
		flatOverrides[k] = v
	}

	overrides = unflattenMap(flatOverrides)

	// save the new config.
	writeBytes, err := yaml.Marshal(overrides)
	if err != nil {
		return err
	}

	overridePath := overridePath()
	if err := util.Save(overridePath, writeBytes, bool(configData.FilePaths.CarefulSaveFiles)); err != nil {
		return err
	}

	if err = configData.OverlayOverrides(overrides); err != nil {
		return err
	}

	configData.Validate()
	return nil
}

func GetConfig() Config {
	ensureConfigValidated()

	configDataLock.RLock()
	defer configDataLock.RUnlock()

	return configData
}

func overridePath() string {
	overridePath := os.Getenv(`CONFIG_PATH`)
	if overridePath == `` {
		overridePath = GetConfig().FilePaths.DataFiles.String() + `/config-overrides.yaml`
	}

	return overridePath
}

func ReloadConfig() error {

	configPath := util.FilePath(`_datafiles/config.yaml`)

	bytes, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	tmpConfigData := Config{}
	err = yaml.Unmarshal(bytes, &tmpConfigData)
	if err != nil {
		return err
	}

	// Build a special lookup to attempt to match old data or even some minor typos
	keyLookups = map[string]string{}
	typeLookups = map[string]string{}
	for k, v := range configData.AllConfigData() {

		if strings.Index(k, `.`) != -1 {

			parts := strings.Split(k, `.`)

			for i := len(parts) - 1; i >= 0; i-- {
				tmpKey := strings.Join(parts[i:], `.`)
				keyLookups[strings.ToLower(tmpKey)] = k

				tmpKey = strings.Join(parts[i:], ``)
				keyLookups[strings.ToLower(tmpKey)] = k

			}

		} else {
			keyLookups[strings.ToLower(k)] = k
		}

		typeLookups[k] = reflect.TypeOf(v).String()
	}

	overridePath := overridePath()

	mudlog.Info("ReloadConfig()", "overridePath", overridePath)

	if _, err := os.Stat(util.FilePath(overridePath)); err == nil {
		if overridePath != `` {

			mudlog.Info("ReloadConfig()", "Loading overrides", true)

			overrideBytes, err := os.ReadFile(util.FilePath(overridePath))
			if err != nil {
				return err
			}

			tmpOverrides := map[string]any{}
			err = yaml.Unmarshal(overrideBytes, &tmpOverrides)
			if err != nil {
				return err
			}

			// Attempt a correction for bad names
			for k, v := range tmpOverrides {
				if newKey, _ := FindFullPath(k); newKey != k {
					tmpOverrides[newKey] = v
					delete(tmpOverrides, k)
				}
			}

			tmpConfigData.SetOverrides(tmpOverrides)
		}
	} else {
		mudlog.Info("ReloadConfig()", "Loading overrides", false)
	}

	tmpConfigData.setEnvAssignments(false)

	tmpConfigData.Validate()

	configDataLock.Lock()
	defer configDataLock.Unlock()
	// Assign it
	configData = tmpConfigData

	return nil
}

func FindFullPath(inputKey string) (properKey string, typeName string) {

	if v, ok := keyLookups[strings.ToLower(inputKey)]; ok {
		return v, typeLookups[v]
	}
	return inputKey, typeLookups[inputKey]
}

// Usage: configs.GetSecret(c.DiscordWebhookUrl)
func GetSecret(v ConfigSecret) string {
	return string(v)
}

// flatten recursively flattens a map[string]any.
// It supports both map[string]any and map[any]any values,
// which is useful when unmarshaling YAML.
func Flatten(input map[string]any) map[string]any {
	flatMap := make(map[string]any)
	flattenHelper("", input, flatMap)
	return flatMap
}

// flattenHelper is a recursive helper that constructs the flattened map.
func flattenHelper(prefix string, input map[string]any, flatMap map[string]any) {
	for key, value := range input {
		// Construct the new key path.
		var newKey string
		if prefix == "" {
			newKey = key
		} else {
			newKey = prefix + "." + key
		}

		// Handle nested maps from YAML unmarshaling, which can be of type map[any]any.
		switch v := value.(type) {
		case map[string]any:
			flattenHelper(newKey, v, flatMap)
		case map[any]any:
			// Convert map[any]any to map[string]any.
			converted := make(map[string]any)
			for k, val := range v {
				if strKey, ok := k.(string); ok {
					converted[strKey] = val
				}
			}
			flattenHelper(newKey, converted, flatMap)
		default:
			flatMap[newKey] = value
		}
	}
}
