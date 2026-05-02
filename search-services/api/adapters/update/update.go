package update

import (
	"context"
	"log/slog"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"yadro.com/course/api/core"
	updatepb "yadro.com/course/proto/update"
)

type Client struct {
	log    *slog.Logger
	client updatepb.UpdateClient
}

func NewClient(address string, log *slog.Logger) (*Client, error) {
	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &Client{
		client: updatepb.NewUpdateClient(conn),
		log:    log,
	}, nil
}

func (c *Client) Ping(ctx context.Context) error {
	_, err := c.client.Ping(ctx, &emptypb.Empty{})
	return err
}

func (c *Client) Status(ctx context.Context) (core.UpdateStatus, error) {
	StatusReply, err := c.client.Status(ctx, &emptypb.Empty{})
	if err != nil {
		return "", err
	}
	var status core.UpdateStatus
	switch StatusReply.Status {
	case updatepb.Status_STATUS_UNSPECIFIED:
		status = core.StatusUpdateIdle
	case updatepb.Status_STATUS_IDLE:
		status = core.StatusUpdateIdle
	case updatepb.Status_STATUS_RUNNING:
		status = core.StatusUpdateRunning
	default:
		status = core.StatusUpdateUnknown
	}
	return status, nil
}

func (c *Client) Stats(ctx context.Context) (core.UpdateStats, error) {
	StatusReply, err := c.client.Stats(ctx, &emptypb.Empty{})
	if err != nil {
		c.log.Error("grpc stats call failed", "error", err)
		return core.UpdateStats{}, err
	}
	return core.UpdateStats{
		WordsTotal:    int(StatusReply.WordsTotal),
		WordsUnique:   int(StatusReply.WordsUnique),
		ComicsFetched: int(StatusReply.ComicsFetched),
		ComicsTotal:   int(StatusReply.ComicsTotal),
	}, nil

}

func (c *Client) Update(ctx context.Context) error {
	_, err := c.client.Update(ctx, &emptypb.Empty{})
	if err != nil {
		if s, ok := status.FromError(err); ok && s.Code() == codes.AlreadyExists {
			return core.ErrAlreadyExists
		}
		return err
	}
	return nil
}

func (c *Client) Drop(ctx context.Context) error {
	_, err := c.client.Drop(ctx, &emptypb.Empty{})
	if err != nil {
		return err
	}
	return nil
}
