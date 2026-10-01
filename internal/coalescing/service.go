package coalescing

import (
	"github.com/iamrafaelmelo/requests-coalescing/internal/cassandra"
	"golang.org/x/sync/singleflight"
)

type Service struct {
	repository *cassandra.Repository
	group      singleflight.Group
}

func NewService(repository *cassandra.Repository) *Service {
	return &Service{
		repository: repository,
	}
}

func (s *Service) GetMessage(id string) (string, string, error) {
	result, err, _ := s.group.Do(id, func() (any, error) {
		content, host, err := s.repository.GetMessage(id)

		if err != nil {
			return nil, err
		}

		return struct {
			Content string
			Host    string
		}{
			Content: content,
			Host:    host,
		}, nil
	})

	if err != nil {
		return "", "", err
	}

	data := result.(struct {
		Content string
		Host    string
	})

	return data.Content, data.Host, nil
}
