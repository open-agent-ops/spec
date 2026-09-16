package conformance

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

type integratedMutation struct {
	Pointer string `json:"pointer"`
	Value   any    `json:"value"`
}

type integratedTraceCorpus struct {
	Format                  string               `json:"format"`
	BaseConsumerExecutionID string               `json:"base_consumer_execution_id"`
	BaseC4Mutations         []integratedMutation `json:"base_c4_mutations"`
	BaseEvidenceMutations   []integratedMutation `json:"base_evidence_mutations"`
	Cases                   []struct {
		ID                     string               `json:"id"`
		PositiveID             string               `json:"positive_id"`
		ConsumerExecutionID    string               `json:"consumer_execution_id"`
		SetupC4Mutations       []integratedMutation `json:"setup_c4_mutations"`
		SetupEvidenceMutations []integratedMutation `json:"setup_evidence_mutations"`
		C4Mutations            []integratedMutation `json:"c4_mutations"`
		EvidenceMutations      []integratedMutation `json:"evidence_mutations"`
		Code                   string               `json:"code"`
		Relation               string               `json:"relation"`
	} `json:"cases"`
}

func TestIntegratedCatalogProtocolAndPairedFixtures(t *testing.T) {
	catalog, err := OpenIntegratedCatalog()
	if err != nil {
		t.Fatal(err)
	}
	protocol, err := OpenEvaluationProtocol()
	if err != nil {
		t.Fatal(err)
	}
	if catalog.EvaluationProtocol != protocol.ProtocolID {
		t.Fatal("integrated catalog does not pin the evaluation protocol")
	}
	fixtures := loadIntegratedTraceFixtures(t)
	v, err := New()
	if err != nil {
		t.Fatal(err)
	}
	for _, fixture := range fixtures.Cases {
		t.Run(fixture.ID, func(t *testing.T) {
			positive := integratedFixtureInput(t, fixtures, fixture.ConsumerExecutionID, fixture.SetupC4Mutations, fixture.SetupEvidenceMutations, nil, nil)
			accepted := v.ValidateIntegrated(positive)
			if !accepted.Accepted || accepted.ConsumerDecision != "allowed" || !accepted.CrossBoundary || len(accepted.Findings) != 0 {
				t.Fatalf("positive pair %s rejected: %+v", fixture.PositiveID, accepted)
			}

			negative := integratedFixtureInput(t, fixtures, fixture.ConsumerExecutionID, fixture.SetupC4Mutations, fixture.SetupEvidenceMutations, fixture.C4Mutations, fixture.EvidenceMutations)
			result := v.ValidateIntegrated(negative)
			if result.Accepted || !result.Syntactic || result.ConsumerDecision != "blocked" || result.CrossBoundary {
				t.Fatalf("negative fixture accepted or misclassified: %+v", result)
			}
			if !hasRelationFinding(result.Findings, fixture.Code, fixture.Relation) {
				t.Fatalf("missing %s/%s in %+v", fixture.Code, fixture.Relation, result.Findings)
			}
		})
	}
}

func TestIntegratedOrderingDoesNotChangeDisposition(t *testing.T) {
	fixtures := loadIntegratedTraceFixtures(t)
	input := integratedFixtureInput(t, fixtures, fixtures.BaseConsumerExecutionID, nil, nil, nil, nil)
	v, err := New()
	if err != nil {
		t.Fatal(err)
	}
	want := v.ValidateIntegrated(input)
	input.C4.ExecutionRecords[0], input.C4.ExecutionRecords[1] = input.C4.ExecutionRecords[1], input.C4.ExecutionRecords[0]
	input.C4.AuthorityReceipts[0], input.C4.AuthorityReceipts[1] = input.C4.AuthorityReceipts[1], input.C4.AuthorityReceipts[0]
	got := v.ValidateIntegrated(input)
	if got.Accepted != want.Accepted || got.ConsumerDecision != want.ConsumerDecision || !reflect.DeepEqual(got.Findings, want.Findings) {
		t.Fatalf("record order changed disposition: want=%+v got=%+v", want, got)
	}
}

