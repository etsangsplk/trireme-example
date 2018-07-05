package policyexample

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"go.aporeto.io/trireme-lib/common"
	"go.aporeto.io/trireme-lib/controller"
	"go.aporeto.io/trireme-lib/policy"
	"go.uber.org/zap"
)

// CustomPolicyResolver is a simple policy engine
type CustomPolicyResolver struct {
	triremeNets []string
	policies    map[string]*CachedPolicy
	controller  controller.TriremeController
}

// CachedPolicy is a policy for a single container as read by a file
type CachedPolicy struct {
	ApplicationACLs *policy.IPRuleList
	NetworkACLs     *policy.IPRuleList
	Dependencies    policy.TagSelectorList
	ExposureRules   policy.TagSelectorList
}

// LoadPolicies loads a set of policies defined in a JSON file
func LoadPolicies(file string) map[string]*CachedPolicy {
	var config map[string]*CachedPolicy

	defaultConfig := &CachedPolicy{
		ApplicationACLs: &policy.IPRuleList{},
		NetworkACLs:     &policy.IPRuleList{},
		Dependencies:    policy.TagSelectorList{},
		ExposureRules:   policy.TagSelectorList{},
	}

	configFile, err := os.Open(file)
	if err != nil {
		configFile.Close() //nolint
		zap.L().Warn("No policy file found - using defaults")
		return map[string]*CachedPolicy{
			"default": defaultConfig,
		}
	}

	jsonParser := json.NewDecoder(configFile)
	err = jsonParser.Decode(&config)
	if err != nil {
		zap.L().Error("Invalid policies - using default")
	}

	config["default"] = defaultConfig

	configFile.Close() //nolint

	zap.L().Info("Using policy from file", zap.String("Policy File", file))

	return config
}

// GetPolicyIndex assumes that one of the labels of the PU is
// PolicyIndex and returns the corresponding value
func GetPolicyIndex(runtimeInfo policy.RuntimeReader) (string, error) {

	tags := runtimeInfo.Tags()

	for _, tag := range tags.GetSlice() {
		parts := strings.SplitN(tag, "=", 2)
		if strings.HasPrefix(parts[0], "@usr:PolicyIndex") || strings.HasPrefix(parts[0], "@usr:user") {
			zap.L().Info("Using policy from file", zap.String("Policy ID", parts[1]))
			return parts[1], nil
		}
	}

	return "", fmt.Errorf("PolicyIndex Not Found")
}

// NewCustomPolicyResolver creates a new example policy engine for the Trireme package
func NewCustomPolicyResolver(controller controller.TriremeController, networks []string, policyFile string) *CustomPolicyResolver {

	policies := LoadPolicies(policyFile)

	return &CustomPolicyResolver{
		triremeNets: networks,
		policies:    policies,
		controller:  controller,
	}
}

// HandlePUEvent implements the Trireme Policy interface. Once policy is resolved
// the resolver must call the controller to enforce the policy.
func (p *CustomPolicyResolver) HandlePUEvent(ctx context.Context, puID string, event common.Event, runtimeInfo policy.RuntimeReader) error {

	zap.L().Info("Resolving policy for container",
		zap.String("containerID", puID),
		zap.String("name", runtimeInfo.Name()),
	)

	policyIndex, err := GetPolicyIndex(runtimeInfo)
	if err != nil {
		zap.L().Warn("Cannot find requested policy index - Associating default policy")
		policyIndex = "default"
	}

	puPolicy, ok := p.policies[policyIndex]
	if !ok {
		return fmt.Errorf("No policy found")
	}

	// For the default policy we accept traffic with the same labels
	if policyIndex == "default" {
		puPolicy.Dependencies = p.createDefaultRules(runtimeInfo)
		puPolicy.ExposureRules = puPolicy.Dependencies
	}

	// Use the bridge IP from Docker.
	ipl := policy.ExtendedMap{}

	containerPolicyInfo := policy.NewPUPolicy(
		puID,
		policy.Police,
		*puPolicy.ApplicationACLs,
		*puPolicy.NetworkACLs,
		puPolicy.Dependencies,
		puPolicy.ExposureRules,
		runtimeInfo.Tags(),
		runtimeInfo.Tags(),
		ipl,
		p.triremeNets,
		[]string{},
		nil,
		nil,
		nil,
		[]string{},
	)

	switch event {
	case common.EventStart:
		return p.controller.Enforce(ctx, puID, containerPolicyInfo, runtimeInfo.(*policy.PURuntime))
	case common.EventPause:
		return p.controller.UnEnforce(ctx, puID, containerPolicyInfo, runtimeInfo.(*policy.PURuntime))
	case common.EventUnpause:
		return p.controller.Enforce(ctx, puID, containerPolicyInfo, runtimeInfo.(*policy.PURuntime))
	case common.EventStop:
		return p.controller.UnEnforce(ctx, puID, containerPolicyInfo, runtimeInfo.(*policy.PURuntime))
	default:
		return nil
	}
}

// CreateRuleDB creates a simple Rule DB that accepts packets from
// containers with the same labels as the instantiated container.
// If any of the labels matches, the packet is accepted.
func (p *CustomPolicyResolver) createDefaultRules(runtimeInfo policy.RuntimeReader) policy.TagSelectorList {

	selectorList := policy.TagSelectorList{}

	tags := runtimeInfo.Tags()

	i := 0

	for _, tag := range tags.GetSlice() {
		parts := strings.SplitN(tag, "=", 2)
		kv := policy.KeyValueOperator{
			Key:      parts[0],
			Value:    []string{parts[1]},
			Operator: policy.Equal,
		}
		tagSelector := policy.TagSelector{
			Clause: []policy.KeyValueOperator{kv},
			Policy: &policy.FlowPolicy{
				Action:   policy.Accept,
				PolicyID: strconv.Itoa(i),
			},
		}
		selectorList = append(selectorList, tagSelector)
		i++
	}

	// Add a default deny policy that rejects always from "namespace=bad"
	kv := policy.KeyValueOperator{
		Key:      "namespace",
		Value:    []string{"bad"},
		Operator: policy.Equal,
	}
	tagSelector := policy.TagSelector{
		Clause: []policy.KeyValueOperator{kv},
		Policy: &policy.FlowPolicy{
			Action:   policy.Reject,
			PolicyID: strconv.Itoa(i),
		},
	}

	selectorList = append(selectorList, tagSelector)
	for i, selector := range selectorList {
		for j, clause := range selector.Clause {
			zap.L().Info("Trireme policy for container",
				zap.String("name", runtimeInfo.Name()),
				zap.Int("selector", i),
				zap.Int("clause", j),
				zap.String("selector", fmt.Sprintf("%#v", clause)),
				zap.String("policy", fmt.Sprintf("%#v", selector.Policy)),
			)
		}
	}

	zap.L().Info("Trireme tags for container",
		zap.String("name", runtimeInfo.Name()),
		zap.String("tags", fmt.Sprintf("%#v", runtimeInfo.Tags())),
	)

	return selectorList

}
