// Copyright 2024 Redpanda Data, Inc.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.md
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0

package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/pprof"
	"os"
	"strings"
	"time"

	cmapiv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	helmControllerAPIv2beta1 "github.com/fluxcd/helm-controller/api/v2beta1"
	helmControllerAPIv2beta2 "github.com/fluxcd/helm-controller/api/v2beta2"
	helmController "github.com/fluxcd/helm-controller/shim"
	"github.com/fluxcd/pkg/runtime/client"
	helper "github.com/fluxcd/pkg/runtime/controller"
	"github.com/fluxcd/pkg/runtime/events"
	"github.com/fluxcd/pkg/runtime/logger"
	"github.com/fluxcd/pkg/runtime/metrics"
	sourceControllerAPIv1 "github.com/fluxcd/source-controller/api/v1"
	sourceControllerAPIv1beta2 "github.com/fluxcd/source-controller/api/v1beta2"
	helmSourceController "github.com/fluxcd/source-controller/shim"
	flag "github.com/spf13/pflag"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/kube"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	redpandav1alpha1 "github.com/redpanda-data/redpanda-operator/src/go/k8s/api/redpanda/v1alpha1"
	redpandav1alpha2 "github.com/redpanda-data/redpanda-operator/src/go/k8s/api/redpanda/v1alpha2"
	vectorizedv1alpha1 "github.com/redpanda-data/redpanda-operator/src/go/k8s/api/vectorized/v1alpha1"
	internalclient "github.com/redpanda-data/redpanda-operator/src/go/k8s/internal/client"
	"github.com/redpanda-data/redpanda-operator/src/go/k8s/internal/controller/pvcunbinder"
	redpandacontrollers "github.com/redpanda-data/redpanda-operator/src/go/k8s/internal/controller/redpanda"
	adminutils "github.com/redpanda-data/redpanda-operator/src/go/k8s/pkg/admin"
	consolepkg "github.com/redpanda-data/redpanda-operator/src/go/k8s/pkg/console"
	"github.com/redpanda-data/redpanda-operator/src/go/k8s/pkg/resources"
	redpandawebhooks "github.com/redpanda-data/redpanda-operator/src/go/k8s/webhooks/redpanda"
)

type RedpandaController string

type OperatorState string

func (r RedpandaController) toString() string {
	return string(r)
}

const (
	defaultConfiguratorContainerImage = "vectorized/configurator"

	AllControllers         = RedpandaController("all")
	NodeController         = RedpandaController("nodeWatcher")
	DecommissionController = RedpandaController("decommission")

	OperatorV1Mode          = OperatorState("Clustered-v1")
	OperatorV2Mode          = OperatorState("Namespaced-v2")
	ClusterControllerMode   = OperatorState("Clustered-Controllers")
	NamespaceControllerMode = OperatorState("Namespaced-Controllers")

	controllerName               = "redpanda-controller"
	helmReleaseControllerName    = "redpanda-helmrelease-controller"
	helmChartControllerName      = "redpanda-helmchart-reconciler"
	helmRepositoryControllerName = "redpanda-helmrepository-controller"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
	getters  = getter.Providers{
		getter.Provider{
			Schemes: []string{"http", "https"},
			New:     getter.NewHTTPGetter,
		},
		getter.Provider{
			Schemes: []string{"oci"},
			New:     getter.NewOCIGetter,
		},
	}

	clientOptions  client.Options
	kubeConfigOpts client.KubeConfigOptions
	logOptions     logger.Options

	storageAdvAddr string

	availableControllers = []string{
		NodeController.toString(),
		DecommissionController.toString(),
	}
)

