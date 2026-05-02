package words

import (
	"context"
	"log/slog"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	wordspb "yadro.com/course/proto/words"
	"yadro.com/course/update/core"
)

type Client struct {
	log    *slog.Logger
	conn   *grpc.ClientConn
	client wordspb.WordsClient
}

func NewClient(address string, log *slog.Logger) (*Client, error) {
	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &Client{
		client: wordspb.NewWordsClient(conn),
		conn:   conn,
		log:    log,
	}, nil
}

func (c *Client) Norm(ctx context.Context, phrase string) ([]string, error) {
	req := wordspb.WordsRequest{Phrase: phrase}
	reply, err := c.client.Norm(ctx, &req)
	if err != nil {
		if s, ok := status.FromError(err); ok && s.Code() == codes.ResourceExhausted {
			return nil, core.ErrBadArguments
		}
		c.log.Error("grpc norm call failed", "error", err)
		return nil, err
	}
	return reply.Words, nil
}

func (c *Client) Ping(ctx context.Context) error {
	_, err := c.client.Ping(ctx, &emptypb.Empty{})
	if err != nil {
		c.log.Error("grpc ping call failed", "error", err)
		return err
	}
	return nil
}

func (c *Client) Close() error {
	if c.conn == nil {
		return nil
	}
	return c.conn.Close()
}
