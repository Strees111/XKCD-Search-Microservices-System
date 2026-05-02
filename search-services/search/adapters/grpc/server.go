package grpc

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	searchpb "yadro.com/course/proto/search"
	"yadro.com/course/search/core"
)

const defaultLimit = 10

func NewServer(service core.Searcher) *Server {
	return &Server{service: service}
}

type Server struct {
	searchpb.UnimplementedSearchServiceServer
	service core.Searcher
}

func (s *Server) Ping(_ context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}

func (s *Server) Search(
	ctx context.Context, req *searchpb.SearchRequest,
) (*searchpb.SearchResponse, error) {
	if req.Limit == 0 {
		req.Limit = defaultLimit
	}
	results, err := s.service.Search(ctx, req.Phrase, int(req.Limit))
	if err != nil {
		if errors.Is(err, core.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "nothing found")
		}
		return nil, err
	}
	comics := make([]*searchpb.Comic, 0, len(results))
	for _, c := range results {
		comics = append(comics, &searchpb.Comic{
			Id:  int64(c.ID),
			Url: c.URL,
		})
	}
	return &searchpb.SearchResponse{Comics: comics}, nil
}

func (s *Server) SearchIndex(ctx context.Context, req *searchpb.SearchRequest) (*searchpb.SearchResponse, error) {
	if req.Limit == 0 {
		req.Limit = defaultLimit
	}
	results, err := s.service.SearchIndex(ctx, req.Phrase, int(req.Limit))
	if err != nil {
		if errors.Is(err, core.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "nothing found")
		}
		return nil, err
	}
	comics := make([]*searchpb.Comic, 0, len(results))
	for _, c := range results {
		comics = append(comics, &searchpb.Comic{
			Id:  int64(c.ID),
			Url: c.URL,
		})
	}
	return &searchpb.SearchResponse{Comics: comics}, nil
}
