package v1alpha1

import (
	v1alpha2 "github.com/redpanda-data/redpanda-operator/src/go/k8s/api/redpanda/v1alpha2"
	corev1 "k8s.io/api/core/v1"
)

// RedpandaClusterSpec defines the desired state of a Redpanda cluster. These settings are the same as those defined in the Redpanda Helm chart. The values in these settings are passed to the Redpanda Helm chart through Flux. For all default values and links to more documentation, see https://docs.redpanda.com/current/reference/redpanda-helm-spec/.
type RedpandaClusterSpec struct {
	// Customizes the labels `app.kubernetes.io/component=<nameOverride>-statefulset` and `app.kubernetes.io/name=<nameOverride>` on the StatefulSet Pods. The default is `redpanda`.
	NameOverride string `json:"nameOverride,omitempty"`
	// Customizes the name of the StatefulSet and Services. The default is `redpanda`.
	FullNameOverride string `json:"fullNameOverride,omitempty"`
	// Customizes the Kubernetes cluster domain. This domain is used to generate the internal domains of the StatefulSet Pods. For details, see https://kubernetes.io/docs/concepts/workloads/controllers/statefulset/#stable-network-id. The default is the `cluster.local` domain.
	ClusterDomain string `json:"clusterDomain,omitempty"`
	// Assigns custom labels to all resources generated by the Redpanda Helm chart. Specify labels as key/value pairs.
	CommonLabels map[string]string `json:"commonLabels,omitempty"`
	// Specifies on which nodes a Pod should be scheduled. These key/value pairs ensure that Pods are scheduled onto nodes with the specified labels.
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`
	// Specifies tolerations to allow Pods to be scheduled onto nodes where they otherwise wouldn’t.
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// Defines the container image settings to use for the Redpanda cluster.
	Image *v1alpha2.RedpandaImage `json:"image,omitempty"`
	// Specifies credentials for a private image repository. For details, see https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/.
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`

	// Deprecated: Use `Enterprise` instead.
	LicenseKey *string `json:"license_key,omitempty"`
	// Deprecated: Use `EnterpriseLicenseSecretRef` instead.
	LicenseSecretRef *v1alpha2.LicenseSecretRef `json:"license_secret_ref,omitempty"`
	// Defines an Enterprise license.
	Enterprise *v1alpha2.Enterprise `json:"enterprise,omitempty"`

	// Defines rack awareness settings.
	RackAwareness *v1alpha2.RackAwareness `json:"rackAwareness,omitempty"`

	// Defines Redpanda Console settings.
	Console *v1alpha2.RedpandaConsole `json:"console,omitempty"`

	// Defines Redpanda Connector settings.
	Connectors *v1alpha2.RedpandaConnectors `json:"connectors,omitempty"`

	// Defines authentication settings for listeners.
	Auth *v1alpha2.Auth `json:"auth,omitempty"`

	// Defines TLS settings for listeners.
	TLS *v1alpha2.TLS `json:"tls,omitempty"`

	// Defines external access settings.
	External *v1alpha2.External `json:"external,omitempty"`

	// Defines the log level settings.
	Logging *v1alpha2.Logging `json:"logging,omitempty"`

	// Defines the log level settings.
	AuditLogging *v1alpha2.AuditLogging `json:"auditLogging,omitempty"`

	// Defines container resource settings.
	Resources *v1alpha2.Resources `json:"resources,omitempty"`

	// Defines settings for the headless ClusterIP Service.
	Service *v1alpha2.Service `json:"service,omitempty"`

	// Defines storage settings for the Redpanda data directory and the Tiered Storage cache.
	Storage *Storage `json:"storage,omitempty"`

	// Defines settings for the post-install hook, which runs after each install or upgrade. For example, this job is responsible for setting the Enterprise license, if specified.
	PostInstallJob *v1alpha2.PostInstallJob `json:"post_install_job,omitempty"`

	// Defines settings for the post-upgrade hook, which runs after each update. For example, this job is responsible for setting cluster configuration properties and restarting services such as Schema Registry, if required.
	PostUpgradeJob *v1alpha2.PostUpgradeJob `json:"post_upgrade_job,omitempty"`

	// Defines settings for the StatefulSet that manages Redpanda brokers.
	Statefulset *v1alpha2.Statefulset `json:"statefulset,omitempty"`

	// Defines settings for the autotuner tool in Redpanda. The autotuner identifies the hardware configuration in the container and optimizes the Linux kernel to give you the best performance.
	Tuning *v1alpha2.Tuning `json:"tuning,omitempty"`

	// Defines settings for listeners, including HTTP Proxy, Schema Registry, the Admin API and the Kafka API.
	Listeners *v1alpha2.Listeners `json:"listeners,omitempty"`

	// Defines configuration properties supported by Redpanda that may not work correctly in a Kubernetes cluster. Changing these values from the defaults comes with some risk. Use these properties to customize various Redpanda configurations that are not available in the `RedpandaClusterSpec`. These values have no impact on the configuration or behavior of the Kubernetes objects deployed by Helm, and therefore should not be modified for the purpose of configuring those objects. Instead, these settings get passed directly to the Redpanda binary at startup.
	Config *v1alpha2.Config `json:"config,omitempty"`

	// Defines Role Based Access Control (RBAC) settings.
	RBAC *v1alpha2.RBAC `json:"rbac,omitempty"`

	// Defines Service account settings.
	ServiceAccount *v1alpha2.ServiceAccount `json:"serviceAccount,omitempty"`

	// Defines settings for monitoring Redpanda.
	Monitoring *v1alpha2.Monitoring `json:"monitoring,omitempty"`

	// Adds the `--force` flag in `helm upgrade` commands. Used for allowing a change of TLS configuration for the RPC listener.
	// Setting `force` to `true` will result in a short period of downtime.
	Force *bool `json:"force,omitempty"`
}

