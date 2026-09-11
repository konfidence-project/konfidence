// Command envtest starts a standalone envtest control plane (kube-apiserver and etcd, no
// nodes or scheduler) with the Konfidence CRDs installed, writes a kubeconfig for it and
// blocks until interrupted. It is the smallest Kubernetes API the operator, the API server
// and kden can be developed against; nothing that runs pods needs it.
package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

func main() {
	// Own FlagSet: controller-runtime registers a global --kubeconfig on import.
	fs := flag.NewFlagSet("envtest", flag.ExitOnError)
	crdDir := fs.String("crd-dir", filepath.Join("test", "data", "crds"), "directory with the CRDs to install")
	kubeconfig := fs.String("kubeconfig", filepath.Join(".tmp", "envtest.kubeconfig"), "where to write the kubeconfig")
	_ = fs.Parse(os.Args[1:])

	if err := run(*crdDir, *kubeconfig); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(crdDir, kubeconfig string) error {
	env := &envtest.Environment{
		CRDDirectoryPaths:     []string{crdDir},
		ErrorIfCRDPathMissing: true,
	}
	cfg, err := env.Start()
	if err != nil {
		return fmt.Errorf("start envtest (is KUBEBUILDER_ASSETS set?): %w", err)
	}
	defer func() { _ = env.Stop() }()

	if err := writeKubeconfig(cfg, kubeconfig); err != nil {
		return err
	}
	fmt.Printf("envtest apiserver at %s\n", cfg.Host)
	fmt.Printf("export KUBECONFIG=%s\n", kubeconfig)
	fmt.Println("press Ctrl+C to stop")

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	fmt.Println("stopping")
	return nil
}

func writeKubeconfig(cfg *rest.Config, path string) error {
	const name = "envtest"
	kc := clientcmdapi.NewConfig()
	kc.Clusters[name] = &clientcmdapi.Cluster{
		Server:                   cfg.Host,
		CertificateAuthorityData: cfg.CAData,
		InsecureSkipTLSVerify:    cfg.Insecure,
	}
	kc.AuthInfos[name] = &clientcmdapi.AuthInfo{
		ClientCertificateData: cfg.CertData,
		ClientKeyData:         cfg.KeyData,
		Token:                 cfg.BearerToken,
	}
	kc.Contexts[name] = &clientcmdapi.Context{Cluster: name, AuthInfo: name}
	kc.CurrentContext = name

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return clientcmd.WriteToFile(*kc, path)
}
