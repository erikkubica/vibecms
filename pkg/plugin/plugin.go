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
type ExtensionPlugin interface {
	GetSubscriptions() ([]*pb.Subscription, error)
	HandleEvent(action string, payload []byte) (*pb.EventResponse, error)
	Shutdown() error
}

// ExtensionGRPCPlugin implements plugin.GRPCPlugin for the extension interface.
type ExtensionGRPCPlugin struct {
	plugin.NetRPCUnsupportedPlugin
	Impl ExtensionPlugin // Only used on the plugin side
}

func (p *ExtensionGRPCPlugin) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	pb.RegisterExtensionPluginServer(s, &GRPCServer{Impl: p.Impl})
	return nil
}

func (p *ExtensionGRPCPlugin) GRPCClient(broker *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &GRPCClient{client: pb.NewExtensionPluginClient(c)}, nil
}

// --- GRPCClient (host side) ---

// GRPCClient is used by the host to call into the plugin process.
type GRPCClient struct {
	client pb.ExtensionPluginClient
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

func (c *GRPCClient) Shutdown() error {
	_, err := c.client.Shutdown(context.Background(), &pb.Empty{})
	return err
}

// --- GRPCServer (plugin side) ---

// GRPCServer wraps an ExtensionPlugin implementation for gRPC serving.
type GRPCServer struct {
	pb.UnimplementedExtensionPluginServer
	Impl ExtensionPlugin
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

func (s *GRPCServer) Shutdown(ctx context.Context, _ *pb.Empty) (*pb.Empty, error) {
	return &pb.Empty{}, s.Impl.Shutdown()
}
