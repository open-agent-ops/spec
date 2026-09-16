package conformance

import (
	"encoding/json"
	"sort"
)

type IntegratedInput struct {
	C4                  RelationInput
	Evidence            EvidenceRelationInput
	ConsumerExecutionID string
}

type IntegratedResult struct {
	Catalog             string                 `json:"catalog"`
	Profile             string                 `json:"profile"`
	EvaluationProtocol  string                 `json:"evaluation_protocol"`
	Syntactic           bool                   `json:"syntactic_valid"`
	C4Relations         bool                   `json:"c4_relation_valid"`
	EvidenceRelations   bool                   `json:"evidence_relation_valid"`
	EvidenceSufficiency string                 `json:"evidence_sufficiency"`
	CrossBoundary       bool                   `json:"cross_boundary_valid"`
	ConsumerDecision    string                 `json:"consumer_decision"`
	Accepted            bool                   `json:"accepted"`
	C4Result            RelationResult         `json:"c4_result"`
	EvidenceResult      EvidenceRelationResult `json:"evidence_result"`
	Findings            []RelationFinding      `json:"findings"`
	Claim               string                 `json:"claim"`
}

// ValidateIntegrated composes the exact C4 and Evidence validators and checks
// only the accepted interface projection. It does not collect evidence, execute
// a change, authenticate a source, or evaluate a deployed system.
func (v *Validator) ValidateIntegrated(input IntegratedInput) IntegratedResult {
	result := IntegratedResult{
		Catalog:             IntegratedCatalogID,
		Profile:             IntegratedProfileID,
		EvaluationProtocol:  EvaluationProtocolID,
		EvidenceSufficiency: "unknown",
		ConsumerDecision:    "unknown",
		Findings:            []RelationFinding{},
	}
	catalog, catalogErr := OpenIntegratedCatalog()
	_, protocolErr := OpenEvaluationProtocol()
	if catalogErr == nil {
		result.Claim = catalog.Claim
	}
	if v == nil || catalogErr != nil || protocolErr != nil {
		addIntegratedFinding(&result, "AOM-INT-REL-001", 2, "CEI-R01", "interface_binding_mismatch", "validator")
		return finishIntegratedResult(result)
	}

	result.C4Result = v.ValidateRelations(input.C4)
	result.EvidenceResult = v.ValidateEvidenceRelations(input.Evidence)
	result.Syntactic = result.C4Result.Complete && result.EvidenceResult.Syntactic
	result.C4Relations = result.C4Result.Accepted
	result.EvidenceRelations = result.EvidenceResult.Relations
	result.EvidenceSufficiency = result.EvidenceResult.Sufficiency
	if !result.Syntactic {
		return finishIntegratedResult(result)
	}

	c4, c4OK := decodeRelationInput(input.C4)
	evidence, evidenceOK := decodeEvidenceInput(input.Evidence)
	execution, executionOK := integratedExecution(c4.Executions, input.ConsumerExecutionID)
	assertion, assertionOK := integratedAssertion(evidence)
	if !c4OK || !evidenceOK || !executionOK || !assertionOK {
		addIntegratedFinding(&result, "AOM-INT-REL-001", 2, "CEI-R01", "interface_binding_mismatch", input.ConsumerExecutionID)
		return finishIntegratedResult(result)
	}

	validateIntegratedBinding(&result, c4, evidence, execution)
	validateIntegratedProjection(&result, evidence, assertion, execution)
	validateIntegratedEffect(&result, evidence, assertion, execution)
	validateIntegratedOutcome(&result, c4, execution)
	validateIntegratedRetry(&result, c4, evidence, assertion, execution)
	validateIntegratedHistory(&result, c4, evidence)
	return finishIntegratedResult(result)
}

func validateIntegratedBinding(result *IntegratedResult, c4 decodedRelationInput, evidence decodedEvidenceInput, execution executionRecord) {
	requirement, ok := c4Requirement(c4.Package.EvidenceRequirements, evidence.Requirement.RequirementID)
	versions, _ := json.Marshal(execution.Target.Versions)
	criticalVersions := canonicalDigest(versions, "")
	valid := ok && requirement.ConsumerRule == evidence.Requirement.ConsumerRule &&
		execution.Target.UID == evidence.Requirement.Target.UID && execution.Target.Generation == evidence.Requirement.Target.Generation &&
		execution.Admission.PolicyEpoch == evidence.Requirement.Target.PolicyEpoch &&
		evidence.Requirement.Target.CriticalVersionsSHA256 == criticalVersions &&
		evidence.Requirement.DecisionTime == execution.Admission.LinearizedAt
	if !valid {
		addIntegratedFinding(result, "AOM-INT-REL-001", 2, "CEI-R01", "interface_binding_mismatch", execution.ExecutionID)
	}
}

