package server

import (
	"errors"
	"log/slog"
	"os"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	ctrl "sigs.k8s.io/controller-runtime"
)

// GetClient returns a clientset that is built once and then reused. The
// rest.Config re-reads rotated service account tokens on its own.
func (c *ServerConfig) GetClient(req *PluginParameters) (kubernetes.Interface, error) {
	c.clientOnce.Do(func() {
		c.client, c.clientErr = c.newClient()
	})
	return c.client, c.clientErr
}

func (c *ServerConfig) newClient() (kubernetes.Interface, error) {
	var config *rest.Config
	var err error

	if c.Local {
		slog.Debug("We are in --local mode")
		kubeconfigPath := ""
		if os.Getenv("KUBECONFIG") != "" {
			slog.Debug("Found KUBECONFIG environment variable", "KUBECONFIG", os.Getenv("KUBECONFIG"))
			kubeconfigPath = os.Getenv("KUBECONFIG")
		} else if home := homedir.HomeDir(); home != "" {
			slog.Debug("Falling back to user home", "HOME", home)
			kubeconfigPath = filepath.Join(home, ".kube", "config")
		}

		if kubeconfigPath == "" {
			return nil, errors.New("Cannot find KUBECONFIG or default kubeconfig file")
		}

		config, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
		if err != nil {
			return nil, err
		}
	} else {
		config, err = ctrl.GetConfig()
		if err != nil {
			return nil, err
		}
	}

	return kubernetes.NewForConfig(config)
}
