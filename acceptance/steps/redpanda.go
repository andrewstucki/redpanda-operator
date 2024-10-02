// Copyright 2024 Redpanda Data, Inc.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.md
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0

package steps

import (
	"context"

	framework "github.com/redpanda-data/redpanda-operator/harpoon"
	"github.com/stretchr/testify/require"
)

func iShouldBeAbleToConnectToCluster(ctx context.Context, t framework.TestingT, cluster string) {
	require.NoError(t, clientsForCluster(ctx, cluster).Kafka(ctx).Ping(ctx))
}