// Storage configures storage-related settings in the Helm values. See https://docs.redpanda.com/current/manage/kubernetes/storage/.
type Storage struct {
	// Specifies the absolute path on the worker node to store the Redpanda data directory. If unspecified, then an `emptyDir` volume is used. If specified but `persistentVolume.enabled` is true, `storage.hostPath` has no effect.
	HostPath *string `json:"hostPath,omitempty"`
	// Configures a PersistentVolumeClaim (PVC) template to create for each Pod. This PVC is used to store the Redpanda data directory.
	PersistentVolume *v1alpha2.PersistentVolume `json:"persistentVolume,omitempty"`
	// Configures storage for the Tiered Storage cache.
	Tiered *Tiered `json:"tiered,omitempty"`
}

// Tiered configures storage for the Tiered Storage cache. See https://docs.redpanda.com/current/manage/kubernetes/tiered-storage-kubernetes/.
type Tiered struct {
	// mountType can be one of:
	//
	// - `none`: Does not mount a volume. Tiered storage will use the same volume as the one defined for the Redpanda data directory.
	// - `hostPath`: Uses the path specified in `hostPath` on the worker node that the Pod is running on.
	// - `emptyDir`: Mounts an empty directory every time the Pod starts.
	// - `persistentVolume`: Creates and mounts a PersistentVolumeClaim using the template defined in `persistentVolume`.
	MountType *string `json:"mountType,omitempty"`
	// Specifies the absolute path on the worker node to store the Tiered Storage cache.
	HostPath *string `json:"hostPath,omitempty"`
	// Configures a PersistentVolumeClaim (PVC) template to create for each Pod. This PVC is used to store the Tiered Storage cache.
	TieredStoragePersistentVolume *v1alpha2.TieredStoragePersistentVolume `json:"persistentVolume,omitempty"`
	// Configures Tiered Storage, which requires an Enterprise license configured in `enterprise.licenseKey` or `enterprised.licenseSecretRef`.
	Config *TieredConfig `json:"config,omitempty"`
	// CredentialSecretRef can be used to set `cloud_storage_secret_key` and/or `cloud_storage_access_key` from referenced Kubernetes Secret
	CredentialsSecretRef *v1alpha2.CredentialSecretRef `json:"credentialsSecretRef,omitempty"`
}

