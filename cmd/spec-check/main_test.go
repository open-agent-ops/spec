package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestSnapshotIndependentRootAndSubdirectory(t *testing.T) {
	for _, nested := range []bool{false, true} {
		name := "root"
		if nested {
			name = "subdirectory"
		}
		t.Run(name, func(t *testing.T) {
			root, err := filepath.EvalSymlinks(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			runGit := func(args ...string) {
				t.Helper()
				cmd := exec.Command("git", append([]string{"-c", "user.name=Fixture", "-c", "user.email=fixture@example.invalid", "-c", "commit.gpgsign=false", "-c", "core.hooksPath=/dev/null"}, args...)...)
				cmd.Dir = root
				if data, err := cmd.CombinedOutput(); err != nil {
					t.Fatalf("git: %v: %s", err, data)
				}
			}
			t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
			t.Setenv("GIT_CONFIG_GLOBAL", "/dev/null")
			runGit("init", "--quiet", "--template=", "--initial-branch=main")
			public := root
			if nested {
				public = filepath.Join(root, "public")
				if err := os.Mkdir(public, 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(root, "private.txt"), []byte("excluded"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			readme := filepath.Join(public, "README.md")
			if err := os.WriteFile(readme, []byte("public fixture\n"), 0644); err != nil {
				t.Fatal(err)
			}
			runGit("add", ".")
			runGit("commit", "--quiet", "-m", "Fixture")
			t.Chdir(public)
			got, err := snapshot(true)
			if err != nil || len(got) != 1 || string(got["README.md"]) != "public fixture\n" {
				t.Fatalf("snapshot=%v err=%v", got, err)
			}
			if err := os.WriteFile(readme, []byte("dirty\n"), 0644); err != nil {
				t.Fatal(err)
			}
			if _, err := snapshot(true); err == nil {
				t.Fatal("dirty tracked file accepted")
			}
			if err := os.WriteFile(readme, []byte("public fixture\n"), 0644); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink("README.md", filepath.Join(public, "link")); err != nil {
				t.Fatal(err)
			}
			runGit("add", ".")
			runGit("commit", "--quiet", "-m", "Symlink fixture")
			if _, err := snapshot(true); err == nil {
				t.Fatal("tracked symlink accepted")
			}
		})
	}
}
