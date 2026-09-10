package repocheck

import "reflect"

// MirrorBootstrapWorkflow admits one fixed v0.4.0 repair, never arbitrary refs,
// repositories, scripts or credentials supplied through dispatch inputs.
func MirrorBootstrapWorkflow(data []byte, lock ToolLock) error {
	pins, err := workflowPins(lock)
	if err != nil {
		return err
	}
	node, err := YAML(data)
	if err != nil {
		return err
	}
	var actual map[string]any
	if node.Decode(&actual) != nil {
		return ErrInput
	}
	if !reflect.DeepEqual(actual, MirrorBootstrapModel(pins)) {
		return ErrRejected
	}
	return nil
}
func MirrorBootstrapModel(pins map[string]string) map[string]any {
	return map[string]any{
		"name": "AOM v0.4.0 mirror main bootstrap",
		"on":   map[string]any{"workflow_dispatch": nil}, "permissions": map[string]any{},
		"concurrency": map[string]any{"group": "aom-release-${{ github.repository }}", "cancel-in-progress": false},
		"jobs": map[string]any{"bootstrap": map[string]any{
			"name":    "AOM / mirror main bootstrap",
			"if":      "${{ github.repository_id == '1345286136' && github.ref == 'refs/heads/main' && github.run_attempt == 1 }}",
			"runs-on": "ubuntu-24.04", "timeout-minutes": 15, "environment": "public-mirror",
			"permissions": map[string]any{},
			"steps": []any{
				map[string]any{"name": "Repair only existing v0.4.0 main", "shell": "bash", "run": mirrorBootstrapCommand,
					"env": map[string]any{"AOM_MIRROR_TOKEN": "${{ secrets.AOM_MIRROR_TOKEN }}", "AOM_MIRROR_REPOSITORY": "${{ vars.AOM_MIRROR_REPOSITORY }}"}},
				map[string]any{"name": "Retain bootstrap observation", "if": "${{ always() }}", "uses": pins["actions/upload-artifact"],
					"with": map[string]any{"name": "mirror-main-bootstrap-${{ github.run_id }}-${{ github.run_attempt }}", "path": "mirror-bootstrap-evidence/receipt.json", "retention-days": 90, "if-no-files-found": "error"}},
			},
		}},
	}
}

const mirrorBootstrapCommand = `python3 - <<'PYCODE'
import json, os, pathlib, subprocess, tempfile, urllib.request
SHA = "100d10849268ff1c36ddb9568b99a9fbf9ff4dbb"
TAG = "refs/tags/v0.4.0"
REMOTE = "https://gitverse.ru/open-agent-ops/spec.git"
out = pathlib.Path("mirror-bootstrap-evidence")
out.mkdir()
receipt = {"version": "v0.4.0", "candidate": SHA, "gitverse_repository_id": 330883,
           "run": os.environ["GITHUB_RUN_ID"], "workflow": os.environ["GITHUB_WORKFLOW_SHA"],
           "attempt": 1, "main": "unknown", "default_main": False}
class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        return None
try:
    assert os.environ["GITHUB_REPOSITORY_ID"] == "1345286136"
    assert os.environ["GITHUB_EVENT_NAME"] == "workflow_dispatch"
    assert os.environ["GITHUB_REF"] == "refs/heads/main" and os.environ["GITHUB_RUN_ATTEMPT"] == "1"
    assert os.environ["GITHUB_SHA"] == os.environ["GITHUB_WORKFLOW_SHA"]
    assert os.environ["AOM_MIRROR_REPOSITORY"] == "open-agent-ops/spec"
    token = os.environ["AOM_MIRROR_TOKEN"]
    assert token and "\n" not in token and "\r" not in token
    opener = urllib.request.build_opener(NoRedirect())
    with opener.open("https://gitverse.ru/open-agent-ops/spec", timeout=30) as response:
        page = response.read(2000001).decode()
        assert len(page) <= 2000000
        page = page.replace('\\"', '"')
        assert '"repoId":330883' in page and '"ownerName":"open-agent-ops"' in page and '"repoName":"spec"' in page
    with tempfile.TemporaryDirectory(prefix="aom-main-bootstrap-") as work:
        askpass = pathlib.Path(work) / "askpass"
        askpass.write_text('#!/bin/sh\ncase "$1" in\n*Username*) printf "%s\\n" "$AOM_MIRROR_TOKEN" ;;\n*Password*) printf "%s\\n" "" ;;\n*) exit 1 ;;\nesac\n')
        askpass.chmod(0o700)
        env = {"PATH": "/usr/bin:/bin", "GIT_TERMINAL_PROMPT": "0", "GIT_CONFIG_NOSYSTEM": "1",
               "GIT_CONFIG_GLOBAL": "/dev/null", "GIT_ALLOW_PROTOCOL": "https"}
        def git(*args, credential=False):
            selected = dict(env)
            if credential:
                selected.update(GIT_ASKPASS=str(askpass), AOM_MIRROR_TOKEN=token)
            result = subprocess.run(["/usr/bin/git", "-c", "credential.helper=", "-c", "http.followRedirects=false", *args],
                                    cwd=work, env=selected, capture_output=True, timeout=180)
            assert result.returncode == 0 and len(result.stdout) <= 1048576
            return result.stdout.decode().strip()
        git("init", "--quiet")
        assert git("ls-remote", "--refs", "https://github.com/open-agent-ops/spec.git", TAG).split() == [SHA, TAG]
        git("fetch", "--no-tags", "--depth=1", "https://github.com/open-agent-ops/spec.git", TAG)
        assert git("rev-parse", "FETCH_HEAD^{commit}") == SHA
        assert git("ls-remote", "--refs", REMOTE, TAG).split() == [SHA, TAG]
        before = git("ls-remote", "--refs", REMOTE, "refs/heads/main").split()
        assert before in ([], [SHA, "refs/heads/main"])
        if not before:
            # One non-force write, no retry or rollback. Unknown outcome is retained.
            git("push", "--atomic", "--porcelain", REMOTE, SHA + ":refs/heads/main", credential=True)
        assert git("ls-remote", "--refs", REMOTE, "refs/heads/main").split() == [SHA, "refs/heads/main"]
        assert git("ls-remote", "--refs", REMOTE, TAG).split() == [SHA, TAG]
        receipt["main"] = "verified"
        receipt["tree"] = git("rev-parse", "FETCH_HEAD^{tree}")
        receipt["default_main"] = git("ls-remote", "--symref", REMOTE, "HEAD").split() == ["ref:", "refs/heads/main", "HEAD", SHA, "HEAD"]
        assert receipt["default_main"]
    print("Existing v0.4.0 main and default HEAD verified")
except Exception:
    print("Mirror main bootstrap incomplete; inspect receipt and reconcile before any new write")
    raise SystemExit(1)
finally:
    (out / "receipt.json").write_text(json.dumps(receipt, indent=2))
PYCODE`
