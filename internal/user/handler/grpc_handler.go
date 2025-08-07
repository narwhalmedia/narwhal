package handler

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/narwhalmedia/narwhal/internal/user/service"
	"github.com/narwhalmedia/narwhal/pkg/auth"
	"github.com/narwhalmedia/narwhal/pkg/models"
	authpb "github.com/narwhalmedia/narwhal/pkg/auth/v1"
	commonpb "github.com/narwhalmedia/narwhal/pkg/common/v1"
	"github.com/narwhalmedia/narwhal/pkg/errors"
	"github.com/narwhalmedia/narwhal/pkg/interfaces"
)

// GRPCHandler implements the AuthService gRPC interface.
type GRPCHandler struct {
	authpb.UnimplementedAuthServiceServer

	authService *service.AuthService
	userService *service.UserService
	logger      interfaces.Logger
}

// NewGRPCHandler creates a new gRPC handler.
func NewGRPCHandler(
	authService *service.AuthService,
	userService *service.UserService,
	logger interfaces.Logger,
) *GRPCHandler {
	return &GRPCHandler{
		authService: authService,
		userService: userService,
		logger:      logger,
	}
}

// Login authenticates a user and returns tokens.
func (h *GRPCHandler) Login(ctx context.Context, req *authpb.LoginRequest) (*authpb.LoginResponse, error) {
	// Extract client info from context
	md, _ := metadata.FromIncomingContext(ctx)
	ipAddress := extractMetadataValue(md, "x-forwarded-for", "x-real-ip")
	userAgent := extractMetadataValue(md, "user-agent")

	// Perform login
	tokens, err := h.authService.Login(
		ctx,
		req.GetUsername(),
		req.GetPassword(),
		req.GetDeviceName(),
		ipAddress,
		userAgent,
	)
	if err != nil {
		return nil, toGRPCError(err)
	}

	// Get user info
	user, err := h.userService.GetUserByUsername(ctx, req.GetUsername())
	if err != nil {
		return nil, toGRPCError(err)
	}

	return &authpb.LoginResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		ExpiresIn:    int64(tokens.ExpiresIn),
		TokenType:    tokens.TokenType,
		User:         domainUserToProto(user),
	}, nil
}

// Logout logs out a user.
func (h *GRPCHandler) Logout(ctx context.Context, req *authpb.LogoutRequest) (*authpb.LogoutResponse, error) {
	// Get user ID from context
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	// Get session ID from context
	sessionID := getSessionIDFromContext(ctx)

	if req.GetAllDevices() {
		err = h.authService.LogoutAll(ctx, userID)
	} else {
		err = h.authService.Logout(ctx, userID, sessionID)
	}

	if err != nil {
		return nil, toGRPCError(err)
	}

	return &authpb.LogoutResponse{}, nil
}

// RefreshToken generates new tokens using a refresh token.
func (h *GRPCHandler) RefreshToken(
	ctx context.Context,
	req *authpb.RefreshTokenRequest,
) (*authpb.RefreshTokenResponse, error) {
	tokens, err := h.authService.RefreshToken(ctx, req.GetRefreshToken())
	if err != nil {
		return nil, toGRPCError(err)
	}

	return &authpb.RefreshTokenResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		ExpiresIn:    int64(tokens.ExpiresIn),
		TokenType:    tokens.TokenType,
	}, nil
}

// ValidateToken validates an access token.
func (h *GRPCHandler) ValidateToken(
	ctx context.Context,
	req *authpb.ValidateTokenRequest,
) (*authpb.ValidateTokenResponse, error) {
	claims, err := h.authService.ValidateToken(ctx, req.GetAccessToken())
	if err != nil {
		return &authpb.ValidateTokenResponse{Valid: false}, nil
	}

	// Convert role to proto
	role := commonpb.UserRole_USER_ROLE_UNSPECIFIED
	if len(claims.Roles) > 0 {
		switch claims.Roles[0] {
		case models.RoleAdmin:
			role = commonpb.UserRole_USER_ROLE_ADMIN
		case models.RoleUser:
			role = commonpb.UserRole_USER_ROLE_USER
		case models.RoleGuest:
			role = commonpb.UserRole_USER_ROLE_GUEST
		}
	}

	return &authpb.ValidateTokenResponse{
		Valid:     true,
		UserId:    claims.UserID,
		Username:  claims.Username,
		Role:      role,
		ExpiresAt: timestamppb.New(claims.ExpiresAt.Time),
	}, nil
}

