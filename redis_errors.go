package beanq

import "github.com/retail-ai-inc/beanq/v4/internal/driver/bredis"

var (
	ErrAmbiguousCommit = bredis.ErrAmbiguousCommit
	ErrClusterRouting  = bredis.ErrClusterRouting
)

type ReplicationNotConfirmedError = bredis.ReplicationNotConfirmedError
type ClusterRoutingError = bredis.ClusterRoutingError
