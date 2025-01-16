package grpc

import (
	context "context"
	"log"
)

func (s *Server) Signup(ctx context.Context, data *SignupRequest) (*User, error) {
	user, err := s.am.CreateUser(ctx, data.Username, data.Password)
	if err != nil {
		log.Printf("Error Signup - %v: %v", data, err)
		return nil, err
	}
	log.Printf("Success Signup - %v", user)
	return &User{Username: user.Username}, nil
}

func (s *Server) Login(ctx context.Context, data *LoginRequest) (*Session, error) {
	session, err := s.am.LoginUser(ctx, data.Username, data.Password)
	if err != nil {
		log.Printf("Error Login - %v: %v", data, err)
		return nil, err
	}
	log.Printf("Success Login - for user %s", session.Username)
	return &Session{Id: session.Id, ValidThrough: session.ValidThrough.String(), Username: session.Username}, nil
}

func (s *Server) Logout(ctx context.Context, data *SessionId) (*Blank, error) {
	err := s.am.InvalidateSession(ctx, data.Id)
	if err != nil {
		log.Printf("Error Logout - %v: %v", data, err)
	} else {
		log.Printf("Success Logout - %v", data)
	}
	return &Blank{}, err
}

func (s *Server) Authenticate(ctx context.Context, data *SessionId) (*User, error) {
	user, err := s.am.GetUserBySessionId(ctx, data.Id)
	if err != nil {
		log.Printf("Error Authenticate - %v: %v", data, err)
		return &User{}, err
	}
	log.Printf("Success Authenticate - %v", user)
	return &User{Username: user.Username}, nil
}

func (s *Server) ChangePassword(ctx context.Context, data *ChangePasswordRequest) (*Blank, error) {
	user, err := s.am.ChangePassword(ctx, data.Username, data.CurrentPassword, data.NewPassword)
	if err != nil {
		log.Printf("Error ChangePassword - %s", data.Username)
	} else {
		log.Printf("Success ChangePassword - for user %s", user.Username)
	}
	return &Blank{}, err
}