// CreateUser creates a new user.
func (h *GRPCHandler) CreateUser(
	ctx context.Context,
	req *authpb.CreateUserRequest,
) (*authpb.CreateUserResponse, error) {
	// Verify caller has admin permissions
	if err := h.requireAdmin(ctx); err != nil {
		return nil, err
	}

	// Create user
	user, err := h.userService.CreateUser(ctx, req.GetUsername(), req.GetEmail(), req.GetPassword(), req.GetUsername())
	if err != nil {
		return nil, toGRPCError(err)
	}

	// Assign role if specified
	if req.GetRole() != commonpb.UserRole_USER_ROLE_UNSPECIFIED {
		roleName := protoRoleToString(req.GetRole())
		if err := h.userService.AssignRole(ctx, user.ID, roleName); err != nil {
			h.logger.Error("Failed to assign role", interfaces.Error(err))
		}
	}

	return &authpb.CreateUserResponse{
		User: domainUserToProto(user),
	}, nil
}

// GetUser retrieves a user by ID.
func (h *GRPCHandler) GetUser(ctx context.Context, req *authpb.GetUserRequest) (*authpb.GetUserResponse, error) {
	// Parse user ID
	userID, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user ID")
	}

	// Get user
	user, err := h.userService.GetUser(ctx, userID)
	if err != nil {
		return nil, toGRPCError(err)
	}

	return &authpb.GetUserResponse{
		User: domainUserToProto(user),
	}, nil
}

// GetCurrentUser retrieves the current authenticated user.
func (h *GRPCHandler) GetCurrentUser(
	ctx context.Context,
	req *authpb.GetCurrentUserRequest,
) (*authpb.GetCurrentUserResponse, error) {
	// Get user ID from context
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	// Get user
	user, err := h.userService.GetUser(ctx, userID)
	if err != nil {
		return nil, toGRPCError(err)
	}

	return &authpb.GetCurrentUserResponse{
		User: domainUserToProto(user),
	}, nil
}

// UpdateUser updates a user.
func (h *GRPCHandler) UpdateUser(
	ctx context.Context,
	req *authpb.UpdateUserRequest,
) (*authpb.UpdateUserResponse, error) {
	// Parse user ID
	userID, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user ID")
	}

	// Check permissions
	currentUserID, _ := getUserIDFromContext(ctx)
	if currentUserID != userID {
		// Only admins can update other users
		if err := h.requireAdmin(ctx); err != nil {
			return nil, err
		}
	}

	// Prepare updates
	updates := make(map[string]interface{})

	if req.GetUpdateMask() != nil {
		for _, path := range req.GetUpdateMask().GetPaths() {
			switch path {
			case "preferences.language":
				if req.GetUser().GetPreferences() != nil {
					// The preferences are now flattened, so we update the user directly.
					// The service layer will handle the map[string]interface{}
					updates["pref_language"] = req.GetUser().GetPreferences().GetLanguage()
				}
			case "preferences.theme":
				if req.GetUser().GetPreferences() != nil {
					updates["pref_theme"] = req.GetUser().GetPreferences().GetTheme()
				}
			case "email":
				updates["email"] = req.GetUser().GetEmail()
			}
		}
	}

	// Update user
	user, err := h.userService.UpdateUser(ctx, userID, updates)
	if err != nil {
		return nil, toGRPCError(err)
	}

	return &authpb.UpdateUserResponse{
		User: domainUserToProto(user),
	}, nil
}

// DeleteUser deletes a user.
func (h *GRPCHandler) DeleteUser(
	ctx context.Context,
	req *authpb.DeleteUserRequest,
) (*authpb.DeleteUserResponse, error) {
	// Verify caller has admin permissions
	if err := h.requireAdmin(ctx); err != nil {
		return nil, err
	}

	// Parse user ID
	userID, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user ID")
	}

	// Delete user
	if err := h.userService.DeleteUser(ctx, userID); err != nil {
		return nil, toGRPCError(err)
	}

	return &authpb.DeleteUserResponse{}, nil
}

// ChangePassword changes a user's password.
func (h *GRPCHandler) ChangePassword(
	ctx context.Context,
	req *authpb.ChangePasswordRequest,
) (*authpb.ChangePasswordResponse, error) {
	// Parse user ID
	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user ID")
	}

	// Check permissions
	currentUserID, _ := getUserIDFromContext(ctx)
	if currentUserID != userID {
		return nil, status.Error(codes.PermissionDenied, "can only change own password")
	}

	// Change password
	if err := h.userService.ChangePassword(ctx, userID, req.GetCurrentPassword(), req.GetNewPassword()); err != nil {
		return nil, toGRPCError(err)
	}

	return &authpb.ChangePasswordResponse{}, nil
}

