package sdk

import (
	"context"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"time"
	ultipa "ultipa-go-sdk/rpc"
	"ultipa-go-sdk/types"
	"ultipa-go-sdk/utils"
)
type ClientInfo struct {
	conn *grpc.ClientConn
	Client ultipa.UltipaRpcsClient
}

func (t *ClientInfo) init(conn *grpc.ClientConn) {
	client := ultipa.NewUltipaRpcsClient(conn)
	t.Client = client
	t.conn = conn
}
func (t *ClientInfo) Close(){
	if t.conn != nil {
		t.conn.Close()
	}
}

type HostManager struct {
	LeaderHost string
	FollowersHost []string
	leaderClientInfo *ClientInfo
	algoClientInfo *ClientInfo
	defaultClientInfo *ClientInfo
}

func (t *HostManager)GetAllHosts()  *[]string{
	hosts := []string{
		t.LeaderHost,
	}
	if t.FollowersHost != nil && len(t.FollowersHost) > 0 {
		hosts = append(hosts, t.FollowersHost...)
	}
	return &hosts

}

type DefaultConfig struct {
	GraphSetName string
	TimeoutWithSeconds uint32
	ResponseWithRequestInfo bool
}

type Connection struct {
	HostManager *HostManager
	metadataKV *[]string
	username string
	password string
	crtFile string
	defaultConfig *DefaultConfig
}

func GetConnection(host string, username string, password string, crtFile string, defaultConfig *DefaultConfig) (*Connection, error){
	connect := Connection{}
	err := connect.init(host, username, password, crtFile, defaultConfig)
	if err != nil {
		return nil, err
	}
	return &connect, nil
}
func (t *Connection) SetDefaultConfig(defaultConfig *DefaultConfig)  {
	if t.defaultConfig == nil {
		t.defaultConfig = &DefaultConfig{ "default", 15, false}
	}
	if defaultConfig != nil {
		if &defaultConfig.GraphSetName != nil {
			t.defaultConfig.GraphSetName = defaultConfig.GraphSetName
		}
		if &defaultConfig.TimeoutWithSeconds != nil {
			t.defaultConfig.TimeoutWithSeconds = defaultConfig.TimeoutWithSeconds
		}
		if &defaultConfig.ResponseWithRequestInfo != nil {
			t.defaultConfig.ResponseWithRequestInfo = defaultConfig.ResponseWithRequestInfo
		}
	}
}
func saveClose(clientInfo *ClientInfo)  {
	if clientInfo != nil {
		clientInfo.Close()
	}
}
func (t *Connection) CloseAll()  {
	if t.HostManager != nil {
		saveClose(t.HostManager.defaultClientInfo)
		saveClose(t.HostManager.leaderClientInfo)
		saveClose(t.HostManager.algoClientInfo)
	}
}
func (t *Connection) createClientInfo(host string) (*ClientInfo, error) {
	var opts []grpc.DialOption
	//opts = append(opts, grpc.WithDefaultCallOptions(grpc.MaxCallSendMsgSize(-1)))
	//opts = append(opts, grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(-1)))
	if len(t.crtFile) == 0 {
		// 兼容2.0
		opts = append(opts, grpc.WithInsecure())
		conn, _ := grpc.Dial(host, opts...)
		clientInfo := ClientInfo{}
		clientInfo.init(conn)
		return &clientInfo, nil
	} else {
		creds, err := credentials.NewClientTLSFromFile(t.crtFile, "ultipa")
		if err != nil {
			return nil, err
		}
		opts = append(opts, grpc.WithTransportCredentials(creds))
		conn, _ := grpc.Dial(host, opts...)
		clientInfo := ClientInfo{}
		clientInfo.init(conn)
		return &clientInfo, nil
	}
}
func (t *Connection) init(host string, username string, password string, crt string, config *DefaultConfig)  error {
	t.username = username
	t.password = password
	t.crtFile = crt
	t.SetDefaultConfig(config)
	kv := []string{"user", username, "password", password}
	t.metadataKV = &kv
	t.HostManager = &HostManager{}
	clientInfo, err := t.createClientInfo(host)
	if err != nil {
		return err
	}
	t.HostManager.defaultClientInfo = clientInfo
	t.HostManager.LeaderHost = host
	// test connection
	_, err1 := t.TestConnect()
	if err1 != nil {
		return err1
	}
	return nil
}

