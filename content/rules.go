package content

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
)

// Baseline rule keys registered in the production registry.
const (
	RuleStandardConditions = "STANDARD_CONDITIONS"
	RuleNavigate           = "NAVIGATE"
	RuleStandardRules      = "STANDARD_RULES"
)

// ConfigEnvelope is the strict schema-versioned configuration envelope required
// by every gameplay rule configuration: {"schemaVersion":N,"parameters":{...}}.
type ConfigEnvelope struct {
	SchemaVersion int             `json:"schemaVersion"`
	Parameters    json.RawMessage `json:"parameters"`
}

// ReleaseContext exposes candidate-release indexes so rule validators can
// resolve same-candidate references without touching a database.
type ReleaseContext struct {
	indexes map[Kind]map[string]struct{}
}

// Contains reports whether the candidate release contains a definition of the
// given kind with the given stable key.
func (c *ReleaseContext) Contains(kind Kind, key string) bool {
	if c == nil || c.indexes == nil {
		return false
	}
	_, ok := c.indexes[kind][key]
	return ok
}

// Validator validates the typed parameters of one registered rule key. The
// configuration envelope has already been decoded and its schema version checked
// before ValidateParameters is invoked.
type Validator interface {
	// Kind returns the content kind served by this validator.
	Kind() Kind
	// RuleKey returns the registered uppercase rule key.
	RuleKey() string
	// SchemaVersion returns the supported configuration schema version.
	SchemaVersion() int
	// ValidateParameters checks the decoded parameters object for one definition.
	ValidateParameters(ctx *ReleaseContext, key string, parameters json.RawMessage) error
}

// Registry is an immutable set of registered rule validators keyed by content
// kind and rule key. It may be constructed in tests so validation mechanics can
// be exercised without publishing invented production balance rules.
type Registry struct {
	validators map[registryKey]Validator
}

type registryKey struct {
	kind    Kind
	ruleKey string
}

// NewRegistry builds a Registry from validators. A later validator for the same
// kind/rule-key pair replaces an earlier one, which lets a caller derive a
// test-only registry from the production baseline.
func NewRegistry(validators ...Validator) *Registry {
	registry := &Registry{validators: make(map[registryKey]Validator, len(validators))}
	for _, validator := range validators {
		if validator == nil {
			continue
		}
		key := registryKey{kind: validator.Kind(), ruleKey: validator.RuleKey()}
		registry.validators[key] = validator
	}
	return registry
}

// StandardRegistry returns the immutable production registry of approved
// baseline gameplay rules. Strategy, Relic, and Synergy collections stay empty
// until approved game-design definitions are supplied.
func StandardRegistry() *Registry {
	return NewRegistry(
		NewSpecValidator(RuleSpec{Kind: KindModifier, RuleKey: RuleStandardConditions, SchemaVersion: 1}),
		NewSpecValidator(RuleSpec{Kind: KindObjective, RuleKey: RuleNavigate, SchemaVersion: 1}),
		NewSpecValidator(RuleSpec{Kind: KindGameRuleSet, RuleKey: RuleStandardRules, SchemaVersion: 1}),
	)
}

// lookup returns the validator registered for a kind/rule-key pair.
func (r *Registry) lookup(kind Kind, ruleKey string) (Validator, bool) {
	validator, ok := r.validators[registryKey{kind: kind, ruleKey: ruleKey}]
	return validator, ok
}

// validateRuleBinding validates the rule key, schema version, strict envelope,
// and typed parameters for one definition. The returned error is unwrapped for
// the caller to add kind/key context.
func (r *Registry) validateRuleBinding(ctx *ReleaseContext, kind Kind, key, ruleKey string, config json.RawMessage) error {
	if !ruleKeyPattern.MatchString(ruleKey) {
		return fmt.Errorf("invalid rule key %q", ruleKey)
	}
	validator, exists := r.lookup(kind, ruleKey)
	if !exists {
		return fmt.Errorf("unknown rule key %q", ruleKey)
	}
	if !isJSONObject(config) {
		return fmt.Errorf("rule configuration must be a JSON object")
	}
	var envelope ConfigEnvelope
	decoder := json.NewDecoder(bytes.NewReader(config))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&envelope); err != nil {
		return fmt.Errorf("invalid rule configuration envelope: %w", err)
	}
	if envelope.SchemaVersion != validator.SchemaVersion() {
		return fmt.Errorf("unsupported rule schema version %d (supported %d)", envelope.SchemaVersion, validator.SchemaVersion())
	}
	if !isJSONObject(envelope.Parameters) {
		return fmt.Errorf("rule parameters must be a JSON object")
	}
	if err := validator.ValidateParameters(ctx, key, envelope.Parameters); err != nil {
		return err
	}
	return nil
}

// ParameterType is the JSON value type a rule parameter must take.
type ParameterType int

const (
	// ParameterInteger requires an integral JSON number.
	ParameterInteger ParameterType = iota
	// ParameterBoolean requires a JSON boolean.
	ParameterBoolean
	// ParameterString requires a JSON string.
	ParameterString
	// ParameterStringList requires a JSON array of strings.
	ParameterStringList
	// ParameterReference requires a JSON string naming a same-candidate definition.
	ParameterReference
)

// FixedPoint declares the explicit scale and inclusive range of a fixed-point
// integer parameter.
type FixedPoint struct {
	Scale int64
	Min   int64
	Max   int64
}

// ParameterSpec declares one typed parameter field inside a rule envelope.
type ParameterSpec struct {
	Name     string
	Type     ParameterType
	Required bool
	// Fixed is required for ParameterInteger and declares scale/range.
	Fixed *FixedPoint
	// RefKind is required for ParameterReference and names the referenced kind.
	RefKind Kind
}

