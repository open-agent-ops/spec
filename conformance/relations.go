package conformance

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strconv"
	"time"
)

const (
	changePackageSchema    = "https://agent-ops.ru/schemas/c4_change_package.schema.json"
	executionRecordSchema  = "https://agent-ops.ru/schemas/c4_execution_record.schema.json"
	authorityReceiptSchema = "https://agent-ops.ru/schemas/c4_authority_receipt.schema.json"
	outcomeSchema          = "https://agent-ops.ru/schemas/c4_outcome.schema.json"
	lifecycleEventSchema   = "https://agent-ops.ru/schemas/c4_lifecycle_event.schema.json"
	checkpointSchema       = "https://agent-ops.ru/schemas/c4_checkpoint.schema.json"
	humanDecisionSchema    = "https://agent-ops.ru/schemas/foundation_human_decision.schema.json"
	outcomeCheckSchema     = "https://agent-ops.ru/schemas/foundation_outcome_check.schema.json"
)

// RelationInput supplies exact JSON resources. Every resource is schema-checked
// before any cross-record relation is evaluated.
type RelationInput struct {
	ChangePackage     json.RawMessage
	HumanDecision     json.RawMessage
	ExecutionRecords  []json.RawMessage
	AuthorityReceipts []json.RawMessage
	Outcome           json.RawMessage
	OutcomeCheck      json.RawMessage
	LifecycleEvents   []json.RawMessage
	Checkpoint        json.RawMessage
}

type RelationFinding struct {
	Validator string `json:"validator"`
	Level     int    `json:"level"`
	Relation  string `json:"relation,omitempty"`
	Code      string `json:"code"`
	Subject   string `json:"subject"`
}

type RelationResult struct {
	Catalog  string            `json:"catalog"`
	Profile  string            `json:"profile"`
	Accepted bool              `json:"accepted"`
	Complete bool              `json:"complete"`
	Findings []RelationFinding `json:"findings"`
	Claim    string            `json:"claim"`
}

type relationTarget struct {
	UID        string `json:"uid"`
	Generation int    `json:"generation"`
	Versions   struct {
		Resource string `json:"resource"`
		Policy   string `json:"policy"`
		Profile  string `json:"profile"`
	} `json:"versions"`
}

type relationAxes struct {
	Autonomy   string `json:"autonomy"`
	Impact     string `json:"impact"`
	Capability string `json:"capability"`
	Guardian   string `json:"guardian"`
}

type evidenceRequirement struct {
	ID           string         `json:"id"`
	ConsumerRule string         `json:"consumer_rule"`
	DecisionTime string         `json:"decision_time"`
	Target       relationTarget `json:"target"`
}

type changePackage struct {
	PackageID            string                `json:"package_id"`
	IntentID             string                `json:"intent_id"`
	OutcomeID            string                `json:"outcome_id"`
	Target               relationTarget        `json:"target"`
	Axes                 relationAxes          `json:"axes"`
	EvidenceRequirements []evidenceRequirement `json:"evidence_requirements"`
	PolicyEpoch          int                   `json:"policy_epoch"`
	ApprovalEpoch        int                   `json:"approval_epoch"`
	NotBefore            string                `json:"not_before"`
	ExpiresAt            string                `json:"expires_at"`
	AttemptBudget        int                   `json:"attempt_budget"`
	ConflictDomainSHA256 string                `json:"conflict_domain_sha256"`
	DecisionRef          string                `json:"decision_ref"`
	DecisionSHA256       string                `json:"decision_sha256"`
	Operation            struct {
		IDempotentSameKey      bool `json:"idempotent_same_key"`
		SupportsFencing        bool `json:"supports_fencing"`
		SupportsReconciliation bool `json:"supports_reconciliation"`
	} `json:"operation"`
}

type evidenceFact struct {
	ID                    string         `json:"id"`
	ConsumerRule          string         `json:"consumer_rule"`
	Applicable            bool           `json:"applicable"`
	Target                relationTarget `json:"target"`
	CheckedAt             string         `json:"checked_at"`
	FreshAtUse            bool           `json:"fresh_at_use"`
	Coverage              string         `json:"coverage"`
	DetectionCapable      bool           `json:"detection_capable"`
	Disposition           string         `json:"disposition"`
	Sufficiency           string         `json:"sufficiency"`
	ProvenanceBound       bool           `json:"provenance_bound"`
	CommonCausesDisclosed bool           `json:"common_causes_disclosed"`
}

