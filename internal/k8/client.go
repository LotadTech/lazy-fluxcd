package k8

import (
	"path/filepath"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

func kubeConfig(kubeconfigPath string) string {
	if kubeconfigPath == "" {
		if home := homedir.HomeDir(); home != "" {
			return filepath.Join(home, ".kube", "config")
		}
	}
	return kubeconfigPath
}

// DynamicClient returns a dynamic client from a kubeconfig file.
// An empty kubeconfigPath defaults to ~/.kube/config.
func DynamicClient(kubeconfigPath string) (dynamic.Interface, error) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeConfig(kubeconfigPath))
	if err != nil {
		return nil, err
	}
	return dynamic.NewForConfig(config)
}

func CurrentContext(kubeconfigPath string) (string, error) {
	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	if p := kubeConfig(kubeconfigPath); p != "" {
		rules.ExplicitPath = p
	}
	cfg, err := rules.Load()
	if err != nil {
		return "", err
	}
	return cfg.CurrentContext, nil
}
