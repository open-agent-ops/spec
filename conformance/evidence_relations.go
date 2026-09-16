package conformance

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"time"
)

const (
	evidenceRequirementSchema = "https://agent-ops.ru/schemas/evidence_requirement.schema.json"
	evidenceProfileSchema     = "https://agent-ops.ru/schemas/evidence_collection_profile.schema.json"
	evidenceReceiptSchema     = "https://agent-ops.ru/schemas/evidence_collection_receipt.schema.json"
	evidenceAssertionSchema   = "https://agent-ops.ru/schemas/evidence_assertion.schema.json"
	evidenceBundleV2Schema    = "https://agent-ops.ru/schemas/evidence_bundle_v2.schema.json"
)

type EvidenceRelationInput struct {
	Requirement json.RawMessage
	Profile     json.RawMessage
	Receipts    []json.RawMessage
	Assertions  []json.RawMessage
	Bundle      json.RawMessage
}

type EvidenceRelationResult struct {
	Catalog     string            `json:"catalog"`
	Profile     string            `json:"profile"`
	Syntactic   bool              `json:"syntactic_valid"`
	Relations   bool              `json:"relation_valid"`
	Sufficiency string            `json:"sufficiency"`
	Accepted    bool              `json:"accepted"`
	Findings    []RelationFinding `json:"findings"`
	Claim       string            `json:"claim"`
}

type evidenceTarget struct {
	Project                string `json:"project"`
	Environment            string `json:"environment"`
	Service                string `json:"service"`
	UID                    string `json:"uid"`
	Generation             int    `json:"generation"`
	PolicyEpoch            int    `json:"policy_epoch"`
	ProfileEpoch           int    `json:"profile_epoch"`
	CriticalVersionsSHA256 string `json:"critical_versions_sha256"`
}

type evidenceRequirementRecord struct {
	RequirementID  string         `json:"requirement_id"`
	ConsumerRule   string         `json:"consumer_rule"`
	DecisionTime   string         `json:"decision_time"`
	Target         evidenceTarget `json:"target"`
	SourceID       string         `json:"source_id"`
	ProfileID      string         `json:"profile_id"`
	MandatoryRoles []string       `json:"mandatory_roles"`
	MaxAgeSeconds  int            `json:"max_age_seconds"`
}

type evidenceProfileRecord struct {
	ProfileID          string         `json:"profile_id"`
	Authorized         bool           `json:"authorized"`
	AuthorizedBy       string         `json:"authorized_by"`
	ValidAt            string         `json:"valid_at"`
	Target             evidenceTarget `json:"target"`
	SourceID           string         `json:"source_id"`
	CollectorVersion   string         `json:"collector_version"`
	QuerySHA256        string         `json:"query_sha256"`
	ParametersSHA256   string         `json:"parameters_sha256"`
	DetectionCapable   bool           `json:"detection_capable"`
	ExpectedCoverage   string         `json:"expected_coverage"`
	AllowNotApplicable bool           `json:"allow_not_applicable"`
	MandatoryRoles     []string       `json:"mandatory_roles"`
}

type evidenceArtifactRecord struct {
	ArtifactID          string   `json:"artifact_id"`
	Content             string   `json:"content"`
	SHA256              string   `json:"sha256"`
	Coverage            string   `json:"coverage"`
	Sampled             bool     `json:"sampled"`
	Truncated           bool     `json:"truncated"`
	DroppedItems        int      `json:"dropped_items"`
	RedactionLossFields []string `json:"redaction_loss_fields"`
	LineageComplete     bool     `json:"lineage_complete"`
	InputArtifactIDs    []string `json:"input_artifact_ids"`
	CommonCauseIDs      []string `json:"common_cause_ids"`
	ProducerKind        string   `json:"producer_kind"`
}

