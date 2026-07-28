package service

import (
	"context"
)

type GraphRequest struct {
	Path  string
	Depth int
	Limit int
}
type GraphResult struct{}

func (s *Service) LinkGraph(ctx context.Context, req GraphRequest) (GraphResult, error) {
	return GraphResult{}, nil
}

type TagRequest struct {
	Hierarchical bool
}
type TagResult struct{}

func (s *Service) TagList(ctx context.Context, req TagRequest) (TagResult, error) {
	return TagResult{}, nil
}

type ListRequest struct{}
type ListResult struct{}

func (s *Service) ListNotes(ctx context.Context, req ListRequest) (ListResult, error) {
	return ListResult{}, nil
}

type MetadataRequest struct {
	Path string
}
type MetadataResult struct{}

func (s *Service) NoteMetadata(ctx context.Context, req MetadataRequest) (MetadataResult, error) {
	return MetadataResult{}, nil
}

// VaultStats replaces the one in service.go, wait, no, I'll update it here and remove it from service.go
