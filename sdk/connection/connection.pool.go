package connection

import (
	"context"
	"errors"
	"google.golang.org/grpc/metadata"
	"log"
	"time"
	ultipa "ultipa-go-sdk/rpc"
	"ultipa-go-sdk/sdk/configuration"
)

type GraphClusterInfo struct {
	Graph         string
	Leader        *Connection
	Followers     []*Connection
	Algos         []*Connection
	LastAlgoIndex int //记录上次使用的 Task 节点索引
}

// handle all connections
type ConnectionPool struct {
	GraphMgr    *GraphManager // graph name : ClusterInfo
	Config      *configuration.UltipaConfig
	Connections map[string]*Connection // Host : Connection
	RandomTick  int
	Actives     []*Connection
	IsRaft      bool
}

func NewConnectionPool(config *configuration.UltipaConfig) (*ConnectionPool, error) {

	if len(config.Hosts) < 1 {
		return nil, errors.New("Error Hosts can not by empty")
	}

	pool := &ConnectionPool{
		Config:      config,
		Connections: map[string]*Connection{},
		GraphMgr:    NewGraphManager(),
	}

	// Init Cluster Manager

	// Get Connections
	err := pool.CreateConnections()

	if err != nil {
		log.Println(err)
	}

	// Refresh Actives
	pool.RefreshActives()

	// Refresh global Cluster info
	err = pool.RefreshClusterInfo("global")

	if err != nil {
		log.Println(err)
	}

	return pool, err
}

func (pool *ConnectionPool) CreateConnections() error {
	var err error

	for _, host := range pool.Config.Hosts {
		conn, _ := NewConnection(host, pool.Config)
		pool.Connections[host] = conn
	}

	return err
}

// 更新查看哪些连接还有效
func (pool *ConnectionPool) RefreshActives() {
	pool.Actives = []*Connection{}
	for _, conn := range pool.Connections {

		ctx, _ := pool.NewContext(nil)

		resp, err := conn.GetClient().SayHello(ctx, &ultipa.HelloUltipaRequest{
			Name: "go sdk refresh",
		})

		if err != nil {
			continue
		}

		if resp.Status == nil || resp.Status.ErrorCode == ultipa.ErrorCode_SUCCESS {
			pool.Actives = append(pool.Actives, conn)
		}

	}
}

// sync cluster info from server
func (pool *ConnectionPool) RefreshClusterInfo(graphName string) error {

	var conn *Connection
	var err error
	if pool.GraphMgr.GetLeader(graphName) == nil {
		conn, err = pool.GetConn(nil)
	} else {
		conn = pool.GraphMgr.GetLeader(graphName)
	}

	if err != nil {
		return err
	}

	ctx, _ := pool.NewContext(&configuration.RequestConfig{GraphName: graphName})
	client := conn.GetClient()
	resp, err := client.GetLeader(ctx, &ultipa.GetLeaderRequest{})

	if resp == nil || err != nil {
		return err
	}

	if resp.Status.ErrorCode == ultipa.ErrorCode_NOT_RAFT_MODE {
		pool.IsRaft = false
	}

	if resp.Status.ErrorCode == ultipa.ErrorCode_RAFT_REDIRECT {
		pool.IsRaft = true
		if pool.Connections[resp.Status.ClusterInfo.Redirect] == nil {
			pool.Connections[resp.Status.ClusterInfo.Redirect], err = NewConnection(resp.Status.ClusterInfo.Redirect, pool.Config)
		}

		pool.SetMasterConn(graphName, pool.Connections[resp.Status.ClusterInfo.Redirect])

		return pool.RefreshClusterInfo(graphName)
	}

	if resp.Status.ErrorCode != ultipa.ErrorCode_SUCCESS {
		// not raft mode
		log.Println(resp.Status.Msg)
	} else {
		pool.IsRaft = true
		c := pool.Connections[resp.Status.ClusterInfo.LeaderAddress]
		pool.GraphMgr.SetLeader(graphName, c)
		pool.GraphMgr.ClearFollower(graphName)

		for _, follower := range resp.Status.ClusterInfo.Followers {
			fconn := pool.Connections[follower.Address]

			if fconn == nil {
				fconn, err = NewConnection(follower.Address, pool.Config)

				if err != nil {
					return err
				}

				pool.Connections[follower.Address] = fconn
			}

			fconn.Host = follower.Address
			fconn.Active = follower.Status
			fconn.SetRoleFromInt32(follower.Role)
			pool.GraphMgr.AddFollower(graphName, fconn)
		}
	}

	return err
}

// Get client by global config
func (pool *ConnectionPool) GetConn(config *configuration.UltipaConfig) (*Connection, error) {

	if pool.Config.Consistency {
		return pool.GetMasterConn(config)
	} else {
		return pool.GetRandomConn(config)
	}
}

// Get master client
func (pool *ConnectionPool) GetMasterConn(config *configuration.UltipaConfig) (*Connection, error) {

	if pool.GraphMgr.GetLeader(config.CurrentGraph) == nil {
		err := pool.RefreshClusterInfo(config.CurrentGraph)

		if err != nil {
			return nil, err
		}
	}

	return pool.GraphMgr.GetLeader(config.CurrentGraph), nil

}

//SetMasterConn (graphName , *conn) Set master client
func (pool *ConnectionPool) SetMasterConn(graphName string, conn *Connection) {
	pool.GraphMgr.SetLeader(graphName, conn)
}

// Get random client
func (pool *ConnectionPool) GetRandomConn(config *configuration.UltipaConfig) (*Connection, error) {
	if len(pool.Actives) < 1 {
		return nil, errors.New("No Actived Connection is found")
	}

	pool.RandomTick++

	return pool.Actives[pool.RandomTick%len(pool.Actives)], nil
}

// Get Task/Analytics client
func (pool *ConnectionPool) GetAnalyticsConn(config *configuration.UltipaConfig) (*Connection, error) {

	gci := pool.GraphMgr.GetGraph(config.CurrentGraph)

	if gci == nil {
		err := pool.RefreshClusterInfo(config.CurrentGraph)
		if err != nil {
			return nil, err
		}
		gci = pool.GraphMgr.GetGraph(config.CurrentGraph)
	}

	return gci.GetAnalyticConn()

}

func (pool *ConnectionPool) Close() error {
	for _, conn := range pool.Connections {
		err := conn.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

// set context with timeout and auth info
func (pool *ConnectionPool) NewContext(config *configuration.RequestConfig) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(pool.Config.Timeout)*time.Second)
	ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs(pool.Config.ToContextKV(config)...))
	return ctx, cancel
}