type evidenceReceiptRecord struct {
	ReceiptID         string                 `json:"receipt_id"`
	RequirementID     string                 `json:"requirement_id"`
	ProfileID         string                 `json:"profile_id"`
	ProfileSHA256     string                 `json:"profile_sha256"`
	Target            evidenceTarget         `json:"target"`
	SourceID          string                 `json:"source_id"`
	AttemptSequence   int                    `json:"attempt_sequence"`
	PreviousReceiptID string                 `json:"previous_receipt_id"`
	CollectorVersion  string                 `json:"collector_version"`
	QuerySHA256       string                 `json:"query_sha256"`
	ParametersSHA256  string                 `json:"parameters_sha256"`
	StartedAt         string                 `json:"started_at"`
	CompletedAt       string                 `json:"completed_at"`
	ReceivedAt        string                 `json:"received_at"`
	Transport         string                 `json:"transport"`
	Parse             string                 `json:"parse"`
	Collection        string                 `json:"collection"`
	ErrorCode         string                 `json:"error_code"`
	Artifact          evidenceArtifactRecord `json:"artifact"`
}

type evidenceAssertionRecord struct {
	AssertionID           string         `json:"assertion_id"`
	RequirementID         string         `json:"requirement_id"`
	ReceiptIDs            []string       `json:"receipt_ids"`
	Target                evidenceTarget `json:"target"`
	SourceID              string         `json:"source_id"`
	Disposition           string         `json:"disposition"`
	Negative              bool           `json:"negative"`
	Coverage              string         `json:"coverage"`
	DetectionCapable      bool           `json:"detection_capable"`
	FreshAtUse            bool           `json:"fresh_at_use"`
	CheckedAt             string         `json:"checked_at"`
	DecisionTime          string         `json:"decision_time"`
	LineageComplete       bool           `json:"lineage_complete"`
	CommonCausesDisclosed bool           `json:"common_causes_disclosed"`
	Independent           bool           `json:"independent"`
	ConflictVisible       bool           `json:"conflict_visible"`
	Sufficiency           string         `json:"sufficiency"`
}

type evidenceBundleElement struct {
	ElementID   string `json:"element_id"`
	Role        string `json:"role"`
	AssertionID string `json:"assertion_id"`
}

type evidenceBundleCompletion struct {
	Role        string   `json:"role"`
	Mode        string   `json:"mode"`
	ElementIDs  []string `json:"element_ids"`
	OwnerRef    string   `json:"owner_ref"`
	DecisionRef string   `json:"decision_ref"`
	ReasonCode  string   `json:"reason_code"`
}

type evidenceBundleRecord struct {
	BundleID         string                     `json:"bundle_id"`
	RequirementID    string                     `json:"requirement_id"`
	ProfileID        string                     `json:"profile_id"`
	ConsumerRule     string                     `json:"consumer_rule"`
	Target           evidenceTarget             `json:"target"`
	State            string                     `json:"state"`
	Elements         []evidenceBundleElement    `json:"elements"`
	Completion       []evidenceBundleCompletion `json:"completion"`
	ReceiptIDs       []string                   `json:"receipt_ids"`
	ComputedComplete bool                       `json:"computed_complete"`
	Sufficiency      string                     `json:"sufficiency"`
	UnresolvedStates []string                   `json:"unresolved_states"`
	BundleSHA256     string                     `json:"bundle_sha256"`
	SealedAt         string                     `json:"sealed_at"`
	SupersedesID     string                     `json:"supersedes_id"`
}

type decodedEvidenceInput struct {
	Requirement evidenceRequirementRecord
	Profile     evidenceProfileRecord
	Receipts    []evidenceReceiptRecord
	Assertions  []evidenceAssertionRecord
	Bundle      evidenceBundleRecord
}

