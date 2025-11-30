package staticservices

import (
	"strings"

	"github.com/nicklaw5/helix/v2"
)

type TwitchHelixService struct {
	nickname string
	client   *helix.Client
}

func (s *TwitchHelixService) Client() *helix.Client {
	return s.client
}

func (s *TwitchHelixService) GetNickname() string {
	return s.nickname
}

func NewTwitchHelixService(client *helix.Client) (*TwitchHelixService, error) {
	s := &TwitchHelixService{client: client}
	err := s.TestConnection()
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (s *TwitchHelixService) TestConnection() error {
	r, err := s.Client().GetUsers(&helix.UsersParams{})
	if err != nil {
		return err
	}
	s.nickname = strings.ToLower(r.Data.Users[0].Login)
	return nil
}
