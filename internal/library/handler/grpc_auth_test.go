package handler_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	librarypb "github.com/narwhalmedia/narwhal/pkg/library/v1"
	"github.com/narwhalmedia/narwhal/internal/library/handler"
	"github.com/narwhalmedia/narwhal/pkg/auth"
	"github.com/narwhalmedia/narwhal/pkg/logger"
)

type AuthTestSuite struct {
	suite.Suite
	handler *handler.GRPCHandler
}

func (suite *AuthTestSuite) SetupTest() {
	// Create handler with nil service for auth testing
	suite.handler = handler.NewGRPCHandler(nil, logger.NewNoop(), nil)
}

func (suite *AuthTestSuite) TestCreateLibrary_NoAuth() {
	// Test without authentication context
	ctx := context.Background()
	req := &librarypb.CreateLibraryRequest{
		Name: "Test Library",
		Path: "/test/path",
	}
	
	resp, err := suite.handler.CreateLibrary(ctx, req)
	
	assert.Nil(suite.T(), resp)
	assert.Error(suite.T(), err)
	
	st, ok := status.FromError(err)
	assert.True(suite.T(), ok)
	assert.Equal(suite.T(), codes.Unauthenticated, st.Code())
	assert.Contains(suite.T(), st.Message(), "user not authenticated")
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
	
	// This will fail at the service layer since service is nil,
	// but it should pass authentication
	resp, err := suite.handler.CreateLibrary(ctx, req)
	
	// Should panic or error differently, not auth error
	assert.Nil(suite.T(), resp)
	assert.Error(suite.T(), err)
	
	st, ok := status.FromError(err)
	if ok {
		assert.NotEqual(suite.T(), codes.Unauthenticated, st.Code())
	}
}

func (suite *AuthTestSuite) TestGetLibrary_NoAuth() {
	ctx := context.Background()
	req := &librarypb.GetLibraryRequest{
		Id: "550e8400-e29b-41d4-a716-446655440000",
	}
	
	resp, err := suite.handler.GetLibrary(ctx, req)
	
	assert.Nil(suite.T(), resp)
	assert.Error(suite.T(), err)
	
	st, ok := status.FromError(err)
	assert.True(suite.T(), ok)
	assert.Equal(suite.T(), codes.Unauthenticated, st.Code())
}

func (suite *AuthTestSuite) TestListLibraries_NoAuth() {
	ctx := context.Background()
	req := &librarypb.ListLibrariesRequest{}
	
	resp, err := suite.handler.ListLibraries(ctx, req)
	
	assert.Nil(suite.T(), resp)
	assert.Error(suite.T(), err)
	
	st, ok := status.FromError(err)
	assert.True(suite.T(), ok)
	assert.Equal(suite.T(), codes.Unauthenticated, st.Code())
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
	
	assert.Nil(suite.T(), resp)
	assert.Error(suite.T(), err)
	
	st, ok := status.FromError(err)
	assert.True(suite.T(), ok)
	assert.Equal(suite.T(), codes.Unauthenticated, st.Code())
}

func (suite *AuthTestSuite) TestDeleteLibrary_NoAuth() {
	ctx := context.Background()
	req := &librarypb.DeleteLibraryRequest{
		Id: "550e8400-e29b-41d4-a716-446655440000",
	}
	
	resp, err := suite.handler.DeleteLibrary(ctx, req)
	
	assert.Nil(suite.T(), resp)
	assert.Error(suite.T(), err)
	
	st, ok := status.FromError(err)
	assert.True(suite.T(), ok)
	assert.Equal(suite.T(), codes.Unauthenticated, st.Code())
}

func TestAuthTestSuite(t *testing.T) {
	suite.Run(t, new(AuthTestSuite))
}