func (v *Validator) ValidateEvidenceRelations(input EvidenceRelationInput) EvidenceRelationResult {
	catalog, catalogErr := OpenEvidenceRelationCatalog()
	result := EvidenceRelationResult{Catalog: EvidenceRelationCatalogID, Profile: "agent-ops.evidence-acquisition@1.0.0", Sufficiency: "unknown", Findings: []RelationFinding{}}
	if catalogErr == nil {
		result.Claim = catalog.Claim
	}
	if v == nil || catalogErr != nil {
		addEvidenceFinding(&result, "AOM-EVD-SCHEMA-001", 1, "", "trace_incomplete", "validator")
		return finishEvidenceResult(result)
	}
	type schemaItem struct {
		subject, schema string
		data            json.RawMessage
	}
	items := []schemaItem{{"requirement", evidenceRequirementSchema, input.Requirement}, {"profile", evidenceProfileSchema, input.Profile}, {"bundle", evidenceBundleV2Schema, input.Bundle}}
	for i, data := range input.Receipts {
		items = append(items, schemaItem{indexedSubject("receipt", i), evidenceReceiptSchema, data})
	}
	for i, data := range input.Assertions {
		items = append(items, schemaItem{indexedSubject("assertion", i), evidenceAssertionSchema, data})
	}
	if len(input.Receipts) == 0 || len(input.Assertions) == 0 {
		addEvidenceFinding(&result, "AOM-EVD-SCHEMA-001", 1, "", "trace_incomplete", "trace")
	}
	for _, item := range items {
		if len(item.data) == 0 {
			addEvidenceFinding(&result, "AOM-EVD-SCHEMA-001", 1, "", "trace_incomplete", item.subject)
			continue
		}
		if checked := v.Validate(item.schema, item.data); !checked.Accepted {
			addEvidenceFinding(&result, "AOM-EVD-SCHEMA-001", 1, "", "schema_invalid", item.subject)
		}
	}
	if len(result.Findings) > 0 {
		return finishEvidenceResult(result)
	}
	result.Syntactic = true
	decoded, ok := decodeEvidenceInput(input)
	if !ok {
		addEvidenceFinding(&result, "AOM-EVD-SCHEMA-001", 1, "", "schema_invalid", "trace")
		return finishEvidenceResult(result)
	}

	validateEvidenceApplicability(&result, decoded, input.Profile)
	validateEvidenceAcquisition(&result, decoded)
	validateEvidenceAssertions(&result, decoded)
	validateEvidenceFreshness(&result, decoded)
	validateEvidenceIndependence(&result, decoded)
	validateEvidenceCompletion(&result, decoded, input.Bundle)
	return finishEvidenceResult(result)
}

func decodeEvidenceInput(input EvidenceRelationInput) (decodedEvidenceInput, bool) {
	var out decodedEvidenceInput
	if json.Unmarshal(input.Requirement, &out.Requirement) != nil || json.Unmarshal(input.Profile, &out.Profile) != nil || json.Unmarshal(input.Bundle, &out.Bundle) != nil {
		return out, false
	}
	for _, raw := range input.Receipts {
		var x evidenceReceiptRecord
		if json.Unmarshal(raw, &x) != nil {
			return out, false
		}
		out.Receipts = append(out.Receipts, x)
	}
	for _, raw := range input.Assertions {
		var x evidenceAssertionRecord
		if json.Unmarshal(raw, &x) != nil {
			return out, false
		}
		out.Assertions = append(out.Assertions, x)
	}
	return out, true
}

func validateEvidenceApplicability(result *EvidenceRelationResult, input decodedEvidenceInput, rawProfile json.RawMessage) {
	r, p, b := input.Requirement, input.Profile, input.Bundle
	if r.RequirementID == "" || b.RequirementID != r.RequirementID || b.ConsumerRule != r.ConsumerRule || b.Target != r.Target {
		addEvidenceFinding(result, "AOM-EVD-REL-001", 2, "EVD-R01", "requirement_not_applicable", r.RequirementID)
	}
	validAt, vok := parseTime(p.ValidAt)
	decision, dok := parseTime(r.DecisionTime)
	if !p.Authorized || p.AuthorizedBy == "" || !vok || !dok || validAt.After(decision) {
		addEvidenceFinding(result, "AOM-EVD-REL-001", 2, "EVD-R02", "profile_unauthorized", p.ProfileID)
	}
	if p.ProfileID != r.ProfileID || b.ProfileID != p.ProfileID || p.Target != r.Target || p.SourceID != r.SourceID || !sameStrings(p.MandatoryRoles, r.MandatoryRoles) {
		addEvidenceFinding(result, "AOM-EVD-REL-001", 2, "EVD-R02", "profile_binding_mismatch", p.ProfileID)
	}
	profileDigest := canonicalDigest(rawProfile, "")
	for _, receipt := range input.Receipts {
		if receipt.ProfileSHA256 != profileDigest || receipt.ProfileID != p.ProfileID || receipt.CollectorVersion != p.CollectorVersion || receipt.QuerySHA256 != p.QuerySHA256 || receipt.ParametersSHA256 != p.ParametersSHA256 {
			addEvidenceFinding(result, "AOM-EVD-REL-001", 2, "EVD-R02", "profile_binding_mismatch", receipt.ReceiptID)
		}
	}
}

