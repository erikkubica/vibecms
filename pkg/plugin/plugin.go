package plugin

import (
	"context"

	"github.com/hashicorp/go-plugin"
	pb "vibecms/pkg/plugin/proto"
	"google.golang.org/grpc"
)

// Handshake is used to ensure the plugin and host are compatible.
var Handshake = plugin.HandshakeConfig{
	ProtocolVersion:  2,
	MagicCookieKey:   "VIBECMS_PLUGIN",
	MagicCookieValue: "vibecms",
}

// PluginMap is the map of plugins we can dispense.
var PluginMap = map[string]plugin.Plugin{
	"extension": &ExtensionGRPCPlugin{},
}

// ExtensionPlugin is the interface that plugin binaries must implement.
// On the plugin side, Initialize receives a *grpc.ClientConn to the host.
// On the host side, GRPCClient.InitializeHost provides the broker-based version.
type ExtensionPlugin interface {
	GetSubscriptions() ([]*pb.Subscription, error)
	HandleEvent(action string, payload []byte) (*pb.EventResponse, error)
	HandleHTTPRequest(req *pb.PluginHTTPRequest) (*pb.PluginHTTPResponse, error)
	Shutdown() error
	Initialize(hostConn *grpc.ClientConn) error
}

// ExtensionGRPCPlugin implements plugin.GRPCPlugin for the extension interface.
type ExtensionGRPCPlugin struct {
	plugin.NetRPCUnsupportedPlugin
	Impl ExtensionPlugin // Only used on the plugin side
}

func (p *ExtensionGRPCPlugin) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	pb.RegisterExtensionPluginServer(s, &GRPCServer{Impl: p.Impl, broker: broker})
	return nil
}

func (p *ExtensionGRPCPlugin) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &GRPCClient{client: pb.NewExtensionPluginClient(c), broker: broker}, nil
}

// --- GRPCClient (host side) ---

// GRPCClient is used by the host to call into the plugin process.
type GRPCClient struct {
	client pb.ExtensionPluginClient
	broker *plugin.GRPCBroker
}

func (c *GRPCClient) GetSubscriptions() ([]*pb.Subscription, error) {
	resp, err := c.client.GetSubscriptions(context.Background(), &pb.Empty{})
	if err != nil {
		return nil, err
	}
	return resp.Subscriptions, nil
}

func (c *GRPCClient) HandleEvent(action string, payload []byte) (*pb.EventResponse, error) {
	return c.client.HandleEvent(context.Background(), &pb.EventRequest{
		Action:  action,
		Payload: payload,
	})
}

func (c *GRPCClient) HandleHTTPRequest(req *pb.PluginHTTPRequest) (*pb.PluginHTTPResponse, error) {
	return c.client.HandleHTTPRequest(context.Background(), req)
}

func (c *GRPCClient) Shutdown() error {
	_, err := c.client.Shutdown(context.Background(), &pb.Empty{})
	return err
}

// Initialize satisfies the ExtensionPlugin interface (unused on the host side).
func (c *GRPCClient) Initialize(hostConn *grpc.ClientConn) error {
	return nil // host side uses InitializeHost instead
}

// InitializeHost starts a gRPC host service on the broker and tells the plugin
// where to connect back. registerServer is called with the *grpc.Server so
// the caller can register the VibeCMSHost service implementation.
func (c *GRPCClient) InitializeHost(registerServer func(s *grpc.Server)) error {
	hostServiceID := c.broker.NextId()
	go c.broker.AcceptAndServe(hostServiceID, func(opts []grpc.ServerOption) *grpc.Server {
		s := grpc.NewServer(opts...)
		registerServer(s)
		return s
	})

	_, err := c.client.Initialize(context.Background(), &pb.InitializeRequest{
		HostServiceId: hostServiceID,
	})
	return err
}

// --- GRPCServer (plugin side) ---

// GRPCServer wraps an ExtensionPlugin implementation for gRPC serving (runs in plugin process).
type GRPCServer struct {
	pb.UnimplementedExtensionPluginServer
	Impl   ExtensionPlugin
	broker *plugin.GRPCBroker
}

func (s *GRPCServer) GetSubscriptions(ctx context.Context, _ *pb.Empty) (*pb.SubscriptionList, error) {
	subs, err := s.Impl.GetSubscriptions()
	if err != nil {
		return nil, err
	}
	return &pb.SubscriptionList{Subscriptions: subs}, nil
}

func (s *GRPCServer) HandleEvent(ctx context.Context, req *pb.EventRequest) (*pb.EventResponse, error) {
	return s.Impl.HandleEvent(req.Action, req.Payload)
}

func (s *GRPCServer) HandleHTTPRequest(ctx context.Context, req *pb.PluginHTTPRequest) (*pb.PluginHTTPResponse, error) {
	return s.Impl.HandleHTTPRequest(req)
}

func (s *GRPCServer) Shutdown(ctx context.Context, _ *pb.Empty) (*pb.Empty, error) {
	return &pb.Empty{}, s.Impl.Shutdown()
}

func (s *GRPCServer) Initialize(ctx context.Context, req *pb.InitializeRequest) (*pb.Empty, error) {
	conn, err := s.broker.Dial(req.HostServiceId)
	if err != nil {
		return nil, err
	}
	if err := s.Impl.Initialize(conn); err != nil {
		return nil, err
	}
	return &pb.Empty{}, nil
}