//nolint:wsl // the init was generated by kubebuilder
func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(cmapiv1.AddToScheme(scheme))
	utilruntime.Must(helmControllerAPIv2beta1.AddToScheme(scheme))
	utilruntime.Must(helmControllerAPIv2beta2.AddToScheme(scheme))
	utilruntime.Must(redpandav1alpha1.AddToScheme(scheme))
	utilruntime.Must(redpandav1alpha2.AddToScheme(scheme))
	utilruntime.Must(sourceControllerAPIv1.AddToScheme(scheme))
	utilruntime.Must(sourceControllerAPIv1beta2.AddToScheme(scheme))
	utilruntime.Must(vectorizedv1alpha1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

// +kubebuilder:rbac:groups=coordination.k8s.io,namespace=default,resources=leases,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,namespace=default,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,namespace=default,resources=events,verbs=create;patch

//nolint:funlen,gocyclo // length looks good
func main() {
	var (
		clusterDomain               string
		metricsAddr                 string
		probeAddr                   string
		pprofAddr                   string
		enableLeaderElection        bool
		webhookEnabled              bool
		configuratorBaseImage       string
		configuratorTag             string
		configuratorImagePullPolicy string
		decommissionWaitInterval    time.Duration
		metricsTimeout              time.Duration
		restrictToRedpandaVersion   string
		namespace                   string
		eventsAddr                  string
		additionalControllers       []string
		operatorMode                bool
		enableHelmControllers       bool
		debug                       bool
		ghostbuster                 bool
		unbindPVCsAfter             time.Duration
		autoDeletePVCs              bool
	)

	flag.StringVar(&eventsAddr, "events-addr", "", "The address of the events receiver.")
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.StringVar(&pprofAddr, "pprof-bind-address", ":8082", "The address the metric endpoint binds to.")
	flag.StringVar(&clusterDomain, "cluster-domain", "cluster.local", "Set the Kubernetes local domain (Kubelet's --cluster-domain)")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.BoolVar(&webhookEnabled, "webhook-enabled", false, "Enable webhook Manager")
	flag.StringVar(&configuratorBaseImage, "configurator-base-image", defaultConfiguratorContainerImage, "Set the configurator base image")
	flag.StringVar(&configuratorTag, "configurator-tag", "latest", "Set the configurator tag")
	flag.StringVar(&configuratorImagePullPolicy, "configurator-image-pull-policy", "Always", "Set the configurator image pull policy")
	flag.DurationVar(&decommissionWaitInterval, "decommission-wait-interval", 8*time.Second, "Set the time to wait for a node decommission to happen in the cluster")
	flag.DurationVar(&metricsTimeout, "metrics-timeout", 8*time.Second, "Set the timeout for a checking metrics Admin API endpoint. If set to 0, then the 2 seconds default will be used")
	flag.BoolVar(&vectorizedv1alpha1.AllowDownscalingInWebhook, "allow-downscaling", true, "Allow to reduce the number of replicas in existing clusters")
	flag.Bool("allow-pvc-deletion", false, "Deprecated: Ignored if specified")
	flag.BoolVar(&vectorizedv1alpha1.AllowConsoleAnyNamespace, "allow-console-any-ns", false, "Allow to create Console in any namespace. Allowing this copies Redpanda SchemaRegistry TLS Secret to namespace (alpha feature)")
	flag.StringVar(&restrictToRedpandaVersion, "restrict-redpanda-version", "", "Restrict management of clusters to those with this version")
	flag.StringVar(&vectorizedv1alpha1.SuperUsersPrefix, "superusers-prefix", "", "Prefix to add in username of superusers managed by operator. This will only affect new clusters, enabling this will not add prefix to existing clusters (alpha feature)")
	flag.BoolVar(&debug, "debug", false, "Set to enable debugging")
	flag.StringVar(&namespace, "namespace", "", "If namespace is set to not empty value, it changes scope of Redpanda operator to work in single namespace")
	flag.BoolVar(&ghostbuster, "unsafe-decommission-failed-brokers", false, "Set to enable decommissioning a failed broker that is configured but does not exist in the StatefulSet (ghost broker). This may result in invalidating valid data")
	_ = flag.CommandLine.MarkHidden("unsafe-decommission-failed-brokers")
	flag.StringSliceVar(&additionalControllers, "additional-controllers", []string{""}, fmt.Sprintf("which controllers to run, available: all, %s", strings.Join(availableControllers, ", ")))
	flag.BoolVar(&operatorMode, "operator-mode", true, "enables to run as an operator, setting this to false will disable cluster (deprecated), redpanda resources reconciliation.")
	flag.BoolVar(&enableHelmControllers, "enable-helm-controllers", true, "if a namespace is defined and operator mode is true, this enables the use of helm controllers to manage fluxcd helm resources.")
	flag.DurationVar(&unbindPVCsAfter, "unbind-pvcs-after", 0, "if not zero, runs the PVCUnbinder controller which attempts to 'unbind' the PVCs' of Pods that are Pending for longer than the given duration")
	flag.BoolVar(&autoDeletePVCs, "auto-delete-pvcs", false, "Use StatefulSet PersistentVolumeClaimRetentionPolicy to auto delete PVCs on scale down and Cluster resource delete.")

	logOptions.BindFlags(flag.CommandLine)
	clientOptions.BindFlags(flag.CommandLine)
	kubeConfigOpts.BindFlags(flag.CommandLine)

	flag.Parse()

	ctrl.SetLogger(logger.NewLogger(logOptions))

	// set the managedFields owner for resources reconciled from Helm charts
	kube.ManagedFieldsManager = controllerName

	if debug {
		go func() {
			pprofMux := http.NewServeMux()
			pprofMux.HandleFunc("/debug/pprof/", pprof.Index)
			pprofMux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
			pprofMux.HandleFunc("/debug/pprof/profile", pprof.Profile)
			pprofMux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
			pprofMux.HandleFunc("/debug/pprof/trace", pprof.Trace)
			pprofServer := &http.Server{
				Addr:              pprofAddr,
				Handler:           pprofMux,
				ReadHeaderTimeout: 3 * time.Second,
			}
			log.Fatal(pprofServer.ListenAndServe())
		}()
	}

	ctx, done := context.WithCancel(context.Background())
	defer done()

	mgrOptions := ctrl.Options{
		Scheme:                  scheme,
		Metrics:                 metricsserver.Options{BindAddress: metricsAddr},
		HealthProbeBindAddress:  probeAddr,
		LeaderElection:          enableLeaderElection,
		LeaderElectionID:        "aa9fc693.vectorized.io",
		LeaderElectionNamespace: namespace,
	}
	if namespace != "" {
		mgrOptions.Cache.DefaultNamespaces = map[string]cache.Config{namespace: {}}
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), mgrOptions)
	if err != nil {
		setupLog.Error(err, "Unable to start manager")
		// nolint:gocritic // this exits without closing the context. That's ok.
		os.Exit(1)
	}

	configurator := resources.ConfiguratorSettings{
		ConfiguratorBaseImage: configuratorBaseImage,
		ConfiguratorTag:       configuratorTag,
		ImagePullPolicy:       corev1.PullPolicy(configuratorImagePullPolicy),
	}

	// init running state values if we are not in operator mode
	operatorRunningState := ClusterControllerMode
	if namespace != "" {
		operatorRunningState = NamespaceControllerMode
	}

	// but if we are in operator mode, then the run state is different
	if operatorMode {
		operatorRunningState = OperatorV1Mode
		if namespace != "" {
			operatorRunningState = OperatorV2Mode
		}
	}

	// Now we start different processes depending on state
	switch operatorRunningState {
	case OperatorV1Mode:
		ctrl.Log.Info("running in v1", "mode", OperatorV1Mode)

		if err = (&redpandacontrollers.ClusterReconciler{
			Client:                    mgr.GetClient(),
			Log:                       ctrl.Log.WithName("controllers").WithName("redpanda").WithName("Cluster"),
			Scheme:                    mgr.GetScheme(),
			AdminAPIClientFactory:     adminutils.NewInternalAdminAPI,
			DecommissionWaitInterval:  decommissionWaitInterval,
			MetricsTimeout:            metricsTimeout,
			RestrictToRedpandaVersion: restrictToRedpandaVersion,
			GhostDecommissioning:      ghostbuster,
			AutoDeletePVCs:            autoDeletePVCs,
		}).WithClusterDomain(clusterDomain).WithConfiguratorSettings(configurator).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "Unable to create controller", "controller", "Cluster")
			os.Exit(1)
		}

		if err = (&redpandacontrollers.ClusterConfigurationDriftReconciler{
			Client:                    mgr.GetClient(),
			Log:                       ctrl.Log.WithName("controllers").WithName("redpanda").WithName("ClusterConfigurationDrift"),
			Scheme:                    mgr.GetScheme(),
			AdminAPIClientFactory:     adminutils.NewInternalAdminAPI,
			RestrictToRedpandaVersion: restrictToRedpandaVersion,
		}).WithClusterDomain(clusterDomain).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "Unable to create controller", "controller", "ClusterConfigurationDrift")
			os.Exit(1)
		}

		if err = redpandacontrollers.NewClusterMetricsController(mgr.GetClient()).
			SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "Unable to create controller", "controller", "ClustersMetrics")
			os.Exit(1)
		}

		if err = (&redpandacontrollers.ConsoleReconciler{
			Client:                  mgr.GetClient(),
			Scheme:                  mgr.GetScheme(),
			Log:                     ctrl.Log.WithName("controllers").WithName("redpanda").WithName("Console"),
			AdminAPIClientFactory:   adminutils.NewInternalAdminAPI,
			Store:                   consolepkg.NewStore(mgr.GetClient(), mgr.GetScheme()),
			EventRecorder:           mgr.GetEventRecorderFor("Console"),
			KafkaAdminClientFactory: consolepkg.NewKafkaAdmin,
		}).WithClusterDomain(clusterDomain).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "Console")
			os.Exit(1)
		}

		var topicEventRecorder *events.Recorder
		if topicEventRecorder, err = events.NewRecorder(mgr, ctrl.Log, eventsAddr, "TopicReconciler"); err != nil {
			setupLog.Error(err, "unable to create event recorder for: TopicReconciler")
			os.Exit(1)
		}

		if err = (&redpandacontrollers.TopicReconciler{
			Client:        mgr.GetClient(),
			Factory:       internalclient.NewFactory(mgr.GetConfig(), mgr.GetClient()),
			Scheme:        mgr.GetScheme(),
			EventRecorder: topicEventRecorder,
		}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "Topic")
			os.Exit(1)
		}

		if unbindPVCsAfter <= 0 {
			setupLog.Info("PVCUnbinder controller not active", "flag", unbindPVCsAfter)
		} else {
			setupLog.Info("starting PVCUnbinder controller", "flag", unbindPVCsAfter)

			if err := (&pvcunbinder.Reconciler{
				Client:  mgr.GetClient(),
				Timeout: unbindPVCsAfter,
			}).SetupWithManager(mgr); err != nil {
				setupLog.Error(err, "unable to create controller", "controller", "PVCUnbinder")
				os.Exit(1)
			}
		}

		// Setup webhooks
		if webhookEnabled {
			setupLog.Info("Setup webhook")
			if err = (&vectorizedv1alpha1.Cluster{}).SetupWebhookWithManager(mgr); err != nil {
				setupLog.Error(err, "Unable to create webhook", "webhook", "RedpandaCluster")
				os.Exit(1)
			}
			hookServer := mgr.GetWebhookServer()
			hookServer.Register("/mutate-redpanda-vectorized-io-v1alpha1-console", &webhook.Admission{
				Handler: &redpandawebhooks.ConsoleDefaulter{
					Client:  mgr.GetClient(),
					Decoder: admission.NewDecoder(scheme),
				},
			})
			hookServer.Register("/validate-redpanda-vectorized-io-v1alpha1-console", &webhook.Admission{
				Handler: &redpandawebhooks.ConsoleValidator{
					Client:  mgr.GetClient(),
					Decoder: admission.NewDecoder(scheme),
				},
			})
		}
	case OperatorV2Mode:
		ctrl.Log.Info("running in v2", "mode", OperatorV2Mode, "helm controllers enabled", enableHelmControllers, "namespace", namespace)

		// if we enable these controllers then run them, otherwise, do not
		//nolint:nestif // not really nested, required.
		if enableHelmControllers {
			storageAddr := ":9090"
			storageAdvAddr = redpandacontrollers.DetermineAdvStorageAddr(storageAddr, setupLog)
			storage := redpandacontrollers.MustInitStorage("/tmp", storageAdvAddr, 60*time.Second, 2, setupLog)

			metricsH := helper.NewMetrics(mgr, metrics.MustMakeRecorder())

			// TODO fill this in with options
			helmOpts := helmController.HelmReleaseReconcilerOptions{
				DependencyRequeueInterval: 30 * time.Second, // The interval at which failing dependencies are reevaluated.
				HTTPRetry:                 9,                // The maximum number of retries when failing to fetch artifacts over HTTP.
				RateLimiter:               workqueue.NewItemExponentialFailureRateLimiter(30*time.Second, 60*time.Second),
			}

			// Helm Release Controller
			var helmReleaseEventRecorder *events.Recorder
			if helmReleaseEventRecorder, err = events.NewRecorder(mgr, ctrl.Log, eventsAddr, "HelmReleaseReconciler"); err != nil {
				setupLog.Error(err, "unable to create event recorder for: HelmReleaseReconciler")
				os.Exit(1)
			}

			// Helm Release Controller
			helmRelease := helmController.HelmReleaseReconcilerFactory{
				Client:           mgr.GetClient(),
				EventRecorder:    helmReleaseEventRecorder,
				ClientOpts:       clientOptions,
				KubeConfigOpts:   kubeConfigOpts,
				FieldManager:     helmReleaseControllerName,
				Metrics:          metricsH,
				GetClusterConfig: ctrl.GetConfig,
			}
			if err = helmRelease.SetupWithManager(ctx, mgr, helmOpts); err != nil {
				setupLog.Error(err, "Unable to create controller", "controller", "HelmRelease")
			}

			// Helm Chart Controller
			var helmChartEventRecorder *events.Recorder
			if helmChartEventRecorder, err = events.NewRecorder(mgr, ctrl.Log, eventsAddr, "HelmChartReconciler"); err != nil {
				setupLog.Error(err, "unable to create event recorder for: HelmChartReconciler")
				os.Exit(1)
			}

			cacheRecorder := helmSourceController.MustMakeCacheMetrics()
			indexTTL := 15 * time.Minute
			helmIndexCache := helmSourceController.NewCache(0, indexTTL)
			chartOpts := helmSourceController.HelmChartReconcilerOptions{
				RateLimiter: helper.GetDefaultRateLimiter(),
			}
			repoOpts := helmSourceController.HelmRepositoryReconcilerOptions{
				RateLimiter: helper.GetDefaultRateLimiter(),
			}
			helmChart := helmSourceController.HelmChartReconcilerFactory{
				Cache:                   helmIndexCache,
				CacheRecorder:           cacheRecorder,
				TTL:                     indexTTL,
				Client:                  mgr.GetClient(),
				RegistryClientGenerator: redpandacontrollers.ClientGenerator,
				Getters:                 getters,
				Metrics:                 metricsH,
				Storage:                 storage,
				EventRecorder:           helmChartEventRecorder,
				ControllerName:          helmChartControllerName,
			}
			if err = helmChart.SetupWithManager(ctx, mgr, chartOpts); err != nil {
				setupLog.Error(err, "Unable to create controller", "controller", "HelmChart")
			}

			// Helm Repository Controller
			var helmRepositoryEventRecorder *events.Recorder
			if helmRepositoryEventRecorder, err = events.NewRecorder(mgr, ctrl.Log, eventsAddr, "HelmRepositoryReconciler"); err != nil {
				setupLog.Error(err, "unable to create event recorder for: HelmRepositoryReconciler")
				os.Exit(1)
			}

			helmRepository := helmSourceController.HelmRepositoryReconcilerFactory{
				Client:         mgr.GetClient(),
				EventRecorder:  helmRepositoryEventRecorder,
				Getters:        getters,
				ControllerName: helmRepositoryControllerName,
				Cache:          helmIndexCache,
				CacheRecorder:  cacheRecorder,
				TTL:            indexTTL,
				Metrics:        metricsH,
				Storage:        storage,
			}

			if err = helmRepository.SetupWithManager(ctx, mgr, repoOpts); err != nil {
				setupLog.Error(err, "Unable to create controller", "controller", "HelmRepository")
			}

			go func() {
				// Block until our controller manager is elected leader. We presume our
				// entire process will terminate if we lose leadership, so we don't need
				// to handle that.
				<-mgr.Elected()

				redpandacontrollers.StartFileServer(storage.BasePath, storageAddr, setupLog)
			}()
		}

		// Redpanda Reconciler
		var redpandaEventRecorder *events.Recorder
		if redpandaEventRecorder, err = events.NewRecorder(mgr, ctrl.Log, eventsAddr, "RedpandaReconciler"); err != nil {
			setupLog.Error(err, "unable to create event recorder for: RedpandaReconciler")
			os.Exit(1)
		}

		if err = (&redpandacontrollers.RedpandaReconciler{
			Client:          mgr.GetClient(),
			Scheme:          mgr.GetScheme(),
			EventRecorder:   redpandaEventRecorder,
			RequeueHelmDeps: 10 * time.Second,
		}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "Redpanda")
			os.Exit(1)
		}

		var topicEventRecorder *events.Recorder
		if topicEventRecorder, err = events.NewRecorder(mgr, ctrl.Log, eventsAddr, "TopicReconciler"); err != nil {
			setupLog.Error(err, "unable to create event recorder for: TopicReconciler")
			os.Exit(1)
		}

		if err = (&redpandacontrollers.TopicReconciler{
			Client:        mgr.GetClient(),
			Factory:       internalclient.NewFactory(mgr.GetConfig(), mgr.GetClient()),
			Scheme:        mgr.GetScheme(),
			EventRecorder: topicEventRecorder,
		}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "Topic")
			os.Exit(1)
		}

		var managedDecommissionEventRecorder *events.Recorder
		if managedDecommissionEventRecorder, err = events.NewRecorder(mgr, ctrl.Log, eventsAddr, "ManagedDecommissionReconciler"); err != nil {
			setupLog.Error(err, "unable to create event recorder for: ManagedDecommissionReconciler")
			os.Exit(1)
		}

		if err = (&redpandacontrollers.ManagedDecommissionReconciler{
			Client:        mgr.GetClient(),
			EventRecorder: managedDecommissionEventRecorder,
		}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "ManagedDecommission")
			os.Exit(1)
		}

		if runThisController(NodeController, additionalControllers) {
			if err = (&redpandacontrollers.RedpandaNodePVCReconciler{
				Client:       mgr.GetClient(),
				OperatorMode: operatorMode,
			}).SetupWithManager(mgr); err != nil {
				setupLog.Error(err, "unable to create controller", "controller", "RedpandaNodePVCReconciler")
				os.Exit(1)
			}
		}

		if runThisController(DecommissionController, additionalControllers) {
			if err = (&redpandacontrollers.DecommissionReconciler{
				Client:                   mgr.GetClient(),
				OperatorMode:             operatorMode,
				DecommissionWaitInterval: decommissionWaitInterval,
			}).SetupWithManager(mgr); err != nil {
				setupLog.Error(err, "unable to create controller", "controller", "DecommissionReconciler")
				os.Exit(1)
			}
		}

		if webhookEnabled {
			setupLog.Info("Setup Redpanda conversion webhook")
			if err = (&redpandav1alpha2.Redpanda{}).SetupWebhookWithManager(mgr); err != nil {
				setupLog.Error(err, "Unable to create webhook", "webhook", "RedpandaConversion")
				os.Exit(1)
			}
		}

	case ClusterControllerMode:
		ctrl.Log.Info("running as a cluster controller", "mode", ClusterControllerMode)
		setupLog.Error(err, "unable to create cluster controllers, not supported")
		os.Exit(1)
	case NamespaceControllerMode:
		ctrl.Log.Info("running as a namespace controller", "mode", NamespaceControllerMode, "namespace", namespace)
		if runThisController(NodeController, additionalControllers) {
			if err = (&redpandacontrollers.RedpandaNodePVCReconciler{
				Client:       mgr.GetClient(),
				OperatorMode: operatorMode,
			}).SetupWithManager(mgr); err != nil {
				setupLog.Error(err, "unable to create controller", "controller", "RedpandaNodePVCReconciler")
				os.Exit(1)
			}
		}

		if runThisController(DecommissionController, additionalControllers) {
			if err = (&redpandacontrollers.DecommissionReconciler{
				Client:                   mgr.GetClient(),
				OperatorMode:             operatorMode,
				DecommissionWaitInterval: decommissionWaitInterval,
			}).SetupWithManager(mgr); err != nil {
				setupLog.Error(err, "unable to create controller", "controller", "DecommissionReconciler")
				os.Exit(1)
			}
		}
	default:
		setupLog.Error(err, "unable unknown state, not supported")
		os.Exit(1)
	}

	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("health", healthz.Ping); err != nil {
		setupLog.Error(err, "Unable to set up health check")
		os.Exit(1)
	}

	if err := mgr.AddReadyzCheck("check", healthz.Ping); err != nil {
		setupLog.Error(err, "Unable to set up ready check")
		os.Exit(1)
	}

	if webhookEnabled {
		hookServer := mgr.GetWebhookServer()
		if err := mgr.AddReadyzCheck("webhook", hookServer.StartedChecker()); err != nil {
			setupLog.Error(err, "unable to create ready check")
			os.Exit(1)
		}

		if err := mgr.AddHealthzCheck("webhook", hookServer.StartedChecker()); err != nil {
			setupLog.Error(err, "unable to create health check")
			os.Exit(1)
		}
	}
	setupLog.Info("Starting manager")

	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "Problem running manager")
		os.Exit(1)
	}
}

func runThisController(rc RedpandaController, controllers []string) bool {
	if len(controllers) == 0 {
		return false
	}

	for _, c := range controllers {
		if RedpandaController(c) == AllControllers || RedpandaController(c) == rc {
			return true
		}
	}
	return false
}
