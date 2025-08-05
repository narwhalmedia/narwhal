package handler_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	commonpb "github.com/narwhalmedia/narwhal/pkg/common/v1"
	librarypb "github.com/narwhalmedia/narwhal/pkg/library/v1"
	"github.com/narwhalmedia/narwhal/internal/library/domain"
	"github.com/narwhalmedia/narwhal/internal/library/handler"
	"github.com/narwhalmedia/narwhal/pkg/auth"
	"github.com/narwhalmedia/narwhal/pkg/errors"
	"github.com/narwhalmedia/narwhal/pkg/logger"
	"github.com/narwhalmedia/narwhal/test/mocks"
)

// Note: To generate the mock, run:
// mockgen -source=internal/library/service/interfaces.go -destination=test/mocks/library_service_mock.go -package=mocks LibraryServiceInterface

type GRPCHandlerTestSuite struct {
	suite.Suite
	ctx            context.Context
	mockService    *mocks.MockLibraryService
	handler        *handler.GRPCHandler
	testLibraryID  uuid.UUID
	testMediaID    uuid.UUID
}

func (suite *GRPCHandlerTestSuite) SetupTest() {
	suite.ctx = context.Background()
	// Add authentication context
	suite.ctx = context.WithValue(suite.ctx, auth.ContextKeyUserID, "test-user-123")
	suite.ctx = context.WithValue(suite.ctx, auth.ContextKeyRoles, []string{"admin"})
	
	suite.mockService = new(mocks.MockLibraryService)
	
	// Create handler with mock service
	suite.handler = handler.NewGRPCHandler(
		suite.mockService,
		logger.NewNoop(),
		nil, // No pagination encoder for tests
	)
	suite.testLibraryID = uuid.New()
	suite.testMediaID = uuid.New()
}

func (suite *GRPCHandlerTestSuite) TearDownTest() {
	suite.mockService.AssertExpectations(suite.T())
}

func (suite *GRPCHandlerTestSuite) TestGetLibrary_Success() {
	// Arrange
	library := &domain.Library{
		ID:        suite.testLibraryID,
		Name:      "Test Library",
		Path:      "/test/path",
		Type:      "movie",
		Enabled:   true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	
	suite.mockService.On("GetLibrary", suite.ctx, suite.testLibraryID).Return(library, nil)
	
	// Act
	req := &librarypb.GetLibraryRequest{Id: suite.testLibraryID.String()}
	resp, err := suite.handler.GetLibrary(suite.ctx, req)
	
	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), resp)
	assert.Equal(suite.T(), library.ID.String(), resp.Id)
	assert.Equal(suite.T(), library.Name, resp.Name)
	assert.Equal(suite.T(), library.Path, resp.Path)
}

func (suite *GRPCHandlerTestSuite) TestGetLibrary_NotFound() {
	// Arrange
	suite.mockService.On("GetLibrary", suite.ctx, suite.testLibraryID).
		Return(nil, errors.NotFound("library not found"))
	
	// Act
	req := &librarypb.GetLibraryRequest{Id: suite.testLibraryID.String()}
	resp, err := suite.handler.GetLibrary(suite.ctx, req)
	
	// Assert
	assert.Nil(suite.T(), resp)
	assert.Error(suite.T(), err)
	st, ok := status.FromError(err)
	assert.True(suite.T(), ok)
	assert.Equal(suite.T(), codes.NotFound, st.Code())
}

func (suite *GRPCHandlerTestSuite) TestGetLibrary_InvalidID() {
	// Act
	req := &librarypb.GetLibraryRequest{Id: "invalid-uuid"}
	resp, err := suite.handler.GetLibrary(suite.ctx, req)
	
	// Assert
	assert.Nil(suite.T(), resp)
	assert.Error(suite.T(), err)
	st, ok := status.FromError(err)
	assert.True(suite.T(), ok)
	assert.Equal(suite.T(), codes.InvalidArgument, st.Code())
}

func (suite *GRPCHandlerTestSuite) TestListLibraries_Success() {
	// Arrange
	libraries := []*domain.Library{
		{
			ID:        uuid.New(),
			Name:      "Movies",
			Path:      "/media/movies",
			Type:      "movie",
			Enabled:   true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:        uuid.New(),
			Name:      "TV Shows",
			Path:      "/media/tv",
			Type:      "tv_show",
			Enabled:   true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}
	
	suite.mockService.On("ListLibraries", suite.ctx, (*bool)(nil)).Return(libraries, nil)
	
	// Act
	req := &librarypb.ListLibrariesRequest{
		Pagination: &commonpb.PaginationRequest{
			PageSize: 10,
		},
	}
	resp, err := suite.handler.ListLibraries(suite.ctx, req)
	
	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), resp)
	assert.Len(suite.T(), resp.Libraries, 2)
	assert.Equal(suite.T(), "Movies", resp.Libraries[0].Name)
	assert.Equal(suite.T(), "TV Shows", resp.Libraries[1].Name)
}

func (suite *GRPCHandlerTestSuite) TestUpdateLibrary_Success() {
	// Arrange
	updatedLibrary := &domain.Library{
		ID:        suite.testLibraryID,
		Name:      "Updated Name",
		Path:      "/test/path",
		Type:      "movie",
		Enabled:   false,
		CreatedAt: time.Now().Add(-24 * time.Hour),
		UpdatedAt: time.Now(),
	}
	
	suite.mockService.On("UpdateLibrary", suite.ctx, suite.testLibraryID, mock.MatchedBy(func(u map[string]interface{}) bool {
		return u["name"] == "Updated Name" && u["enabled"] == false
	})).Return(updatedLibrary, nil)
	
	// Act
	req := &librarypb.UpdateLibraryRequest{
		Id: suite.testLibraryID.String(),
		Library: &librarypb.Library{
			Name:     "Updated Name",
			AutoScan: false,
		},
	}
	resp, err := suite.handler.UpdateLibrary(suite.ctx, req)
	
	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), resp)
	assert.Equal(suite.T(), "Updated Name", resp.Name)
	assert.False(suite.T(), resp.AutoScan)
}

func (suite *GRPCHandlerTestSuite) TestDeleteLibrary_Success() {
	// Arrange
	suite.mockService.On("DeleteLibrary", suite.ctx, suite.testLibraryID).Return(nil)
	
	// Act
	req := &librarypb.DeleteLibraryRequest{Id: suite.testLibraryID.String()}
	resp, err := suite.handler.DeleteLibrary(suite.ctx, req)
	
	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), resp)
}

func (suite *GRPCHandlerTestSuite) TestDeleteLibrary_NotFound() {
	// Arrange
	suite.mockService.On("DeleteLibrary", suite.ctx, suite.testLibraryID).
		Return(errors.NotFound("library not found"))
	
	// Act
	req := &librarypb.DeleteLibraryRequest{Id: suite.testLibraryID.String()}
	resp, err := suite.handler.DeleteLibrary(suite.ctx, req)
	
	// Assert
	assert.Nil(suite.T(), resp)
	assert.Error(suite.T(), err)
	st, ok := status.FromError(err)
	assert.True(suite.T(), ok)
	assert.Equal(suite.T(), codes.NotFound, st.Code())
}

func TestGRPCHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(GRPCHandlerTestSuite))
}