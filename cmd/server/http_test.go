package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

// useFakeClient makes the handler list namespaces from a fake clientset.
// Tests that use it must not run in parallel.
func useFakeClient(t *testing.T, namespaces ...*corev1.Namespace) {
	t.Helper()
	cs := fake.NewClientset()
	for _, ns := range namespaces {
		if _, err := cs.CoreV1().Namespaces().Create(context.Background(), ns, metav1.CreateOptions{}); err != nil {
			t.Fatal(err)
		}
	}
	orig := getClient
	getClient = func(*ServerConfig, *PluginParameters) (kubernetes.Interface, error) { return cs, nil }
	t.Cleanup(func() { getClient = orig })
}

func namespace(name string, labels map[string]string) *corev1.Namespace {
	return &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: name, Labels: labels}}
}

func post(t *testing.T, c *ServerConfig, token, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/getparams.execute", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	c.secretsHandler(context.Background())(rec, req)
	return rec
}

func namespaceNames(t *testing.T, rec *httptest.ResponseRecorder) []string {
	t.Helper()
	var got ResponseBody
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	names := []string{}
	for _, p := range got.Output.Parameters {
		names = append(names, *p.Namespace)
	}
	sort.Strings(names)
	return names
}

func TestLabelSelector(t *testing.T) {
	useFakeClient(t,
		namespace("s1", map[string]string{"example.com/tier": "s"}),
		namespace("bad", map[string]string{"example.com/tier": "xl"}),
		namespace("none", nil),
	)

	tests := []struct {
		name     string
		selector string
		status   int
		want     string
	}{
		{"absent", ``, http.StatusOK, "bad,none,s1"},
		{"empty matchLabels", `"labelSelector":{"matchLabels":{}}`, http.StatusOK, "bad,none,s1"},
		{"empty matchExpressions", `"labelSelector":{"matchExpressions":[]}`, http.StatusOK, "bad,none,s1"},
		{"matchLabels", `"labelSelector":{"matchLabels":{"example.com/tier":"s"}}`, http.StatusOK, "s1"},
		{"In", `"labelSelector":{"matchExpressions":[{"key":"example.com/tier","operator":"In","values":["s","m","l"]}]}`, http.StatusOK, "s1"},
		{"Exists", `"labelSelector":{"matchExpressions":[{"key":"example.com/tier","operator":"Exists"}]}`, http.StatusOK, "bad,s1"},
		{"invalid operator", `"labelSelector":{"matchExpressions":[{"key":"example.com/tier","operator":"Bogus"}]}`, http.StatusBadRequest, ""},
		{"In without values", `"labelSelector":{"matchExpressions":[{"key":"example.com/tier","operator":"In"}]}`, http.StatusBadRequest, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := post(t, &ServerConfig{ListenToken: "t"}, "t", `{"input":{"parameters":{`+tt.selector+`}}}`)
			if rec.Code != tt.status {
				t.Fatalf("status %d, want %d: %s", rec.Code, tt.status, rec.Body)
			}
			if tt.status != http.StatusOK {
				return
			}
			if got := strings.Join(namespaceNames(t, rec), ","); got != tt.want {
				t.Errorf("namespaces = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestReturnsLabels(t *testing.T) {
	useFakeClient(t,
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
			Name:        "team-a",
			Labels:      map[string]string{"example.com/tier": "s"},
			Annotations: map[string]string{"example.com/secret-ish": "x"},
		}},
		namespace("unlabelled", nil),
	)

	rec := post(t, &ServerConfig{ListenToken: "t"}, "t", `{"input":{"parameters":{}}}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body)
	}

	var got struct {
		Output struct {
			Parameters []map[string]any `json:"parameters"`
		} `json:"output"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	byName := map[string]map[string]any{}
	for _, p := range got.Output.Parameters {
		byName[p["namespace"].(string)] = p
	}

	a := byName["team-a"]
	if a == nil {
		t.Fatalf("team-a missing: %s", rec.Body)
	}
	if tier := a["labels"].(map[string]any)["example.com/tier"]; tier != "s" {
		t.Errorf("tier label = %v, want s", tier)
	}
	if _, ok := a["annotations"]; ok {
		t.Errorf("annotations must not be returned: %s", rec.Body)
	}

	// A namespace without labels must get an empty object, not null.
	u := byName["unlabelled"]
	if u == nil {
		t.Fatalf("unlabelled missing: %s", rec.Body)
	}
	if _, ok := u["labels"].(map[string]any); !ok {
		t.Errorf("labels = %#v, want empty object", u["labels"])
	}
}