type executionRecord struct {
	ExecutionID    string         `json:"execution_id"`
	PackageID      string         `json:"package_id"`
	PackageSHA256  string         `json:"package_sha256"`
	IntentID       string         `json:"intent_id"`
	OutcomeID      string         `json:"outcome_id"`
	Target         relationTarget `json:"target"`
	Axes           relationAxes   `json:"axes"`
	DecisionRef    string         `json:"decision_ref"`
	DecisionSHA256 string         `json:"decision_sha256"`
	HumanStart     struct {
		ID            string `json:"id"`
		OccurredAt    string `json:"occurred_at"`
		Authenticated bool   `json:"authenticated"`
	} `json:"human_start"`
	Reservation struct {
		ID                   string `json:"id"`
		ConflictDomainSHA256 string `json:"conflict_domain_sha256"`
		Fence                int    `json:"fence"`
		HeldAtAdmission      bool   `json:"held_at_admission"`
	} `json:"reservation"`
	Attempt struct {
		ID                string `json:"id"`
		Sequence          int    `json:"sequence"`
		IdempotencyKey    string `json:"idempotency_key"`
		RetryOf           string `json:"retry_of"`
		RetryBasis        string `json:"retry_basis"`
		ReconciliationRef string `json:"reconciliation_ref"`
	} `json:"attempt"`
	Admission struct {
		Authorized    bool   `json:"authorized"`
		LinearizedAt  string `json:"linearized_at"`
		PolicyEpoch   int    `json:"policy_epoch"`
		ApprovalEpoch int    `json:"approval_epoch"`
	} `json:"admission"`
	EvidenceFacts []evidenceFact `json:"evidence_facts"`
	Effect        struct {
		State                 string `json:"state"`
		EvidenceRequirementID string `json:"evidence_requirement_id"`
	} `json:"effect"`
	OpenObligationIDs []string `json:"open_obligation_ids"`
}

type authorityReceipt struct {
	ReceiptID      string         `json:"receipt_id"`
	ExecutionID    string         `json:"execution_id"`
	PackageSHA256  string         `json:"package_sha256"`
	DecisionRef    string         `json:"decision_ref"`
	DecisionSHA256 string         `json:"decision_sha256"`
	HumanStartID   string         `json:"human_start_id"`
	Target         relationTarget `json:"target"`
	PolicyEpoch    int            `json:"policy_epoch"`
	ApprovalEpoch  int            `json:"approval_epoch"`
	ReservationID  string         `json:"reservation_id"`
	Fence          int            `json:"fence"`
	Authorized     bool           `json:"authorized"`
	AdmittedAt     string         `json:"admitted_at"`
}

type outcomeRecord struct {
	OutcomeID          string         `json:"outcome_id"`
	IntentID           string         `json:"intent_id"`
	ExecutionID        string         `json:"execution_id"`
	Target             relationTarget `json:"target"`
	EffectState        string         `json:"effect_state"`
	State              string         `json:"state"`
	Completeness       string         `json:"completeness"`
	EvidenceFactIDs    []string       `json:"evidence_fact_ids"`
	OutcomeCheckRef    string         `json:"outcome_check_ref"`
	OutcomeCheckSHA256 string         `json:"outcome_check_sha256"`
	EvaluatedAt        string         `json:"evaluated_at"`
	OpenObligationIDs  []string       `json:"open_obligation_ids"`
}

type lifecycleEvent struct {
	EventID             string  `json:"event_id"`
	Sequence            int     `json:"sequence"`
	PreviousEventSHA256 *string `json:"previous_event_sha256"`
	Type                string  `json:"type"`
	PackageSHA256       string  `json:"package_sha256"`
	ExecutionID         string  `json:"execution_id"`
	AttemptSequence     int     `json:"attempt_sequence"`
	ObligationID        string  `json:"obligation_id"`
	Fence               int     `json:"fence"`
}