func validateIntegratedProjection(result *IntegratedResult, evidence decodedEvidenceInput, assertion evidenceAssertionRecord, execution executionRecord) {
	fact, factOK := c4Fact(execution.EvidenceFacts, evidence.Requirement.RequirementID)
	if !factOK || !fact.Applicable || evidence.Requirement.RequirementID == "" || !evidence.Profile.Authorized {
		addIntegratedFinding(result, "AOM-INT-REL-001", 2, "CEI-R01", "evidence_applicability_mismatch", execution.ExecutionID)
		return
	}
	if fact.FreshAtUse != assertion.FreshAtUse || fact.CheckedAt != assertion.DecisionTime || assertion.DecisionTime != execution.Admission.LinearizedAt {
		addIntegratedFinding(result, "AOM-INT-REL-001", 2, "CEI-R02", "evidence_freshness_mismatch", execution.ExecutionID)
	}
	detection := assertion.DetectionCapable && evidence.Profile.DetectionCapable
	if fact.Coverage != assertion.Coverage || fact.DetectionCapable != detection {
		addIntegratedFinding(result, "AOM-INT-REL-001", 2, "CEI-R03", "observation_coverage_mismatch", execution.ExecutionID)
	}
	if fact.Disposition != assertion.Disposition {
		addIntegratedFinding(result, "AOM-INT-REL-001", 2, "CEI-R04", "assertion_disposition_mismatch", execution.ExecutionID)
	}
	if fact.Sufficiency != evidence.Bundle.Sufficiency || assertion.Sufficiency != evidence.Bundle.Sufficiency {
		addIntegratedFinding(result, "AOM-INT-REL-001", 2, "CEI-R05", "evidence_sufficiency_mismatch", execution.ExecutionID)
	}
	lineage := assertion.LineageComplete
	for _, receipt := range evidence.Receipts {
		lineage = lineage && receipt.Artifact.LineageComplete
	}
	if fact.ProvenanceBound != lineage {
		addIntegratedFinding(result, "AOM-INT-REL-001", 2, "CEI-R06", "provenance_binding_mismatch", execution.ExecutionID)
	}
	if fact.CommonCausesDisclosed != assertion.CommonCausesDisclosed {
		addIntegratedFinding(result, "AOM-INT-REL-001", 2, "CEI-R07", "common_cause_disclosure_mismatch", execution.ExecutionID)
	}
}

func validateIntegratedEffect(result *IntegratedResult, evidence decodedEvidenceInput, assertion evidenceAssertionRecord, execution executionRecord) {
	if execution.Effect.State == "unknown" {
		return
	}
	for _, receipt := range evidence.Receipts {
		if receipt.Artifact.ProducerKind == "executor_response" && containsString(assertion.ReceiptIDs, receipt.ReceiptID) {
			addIntegratedFinding(result, "AOM-INT-REL-002", 5, "CEI-R06", "executor_response_effect_claim", execution.ExecutionID)
		}
	}
	if execution.Effect.State == "not_applied" && (!assertion.Negative || assertion.Coverage != "complete" || !assertion.DetectionCapable || !evidence.Profile.DetectionCapable || assertion.Disposition != "supported") {
		addIntegratedFinding(result, "AOM-INT-REL-002", 5, "CEI-R03", "negative_effect_observation_incomplete", execution.ExecutionID)
	}
}

func validateIntegratedOutcome(result *IntegratedResult, c4 decodedRelationInput, execution executionRecord) {
	if c4.Outcome.ExecutionID != execution.ExecutionID || c4.Outcome.State != "succeeded" {
		return
	}
	if !result.EvidenceResult.Accepted || result.EvidenceResult.Sufficiency != "sufficient" {
		addIntegratedFinding(result, "AOM-INT-REL-002", 5, "CEI-R05", "outcome_evidence_insufficient", c4.Outcome.OutcomeID)
	}
}

