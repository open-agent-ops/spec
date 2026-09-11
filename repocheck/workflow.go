package repocheck

import (
	"fmt"
	"reflect"
	"sort"
)

type ActionPin struct {
	Repository   string `json:"repository"`
	Version      string `json:"version"`
	Commit       string `json:"commit"`
	ActionSHA256 string `json:"action_sha256"`
	Source       string `json:"source"`
}
type ToolLock struct {
	Schema    string      `json:"schema"`
	GoVersion string      `json:"go_version"`
	GoMinimum string      `json:"go_minimum"`
	Actions   []ActionPin `json:"actions"`
	Execution string      `json:"execution"`
}

// Workflow validates the AST against a closed executable catalog. The expected
// commands are fixed source literals, not supplied by the workflow under test.
func Workflow(data []byte, lock ToolLock, release bool) error {
	pins, e := workflowPins(lock)
	if e != nil {
		return e
	}

	node, e := YAML(data)
	if e != nil {
		return e
	}
	var actual map[string]any
	if e := node.Decode(&actual); e != nil {
		return Invalidf("workflow: %v", e)
	}
	expected := WorkflowModel(lock, pins, release)
	if !reflect.DeepEqual(actual, expected) {
		return Rejectedf("workflow differs from the fixed model at %s", firstDifference(actual, expected, "$"))
	}
	return nil
}

// firstDifference returns the JSON-ish path and both values at the first point
// where two decoded YAML documents diverge. Values are truncated; the workflow
// under test is repository source, not contributor prose.
func firstDifference(actual, expected any, path string) string {
	const limit = 80
	show := func(v any) string {
		s := fmt.Sprintf("%v", v)
		if v == nil {
			s = "<absent>"
		}
		if len(s) > limit {
			s = s[:limit] + "..."
		}
		return s
	}
	switch e := expected.(type) {
	case map[string]any:
		a, ok := actual.(map[string]any)
		if !ok {
			return fmt.Sprintf("%s: got %T, want mapping", path, actual)
		}
		keys := make([]string, 0, len(e)+len(a))
		for k := range e {
			keys = append(keys, k)
		}
		for k := range a {
			if _, dup := e[k]; !dup {
				keys = append(keys, k)
			}
		}
		sort.Strings(keys)
		for _, k := range keys {
			av, aok := a[k]
			ev, eok := e[k]
			switch {
			case !aok:
				return fmt.Sprintf("%s.%s: missing, want %s", path, k, show(ev))
			case !eok:
				return fmt.Sprintf("%s.%s: unexpected key", path, k)
			case !reflect.DeepEqual(av, ev):
				return firstDifference(av, ev, path+"."+k)
			}
		}
	case []any:
		a, ok := actual.([]any)
		if !ok {
			return fmt.Sprintf("%s: got %T, want sequence", path, actual)
		}
		for i := 0; i < len(a) && i < len(e); i++ {
			if !reflect.DeepEqual(a[i], e[i]) {
				return firstDifference(a[i], e[i], fmt.Sprintf("%s[%d]", path, i))
			}
		}
		if len(a) != len(e) {
			return fmt.Sprintf("%s: %d items, want %d", path, len(a), len(e))
		}
	}
	if !reflect.DeepEqual(actual, expected) {
		return fmt.Sprintf("%s: got %s, want %s", path, show(actual), show(expected))
	}
	return path + ": no difference"
}