const (
	TIMEOUT_DEFAUL time.Duration = time.Minute
)

func (t *Connection) chooseClient(timeout time.Duration) (_clientInfo *ClientInfo, _context context.Context, _cancelFunc context.CancelFunc) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	kv := []string{"graph_name", t.defaultConfig.GraphSetName}
	kv = append(kv, *t.metadataKV...)
	ctx = metadata.AppendToOutgoingContext(ctx, kv...)
	//ctx = metadata.AppendToOutgoingContext(ctx, *t.metadataKV...)
	//defer cancel()
	clientInfo := t.HostManager.leaderClientInfo
	if clientInfo == nil {
		clientInfo = t.HostManager.defaultClientInfo
	}
	return clientInfo, ctx, cancel
}
func (t *Connection) TestConnect()  (bool, error) {
	clientInfo, ctx, cancel := t.chooseClient(time.Second * 10)
	defer cancel()
	name := "MyTest"
	res, err := clientInfo.Client.SayHello(ctx, &ultipa.HelloUltipaRequest{
		Name: name,
	})
	if err != nil {
		return false, err
	}
	if res.Message != name + " Welcome To Ultipa!"{
		return false, err
	}
	return true, nil
}

type RaftLeaderResSimple struct {
	Code ultipa.ErrorCode
	Message string
	LeaderHost string
	FollowersHost []string
}
func (t *Connection) autoGetRaftLeader(host string) (*RaftLeaderResSimple,error){
	conn, err := GetConnection(host, t.username, t.password, t.crtFile, t.defaultConfig)
	// 用一次就关掉
	defer conn.CloseAll()
	if err != nil {
		return nil, err
	}
	res := conn.GetLeaderReuqest()
	errorCode := res.Status.Code
	switch errorCode {
	case types.ErrorCode_SUCCESS:
		followers := res.Status.ClusterInfo.RaftPeers
		return &RaftLeaderResSimple{
			Code: errorCode,
			LeaderHost: host,
			FollowersHost: utils.Remove(followers, host),
		}, nil
	case types.ErrorCode_NOT_RAFT_MODE:
		return &RaftLeaderResSimple{
			Code:          types.ErrorCode_SUCCESS,
			LeaderHost:    host,
			FollowersHost: nil,
		}, nil
	case types.ErrorCode_RAFT_REDIRECT:
		return t.autoGetRaftLeader(res.Status.ClusterInfo.Redirect)
	}
	return &RaftLeaderResSimple{
		Code: errorCode,
		Message: res.Status.Message,
	}, nil
}
func (t *Connection)  RefreshRaftLeader() error{
	hosts := t.HostManager.GetAllHosts()
	for _, host := range *hosts{
		res, err := t.autoGetRaftLeader(host)
		if err != nil {
			return err
		}
		if res.Code == types.ErrorCode_SUCCESS {
			leaderHost := res.LeaderHost
			followersHost := res.FollowersHost
			t.HostManager.LeaderHost = leaderHost
			leaderClient, _ := t.createClientInfo(leaderHost)
			t.HostManager.leaderClientInfo = leaderClient
			t.HostManager.FollowersHost = followersHost
			if followersHost != nil && len(followersHost) > 0 {
				_c, _ := t.createClientInfo(followersHost[0])
				t.HostManager.algoClientInfo = _c
			} else {
				t.HostManager.algoClientInfo = leaderClient
			}
			return nil
		}
		return fmt.Errorf("%v - %v", res.Code.String(), res.Message)
	}
	return fmt.Errorf("Unknow error! ")

}