func validateEvidenceAcquisition(result *EvidenceRelationResult, input decodedEvidenceInput) {
	receipts := append([]evidenceReceiptRecord(nil), input.Receipts...)
	sort.Slice(receipts, func(i, j int) bool { return receipts[i].AttemptSequence < receipts[j].AttemptSequence })
	seenIDs := map[string]bool{}
	for i, receipt := range receipts {
		if seenIDs[receipt.ReceiptID] || receipt.AttemptSequence != i+1 || (i == 0 && receipt.PreviousReceiptID != "") || (i > 0 && receipt.PreviousReceiptID != receipts[i-1].ReceiptID) {
			addEvidenceFinding(result, "AOM-EVD-REL-002", 3, "EVD-R03", "attempt_history_invalid", receipt.ReceiptID)
		}
		seenIDs[receipt.ReceiptID] = true
		if receipt.RequirementID != input.Requirement.RequirementID || receipt.ProfileID != input.Profile.ProfileID || receipt.Target != input.Requirement.Target || receipt.SourceID != input.Requirement.SourceID {
			addEvidenceFinding(result, "AOM-EVD-REL-002", 3, "EVD-R04", "receipt_binding_mismatch", receipt.ReceiptID)
		}
		started, sok := parseTime(receipt.StartedAt)
		completed, cok := parseTime(receipt.CompletedAt)
		received, rok := parseTime(receipt.ReceivedAt)
		if !sok || !cok || !rok || completed.Before(started) || received.Before(completed) {
			addEvidenceFinding(result, "AOM-EVD-REL-002", 3, "EVD-R04", "receipt_binding_mismatch", receipt.ReceiptID)
		}
		failure := receipt.Transport != "success" || receipt.Parse != "success" || receipt.Collection != "complete"
		if failure && receipt.ErrorCode == "" {
			addEvidenceFinding(result, "AOM-EVD-REL-002", 3, "EVD-R04", "acquisition_failure_hidden", receipt.ReceiptID)
		}
		digest := sha256.Sum256([]byte(receipt.Artifact.Content))
		if receipt.Artifact.SHA256 != hex.EncodeToString(digest[:]) {
			addEvidenceFinding(result, "AOM-EVD-REL-003", 3, "EVD-R05", "artifact_digest_mismatch", receipt.Artifact.ArtifactID)
		}
		complete := !failure && receipt.Artifact.Coverage == "complete" && !receipt.Artifact.Sampled && !receipt.Artifact.Truncated && receipt.Artifact.DroppedItems == 0
		if !complete {
			addEvidenceFinding(result, "AOM-EVD-REL-003", 3, "EVD-R05", "acquisition_incomplete", receipt.ReceiptID)
		}
		if !receipt.Artifact.LineageComplete || (receipt.Artifact.ProducerKind == "transform" && len(receipt.Artifact.InputArtifactIDs) == 0) {
			addEvidenceFinding(result, "AOM-EVD-REL-003", 3, "EVD-R06", "lineage_incomplete", receipt.Artifact.ArtifactID)
		}
		if len(receipt.Artifact.RedactionLossFields) != 0 {
			addEvidenceFinding(result, "AOM-EVD-REL-003", 3, "EVD-R06", "redaction_loss_hidden", receipt.Artifact.ArtifactID)
		}
	}
}

