package handler_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/narwhalmedia/narwhal/internal/library/handler"
	"github.com/narwhalmedia/narwhal/pkg/auth"
	librarypb "github.com/narwhalmedia/narwhal/pkg/library/v1"
	"github.com/narwhalmedia/narwhal/pkg/logger"
	"github.com/narwhalmedia/narwhal/test/mocks"
)

type AuthTestSuite struct {
	suite.Suite

	handler     *handler.GRPCHandler
	mockService *mocks.MockLibraryService
}

func (suite *AuthTestSuite) SetupTest() {
	// Create mock service
	suite.mockService = new(mocks.MockLibraryService)

	// Create handler with mock service for auth testing
	suite.handler = handler.NewGRPCHandler(suite.mockService, logger.NewNoop(), nil)
}

func (suite *AuthTestSuite) TestCreateLibrary_NoAuth() {
	// Test without authentication context
	ctx := context.Background()
	req := &librarypb.CreateLibraryRequest{
		Name: "Test Library",
		Path: "/test/path",
	}

	resp, err := suite.handler.CreateLibrary(ctx, req)

	suite.Nil(resp)
	suite.Require().Error(err)

	st, ok := status.FromError(err)
	suite.True(ok)
	suite.Equal(codes.Unauthenticated, st.Code())
	suite.Contains(st.Message(), "user not authenticated")
}

func (suite *AuthTestSuite) TestCreateLibrary_WithAuth() {
	// Test with authentication context
	ctx := context.Background()
	ctx = context.WithValue(ctx, auth.ContextKeyUserID, "test-user-123")
	ctx = context.WithValue(ctx, auth.ContextKeyRoles, []string{"admin"})

	req := &librarypb.CreateLibraryRequest{
		Name: "Test Library",
		Path: "/test/path",
	}

	// Set up mock expectation
	suite.mockService.On("CreateLibrary", ctx, mock.AnythingOfType("*domain.Library")).
		Return(nil)

	// Should succeed with authentication
	resp, err := suite.handler.CreateLibrary(ctx, req)

	// Should succeed
	suite.Require().NoError(err)
	suite.NotNil(resp)
	suite.NotNil(resp.GetLibrary())
	suite.Equal("Test Library", resp.GetLibrary().GetName())
	suite.Equal("/test/path", resp.GetLibrary().GetPath())

	// Verify the service was called
	suite.mockService.AssertExpectations(suite.T())
}

func (suite *AuthTestSuite) TestGetLibrary_NoAuth() {
	ctx := context.Background()
	req := &librarypb.GetLibraryRequest{
		Id: "550e8400-e29b-41d4-a716-446655440000",
	}

	resp, err := suite.handler.GetLibrary(ctx, req)

	suite.Nil(resp)
	suite.Require().Error(err)

	st, ok := status.FromError(err)
	suite.True(ok)
	suite.Equal(codes.Unauthenticated, st.Code())
}

func (suite *AuthTestSuite) TestListLibraries_NoAuth() {
	ctx := context.Background()
	req := &librarypb.ListLibrariesRequest{}

	resp, err := suite.handler.ListLibraries(ctx, req)

	suite.Nil(resp)
	suite.Require().Error(err)

	st, ok := status.FromError(err)
	suite.True(ok)
	suite.Equal(codes.Unauthenticated, st.Code())
}

func (suite *AuthTestSuite) TestUpdateLibrary_NoAuth() {
	ctx := context.Background()
	req := &librarypb.UpdateLibraryRequest{
		Id: "550e8400-e29b-41d4-a716-446655440000",
		Library: &librarypb.Library{
			Name: "Updated",
		},
	}

	resp, err := suite.handler.UpdateLibrary(ctx, req)

	suite.Nil(resp)
	suite.Require().Error(err)

	st, ok := status.FromError(err)
	suite.True(ok)
	suite.Equal(codes.Unauthenticated, st.Code())
}

func (suite *AuthTestSuite) TestDeleteLibrary_NoAuth() {
	ctx := context.Background()
	req := &librarypb.DeleteLibraryRequest{
		Id: "550e8400-e29b-41d4-a716-446655440000",
	}

	resp, err := suite.handler.DeleteLibrary(ctx, req)

	suite.Nil(resp)
	suite.Require().Error(err)

	st, ok := status.FromError(err)
	suite.True(ok)
	suite.Equal(codes.Unauthenticated, st.Code())
}

func TestAuthTestSuite(t *testing.T) {
	suite.Run(t, new(AuthTestSuite))
}