type checkpointRecord struct {
	PackageSHA256     string   `json:"package_sha256"`
	LogTailSequence   int      `json:"log_tail_sequence"`
	LogTailSHA256     string   `json:"log_tail_sha256"`
	AttemptsConsumed  int      `json:"attempts_consumed"`
	PolicyEpoch       int      `json:"policy_epoch"`
	ApprovalEpoch     int      `json:"approval_epoch"`
	CurrentFence      int      `json:"current_fence"`
	ReservationHeld   bool     `json:"reservation_held"`
	OpenObligationIDs []string `json:"open_obligation_ids"`
}

type humanDecision struct {
	Decision      string `json:"decision"`
	NotBefore     string `json:"not_before"`
	ExpiresAt     string `json:"expires_at"`
	Transition    string `json:"transition_id"`
	BlockedAction string `json:"blocked_action"`
	Ref           struct {
		ID string `json:"id"`
	} `json:"ref"`
	IntentRef struct {
		ID string `json:"id"`
	} `json:"intent_ref"`
	AuthorityScope struct {
		Transition    string `json:"transition_id"`
		BlockedAction string `json:"blocked_action"`
		IntentRef     struct {
			ID string `json:"id"`
		} `json:"intent_ref"`
	} `json:"authority_scope"`
}

type outcomeCheck struct {
	Ref struct {
		ID string `json:"id"`
	} `json:"ref"`
	IntentRef struct {
		ID string `json:"id"`
	} `json:"intent_ref"`
	DecisionRef struct {
		ID string `json:"id"`
	} `json:"decision_ref"`
}

type decodedRelationInput struct {
	Package    changePackage
	Decision   humanDecision
	Executions []executionRecord
	Receipts   []authorityReceipt
	Outcome    outcomeRecord
	Check      outcomeCheck
	Events     []lifecycleEvent
	Checkpoint checkpointRecord
}

// ValidateRelations checks the accepted bounded relation slice. It does not
// authenticate inputs or execute, authorize, reconcile, or release a change.
func (v *Validator) ValidateRelations(input RelationInput) RelationResult {
	catalog, catalogErr := OpenRelationCatalog()
	result := RelationResult{
		Catalog:  RelationCatalogID,
		Profile:  "agent-ops.c4-human-approved-apply@1.0.0",
		Findings: []RelationFinding{},
	}
	if catalogErr == nil {
		result.Claim = catalog.Claim
	}
	if v == nil || catalogErr != nil {
		addRelationFinding(&result, "AOM-C4-SCHEMA-001", 1, "", "trace_incomplete", "validator")
		return finishRelationResult(result)
	}

	type schemaItem struct {
		subject string
		schema  string
		data    json.RawMessage
	}
	items := []schemaItem{
		{"change_package", changePackageSchema, input.ChangePackage},
		{"human_decision", humanDecisionSchema, input.HumanDecision},
		{"outcome", outcomeSchema, input.Outcome},
		{"outcome_check", outcomeCheckSchema, input.OutcomeCheck},
		{"checkpoint", checkpointSchema, input.Checkpoint},
	}
	for i, data := range input.ExecutionRecords {
		items = append(items, schemaItem{subject: indexedSubject("execution", i), schema: executionRecordSchema, data: data})
	}
	for i, data := range input.AuthorityReceipts {
		items = append(items, schemaItem{subject: indexedSubject("receipt", i), schema: authorityReceiptSchema, data: data})
	}
	for i, data := range input.LifecycleEvents {
		items = append(items, schemaItem{subject: indexedSubject("event", i), schema: lifecycleEventSchema, data: data})
	}
	if len(input.ExecutionRecords) == 0 || len(input.ExecutionRecords) != len(input.AuthorityReceipts) || len(input.LifecycleEvents) == 0 {
		addRelationFinding(&result, "AOM-C4-SCHEMA-001", 1, "", "trace_incomplete", "trace")
	}
	for _, item := range items {
		if len(item.data) == 0 {
			addRelationFinding(&result, "AOM-C4-SCHEMA-001", 1, "", "trace_incomplete", item.subject)
			continue
		}
		if checked := v.Validate(item.schema, item.data); !checked.Accepted {
			addRelationFinding(&result, "AOM-C4-SCHEMA-001", 1, "", "schema_invalid", item.subject)
		}
	}
	if len(result.Findings) > 0 {
		return finishRelationResult(result)
	}

	decoded, ok := decodeRelationInput(input)
	if !ok {
		addRelationFinding(&result, "AOM-C4-SCHEMA-001", 1, "", "schema_invalid", "trace")
		return finishRelationResult(result)
	}
	result.Complete = true
	packageDigest := canonicalDigest(input.ChangePackage, "")
	approvalSubjectDigest := canonicalDigestOmitting(input.ChangePackage, "decision_ref", "decision_sha256")
	decisionDigest := canonicalDigest(input.HumanDecision, "")
	checkDigest := canonicalDigest(input.OutcomeCheck, "")

	validatePackageBindings(&result, decoded, packageDigest)
	validateAuthority(&result, decoded, decisionDigest, approvalSubjectDigest)
	validateReservations(&result, decoded)
	validateEvidenceAtUse(&result, decoded)
	validateRetry(&result, decoded)
	validateAttemptBudget(&result, decoded)
	validateLifecycle(&result, decoded, input.LifecycleEvents, packageDigest)
	validateCheckpoint(&result, decoded, input.LifecycleEvents, packageDigest)
	validateOutcome(&result, decoded, checkDigest)
	return finishRelationResult(result)
}

