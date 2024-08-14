package clients

import (
	"context"
	"encoding/json"

	"github.com/redpanda-data/common-go/rpadmin"
	redpandachart "github.com/redpanda-data/helm-charts/charts/redpanda"
	"github.com/redpanda-data/helm-charts/pkg/gotohelm/helmette"
	"github.com/redpanda-data/helm-charts/pkg/kube"
	"github.com/redpanda-data/helm-charts/pkg/redpanda"
	redpandav1alpha2 "github.com/redpanda-data/redpanda-operator/src/go/k8s/api/redpanda/v1alpha2"
	"github.com/twmb/franz-go/pkg/kgo"
)

func releaseAndPartialsFor(cluster *redpandav1alpha2.Redpanda) (release helmette.Release, partial redpandachart.PartialValues, err error) {
	var values []byte

	values, err = json.Marshal(cluster.Spec.ClusterSpec)
	if err != nil {
		return
	}

	err = json.Unmarshal(values, &partial)
	if err != nil {
		return
	}

	release = helmette.Release{
		Name:      cluster.Name,
		Namespace: cluster.Namespace,
		Service:   "redpanda",
		IsInstall: true,
	}

	return
}

// RedpandaAdminForCluster returns a simple kgo.Client able to communicate with the given cluster specified via a Redpanda cluster.
func (c *clientFactory) redpandaAdminForCluster(ctx context.Context, cluster *redpandav1alpha2.Redpanda) (*rpadmin.AdminAPI, error) {
	release, partials, err := releaseAndPartialsFor(cluster)
	if err != nil {
		return nil, err
	}

	config := kube.RestToConfig(c.config)
	return redpanda.AdminClient(config, release, partials, c.dialer)
}

// KafkaForCluster returns a simple kgo.Client able to communicate with the given cluster specified via a Redpanda cluster.
func (c *clientFactory) kafkaForCluster(ctx context.Context, cluster *redpandav1alpha2.Redpanda, opts ...kgo.Opt) (*kgo.Client, error) {
	release, partials, err := releaseAndPartialsFor(cluster)
	if err != nil {
		return nil, err
	}

	config := kube.RestToConfig(c.config)
	return redpanda.KafkaClient(config, release, partials, c.dialer, opts...)
}