func validateIntegratedRetry(result *IntegratedResult, c4 decodedRelationInput, evidence decodedEvidenceInput, assertion evidenceAssertionRecord, execution executionRecord) {
	executions := append([]executionRecord(nil), c4.Executions...)
	sort.Slice(executions, func(i, j int) bool { return executions[i].Attempt.Sequence < executions[j].Attempt.Sequence })
	for i := 1; i < len(executions); i++ {
		if executions[i-1].ExecutionID != execution.ExecutionID || executions[i].Attempt.RetryBasis != "complete_not_applied" {
			continue
		}
		if executions[i-1].Effect.State != "not_applied" || !assertion.Negative || assertion.Coverage != "complete" || !assertion.DetectionCapable || !result.EvidenceResult.Accepted {
			addIntegratedFinding(result, "AOM-INT-REL-002", 5, "C4-R08", "retry_evidence_insufficient", executions[i].ExecutionID)
		}
	}
}

func integratedAssertion(evidence decodedEvidenceInput) (evidenceAssertionRecord, bool) {
	referenced := map[string]bool{}
	for _, element := range evidence.Bundle.Elements {
		referenced[element.AssertionID] = true
	}
	var selected evidenceAssertionRecord
	found := false
	for _, assertion := range evidence.Assertions {
		if assertion.RequirementID != evidence.Requirement.RequirementID || !referenced[assertion.AssertionID] {
			continue
		}
		if !found || assertion.AssertionID < selected.AssertionID {
			selected = assertion
			found = true
		}
	}
	return selected, found
}

func validateIntegratedHistory(result *IntegratedResult, c4 decodedRelationInput, evidence decodedEvidenceInput) {
	hiddenEvidenceHistory := evidence.Bundle.SupersedesID != "" && len(evidence.Bundle.UnresolvedStates) == 0
	open := map[string]bool{}
	for _, event := range c4.Events {
		switch event.Type {
		case "obligation_opened":
			open[event.ObligationID] = true
		case "obligation_resolved":
			delete(open, event.ObligationID)
		}
	}
	hiddenC4History := false
	for id := range open {
		if !containsString(c4.Checkpoint.OpenObligationIDs, id) {
			hiddenC4History = true
		}
	}
	if hiddenEvidenceHistory && hiddenC4History {
		addIntegratedFinding(result, "AOM-INT-REL-002", 5, "C4-R12", "cross_boundary_history_incomplete", "checkpoint:bundle")
	}
}

func integratedExecution(executions []executionRecord, id string) (executionRecord, bool) {
	for _, execution := range executions {
		if execution.ExecutionID == id {
			return execution, true
		}
	}
	return executionRecord{}, false
}

func c4Requirement(requirements []evidenceRequirement, id string) (evidenceRequirement, bool) {
	for _, requirement := range requirements {
		if requirement.ID == id {
			return requirement, true
		}
	}
	return evidenceRequirement{}, false
}

func c4Fact(facts []evidenceFact, id string) (evidenceFact, bool) {
	for _, fact := range facts {
		if fact.ID == id {
			return fact, true
		}
	}
	return evidenceFact{}, false
}

func addIntegratedFinding(result *IntegratedResult, validator string, level int, relation, code, subject string) {
	finding := RelationFinding{Validator: validator, Level: level, Relation: relation, Code: code, Subject: subject}
	for _, existing := range result.Findings {
		if existing == finding {
			return
		}
	}
	result.Findings = append(result.Findings, finding)
}

func finishIntegratedResult(result IntegratedResult) IntegratedResult {
	sort.Slice(result.Findings, func(i, j int) bool {
		a, b := result.Findings[i], result.Findings[j]
		if a.Validator != b.Validator {
			return a.Validator < b.Validator
		}
		if a.Code != b.Code {
			return a.Code < b.Code
		}
		if a.Relation != b.Relation {
			return a.Relation < b.Relation
		}
		return a.Subject < b.Subject
	})
	result.CrossBoundary = result.Syntactic && len(result.Findings) == 0
	result.Accepted = result.C4Result.Accepted && result.EvidenceResult.Accepted && result.CrossBoundary
	if result.Syntactic {
		result.ConsumerDecision = "blocked"
		if result.Accepted {
			result.ConsumerDecision = "allowed"
		}
	}
	return result
}
