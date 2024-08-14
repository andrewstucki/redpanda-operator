package clients

import (
	"context"
	"errors"

	"github.com/redpanda-data/common-go/rpadmin"
	redpandav1alpha2 "github.com/redpanda-data/redpanda-operator/src/go/k8s/api/redpanda/v1alpha2"
	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/kmsg"
	"k8s.io/apimachinery/pkg/types"
)

func (c *ClientFactory) UserClient(ctx context.Context, user *redpandav1alpha2.User) (*UserClient, error) {
	kafkaClient, err := c.KafkaForUser(ctx, user)
	if err != nil {
		return nil, err
	}

	adminClient, err := c.RedpandaAdminForUser(ctx, user)
	if err != nil {
		return nil, err
	}

	kafkaAdminClient := kadm.NewClient(kafkaClient)
	brokerAPI, err := kafkaAdminClient.ApiVersions(ctx)
	if err != nil {
		return nil, err
	}

	for _, api := range brokerAPI {
		_, _, supported := api.KeyVersions(kmsg.DescribeUserSCRAMCredentials.Int16())
		if supported {
			return newClient(user, c.Client, kafkaClient, kafkaAdminClient, adminClient, true), nil
		}
	}

	return newClient(user, c.Client, kafkaClient, kafkaAdminClient, adminClient, false), nil
}

func (c *ClientFactory) KafkaForUser(ctx context.Context, user *redpandav1alpha2.User) (*kgo.Client, error) {
	if user.Spec.ClusterRef != nil {
		var cluster redpandav1alpha2.Redpanda
		if err := c.Get(ctx, types.NamespacedName{Name: user.Spec.ClusterRef.Name, Namespace: user.Namespace}, &cluster); err != nil {
			return nil, err
		}
		return c.KafkaForCluster(ctx, &cluster)
	}

	if user.Spec.KafkaAPISpec != nil {
		return c.KafkaForSpec(ctx, user.Namespace, nil, user.Spec.KafkaAPISpec)
	}

	return nil, errors.New("unable to determine cluster connection info")
}

func (c *ClientFactory) RedpandaAdminForUser(ctx context.Context, user *redpandav1alpha2.User) (*rpadmin.AdminAPI, error) {
	if user.Spec.ClusterRef != nil {
		var cluster redpandav1alpha2.Redpanda
		if err := c.Get(ctx, types.NamespacedName{Name: user.Spec.ClusterRef.Name, Namespace: user.Namespace}, &cluster); err != nil {
			return nil, err
		}
		return c.RedpandaAdminForCluster(ctx, &cluster)
	}

	if user.Spec.AdminAPISpec != nil {
		return c.RedpandaAdminForSpec(ctx, user.Namespace, user.Spec.AdminAPISpec)
	}

	return nil, errors.New("unable to determine cluster connection info")
}