func decodeRelationInput(input RelationInput) (decodedRelationInput, bool) {
	var out decodedRelationInput
	if json.Unmarshal(input.ChangePackage, &out.Package) != nil || json.Unmarshal(input.HumanDecision, &out.Decision) != nil ||
		json.Unmarshal(input.Outcome, &out.Outcome) != nil || json.Unmarshal(input.OutcomeCheck, &out.Check) != nil ||
		json.Unmarshal(input.Checkpoint, &out.Checkpoint) != nil {
		return decodedRelationInput{}, false
	}
	for _, raw := range input.ExecutionRecords {
		var value executionRecord
		if json.Unmarshal(raw, &value) != nil {
			return decodedRelationInput{}, false
		}
		out.Executions = append(out.Executions, value)
	}
	for _, raw := range input.AuthorityReceipts {
		var value authorityReceipt
		if json.Unmarshal(raw, &value) != nil {
			return decodedRelationInput{}, false
		}
		out.Receipts = append(out.Receipts, value)
	}
	for _, raw := range input.LifecycleEvents {
		var value lifecycleEvent
		if json.Unmarshal(raw, &value) != nil {
			return decodedRelationInput{}, false
		}
		out.Events = append(out.Events, value)
	}
	return out, true
}

func validatePackageBindings(result *RelationResult, input decodedRelationInput, packageDigest string) {
	for _, execution := range input.Executions {
		if execution.PackageSHA256 != packageDigest || execution.PackageID != input.Package.PackageID || execution.IntentID != input.Package.IntentID || execution.OutcomeID != input.Package.OutcomeID {
			addRelationFinding(result, "AOM-C4-REL-001", 2, "C4-R01", "package_digest_mismatch", execution.ExecutionID)
		}
		if execution.Target != input.Package.Target {
			addRelationFinding(result, "AOM-C4-REL-001", 2, "C4-R01", "target_binding_mismatch", execution.ExecutionID)
		}
		if execution.Axes != input.Package.Axes {
			addRelationFinding(result, "AOM-C4-REL-001", 2, "C4-R01", "axis_binding_mismatch", execution.ExecutionID)
		}
	}
	for _, receipt := range input.Receipts {
		if receipt.PackageSHA256 != packageDigest {
			addRelationFinding(result, "AOM-C4-REL-001", 2, "C4-R01", "package_digest_mismatch", receipt.ReceiptID)
		}
		if receipt.Target != input.Package.Target {
			addRelationFinding(result, "AOM-C4-REL-001", 2, "C4-R01", "target_binding_mismatch", receipt.ReceiptID)
		}
	}
}