func validateEvidenceAssertions(result *EvidenceRelationResult, input decodedEvidenceInput) {
	receipts := map[string]evidenceReceiptRecord{}
	for _, receipt := range input.Receipts {
		receipts[receipt.ReceiptID] = receipt
	}
	for _, assertion := range input.Assertions {
		valid := assertion.RequirementID == input.Requirement.RequirementID && assertion.SourceID == input.Requirement.SourceID && assertion.LineageComplete
		for _, id := range assertion.ReceiptIDs {
			receipt, ok := receipts[id]
			if !ok || receipt.RequirementID != assertion.RequirementID {
				valid = false
			}
		}
		if !valid || assertion.Disposition == "unknown" || assertion.Sufficiency != "sufficient" {
			addEvidenceFinding(result, "AOM-EVD-REL-004", 4, "EVD-R07", "assertion_unsupported", assertion.AssertionID)
		}
		if assertion.Negative && assertion.Disposition == "supported" && (assertion.Coverage != "complete" || !assertion.DetectionCapable || !input.Profile.DetectionCapable) {
			addEvidenceFinding(result, "AOM-EVD-REL-004", 4, "EVD-R08", "negative_observation_invalid", assertion.AssertionID)
		}
		if assertion.Disposition == "conflicted" && !assertion.ConflictVisible {
			addEvidenceFinding(result, "AOM-EVD-REL-004", 4, "EVD-R07", "conflict_hidden", assertion.AssertionID)
		}
	}
}

func validateEvidenceFreshness(result *EvidenceRelationResult, input decodedEvidenceInput) {
	decision, dok := parseTime(input.Requirement.DecisionTime)
	for _, assertion := range input.Assertions {
		checked, cok := parseTime(assertion.CheckedAt)
		_, aok := parseTime(assertion.DecisionTime)
		fresh := assertion.Target == input.Requirement.Target && assertion.DecisionTime == input.Requirement.DecisionTime && dok && cok && aok && !checked.After(decision) && decision.Sub(checked) <= time.Duration(input.Requirement.MaxAgeSeconds)*time.Second
		if !assertion.FreshAtUse || !fresh {
			addEvidenceFinding(result, "AOM-EVD-REL-005", 5, "EVD-R09", "evidence_stale_at_use", assertion.AssertionID)
		}
		if !fresh && (assertion.FreshAtUse || assertion.Sufficiency == "sufficient" || input.Bundle.Sufficiency == "sufficient") {
			addEvidenceFinding(result, "AOM-EVD-REL-005", 5, "EVD-R14", "invalidation_not_propagated", assertion.AssertionID)
		}
	}
}

func validateEvidenceIndependence(result *EvidenceRelationResult, input decodedEvidenceInput) {
	receipts := map[string]evidenceReceiptRecord{}
	for _, receipt := range input.Receipts {
		receipts[receipt.ReceiptID] = receipt
	}
	for _, assertion := range input.Assertions {
		if !assertion.CommonCausesDisclosed || !assertion.Independent {
			addEvidenceFinding(result, "AOM-EVD-REL-006", 4, "EVD-R10", "independence_unsupported", assertion.AssertionID)
		}
		for _, id := range assertion.ReceiptIDs {
			if receipt, ok := receipts[id]; ok && receipt.Artifact.ProducerKind == "executor_response" {
				addEvidenceFinding(result, "AOM-EVD-REL-006", 4, "EVD-R10", "executor_response_not_effect_proof", assertion.AssertionID)
			}
		}
	}
}

