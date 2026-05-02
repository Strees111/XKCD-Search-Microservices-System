package search

import (
	"context"
	"log/slog"
	"math"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"
	"yadro.com/course/api/core"
	searchpb "yadro.com/course/proto/search"
)

type Client struct {
	log    *slog.Logger
	client searchpb.SearchServiceClient
}

func NewClient(address string, log *slog.Logger) (*Client, error) {
	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &Client{
		client: searchpb.NewSearchServiceClient(conn),
		log:    log,
	}, nil
}

func (c *Client) Ping(ctx context.Context) error {
	_, err := c.client.Ping(ctx, &emptypb.Empty{})
	return err
}

func (c *Client) Search(ctx context.Context, phrase string, limit int) ([]core.Comics, error) {
	if limit < math.MinInt32 || limit > math.MaxInt32 {
		return nil, core.ErrBadArguments
	}
	response, err := c.client.Search(ctx, &searchpb.SearchRequest{
		Phrase: phrase,
		Limit:  int64(limit),
	})
	if err != nil {
		c.log.Error("search error", "error", err)
		return nil, err
	}

	comics := make([]core.Comics, len(response.Comics))
	for i, comic := range response.Comics {
		comics[i] = core.Comics{
			ID:  int(comic.Id),
			URL: comic.Url,
		}
	}
	return comics, nil
}

func (c *Client) SearchIndex(ctx context.Context, phrase string, limit int) ([]core.Comics, error) {
	if limit < math.MinInt32 || limit > math.MaxInt32 {
		return nil, core.ErrBadArguments
	}
	response, err := c.client.SearchIndex(ctx, &searchpb.SearchRequest{
		Phrase: phrase,
		Limit:  int64(limit),
	})
	if err != nil {
		c.log.Error("search error", "error", err)
		return nil, err
	}

	comics := make([]core.Comics, len(response.Comics))
	for i, comic := range response.Comics {
		comics[i] = core.Comics{
			ID:  int(comic.Id),
			URL: comic.Url,
		}
	}
	return comics, nil
}