type CloudStorageEnabledString string

const (
	trueStr  = "true"
	falseStr = "false"
)

func (s *CloudStorageEnabledString) UnmarshalJSON(text []byte) error {
	switch string(text) {
	case trueStr:
		*s = trueStr
	case falseStr:
		*s = falseStr
	}

	return nil
}

// TieredConfig configures Tiered Storage, which requires an Enterprise license configured in `enterprise.licenseKey` or `enterprise.licenseSecretRef`.TieredConfig is a top-level field of the Helm values.
type TieredConfig struct {
	// Enables Tiered Storage if a license key is provided. See https://docs.redpanda.com/docs/reference/cluster-properties/#cloud_storage_enabled.
	CloudStorageEnabled *CloudStorageEnabledString `json:"cloud_storage_enabled,omitempty"`
	// See https://docs.redpanda.com/docs/reference/cluster-properties/#cloud_storage_api_endpoint.
	CloudStorageAPIEndpoint *string `json:"cloud_storage_api_endpoint,omitempty"`
	// See https://docs.redpanda.com/current/reference/cluster-properties/#cloud_storage_api_endpoint_port.
	CloudStorageAPIEndpointPort *int `json:"cloud_storage_api_endpoint_port,omitempty"`
	// See https://docs.redpanda.com/current/reference/cluster-properties/#cloud_storage_bucket.
	CloudStorageBucket *string `json:"cloud_storage_bucket,omitempty"`
	// See https://docs.redpanda.com/docs/reference/cluster-properties/#cloud_storage_azure_container.
	CloudStorageAzureContainer *string `json:"cloud_storage_azure_container,omitempty"`
	// See https://docs.redpanda.com/docs/reference/cluster-properties/#cloud_storage_azure_managed_identity_id.
	CloudStorageAzureManagedIdentityID *string `json:"cloud_storage_azure_managed_identity_id,omitempty"`
	// See https://docs.redpanda.com/docs/reference/cluster-properties/#cloud_storage_azure_storage_account.
	CloudStorageAzureStorageAccount *string `json:"cloud_storage_azure_storage_account,omitempty"`
	// See https://docs.redpanda.com/docs/reference/cluster-properties/#cloud_storage_azure_shared_key.
	CloudStorageAzureSharedKey *string `json:"cloud_storage_azure_shared_key,omitempty"`
	// See https://docs.redpanda.com/docs/reference/cluster-properties/#cloud_storage_azure_adls_endpoint.
	CloudStorageAzureADLSEndpoint *string `json:"cloud_storage_azure_adls_endpoint,omitempty"`
	// See https://docs.redpanda.com/docs/reference/cluster-properties/#cloud_storage_azure_adls_port.
	CloudStorageAzureADLSPort *int `json:"cloud_storage_azure_adls_port,omitempty"`
	// See https://docs.redpanda.com/current/reference/tunable-properties/#cloud_storage_cache_check_interval.
	CloudStorageCacheCheckInterval *int `json:"cloud_storage_cache_check_interval,omitempty"`
	// See https://docs.redpanda.com/current/reference/node-properties/#cloud_storage_cache_directory.
	CloudStorageCacheDirectory *string `json:"cloud_storage_cache_directory,omitempty"`
	// See https://docs.redpanda.com/current/reference/cluster-properties/#cloud_storage_cache_size.
	CloudStorageCacheSize *string `json:"cloud_storage_cache_size,omitempty"`
	// See https://docs.redpanda.com/current/reference/cluster-properties/#cloud_storage_credentials_source.
	CloudStorageCredentialsSource *string `json:"cloud_storage_credentials_source,omitempty"`
	// See https://docs.redpanda.com/current/reference/cluster-properties/#cloud_storage_disable_tls.
	CloudStorageDisableTLS *bool `json:"cloud_storage_disable_tls,omitempty"`
	// See https://docs.redpanda.com/current/reference/tunable-properties/#cloud_storage_enable_remote_read.
	CloudStorageEnableRemoteRead *bool `json:"cloud_storage_enable_remote_read,omitempty"`
	// See https://docs.redpanda.com/current/reference/tunable-properties/#cloud_storage_enable_remote_write.
	CloudStorageEnableRemoteWrite *bool `json:"cloud_storage_enable_remote_write,omitempty"`
	// See https://docs.redpanda.com/current/reference/tunable-properties/#cloud_storage_initial_backoff_ms.
	CloudStorageInitialBackoffMs *int `json:"cloud_storage_initial_backoff_ms,omitempty"`
	// See https://docs.redpanda.com/current/reference/tunable-properties/#cloud_storage_manifest_upload_timeout_ms.
	CloudStorageManifestUploadTimeoutMs *int `json:"cloud_storage_manifest_upload_timeout_ms,omitempty"`
	// See https://docs.redpanda.com/current/reference/tunable-properties/#cloud_storage_max_connection_idle_time_ms.
	CloudStorageMaxConnectionIdleTimeMs *int `json:"cloud_storage_max_connection_idle_time_ms,omitempty"`
	// See https://docs.redpanda.com/current/reference/cluster-properties/#cloud_storage_max_connections.
	CloudStorageMaxConnections *int `json:"cloud_storage_max_connections,omitempty"`
	// Deprecated: See https://docs.redpanda.com/current/reference/tunable-properties/#cloud_storage_reconciliation_interval_ms.
	CloudStorageReconciliationIntervalMs *int `json:"cloud_storage_reconciliation_interval_ms,omitempty"`
	// See https://docs.redpanda.com/current/reference/cluster-properties/#cloud_storage_region.
	CloudStorageRegion *string `json:"cloud_storage_region,omitempty"`
	// See https://docs.redpanda.com/current/reference/tunable-properties/#cloud_storage_segment_max_upload_interval_sec.
	CloudStorageSegmentMaxUploadIntervalSec *int `json:"cloud_storage_segment_max_upload_interval_sec,omitempty"`
	// See https://docs.redpanda.com/current/reference/tunable-properties/#cloud_storage_segment_upload_timeout_ms.
	CloudStorageSegmentUploadTimeoutMs *int `json:"cloud_storage_segment_upload_timeout_ms,omitempty"`
	// See https://docs.redpanda.com/current/reference/cluster-properties/#cloud_storage_trust_file.
	CloudStorageTrustFile *string `json:"cloud_storage_trust_file,omitempty"`
	// See https://docs.redpanda.com/current/reference/tunable-properties/#cloud_storage_upload_ctrl_d_coeff.
	CloudStorageUploadCtrlDCoeff *int `json:"cloud_storage_upload_ctrl_d_coeff,omitempty"`
	// See https://docs.redpanda.com/current/reference/tunable-properties/#cloud_storage_upload_ctrl_max_shares.
	CloudStorageUploadCtrlMaxShares *int `json:"cloud_storage_upload_ctrl_max_shares,omitempty"`
	// See https://docs.redpanda.com/current/reference/tunable-properties/#cloud_storage_upload_ctrl_min_shares.
	CloudStorageUploadCtrlMinShares *int `json:"cloud_storage_upload_ctrl_min_shares,omitempty"`
	// See https://docs.redpanda.com/current/reference/tunable-properties/#cloud_storage_upload_ctrl_p_coeff.
	CloudStorageUploadCtrlPCoeff *int `json:"cloud_storage_upload_ctrl_p_coeff,omitempty"`
	// See https://docs.redpanda.com/current/reference/tunable-properties/#cloud_storage_upload_ctrl_update_interval_ms.
	CloudStorageUploadCtrlUpdateIntervalMs *int `json:"cloud_storage_upload_ctrl_update_interval_ms,omitempty"`
}