func validateEvidenceCompletion(result *EvidenceRelationResult, input decodedEvidenceInput, rawBundle json.RawMessage) {
	elements := map[string]evidenceBundleElement{}
	assertions := map[string]bool{}
	for _, assertion := range input.Assertions {
		assertions[assertion.AssertionID] = true
	}
	for _, element := range input.Bundle.Elements {
		if elements[element.ElementID].ElementID != "" || !assertions[element.AssertionID] {
			addEvidenceFinding(result, "AOM-EVD-REL-007", 4, "EVD-R11", "completion_element_invalid", element.ElementID)
		}
		elements[element.ElementID] = element
	}
	roles := map[string]bool{}
	completionOK := len(input.Bundle.Elements) > 0
	for _, completion := range input.Bundle.Completion {
		if roles[completion.Role] {
			addEvidenceFinding(result, "AOM-EVD-REL-007", 4, "EVD-R11", "completion_role_invalid", completion.Role)
			completionOK = false
		}
		roles[completion.Role] = true
		switch completion.Mode {
		case "present":
			if len(completion.ElementIDs) == 0 {
				addEvidenceFinding(result, "AOM-EVD-REL-007", 4, "EVD-R11", "completion_element_invalid", completion.Role)
				completionOK = false
			}
			for _, id := range completion.ElementIDs {
				element, ok := elements[id]
				if !ok || element.Role != completion.Role {
					addEvidenceFinding(result, "AOM-EVD-REL-007", 4, "EVD-R11", "completion_element_invalid", completion.Role+":"+id)
					completionOK = false
				}
			}
		case "not_applicable":
			if !input.Profile.AllowNotApplicable || completion.OwnerRef == "" || completion.DecisionRef == "" || completion.ReasonCode == "" || len(completion.ElementIDs) != 0 {
				addEvidenceFinding(result, "AOM-EVD-REL-007", 4, "EVD-R11", "not_applicable_unauthorized", completion.Role)
				completionOK = false
			}
		}
	}
	for _, role := range input.Requirement.MandatoryRoles {
		if !roles[role] {
			addEvidenceFinding(result, "AOM-EVD-REL-007", 4, "EVD-R11", "completion_role_invalid", role)
			completionOK = false
		}
	}
	receiptIDs := map[string]bool{}
	for _, id := range input.Bundle.ReceiptIDs {
		receiptIDs[id] = true
	}
	for _, receipt := range input.Receipts {
		if !receiptIDs[receipt.ReceiptID] {
			addEvidenceFinding(result, "AOM-EVD-REL-007", 4, "EVD-R11", "bundle_history_incomplete", receipt.ReceiptID)
			completionOK = false
		}
	}
	if input.Bundle.SupersedesID != "" && len(input.Bundle.UnresolvedStates) == 0 {
		addEvidenceFinding(result, "AOM-EVD-REL-007", 4, "EVD-R11", "bundle_history_incomplete", input.Bundle.BundleID)
		completionOK = false
	}
	expectedDigest := canonicalDigestOmitting(rawBundle, "bundle_sha256")
	if input.Bundle.BundleSHA256 != expectedDigest {
		addEvidenceFinding(result, "AOM-EVD-REL-007", 4, "EVD-R12", "bundle_digest_mismatch", input.Bundle.BundleID)
		completionOK = false
	}
	sealOK := input.Bundle.State == "sealed" && input.Bundle.ComputedComplete && completionOK && len(input.Bundle.UnresolvedStates) == 0
	if !sealOK {
		addEvidenceFinding(result, "AOM-EVD-REL-007", 4, "EVD-R12", "bundle_seal_invalid", input.Bundle.BundleID)
	}
	if input.Bundle.Sufficiency != "sufficient" {
		addEvidenceFinding(result, "AOM-EVD-REL-008", 5, "EVD-R13", "consumer_sufficiency_invalid", input.Bundle.BundleID)
	}
}

func addEvidenceFinding(result *EvidenceRelationResult, validator string, level int, relation, code, subject string) {
	finding := RelationFinding{Validator: validator, Level: level, Relation: relation, Code: code, Subject: subject}
	for _, existing := range result.Findings {
		if existing == finding {
			return
		}
	}
	result.Findings = append(result.Findings, finding)
}

func finishEvidenceResult(result EvidenceRelationResult) EvidenceRelationResult {
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
	relationValid := result.Syntactic
	sufficiencyValid := result.Syntactic
	for _, finding := range result.Findings {
		if finding.Level >= 2 && finding.Level <= 4 {
			relationValid = false
			sufficiencyValid = false
		}
		if finding.Level == 5 {
			sufficiencyValid = false
		}
	}
	result.Relations = relationValid
	if result.Syntactic {
		result.Sufficiency = "insufficient"
		if sufficiencyValid {
			result.Sufficiency = "sufficient"
		}
	}
	result.Accepted = result.Relations && result.Sufficiency == "sufficient"
	return result
}