// WorkflowModel is also the generator input. Independent actionlint and private
// fixture checks are required in addition to this closed AST admission check.
func WorkflowModel(lock ToolLock, pins map[string]string, release bool) map[string]any {
	read := map[string]any{"contents": "read"}
	checkout := func(ref string) any {
		return map[string]any{"uses": pins["actions/checkout"], "with": map[string]any{"persist-credentials": false, "fetch-depth": 0, "ref": ref}}
	}
	upload := func(name string) any {
		return map[string]any{"name": "Retain complete evidence", "if": "${{ always() }}", "uses": pins["actions/upload-artifact"], "with": map[string]any{"name": name, "path": ".aom-evidence/", "if-no-files-found": "error", "retention-days": 90, "include-hidden-files": true}}
	}
	job := func(name, command string, timeout int, ref string) map[string]any {
		return map[string]any{"name": "AOM / " + name, "runs-on": "ubuntu-24.04", "permissions": read, "timeout-minutes": timeout, "steps": []any{checkout(ref), map[string]any{"name": "Verify and provision tools", "run": "sh scripts/provision.sh"}, map[string]any{"name": "Execute bounded check", "run": command}, upload(name + "-${{ github.run_id }}-${{ github.run_attempt }}")}}
	}
	env := map[string]any{"GOTOOLCHAIN": "local", "GOWORK": "off", "AOM_REPOSITORY": "${{ github.repository }}", "AOM_SHA": "${{ github.sha }}", "AOM_HEAD": "${{ github.event.pull_request.head.sha || github.sha }}", "AOM_BASE": "${{ github.event.pull_request.base.sha || github.sha }}", "AOM_RUN": "${{ github.run_id }}", "AOM_ATTEMPT": "${{ github.run_attempt }}", "AOM_WORKFLOW": "${{ github.workflow_sha }}", "AOM_EVENT": "${{ github.event_name }}"}
	if !release {
		jobs := map[string]any{}
		for _, name := range []string{"policy", "conformance", "docs", "supply-chain"} {
			jobs[name] = job(name, "go run -mod=readonly ./cmd/spec-check gate "+name, 20, "${{ github.sha }}")
		}
		for _, name := range []string{"heavy", "reproducibility"} {
			j := job(name, "go run -mod=readonly ./cmd/spec-check gate "+name, 45, "${{ github.sha }}")
			j["if"] = "${{ github.event_name != 'pull_request' }}"
			jobs[name] = j
		}
		j := job("aggregate", "go run -mod=readonly ./cmd/spec-check gate aggregate", 20, "${{ github.sha }}")
		j["if"] = "${{ always() }}"
		j["needs"] = []any{"policy", "conformance", "docs", "supply-chain", "heavy", "reproducibility"}
		j["env"] = map[string]any{"AOM_NEEDS": "${{ toJSON(needs) }}"}
		steps := j["steps"].([]any)
		steps = append(steps[:2], append([]any{map[string]any{"uses": pins["actions/download-artifact"], "with": map[string]any{"pattern": "*-${{ github.run_id }}-${{ github.run_attempt }}", "merge-multiple": true, "path": ".aom-evidence/"}}}, steps[2:]...)...)
		j["steps"] = steps
		jobs["aggregate"] = j
		return map[string]any{"name": "AOM checks", "on": map[string]any{"pull_request": map[string]any{"types": []any{"opened", "synchronize", "reopened", "ready_for_review"}}, "push": map[string]any{"branches": []any{"main"}}, "schedule": []any{map[string]any{"cron": "0 3 * * 1"}}, "workflow_dispatch": nil}, "permissions": map[string]any{}, "env": env, "concurrency": map[string]any{"group": "aom-ci-${{ github.repository }}-${{ github.event.pull_request.number || github.ref }}", "cancel-in-progress": true}, "jobs": jobs}
	}
	env["AOM_CANDIDATE"] = "${{ inputs.candidate_sha }}"
	env["AOM_VERSION"] = "${{ inputs.version }}"
	env["AOM_INITIATOR"] = "${{ github.actor_id }}"
	env["AOM_REF"] = "${{ github.ref }}"
	build := job("release-build", "go run -mod=readonly ./cmd/spec-check prepare-release", 45, "${{ github.sha }}")
	build["if"] = "${{ github.ref == 'refs/heads/main' && github.run_attempt == 1 && inputs.candidate_sha == github.sha }}"
	build["steps"] = []any{checkout("${{ github.sha }}"), map[string]any{"run": "sh scripts/provision.sh"}, map[string]any{"run": "go run -mod=readonly ./cmd/spec-check prepare-release"}, map[string]any{"uses": pins["actions/upload-artifact"], "with": map[string]any{"name": "release-candidate-${{ github.run_id }}", "path": ".aom-release/", "if-no-files-found": "error", "retention-days": 90, "include-hidden-files": true}}}
	build["steps"] = append(build["steps"].([]any), upload("release-evidence-${{ github.run_id }}-${{ github.run_attempt }}"))
	publish := map[string]any{"name": "AOM / publish", "runs-on": "ubuntu-24.04", "permissions": map[string]any{"contents": "read", "actions": "read"}, "timeout-minutes": 30, "environment": "public-release", "needs": []any{"build"}, "env": map[string]any{"AOM_OWNER_IDS": "${{ vars.AOM_OWNER_IDS }}"}, "steps": []any{checkout("${{ github.sha }}"), map[string]any{"uses": pins["actions/download-artifact"], "with": map[string]any{"name": "release-candidate-${{ github.run_id }}", "path": ".aom-release/"}}, map[string]any{"id": "publisher-token", "name": "Create scoped publisher token", "uses": pins["actions/create-github-app-token"], "with": map[string]any{"client-id": "${{ vars.AOM_PUBLISHER_CLIENT_ID }}", "private-key": "${{ secrets.AOM_PUBLISHER_PRIVATE_KEY }}", "owner": "open-agent-ops", "repositories": "spec", "permission-contents": "write", "permission-actions": "read", "skip-token-revoke": false}}, map[string]any{"env": map[string]any{"AOM_GITHUB_TOKEN": "${{ steps.publisher-token.outputs.token }}"}, "run": "chmod 700 .aom-release/spec-release\n.aom-release/spec-release publish > .aom-release/publication-receipt.json"}}}
	mirror := map[string]any{"name": "AOM / mirror", "runs-on": "ubuntu-24.04", "permissions": read, "timeout-minutes": 30, "environment": "public-mirror", "needs": []any{"publish"}, "env": map[string]any{"AOM_MIRROR_TOKEN": "${{ secrets.AOM_MIRROR_TOKEN }}", "AOM_MIRROR_REPOSITORY": "${{ vars.AOM_MIRROR_REPOSITORY }}"}, "steps": []any{checkout("${{ github.sha }}"), map[string]any{"uses": pins["actions/download-artifact"], "with": map[string]any{"name": "release-candidate-${{ github.run_id }}", "path": ".aom-release/"}}, map[string]any{"run": "chmod 700 .aom-release/spec-release\n.aom-release/spec-release mirror > .aom-release/mirror-receipt.json"}}}
	for name, j := range map[string]map[string]any{"publication": publish, "mirror": mirror} {
		j["steps"] = append(j["steps"].([]any), map[string]any{"name": "Retain publication observation", "if": "${{ always() }}", "uses": pins["actions/upload-artifact"], "with": map[string]any{"name": name + "-receipt-${{ github.run_id }}-${{ github.run_attempt }}", "path": ".aom-release/" + name + "-receipt.json", "if-no-files-found": "error", "retention-days": 90, "include-hidden-files": true}})
	}
	return map[string]any{"name": "AOM release", "on": map[string]any{"workflow_dispatch": map[string]any{"inputs": map[string]any{"candidate_sha": map[string]any{"description": "Exact reviewed main commit (40 lowercase hex)", "required": true, "type": "string"}, "version": map[string]any{"description": "Stable module SemVer v0.x.y or v1.x.y", "required": true, "type": "string"}}}}, "permissions": map[string]any{}, "env": env, "concurrency": map[string]any{"group": "aom-release-${{ github.repository }}", "cancel-in-progress": false}, "jobs": map[string]any{"build": build, "publish": publish, "mirror": mirror}}
}

