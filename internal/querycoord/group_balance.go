package querycoord

import (
	"sort"
)

type Balancer interface {
	AddNode(nodeID int64) ([]*balancePlan, error)
	RemoveNode(nodeID int64) []*balancePlan
	Rebalance() []*balancePlan
}

// Plan for adding/removing node from replica,
// adds node into targetReplica,
// removes node from sourceReplica.
// Set the replica ID to invalidReplicaID to avoid adding/removing into/from replica
type balancePlan struct {
	nodes         []UniqueID
	sourceReplica UniqueID
	targetReplica UniqueID
}

type replicaBalancer struct {
	meta    Meta
	cluster Cluster
}

func newReplicaBalancer(meta Meta, cluster Cluster) *replicaBalancer {
	return &replicaBalancer{meta, cluster}
}

func (b *replicaBalancer) AddNode(nodeID int64) ([]*balancePlan, error) {
	// allocate this node to all collections replicas
	var ret []*balancePlan
	collections := b.meta.showCollections()
	for _, c := range collections {
		replicas, err := b.meta.getReplicasByCollectionID(c.GetCollectionID())
		if err != nil {
			return nil, err
		}
		if len(replicas) == 0 {
			continue
		}

		foundNode := false
		for _, replica := range replicas {
			for _, replicaNode := range replica.NodeIds {
				if replicaNode == nodeID {
					foundNode = true
					break
				}
			}

			if foundNode {
				break
			}
		}

		// This node is serving this collection
		if foundNode {
			continue
		}

		replicaAvailableMemory := make(map[UniqueID]uint64, len(replicas))
		for _, replica := range replicas {
			replicaAvailableMemory[replica.ReplicaID] = getReplicaAvailableMemory(b.cluster, replica)
		}
		sort.Slice(replicas, func(i, j int) bool {
			replicai := replicas[i].ReplicaID
			replicaj := replicas[j].ReplicaID

			return replicaAvailableMemory[replicai] < replicaAvailableMemory[replicaj]
		})

		ret = append(ret, &balancePlan{
			nodes:         []UniqueID{nodeID},
			sourceReplica: invalidReplicaID,
			targetReplica: replicas[0].GetReplicaID(),
		})
	}
	return ret, nil
}

func (b *replicaBalancer) RemoveNode(nodeID int64) []*balancePlan {
	// for this version, querynode does not support move from a replica to another
	return nil
}

func (b *replicaBalancer) Rebalance() []*balancePlan {
	return nil
}
