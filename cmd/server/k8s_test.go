package server

import (
	"os"
	"path/filepath"
	"testing"
)

const testKubeconfig = `apiVersion: v1
kind: Config
clusters:
- name: test
  cluster:
    server: https://127.0.0.1:1
contexts:
- name: test
  context:
    cluster: test
    user: test
current-context: test
users:
- name: test
  user:
    token: test
`

func TestGetClientIsReused(t *testing.T) {
	path := filepath.Join(t.TempDir(), "kubeconfig")
	if err := os.WriteFile(path, []byte(testKubeconfig), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("KUBECONFIG", path)

	c := &ServerConfig{Local: true}
	first, err := c.GetClient(nil)
	if err != nil {
		t.Fatal(err)
	}
	second, err := c.GetClient(nil)
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Error("GetClient built a new client on the second call")
	}
}

func TestGetClientReturnsError(t *testing.T) {
	t.Setenv("KUBECONFIG", filepath.Join(t.TempDir(), "missing"))

	c := &ServerConfig{Local: true}
	if _, err := c.GetClient(nil); err == nil {
		t.Fatal("expected an error for a missing kubeconfig")
	}
}