func TestIntegratedUnknownCannotImproveDecision(t *testing.T) {
	fixtures := loadIntegratedTraceFixtures(t)
	input := integratedFixtureInput(t, fixtures, fixtures.BaseConsumerExecutionID, nil, nil, nil, []integratedMutation{
		{Pointer: "/assertions/0/disposition", Value: "unknown"},
		{Pointer: "/assertions/0/sufficiency", Value: "unknown"},
		{Pointer: "/bundle/sufficiency", Value: "unknown"},
	})
	v, err := New()
	if err != nil {
		t.Fatal(err)
	}
	result := v.ValidateIntegrated(input)
	if result.Accepted || result.ConsumerDecision != "blocked" || result.EvidenceSufficiency == "sufficient" {
		t.Fatalf("unknown improved the consuming decision: %+v", result)
	}
}

func TestMalformedIntegratedInputDoesNotPanic(t *testing.T) {
	v, err := New()
	if err != nil {
		t.Fatal(err)
	}
	result := v.ValidateIntegrated(IntegratedInput{})
	if result.Accepted || result.Syntactic || result.ConsumerDecision != "unknown" {
		t.Fatalf("malformed input accepted: %+v", result)
	}
}

func TestEvaluationProtocolKeepsEveryAssignedOutcomeInDenominator(t *testing.T) {
	protocol, err := OpenEvaluationProtocol()
	if err != nil {
		t.Fatal(err)
	}
	for _, outcome := range []string{"succeeded", "failed", "timeout", "partial", "unknown"} {
		if !containsString(protocol.DenominatorOutcomes, outcome) {
			t.Fatalf("%s omitted from denominator", outcome)
		}
	}
	for _, metric := range []string{"unauthorized_effect", "checkpoint_refinement_violation", "false_complete_acceptance", "negative_evidence_false_claim", "snapshot_method_reproducibility"} {
		if !containsString(protocol.PrimaryMetrics, metric) {
			t.Fatalf("required primary metric missing: %s", metric)
		}
	}
}

