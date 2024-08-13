package kafka

import (
	"context"
	"encoding/json"

	redpandachart "github.com/redpanda-data/helm-charts/charts/redpanda"
	"github.com/redpanda-data/helm-charts/pkg/gotohelm/helmette"
	"github.com/redpanda-data/helm-charts/pkg/kube"
	redpandav1alpha2 "github.com/redpanda-data/redpanda-operator/src/go/k8s/api/redpanda/v1alpha2"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

func (c *ClientFactory) ClusterResources(ctx context.Context, cluster *redpandav1alpha2.Redpanda) ([]client.Object, error) {
	release, partials, err := releaseAndPartialsFor(cluster)
	if err != nil {
		return nil, err
	}

	dot, err := redpandachart.Dot(release, partials)
	if err != nil {
		return nil, err
	}

	dot.KubeConfig = kube.RestToConfig(c.config)

	return redpandachart.DotTemplate(dot, release, partials)
}