func validateAuthority(result *RelationResult, input decodedRelationInput, decisionDigest, approvalSubjectDigest string) {
	expectedAction := "c4-package-sha256:" + approvalSubjectDigest
	decisionOK := input.Package.DecisionRef == input.Decision.Ref.ID && input.Package.DecisionSHA256 == decisionDigest &&
		input.Decision.Decision == "approve" && input.Decision.IntentRef.ID == input.Package.IntentID &&
		input.Decision.Transition == "c4_apply_change_package" && input.Decision.BlockedAction == expectedAction &&
		input.Decision.AuthorityScope.IntentRef.ID == input.Package.IntentID &&
		input.Decision.AuthorityScope.Transition == input.Decision.Transition &&
		input.Decision.AuthorityScope.BlockedAction == expectedAction
	if !decisionOK {
		addRelationFinding(result, "AOM-C4-REL-002", 3, "C4-R02", "decision_binding_mismatch", input.Package.PackageID)
	}
	packageStart, packageStartOK := parseTime(input.Package.NotBefore)
	packageEnd, packageEndOK := parseTime(input.Package.ExpiresAt)
	decisionStart, decisionStartOK := parseTime(input.Decision.NotBefore)
	decisionEnd, decisionEndOK := parseTime(input.Decision.ExpiresAt)
	receipts := receiptByExecution(input.Receipts)
	for _, execution := range input.Executions {
		receipt, receiptOK := receipts[execution.ExecutionID]
		admission, admissionOK := parseTime(execution.Admission.LinearizedAt)
		start, startOK := parseTime(execution.HumanStart.OccurredAt)
		if execution.DecisionRef != input.Package.DecisionRef || execution.DecisionSHA256 != decisionDigest || !receiptOK ||
			receipt.DecisionRef != input.Package.DecisionRef || receipt.DecisionSHA256 != decisionDigest {
			addRelationFinding(result, "AOM-C4-REL-002", 3, "C4-R02", "decision_binding_mismatch", execution.ExecutionID)
		}
		if execution.Admission.PolicyEpoch != input.Package.PolicyEpoch || execution.Admission.ApprovalEpoch != input.Package.ApprovalEpoch || !receiptOK ||
			receipt.PolicyEpoch != input.Package.PolicyEpoch || receipt.ApprovalEpoch != input.Package.ApprovalEpoch {
			addRelationFinding(result, "AOM-C4-REL-002", 3, "C4-R02", "authority_epoch_stale", execution.ExecutionID)
		}
		if !admissionOK || !packageStartOK || !packageEndOK || !decisionStartOK || !decisionEndOK || admission.Before(packageStart) || admission.After(packageEnd) || admission.Before(decisionStart) || admission.After(decisionEnd) {
			addRelationFinding(result, "AOM-C4-REL-002", 3, "C4-R02", "authority_time_invalid", execution.ExecutionID)
		}
		if !execution.HumanStart.Authenticated || !startOK || !admissionOK || start.After(admission) || !receiptOK || receipt.HumanStartID != execution.HumanStart.ID {
			addRelationFinding(result, "AOM-C4-REL-002", 3, "C4-R03", "human_start_invalid", execution.ExecutionID)
		}
		if !execution.Admission.Authorized || !receiptOK || !receipt.Authorized || receipt.AdmittedAt != execution.Admission.LinearizedAt {
			addRelationFinding(result, "AOM-C4-REL-002", 3, "C4-R06", "admission_unauthorized", execution.ExecutionID)
		}
	}
}

func validateReservations(result *RelationResult, input decodedRelationInput) {
	receipts := receiptByExecution(input.Receipts)
	for _, execution := range input.Executions {
		receipt, ok := receipts[execution.ExecutionID]
		if !execution.Reservation.HeldAtAdmission || execution.Reservation.ConflictDomainSHA256 != input.Package.ConflictDomainSHA256 || !ok || receipt.ReservationID != execution.Reservation.ID {
			addRelationFinding(result, "AOM-C4-REL-003", 3, "C4-R04", "reservation_not_held", execution.ExecutionID)
		}
		if !input.Package.Operation.SupportsFencing || execution.Reservation.Fence < 1 || !ok || receipt.Fence != execution.Reservation.Fence {
			addRelationFinding(result, "AOM-C4-REL-003", 3, "C4-R05", "fence_stale", execution.ExecutionID)
		}
	}
}