// CheckPermission checks if a user has a specific permission.
func (h *GRPCHandler) CheckPermission(
	ctx context.Context,
	req *authpb.CheckPermissionRequest,
) (*authpb.CheckPermissionResponse, error) {
	// Parse user ID
	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user ID")
	}

	// Get user
	user, err := h.userService.GetUser(ctx, userID)
	if err != nil {
		return nil, toGRPCError(err)
	}

	// Check permission
	allowed := user.HasPermission(req.GetResource(), req.GetAction())

	response := &authpb.CheckPermissionResponse{
		Allowed: allowed,
	}

	if !allowed {
		response.Reason = "user does not have required permission"
	}

	return response, nil
}

// GetUserPermissions gets all permissions for a user.
func (h *GRPCHandler) GetUserPermissions(
	ctx context.Context,
	req *authpb.GetUserPermissionsRequest,
) (*authpb.GetUserPermissionsResponse, error) {
	// Parse user ID
	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user ID")
	}

	// Get user
	user, err := h.userService.GetUser(ctx, userID)
	if err != nil {
		return nil, toGRPCError(err)
	}

	// Collect all permissions
	permissions := make([]*authpb.Permission, 0)
	seen := make(map[string]bool)

	for _, role := range user.Roles {
		for _, perm := range role.Permissions {
			key := perm.Resource + ":" + perm.Action
			if !seen[key] {
				seen[key] = true
				permissions = append(permissions, &authpb.Permission{
					Resource: perm.Resource,
					Action:   perm.Action,
				})
			}
		}
	}

	return &authpb.GetUserPermissionsResponse{
		Permissions: permissions,
	}, nil
}

// Helper functions

func (h *GRPCHandler) requireAdmin(ctx context.Context) error {
	claims := getClaimsFromContext(ctx)
	if claims == nil {
		return status.Error(codes.Unauthenticated, "user not authenticated")
	}

	for _, role := range claims.Roles {
		if role == models.RoleAdmin {
			return nil
		}
	}

	return status.Error(codes.PermissionDenied, "admin access required")
}

func getUserIDFromContext(ctx context.Context) (uuid.UUID, error) {
	claims := getClaimsFromContext(ctx)
	if claims == nil {
		return uuid.Nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	return uuid.Parse(claims.UserID)
}

func getSessionIDFromContext(ctx context.Context) string {
	claims := getClaimsFromContext(ctx)
	if claims == nil {
		return ""
	}
	return claims.SessionID
}

func getClaimsFromContext(ctx context.Context) *auth.CustomClaims {
	if claims, ok := ctx.Value("claims").(*auth.CustomClaims); ok {
		return claims
	}
	return nil
}

func extractMetadataValue(md metadata.MD, keys ...string) string {
	for _, key := range keys {
		if values := md.Get(key); len(values) > 0 {
			return values[0]
		}
	}
	return ""
}

func domainUserToProto(user *models.User) *authpb.User {
	proto := &authpb.User{
		Id:       user.ID.String(),
		Username: user.Username,
		Email:    user.Email,
		Active:   user.IsActive,
		Created:  timestamppb.New(user.CreatedAt),
		Updated:  timestamppb.New(user.UpdatedAt),
	}

	// Set role
	if len(user.Roles) > 0 {
		switch user.Roles[0].Name {
		case models.RoleAdmin:
			proto.Role = commonpb.UserRole_USER_ROLE_ADMIN
		case models.RoleUser:
			proto.Role = commonpb.UserRole_USER_ROLE_USER
		case models.RoleGuest:
			proto.Role = commonpb.UserRole_USER_ROLE_GUEST
		}
	}

	// Set preferences
	proto.Preferences = &authpb.UserPreferences{
		Language:         user.PrefLanguage,
		Theme:            user.PrefTheme,
		DefaultQuality:   user.PrefPreferredQuality,
		SubtitleLanguage: user.PrefSubtitleLanguage,
		AutoPlay:         user.PrefAutoPlayNext,
	}

	// Set last login
	if user.LastLoginAt != nil {
		proto.LastLogin = timestamppb.New(*user.LastLoginAt)
	}

	return proto
}

func protoRoleToString(role commonpb.UserRole) string {
	switch role {
	case commonpb.UserRole_USER_ROLE_ADMIN:
		return models.RoleAdmin
	case commonpb.UserRole_USER_ROLE_USER:
		return models.RoleUser
	case commonpb.UserRole_USER_ROLE_GUEST:
		return models.RoleGuest
	default:
		return models.RoleUser
	}
}

func toGRPCError(err error) error {
	if err == nil {
		return nil
	}

	switch {
	case errors.IsNotFound(err):
		return status.Error(codes.NotFound, err.Error())
	case errors.IsConflict(err):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.IsBadRequest(err):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.IsUnauthorized(err):
		return status.Error(codes.Unauthenticated, err.Error())
	case errors.IsForbidden(err):
		return status.Error(codes.PermissionDenied, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}