// TotalSpec requires all named integer parameters to be present and to sum
// exactly to Total, detecting overflow before summing.
type TotalSpec struct {
	Fields []string
	Total  int64
}

// RuleSpec declaratively describes one typed gameplay rule key. It is the shared
// descriptor used by the production baseline and by test-only rule descriptors,
// keeping typed rule interpretation centralized for later settlement executors.
type RuleSpec struct {
	Kind              Kind
	RuleKey           string
	SchemaVersion     int
	Parameters        []ParameterSpec
	MutuallyExclusive [][]string
	Total             *TotalSpec
}

// SpecValidator is a descriptor-driven rule validator.
type SpecValidator struct {
	spec RuleSpec
}

// NewSpecValidator builds a SpecValidator from a rule descriptor.
func NewSpecValidator(spec RuleSpec) *SpecValidator {
	return &SpecValidator{spec: spec}
}

// Kind returns the content kind served by this validator.
func (v *SpecValidator) Kind() Kind { return v.spec.Kind }

// RuleKey returns the registered uppercase rule key.
func (v *SpecValidator) RuleKey() string { return v.spec.RuleKey }

// SchemaVersion returns the supported configuration schema version.
func (v *SpecValidator) SchemaVersion() int { return v.spec.SchemaVersion }

// ValidateParameters decodes parameters strictly, rejecting unknown fields and
// enforcing declared types, ranges, references, mutual exclusions, and totals.
func (v *SpecValidator) ValidateParameters(ctx *ReleaseContext, _ string, parameters json.RawMessage) error {
	if !isJSONObject(parameters) {
		return fmt.Errorf("parameters must be a JSON object")
	}
	rawFields := make(map[string]json.RawMessage)
	if err := json.Unmarshal(parameters, &rawFields); err != nil {
		return fmt.Errorf("decode parameters: %w", err)
	}

	declared := make(map[string]ParameterSpec, len(v.spec.Parameters))
	for _, spec := range v.spec.Parameters {
		declared[spec.Name] = spec
	}
	for name := range rawFields {
		if _, ok := declared[name]; !ok {
			return fmt.Errorf("unknown parameter %q", name)
		}
	}
	for _, spec := range v.spec.Parameters {
		raw, present := rawFields[spec.Name]
		if !present {
			if spec.Required {
				return fmt.Errorf("missing required parameter %q", spec.Name)
			}
			continue
		}
		if err := v.validateParameter(ctx, spec, raw); err != nil {
			return err
		}
	}
	for _, group := range v.spec.MutuallyExclusive {
		present := 0
		for _, name := range group {
			if _, ok := rawFields[name]; ok {
				present++
			}
		}
		if present > 1 {
			return fmt.Errorf("parameters %v are mutually exclusive", group)
		}
	}
	if v.spec.Total != nil {
		return v.validateTotal(rawFields)
	}
	return nil
}

func (v *SpecValidator) validateParameter(ctx *ReleaseContext, spec ParameterSpec, raw json.RawMessage) error {
	switch spec.Type {
	case ParameterInteger:
		fixed := spec.Fixed
		if fixed == nil {
			fixed = &FixedPoint{Scale: ContentWeightScale, Min: 0, Max: ContentWeightScale}
		}
		number, err := decodeInteger(raw)
		if err != nil {
			return fmt.Errorf("parameter %q must be an integer", spec.Name)
		}
		if number < fixed.Min || number > fixed.Max {
			return fmt.Errorf("parameter %q is outside the range [%d, %d]", spec.Name, fixed.Min, fixed.Max)
		}
		return nil
	case ParameterBoolean:
		var value bool
		if err := json.Unmarshal(raw, &value); err != nil {
			return fmt.Errorf("parameter %q must be a boolean", spec.Name)
		}
		return nil
	case ParameterString:
		var value string
		if err := json.Unmarshal(raw, &value); err != nil {
			return fmt.Errorf("parameter %q must be a string", spec.Name)
		}
		return nil
	case ParameterStringList:
		var values []string
		if err := json.Unmarshal(raw, &values); err != nil {
			return fmt.Errorf("parameter %q must be a list of strings", spec.Name)
		}
		return nil
	case ParameterReference:
		var value string
		if err := json.Unmarshal(raw, &value); err != nil {
			return fmt.Errorf("parameter %q must be a string", spec.Name)
		}
		if !ctx.Contains(spec.RefKind, value) {
			return fmt.Errorf("parameter %q references unknown %s %q", spec.Name, spec.RefKind, value)
		}
		return nil
	default:
		return fmt.Errorf("parameter %q has an unsupported type", spec.Name)
	}
}

func (v *SpecValidator) validateTotal(rawFields map[string]json.RawMessage) error {
	var sum int64
	for _, field := range v.spec.Total.Fields {
		raw, ok := rawFields[field]
		if !ok {
			return fmt.Errorf("total requires parameter %q", field)
		}
		value, err := decodeInteger(raw)
		if err != nil {
			return fmt.Errorf("total parameter %q must be an integer", field)
		}
		if value > math.MaxInt64-sum {
			return fmt.Errorf("total for parameters %v overflows the fixed-point scale", v.spec.Total.Fields)
		}
		sum += value
	}
	if sum != v.spec.Total.Total {
		return fmt.Errorf("parameters %v total %d, want %d", v.spec.Total.Fields, sum, v.spec.Total.Total)
	}
	return nil
}

func decodeInteger(raw json.RawMessage) (int64, error) {
	var number json.Number
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&number); err != nil {
		return 0, err
	}
	return number.Int64()
}