func validateEvidenceAtUse(result *RelationResult, input decodedRelationInput) {
	for _, execution := range input.Executions {
		facts := map[string]evidenceFact{}
		for _, fact := range execution.EvidenceFacts {
			facts[fact.ID] = fact
		}
		for _, requirement := range input.Package.EvidenceRequirements {
			fact, ok := facts[requirement.ID]
			valid := ok && fact.ConsumerRule == requirement.ConsumerRule && fact.Applicable && fact.Target == requirement.Target && fact.Target == execution.Target &&
				fact.CheckedAt == execution.Admission.LinearizedAt && fact.FreshAtUse && fact.Coverage == "complete" && fact.DetectionCapable &&
				fact.Disposition == "supported" && fact.Sufficiency == "sufficient" && fact.ProvenanceBound && fact.CommonCausesDisclosed
			if !valid {
				addRelationFinding(result, "AOM-C4-REL-004", 5, "CEI-R02", "evidence_invalid_at_use", execution.ExecutionID+":"+requirement.ID)
			}
		}
	}
}

func validateRetry(result *RelationResult, input decodedRelationInput) {
	executions := append([]executionRecord(nil), input.Executions...)
	sort.Slice(executions, func(i, j int) bool { return executions[i].Attempt.Sequence < executions[j].Attempt.Sequence })
	for i, execution := range executions {
		if i == 0 {
			if execution.Attempt.RetryBasis != "initial" || execution.Attempt.RetryOf != "" {
				addRelationFinding(result, "AOM-C4-REL-005", 4, "C4-R08", "retry_unsafe", execution.ExecutionID)
			}
			continue
		}
		previous := executions[i-1]
		safe := execution.Attempt.RetryOf == previous.Attempt.ID
		switch execution.Attempt.RetryBasis {
		case "same_key":
			safe = safe && input.Package.Operation.IDempotentSameKey && execution.Attempt.IdempotencyKey == previous.Attempt.IdempotencyKey
		case "complete_not_applied":
			safe = safe && previous.Effect.State == "not_applied" && hasSupportingFact(previous, previous.Effect.EvidenceRequirementID)
		case "authorized_reconciliation":
			safe = safe && input.Package.Operation.SupportsReconciliation && execution.Attempt.ReconciliationRef != ""
		default:
			safe = false
		}
		if !safe {
			addRelationFinding(result, "AOM-C4-REL-005", 4, "C4-R08", "retry_unsafe", execution.ExecutionID)
		}
	}
}

func validateAttemptBudget(result *RelationResult, input decodedRelationInput) {
	seen := map[int]bool{}
	maxSequence := 0
	for _, execution := range input.Executions {
		sequence := execution.Attempt.Sequence
		if sequence > input.Package.AttemptBudget || seen[sequence] {
			addRelationFinding(result, "AOM-C4-REL-006", 4, "C4-R07", "attempt_budget_exceeded", execution.ExecutionID)
		}
		seen[sequence] = true
		if sequence > maxSequence {
			maxSequence = sequence
		}
	}
	if input.Checkpoint.AttemptsConsumed < maxSequence {
		addRelationFinding(result, "AOM-C4-REL-006", 4, "C4-R07", "attempt_budget_reset", "checkpoint")
	}
}

