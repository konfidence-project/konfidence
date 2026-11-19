package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/konfidence-project/konfidence/pkg/build"
)

func TestHelp(t *testing.T) {
	for _, args := range [][]string{{}, {"--help"}, {"run", "--help"}} {
		command := NewCommand()
		var output bytes.Buffer
		command.SetOut(&output)
		command.SetArgs(args)
		if err := command.Execute(); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(output.String(), "kden-conformance") {
			t.Fatal("expected command name in output")
		}
		if len(args) == 2 {
			if !strings.Contains(output.String(), "--ocm-registry-config") {
				t.Fatal("expected execution flags in run help")
			}
		} else if !strings.Contains(output.String(), "run") || !strings.Contains(output.String(), "version") {
			t.Fatal("expected run and version subcommands in root help")
		}
		if strings.Contains(output.String(), "Executed ") {
			t.Fatal("help must not execute the suite")
		}
	}
}

func TestArgumentErrors(t *testing.T) {
	for _, args := range [][]string{
		{"unexpected"}, {"--unknown"}, {"--version"}, {"--context", "test"},
		{"--registry-config", "config.json"},
		{"version", "unexpected"}, {"version", "--output", "json"},
		{"run"}, {"run", "unexpected"},
		{"run", "--artifact-reference", "registry.example.com/app:1.0.0"},
		{"run", "--deployment-target-spec", "target.yaml"},
		{"--artifact-reference", "registry.example.com/app:1.0.0"},
	} {
		command := NewCommand()
		command.SetArgs(args)
		if err := command.Execute(); err == nil {
			t.Fatalf("expected argument error for %v", args)
		}
	}
}

func TestVersion(t *testing.T) {
	command := NewCommand()
	command.SetArgs([]string{"version"})
	var stdout, stderr bytes.Buffer
	command.SetOut(&stdout)
	command.SetErr(&stderr)
	// No suite inputs or credentials are needed by the version subcommand.
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
	var info build.Info
	if err := json.Unmarshal(stdout.Bytes(), &info); err != nil {
		t.Fatal(err)
	}
	if info != build.Current() {
		t.Fatalf("unexpected build metadata: %+v", info)
	}
	if stderr.Len() != 0 {
		t.Fatalf("unexpected stderr: %s", stderr.String())
	}
}

func TestEmptySuite(t *testing.T) {
	directory := t.TempDir()
	t.Setenv("DOCKER_CONFIG", directory)
	files := map[string]string{
		"kubeconfig": `
apiVersion: v1
kind: Config
clusters:
- name: test
  cluster:
    server: https://127.0.0.1:1
contexts:
- name: test
  context:
    cluster: test
current-context: test
`,
		"target.yaml": "artifactType: app.example.com",
	}
	for name, data := range files {
		if err := os.WriteFile(filepath.Join(directory, name), []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	command := NewCommand()
	command.SetArgs([]string{
		"run",
		"--kubeconfig", filepath.Join(directory, "kubeconfig"),
		"--artifact-reference", "registry.example.com/app:1.0.0",
		"--deployment-target-spec", filepath.Join(directory, "target.yaml"),
	})
	var output bytes.Buffer
	command.SetOut(&output)
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "Executed 0") || !strings.Contains(output.String(), "No conformance checks") {
		t.Fatalf("unexpected empty-suite output: %s", output.String())
	}
}
