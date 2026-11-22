// Package grpcusers — gRPC-транспорт users-service (users.v1).
//
// Синхронные межсервисные операции с профилями. Основной поток создания
// профилей — асинхронный (событие user.created из NATS); gRPC держим для
// сценариев, где нужен синхронный ответ (например, будущий sync-вызов auth).
package grpcusers

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	usersv1 "github.com/rosterra/services/users/gen/users/v1"
	"github.com/rosterra/services/users/internal/domain"
	"github.com/rosterra/services/users/internal/service"
)

// Server — реализация users.v1.UsersService.
type Server struct {
	usersv1.UnimplementedUsersServiceServer
	svc *service.Service
	log *slog.Logger
}

// NewServer — fx-провайдер gRPC-сервера.
func NewServer(svc *service.Service, log *slog.Logger) *Server {
	return &Server{svc: svc, log: log}
}

// CreateProfile создаёт профиль (идемпотентно).
func (s *Server) CreateProfile(ctx context.Context, req *usersv1.CreateProfileRequest) (*usersv1.CreateProfileResponse, error) {
	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "user_id must be a UUID")
	}
	profile, err := s.svc.CreateProfile(ctx, userID, req.GetDisplayName(), req.GetAvatarUrl())
	if err != nil {
		return nil, s.mapErr("create", err)
	}
	return &usersv1.CreateProfileResponse{Profile: toProto(profile)}, nil
}

// GetProfile возвращает профиль.
func (s *Server) GetProfile(ctx context.Context, req *usersv1.GetProfileRequest) (*usersv1.GetProfileResponse, error) {
	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "user_id must be a UUID")
	}
	profile, err := s.svc.GetProfile(ctx, userID)
	if err != nil {
		return nil, s.mapErr("get", err)
	}
	return &usersv1.GetProfileResponse{Profile: toProto(profile)}, nil
}

// UpdateProfile обновляет профиль.
func (s *Server) UpdateProfile(ctx context.Context, req *usersv1.UpdateProfileRequest) (*usersv1.UpdateProfileResponse, error) {
	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "user_id must be a UUID")
	}
	upd := domain.UpdateProfile{
		DisplayName: strPtr(req.GetDisplayName()),
		AvatarURL:   strPtr(req.GetAvatarUrl()),
		Notes:       strPtr(req.GetNotes()),
	}
	profile, err := s.svc.UpdateProfile(ctx, userID, upd)
	if err != nil {
		return nil, s.mapErr("update", err)
	}
	return &usersv1.UpdateProfileResponse{Profile: toProto(profile)}, nil
}

// mapErr — доменная ошибка → gRPC-статус.
func (s *Server) mapErr(op string, err error) error {
	switch {
	case err == nil:
		return nil
	case err == domain.ErrNotFound:
		return status.Error(codes.NotFound, "profile not found")
	default:
		s.log.Error("grpc op failed", "op", op, "err", err)
		return status.Error(codes.Internal, "internal error")
	}
}

// toProto — домен → proto.
func toProto(p *domain.Profile) *usersv1.Profile {
	avatar, notes := "", ""
	if p.AvatarURL != nil {
		avatar = *p.AvatarURL
	}
	if p.Notes != nil {
		notes = *p.Notes
	}
	return &usersv1.Profile{
		UserId:      p.UserID.String(),
		DisplayName: p.DisplayName,
		AvatarUrl:   avatar,
		Notes:       notes,
		CreatedAt:   timestamppb.New(p.CreatedAt),
		UpdatedAt:   timestamppb.New(p.UpdatedAt),
	}
}

// strPtr — "" → nil (не менять), значение → указатель.
func strPtr(v string) *string {
	if v == "" {
		return nil
	}
	return &v
}
