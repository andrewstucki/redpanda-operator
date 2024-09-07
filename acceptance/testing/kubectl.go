package testing

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
)

type KubectlOptions struct {
	ConfigPath string
	Namespace  string
	Env        map[string]string
}

func NewKubectlOptions(config string) *KubectlOptions {
	return &KubectlOptions{
		ConfigPath: config,
		Env:        make(map[string]string),
	}
}

func (o *KubectlOptions) Clone() *KubectlOptions {
	opts := &KubectlOptions{
		ConfigPath: o.ConfigPath,
		Namespace:  o.Namespace,
		Env:        make(map[string]string),
	}

	for k, v := range o.Env {
		opts.Env[k] = v
	}

	return opts
}

func (o *KubectlOptions) WithNamespace(namespace string) *KubectlOptions {
	o.Namespace = namespace
	return o
}

func (o *KubectlOptions) WithEnv(key, value string) *KubectlOptions {
	o.Env[key] = value
	return o
}

func (o *KubectlOptions) args(args []string) []string {
	var cmdArgs []string
	if o.ConfigPath != "" {
		cmdArgs = append(cmdArgs, "--kubeconfig", o.ConfigPath)
	}
	if o.Namespace != "" && !slices.Contains(args, "-n") && !slices.Contains(args, "--namespace") {
		cmdArgs = append(cmdArgs, "--namespace", o.Namespace)
	}

	return append(cmdArgs, args...)
}

func (o *KubectlOptions) environment() []string {
	env := os.Environ()
	for key, value := range o.Env {
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}
	return env
}

func (o *KubectlOptions) merge(options *KubectlOptions) *KubectlOptions {
	if options.ConfigPath != "" {
		o.ConfigPath = options.ConfigPath
	}
	if options.Namespace != "" {
		o.Namespace = options.Namespace
	}
	for k, v := range options.Env {
		o.Env[k] = v
	}
	return o
}

func defaultOptions() *KubectlOptions {
	return &KubectlOptions{
		ConfigPath: clientcmd.RecommendedHomeFile,
		Namespace:  metav1.NamespaceDefault,
		Env:        make(map[string]string),
	}
}

func KubectlDelete(ctx context.Context, fileOrDirectory string, options ...*KubectlOptions) (string, error) {
	mergedOptions := defaultOptions()
	for _, option := range options {
		mergedOptions = mergedOptions.merge(option)
	}

	return kubectl(ctx, mergedOptions, "delete", "-f", filepath.Join("fixtures", fileOrDirectory))
}

func KubectlApply(ctx context.Context, fileOrDirectory string, options ...*KubectlOptions) (string, error) {
	mergedOptions := defaultOptions()
	for _, option := range options {
		mergedOptions = mergedOptions.merge(option)
	}

	return kubectl(ctx, mergedOptions, "apply", "-f", filepath.Join("fixtures", fileOrDirectory))
}

func kubectl(ctx context.Context, options *KubectlOptions, args ...string) (string, error) {
	command := exec.Command("kubectl", options.args(args)...)
	command.Env = options.environment()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		// signal a cancel on the command to make
		// it responsive to upstream context cancelation
		<-ctx.Done()
		if !command.ProcessState.Exited() {
			command.Cancel()
		}
	}()

	output, err := command.CombinedOutput()
	return string(output), err
}
