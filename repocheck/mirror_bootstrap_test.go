package repocheck_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/open-agent-ops/spec/repocheck"
	"go.yaml.in/yaml/v3"
)

func TestBootstrapWorkflowClosedBoundary(t *testing.T) {
	raw, e := os.ReadFile("../process/toolchain.lock.json")
	if e != nil {
		t.Fatal(e)
	}
	var lock repocheck.ToolLock
	if e = json.Unmarshal(raw, &lock); e != nil {
		t.Fatal(e)
	}
	raw, e = os.ReadFile("../.github/workflows/mirror-bootstrap.yml")
	if e != nil {
		t.Fatal(e)
	}
	if e = repocheck.MirrorBootstrapWorkflow(raw, lock); e != nil {
		t.Fatal(e)
	}
	for _, pair := range [][2]string{{"environment: public-mirror", "environment: public-release"}, {"refs/heads/main", "refs/heads/other"}, {"100d10849268ff1c36ddb9568b99a9fbf9ff4dbb", strings.Repeat("b", 40)}, {"https://gitverse.ru/open-agent-ops/spec.git", "https://example.invalid/spec.git"}, {"timeout-minutes: 15", "timeout-minutes: 60"}, {"permissions: {}", "permissions: write-all"}, {"--atomic", "--force"}, {"github.run_attempt == 1", "github.run_attempt > 0"}, {"secrets.AOM_MIRROR_TOKEN", "secrets.AOM_PUBLISHER_PRIVATE_KEY"}} {
		if !strings.Contains(string(raw), pair[0]) {
			t.Fatalf("test mutation absent: %s", pair[0])
		}
		if e := repocheck.MirrorBootstrapWorkflow([]byte(strings.ReplaceAll(string(raw), pair[0], pair[1])), lock); e == nil {
			t.Fatalf("mutation accepted: %v", pair)
		}
	}
	var m map[string]any
	if e = yaml.Unmarshal(raw, &m); e != nil {
		t.Fatal(e)
	}
	m["on"] = map[string]any{"pull_request_target": nil}
	b, e := yaml.Marshal(m)
	if e != nil {
		t.Fatal(e)
	}
	if repocheck.MirrorBootstrapWorkflow(b, lock) == nil {
		t.Fatal("untrusted trigger admitted")
	}
}

func TestBootstrapProbeWithSyntheticProvider(t *testing.T) {
	// Execute the actual inline Python against a bounded fake transport. No Git,
	// credential store or network access occurs in these scenarios.
	pins := map[string]string{"actions/upload-artifact": "fixture"}
	model := repocheck.MirrorBootstrapModel(pins)
	job := model["jobs"].(map[string]any)["bootstrap"].(map[string]any)
	command := job["steps"].([]any)[0].(map[string]any)["run"].(string)
	code := strings.TrimSuffix(strings.TrimPrefix(command, "python3 - <<'PYCODE'\n"), "PYCODE")
	harness := `import sys,json,os,types,subprocess,urllib.request,pathlib
request=json.load(sys.stdin); mode=request['mode']; sha='100d10849268ff1c36ddb9568b99a9fbf9ff4dbb'; writes=[]; existing=mode=='matching'
os.environ.update(GITHUB_REPOSITORY_ID='1345286136',GITHUB_EVENT_NAME='workflow_dispatch',GITHUB_REF='refs/heads/main',GITHUB_RUN_ATTEMPT='1',GITHUB_SHA='a'*40,GITHUB_WORKFLOW_SHA='a'*40,GITHUB_RUN_ID='123',AOM_MIRROR_REPOSITORY='open-agent-ops/spec',AOM_MIRROR_TOKEN='synthetic-canary-not-a-credential')
if mode=='replay':os.environ['GITHUB_RUN_ATTEMPT']='2'
if mode=='wrong_repo':os.environ['AOM_MIRROR_REPOSITORY']='foreign/spec'
class Response:
 def __enter__(self):return self
 def __exit__(self,*args):pass
 def read(self,limit):return b'{"repoId":330883,"ownerName":"open-agent-ops","repoName":"spec"}'
class Opener:
 def open(self,url,timeout):
  assert url=='https://gitverse.ru/open-agent-ops/spec' and timeout==30
  return Response()
urllib.request.build_opener=lambda *a:Opener()
def run(argv,**kw):
 global existing
 assert argv[:5]==['/usr/bin/git','-c','credential.helper=','-c','http.followRedirects=false']
 args=argv[5:];out='';status=0
 if args[0]=='ls-remote':
  if args[-1]=='refs/tags/v0.4.0':out=(('b'*40) if mode=='wrong_tag' else sha)+'\trefs/tags/v0.4.0'
  elif args[-1]=='refs/heads/main':
   if mode=='different_main':out='b'*40+'\trefs/heads/main'
   elif existing:out=sha+'\trefs/heads/main'
  elif args[-1]=='HEAD':out='ref: refs/heads/'+('master' if mode=='wrong_default' else 'main')+'\tHEAD\n'+sha+'\tHEAD'
  else:raise AssertionError(args)
 elif args[0]=='rev-parse':out=sha if args[-1]=='FETCH_HEAD^{commit}' else 'f77fd1cfa565d5bda723e4fef689571a63308d3e'
 elif args[0]=='push':
  assert args==['push','--atomic','--porcelain','https://gitverse.ru/open-agent-ops/spec.git',sha+':refs/heads/main']
  assert kw['env']['AOM_MIRROR_TOKEN']=='synthetic-canary-not-a-credential'
  writes.append(args);existing=True
  if mode=='write_unknown':status=1
 elif args[0] not in ['init','fetch']:raise AssertionError(args)
 return types.SimpleNamespace(returncode=status,stdout=out.encode())
subprocess.run=run
status=0
try:exec(compile(request['code'],'actual-bootstrap','exec'),{})
except SystemExit as e:status=e.code
receipt=json.loads(pathlib.Path('mirror-bootstrap-evidence/receipt.json').read_text())
assert 'synthetic-canary' not in json.dumps(receipt)
if mode in ['new','matching']:
 assert status==0 and receipt['main']=='verified' and receipt['default_main']
 assert len(writes)==(1 if mode=='new' else 0)
else:
 assert status==1
 assert len(writes)==(1 if mode in ['wrong_default','write_unknown'] else 0)
 if mode=='wrong_default':assert receipt['main']=='verified' and not receipt['default_main']
print('synthetic provider assertions passed')
`
	for _, mode := range []string{"new", "matching", "wrong_repo", "wrong_tag", "different_main", "replay", "wrong_default", "write_unknown"} {
		t.Run(mode, func(t *testing.T) {
			payload, e := json.Marshal(map[string]string{"mode": mode, "code": code})
			if e != nil {
				t.Fatal(e)
			}
			cmd := exec.Command("python3", "-c", harness)
			cmd.Dir = t.TempDir()
			cmd.Stdin = strings.NewReader(string(payload))
			out, e := cmd.CombinedOutput()
			if e != nil {
				t.Fatalf("%v: %s", e, out)
			}
		})
	}
}
