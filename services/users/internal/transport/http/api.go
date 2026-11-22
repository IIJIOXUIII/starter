package http

// Реализация gen.StrictServerInterface: профили (см. api/openapi/users.yaml).

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/oapi-codegen/runtime/types"

	"github.com/rosterra/services/users/internal/domain"
	"github.com/rosterra/services/users/internal/service"
	gen "github.com/rosterra/services/users/internal/transport/http/api/gen"
)

// ListProfiles — batch по id, только admin (агрегация admin-списка в BFF).
func (s *Server) ListProfiles(ctx context.Context, request gen.ListProfilesRequestObject) (gen.ListProfilesResponseObject, error) {
	claims := s.claims(ctx)
	if claims == nil {
		return gen.ListProfiles403JSONResponse(errBody(codeUnauthorized)), nil
	}
	if !isAdmin(claims) {
		return gen.ListProfiles403JSONResponse(errBody(codeForbidden)), nil
	}

	ids, badReq := parseUUIDs(request.Params.Ids)
	if badReq != nil {
		return gen.ListProfiles400JSONResponse(*badReq), nil
	}

	profiles, err := s.svc.ListProfiles(ctx, ids)
	if err != nil {
		var ve *service.ValidateError
		if errors.As(err, &ve) {
			return gen.ListProfiles400JSONResponse(*validationError(ve)), nil
		}
		if e := domainError(err); e != nil {
			return gen.ListProfiles403JSONResponse(*e), nil
		}
		return nil, err
	}

	items := make([]gen.Profile, 0, len(profiles))
	for i := range profiles {
		items = append(items, profileToGen(&profiles[i]))
	}
	return gen.ListProfiles200JSONResponse(gen.ProfileList{Items: items}), nil
}

// GetProfile — владелец или admin.
func (s *Server) GetProfile(ctx context.Context, request gen.GetProfileRequestObject) (gen.GetProfileResponseObject, error) {
	claims := s.claims(ctx)
	if claims == nil {
		return gen.GetProfile404JSONResponse(errBody(codeUnauthorized)), nil
	}
	if claims.UserID != uuid.UUID(request.UserId) && !isAdmin(claims) {
		return gen.GetProfile404JSONResponse(errBody(codeForbidden)), nil
	}

	profile, err := s.svc.GetProfile(ctx, uuidFromGen(request.UserId))
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return gen.GetProfile404JSONResponse(errBody(codeNotFound)), nil
		}
		return nil, err
	}
	return gen.GetProfile200JSONResponse(profileToGen(profile)), nil
}

// UpdateProfile — владелец или admin; notes — только admin.
func (s *Server) UpdateProfile(ctx context.Context, request gen.UpdateProfileRequestObject) (gen.UpdateProfileResponseObject, error) {
	claims := s.claims(ctx)
	if claims == nil {
		return gen.UpdateProfile403JSONResponse(errBody(codeUnauthorized)), nil
	}
	isAdminUser := isAdmin(claims)
	if claims.UserID != uuid.UUID(request.UserId) && !isAdminUser {
		return gen.UpdateProfile403JSONResponse(errBody(codeForbidden)), nil
	}
	// notes — служебное поле, редактирует только admin.
	if request.Body != nil && request.Body.Notes != nil && !isAdminUser {
		return gen.UpdateProfile403JSONResponse(errBody(codeForbidden)), nil
	}

	var upd domain.UpdateProfile
	if request.Body != nil {
		upd = domain.UpdateProfile{
			DisplayName: request.Body.DisplayName,
			AvatarURL:   request.Body.AvatarUrl,
			Notes:       request.Body.Notes,
		}
	}

	profile, err := s.svc.UpdateProfile(ctx, uuidFromGen(request.UserId), upd)
	if err != nil {
		var ve *service.ValidateError
		if errors.As(err, &ve) {
			return gen.UpdateProfile400JSONResponse(*validationError(ve)), nil
		}
		if errors.Is(err, domain.ErrNotFound) {
			return gen.UpdateProfile404JSONResponse(errBody(codeNotFound)), nil
		}
		if e := domainError(err); e != nil {
			return gen.UpdateProfile403JSONResponse(*e), nil
		}
		return nil, err
	}
	return gen.UpdateProfile200JSONResponse(profileToGen(profile)), nil
}

// uuidFromGen — конверсия oapi UUID → google UUID (алиас одного типа).
func uuidFromGen(id types.UUID) uuid.UUID {
	return uuid.UUID(id)
}