func loadIntegratedTraceFixtures(t *testing.T) integratedTraceCorpus {
	t.Helper()
	b, err := os.ReadFile("integratedtestdata/traces.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixtures integratedTraceCorpus
	if json.Unmarshal(b, &fixtures) != nil || fixtures.Format != integratedFixtureFormatID || fixtures.BaseConsumerExecutionID == "" || len(fixtures.Cases) != 5 {
		t.Fatal("integrated fixture corpus invalid")
	}
	return fixtures
}

func integratedFixtureInput(t *testing.T, fixtures integratedTraceCorpus, consumerExecutionID string, setupC4, setupEvidence, c4Mutations, evidenceMutations []integratedMutation) IntegratedInput {
	t.Helper()
	c4 := cloneFixture(t, loadTraceFixtures(t).Base)
	evidence := cloneFixture(t, loadEvidenceTraceFixtures(t).Base)
	applyIntegratedMutations(t, c4, fixtures.BaseC4Mutations)
	applyIntegratedMutations(t, evidence, fixtures.BaseEvidenceMutations)
	applyIntegratedMutations(t, c4, setupC4)
	applyIntegratedMutations(t, evidence, setupEvidence)
	applyIntegratedMutations(t, c4, c4Mutations)
	applyIntegratedMutations(t, evidence, evidenceMutations)
	normalizeIntegratedC4(t, c4)
	normalizeIntegratedEvidence(t, evidence, c4)
	return IntegratedInput{
		C4:                  relationInputFromFixture(t, c4),
		Evidence:            evidenceInputFromFixture(t, evidence),
		ConsumerExecutionID: consumerExecutionID,
	}
}

func applyIntegratedMutations(t *testing.T, root map[string]any, mutations []integratedMutation) {
	t.Helper()
	for _, mutation := range mutations {
		setFixturePointer(t, root, mutation.Pointer, mutation.Value)
	}
}

func normalizeIntegratedC4(t *testing.T, root map[string]any) {
	t.Helper()
	changePackage := fixtureObject(t, root["change_package"])
	decision := fixtureObject(t, root["human_decision"])
	approvalSubject := fixtureDigest(t, changePackage, "decision_ref", "decision_sha256")
	blockedAction := "c4-package-sha256:" + approvalSubject
	decision["blocked_action"] = blockedAction
	fixtureObject(t, decision["authority_scope"])["blocked_action"] = blockedAction
	decisionDigest := fixtureDigest(t, decision)
	changePackage["decision_sha256"] = decisionDigest
	for _, value := range fixtureList(t, root["execution_records"]) {
		fixtureObject(t, value)["decision_sha256"] = decisionDigest
	}
	for _, value := range fixtureList(t, root["authority_receipts"]) {
		fixtureObject(t, value)["decision_sha256"] = decisionDigest
	}

	packageDigest := fixtureDigest(t, changePackage)
	for _, value := range fixtureList(t, root["execution_records"]) {
		fixtureObject(t, value)["package_sha256"] = packageDigest
	}
	for _, value := range fixtureList(t, root["authority_receipts"]) {
		fixtureObject(t, value)["package_sha256"] = packageDigest
	}
	events := fixtureList(t, root["lifecycle_events"])
	for i, value := range events {
		event := fixtureObject(t, value)
		event["package_sha256"] = packageDigest
		if i == 0 {
			event["previous_event_sha256"] = nil
		} else {
			event["previous_event_sha256"] = fixtureDigest(t, fixtureObject(t, events[i-1]), "previous_event_sha256")
		}
	}
	checkpoint := fixtureObject(t, root["checkpoint"])
	checkpoint["package_sha256"] = packageDigest
	checkpoint["log_tail_sequence"] = len(events)
	checkpoint["log_tail_sha256"] = fixtureDigest(t, fixtureObject(t, events[len(events)-1]), "previous_event_sha256")
	outcome := fixtureObject(t, root["outcome"])
	outcome["outcome_check_sha256"] = fixtureDigest(t, fixtureObject(t, root["outcome_check"]))
}

func normalizeIntegratedEvidence(t *testing.T, evidence, c4 map[string]any) {
	t.Helper()
	changePackage := fixtureObject(t, c4["change_package"])
	c4Target := fixtureObject(t, changePackage["target"])
	versionsDigest := fixtureDigest(t, fixtureObject(t, c4Target["versions"]))
	policyEpoch := changePackage["policy_epoch"]
	setTarget := func(record map[string]any) {
		target := fixtureObject(t, record["target"])
		target["uid"] = c4Target["uid"]
		target["generation"] = c4Target["generation"]
		target["policy_epoch"] = policyEpoch
		target["critical_versions_sha256"] = versionsDigest
	}
	requirement := fixtureObject(t, evidence["requirement"])
	profile := fixtureObject(t, evidence["profile"])
	setTarget(requirement)
	setTarget(profile)
	for _, value := range fixtureList(t, evidence["receipts"]) {
		receipt := fixtureObject(t, value)
		setTarget(receipt)
		artifact := fixtureObject(t, receipt["artifact"])
		sum := sha256.Sum256([]byte(artifact["content"].(string)))
		artifact["sha256"] = hex.EncodeToString(sum[:])
	}
	for _, value := range fixtureList(t, evidence["assertions"]) {
		setTarget(fixtureObject(t, value))
	}
	bundle := fixtureObject(t, evidence["bundle"])
	setTarget(bundle)
	profileDigest := fixtureDigest(t, profile)
	for _, value := range fixtureList(t, evidence["receipts"]) {
		fixtureObject(t, value)["profile_sha256"] = profileDigest
	}
	bundle["bundle_sha256"] = fixtureDigest(t, bundle, "bundle_sha256")
}

func fixtureObject(t *testing.T, value any) map[string]any {
	t.Helper()
	object, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("fixture object: %T", value)
	}
	return object
}

func fixtureList(t *testing.T, value any) []any {
	t.Helper()
	list, ok := value.([]any)
	if !ok {
		t.Fatalf("fixture list: %T", value)
	}
	return list
}

func fixtureDigest(t *testing.T, value map[string]any, omit ...string) string {
	t.Helper()
	b, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return canonicalDigestOmitting(b, omit...)
}
