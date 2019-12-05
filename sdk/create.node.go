package sdk

import (
	"context"
	// "fmt"
	"log"
	"time"
	"ultipa-go-sdk/rpc"
	"ultipa-go-sdk/utils"
)

// message InsertRequest {
//   repeated InsertNode nodes = 1;
//   repeated InsertEdge edges = 2;
// }

// message InsertReply {
//   enum STATUS{
//     OK = 0;
//     FAILED = 1;
//   }
//   STATUS status = 1;
//   int32 time_cost = 2;
//   repeated int32 ids = 3;
// }

// DeleteNodes update node data to db
func CreateNodes(client ultipa.UltipaRpcsClient, nodes []utils.Node) *ultipa.InsertReply {

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)

	defer cancel()

	newNodes := utils.ToRpcNodes(nodes)

	var Nodes []*ultipa.InsertNode

	for _, n := range newNodes {
		var Node ultipa.InsertNode

		for _, v := range n.Values {
			var value ultipa.InsertValues
			value.Key = v.Key
			value.Value = v.Value
			Node.Values = append(Node.Values, &value)
		}
		Nodes = append(Nodes, &Node)
	}

	msg, err := client.Insert(ctx, &ultipa.InsertRequest{
		Nodes: Nodes,
	})

	if err != nil {
		log.Fatalf("[Error] create Node error: %v", err)
	}

	return msg

}