func validateLifecycle(result *RelationResult, input decodedRelationInput, raw []json.RawMessage, packageDigest string) {
	attemptEvents := map[int]bool{}
	for i, event := range input.Events {
		if event.Sequence != i+1 {
			addRelationFinding(result, "AOM-C4-REL-007", 4, "C4-R07", "lifecycle_sequence_invalid", event.EventID)
		}
		if event.PackageSHA256 != packageDigest {
			addRelationFinding(result, "AOM-C4-REL-001", 2, "C4-R01", "package_digest_mismatch", event.EventID)
		}
		if i == 0 {
			if event.PreviousEventSHA256 != nil {
				addRelationFinding(result, "AOM-C4-REL-007", 4, "C4-R12", "lifecycle_chain_invalid", event.EventID)
			}
		} else {
			expected := canonicalDigest(raw[i-1], "previous_event_sha256")
			if event.PreviousEventSHA256 == nil || *event.PreviousEventSHA256 != expected {
				addRelationFinding(result, "AOM-C4-REL-007", 4, "C4-R12", "lifecycle_chain_invalid", event.EventID)
			}
		}
		if event.Type == "attempt_admitted" {
			attemptEvents[event.AttemptSequence] = true
		}
	}
	for _, execution := range input.Executions {
		if !attemptEvents[execution.Attempt.Sequence] {
			addRelationFinding(result, "AOM-C4-REL-007", 4, "C4-R07", "lifecycle_sequence_invalid", execution.ExecutionID)
		}
	}
}

func validateCheckpoint(result *RelationResult, input decodedRelationInput, raw []json.RawMessage, packageDigest string) {
	if len(input.Events) == 0 {
		return
	}
	last := input.Events[len(input.Events)-1]
	lastDigest := canonicalDigest(raw[len(raw)-1], "previous_event_sha256")
	if input.Checkpoint.LogTailSequence != last.Sequence || input.Checkpoint.LogTailSHA256 != lastDigest {
		addRelationFinding(result, "AOM-C4-REL-008", 4, "C4-R12", "checkpoint_tail_mismatch", "checkpoint")
	}
	open := map[string]bool{}
	terminalRelease := false
	for _, event := range input.Events {
		switch event.Type {
		case "obligation_opened":
			open[event.ObligationID] = true
		case "obligation_resolved":
			delete(open, event.ObligationID)
		case "terminal_released":
			terminalRelease = true
		}
	}
	openList := make([]string, 0, len(open))
	for id := range open {
		openList = append(openList, id)
	}
	if !sameStrings(openList, input.Checkpoint.OpenObligationIDs) {
		addRelationFinding(result, "AOM-C4-REL-008", 4, "C4-R11", "checkpoint_obligation_omitted", "checkpoint")
	}
	latest := input.Executions[0]
	for _, execution := range input.Executions[1:] {
		if execution.Attempt.Sequence > latest.Attempt.Sequence {
			latest = execution
		}
	}
	if input.Checkpoint.PackageSHA256 != packageDigest || input.Checkpoint.PolicyEpoch != latest.Admission.PolicyEpoch ||
		input.Checkpoint.ApprovalEpoch != latest.Admission.ApprovalEpoch || input.Checkpoint.CurrentFence != latest.Reservation.Fence ||
		input.Checkpoint.ReservationHeld != latest.Reservation.HeldAtAdmission {
		addRelationFinding(result, "AOM-C4-REL-008", 4, "C4-R12", "checkpoint_state_mismatch", "checkpoint")
	}
	if terminalRelease && len(openList) > 0 {
		addRelationFinding(result, "AOM-C4-REL-009", 5, "C4-R13", "terminal_release_with_obligation", "lifecycle")
	}
}