func NeedsSuccess(data []byte, full bool) error {
	value, e := JSON(data)
	if e != nil {
		return e
	}
	m, ok := value.(map[string]any)
	if !ok || len(m) != 6 {
		return Rejectedf("needs: expected an object with 6 jobs, got %d", len(m))
	}
	for _, name := range []string{"policy", "conformance", "docs", "supply-chain", "heavy", "reproducibility"} {
		item, ok := m[name].(map[string]any)
		if !ok {
			return Rejectedf("needs: job %s missing", name)
		}
		result, ok := item["result"].(string)
		if !ok {
			return Rejectedf("needs: job %s has no string result", name)
		}
		wanted := "success"
		if !full && (name == "heavy" || name == "reproducibility") {
			wanted = "skipped"
		}
		if result != wanted {
			return Rejectedf("needs: job %s result %q, want %q", name, result, wanted)
		}
	}
	return nil
}
func CheckName(job string) string { return fmt.Sprintf("AOM / %s", job) }

func workflowPins(lock ToolLock) (map[string]string, error) {
	if lock.Schema != "aom04a.toolchain-lock.v1" || lock.GoVersion != "1.26.8" || lock.GoMinimum != "1.25.13" || lock.Execution != "github-hosted-native" {
		return nil, Invalidf("toolchain lock header (schema %q, go %q/%q, execution %q)", lock.Schema, lock.GoVersion, lock.GoMinimum, lock.Execution)
	}
	pins := map[string]string{}
	versions := map[string]string{"actions/checkout": "v7.0.1", "actions/upload-artifact": "v7.0.1", "actions/download-artifact": "v8.0.1", "actions/create-github-app-token": "v3.2.0"}
	for _, p := range lock.Actions {
		switch {
		case versions[p.Repository] != p.Version:
			return nil, Invalidf("toolchain lock action %s version %q, want %q", p.Repository, p.Version, versions[p.Repository])
		case !Commit(p.Commit):
			return nil, Invalidf("toolchain lock action %s commit malformed", p.Repository)
		case !Digest(p.ActionSHA256):
			return nil, Invalidf("toolchain lock action %s action_sha256 malformed", p.Repository)
		case pins[p.Repository] != "":
			return nil, Invalidf("toolchain lock action %s duplicated", p.Repository)
		}
		pins[p.Repository] = p.Repository + "@" + p.Commit
	}
	if len(pins) != 4 {
		return nil, Invalidf("toolchain lock pins %d actions, want 4", len(pins))
	}
	return pins, nil
}