func validateOutcome(result *RelationResult, input decodedRelationInput, checkDigest string) {
	executions := map[string]executionRecord{}
	for _, execution := range input.Executions {
		executions[execution.ExecutionID] = execution
	}
	execution, ok := executions[input.Outcome.ExecutionID]
	if !ok || input.Outcome.OutcomeID != input.Package.OutcomeID || input.Outcome.IntentID != input.Package.IntentID || input.Outcome.Target != input.Package.Target ||
		input.Outcome.OutcomeCheckRef != input.Check.Ref.ID || input.Outcome.OutcomeCheckSHA256 != checkDigest || input.Check.IntentRef.ID != input.Package.IntentID || input.Check.DecisionRef.ID != input.Package.DecisionRef {
		addRelationFinding(result, "AOM-C4-REL-009", 5, "C4-R10", "outcome_binding_mismatch", input.Outcome.OutcomeID)
		return
	}
	if input.Outcome.EffectState != execution.Effect.State {
		addRelationFinding(result, "AOM-C4-REL-009", 5, "C4-R09", "outcome_binding_mismatch", input.Outcome.OutcomeID)
	}
	evaluatedAt, evaluatedAtOK := parseTime(input.Outcome.EvaluatedAt)
	facts := make(map[string]evidenceFact, len(execution.EvidenceFacts))
	for _, fact := range execution.EvidenceFacts {
		facts[fact.ID] = fact
	}
	for _, id := range input.Outcome.EvidenceFactIDs {
		fact, found := facts[id]
		checkedAt, checkedAtOK := parseTime(fact.CheckedAt)
		if !found || !checkedAtOK || !evaluatedAtOK || checkedAt.After(evaluatedAt) {
			addRelationFinding(result, "AOM-C4-REL-009", 5, "C4-R10", "outcome_unsupported", input.Outcome.OutcomeID)
		}
	}
	if input.Outcome.EffectState != "unknown" && !containsString(input.Outcome.EvidenceFactIDs, execution.Effect.EvidenceRequirementID) {
		addRelationFinding(result, "AOM-C4-REL-009", 5, "C4-R09", "outcome_unsupported", input.Outcome.OutcomeID)
	}
	if !sameStrings(input.Outcome.OpenObligationIDs, input.Checkpoint.OpenObligationIDs) {
		addRelationFinding(result, "AOM-C4-REL-009", 5, "C4-R11", "outcome_binding_mismatch", input.Outcome.OutcomeID)
	}
	if input.Outcome.State == "succeeded" && (input.Outcome.EffectState != "applied" || input.Outcome.Completeness != "complete" || len(input.Outcome.OpenObligationIDs) != 0) {
		addRelationFinding(result, "AOM-C4-REL-009", 5, "C4-R10", "outcome_unsupported", input.Outcome.OutcomeID)
	}
}

func receiptByExecution(receipts []authorityReceipt) map[string]authorityReceipt {
	out := make(map[string]authorityReceipt, len(receipts))
	for _, receipt := range receipts {
		out[receipt.ExecutionID] = receipt
	}
	return out
}

func hasSupportingFact(execution executionRecord, id string) bool {
	for _, fact := range execution.EvidenceFacts {
		if fact.ID == id && fact.Applicable && fact.FreshAtUse && fact.Coverage == "complete" && fact.DetectionCapable && fact.Disposition == "supported" && fact.Sufficiency == "sufficient" && fact.ProvenanceBound && fact.CommonCausesDisclosed {
			return true
		}
	}
	return false
}

func parseTime(value string) (time.Time, bool) {
	t, err := time.Parse(time.RFC3339, value)
	return t, err == nil
}

func canonicalDigest(data []byte, omit string) string {
	if omit == "" {
		return canonicalDigestOmitting(data)
	}
	return canonicalDigestOmitting(data, omit)
}

func canonicalDigestOmitting(data []byte, omitted ...string) string {
	value, err := Decode(data)
	if err != nil {
		return ""
	}
	if object, ok := value.(map[string]any); ok {
		for _, field := range omitted {
			delete(object, field)
		}
	}
	canonical, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(canonical)
	return hex.EncodeToString(sum[:])
}

func indexedSubject(prefix string, index int) string {
	return prefix + "[" + strconv.Itoa(index) + "]"
}

func sameStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	a := append([]string(nil), left...)
	b := append([]string(nil), right...)
	sort.Strings(a)
	sort.Strings(b)
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func containsString(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func addRelationFinding(result *RelationResult, validator string, level int, relation, code, subject string) {
	finding := RelationFinding{Validator: validator, Level: level, Relation: relation, Code: code, Subject: subject}
	for _, existing := range result.Findings {
		if existing == finding {
			return
		}
	}
	result.Findings = append(result.Findings, finding)
}

func finishRelationResult(result RelationResult) RelationResult {
	sort.Slice(result.Findings, func(i, j int) bool {
		left, right := result.Findings[i], result.Findings[j]
		if left.Validator != right.Validator {
			return left.Validator < right.Validator
		}
		if left.Code != right.Code {
			return left.Code < right.Code
		}
		if left.Relation != right.Relation {
			return left.Relation < right.Relation
		}
		return left.Subject < right.Subject
	})
	result.Accepted = result.Complete && len(result.Findings) == 0
	return result
}
