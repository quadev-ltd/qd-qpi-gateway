package authentication

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"github.com/golang/mock/gomock"
	jwtCommonMock "github.com/quadev-ltd/qd-common/pkg/jwt/mock"
	loggerCommon "github.com/quadev-ltd/qd-common/pkg/log"
	loggerCommonMock "github.com/quadev-ltd/qd-common/pkg/log/mock"
	"github.com/stretchr/testify/assert"

	"github.com/quadev-ltd/qd-qpi-gateway/internal/authentication/mock"
)

func createTestContext(method, path string, body []byte, authHeader *string) (*gin.Context, *httptest.ResponseRecorder) {
	// Initialize Gin engine
	gin.SetMode(gin.TestMode)

	// Create a request
	req := httptest.NewRequest(method, path, bytes.NewBuffer(body))

	// Create a response recorder
	w := httptest.NewRecorder()

	// Create the context
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	// You can add additional setup here, such as setting headers
	c.Request.Header.Set("Content-Type", "application/json")
	if authHeader != nil {
		c.Request.Header.Set("Authorization", *authHeader)
	}

	return c, w
}

func createTestContextWithLogger(logger loggerCommon.Loggerer, authHeader *string) (*gin.Context, *httptest.ResponseRecorder) {
	ctx, w := createTestContext("GET", "/test", nil, authHeader)
	newCtx := context.WithValue(ctx.Request.Context(), loggerCommon.LoggerKey, logger)
	newReq := ctx.Request.WithContext(newCtx)
	ctx.Request = newReq
	return ctx, w
}

func TestMiddleware(t *testing.T) {
	// RequestPublicKey
	t.Run("Request_Public_Key_Error", func(t *testing.T) {
		controller := gomock.NewController(t)
		defer controller.Finish()
		serviceMock := mock.NewMockServiceClienter(controller)
		errorExample := errors.New("example error")
		correlationID := "example-correlation-id"

		serviceMock.EXPECT().GetPublicKey(gomock.Any()).Return(nil, errorExample)

		publicKey, err := RequestPublicKey(serviceMock, correlationID)

		assert.Error(t, err)
		assert.Nil(t, publicKey)
		assert.Equal(t, "Could not obtain public key: example error", err.Error())
	})

	t.Run("Request_Public_Key_Success", func(t *testing.T) {
		controller := gomock.NewController(t)
		defer controller.Finish()
		serviceMock := mock.NewMockServiceClienter(controller)
		correlationID := "example-correlation-id"
		publicKeyExample := "example-key"

		serviceMock.EXPECT().GetPublicKey(gomock.Any()).Return(&publicKeyExample, nil)

		publicKey, err := RequestPublicKey(serviceMock, correlationID)

		assert.Nil(t, err)
		assert.Equal(t, *publicKey, publicKeyExample)
	})

	// RequireAuthentication
	t.Run("RequireAuthentication_No_Logger_Error", func(t *testing.T) {
		controller := gomock.NewController(t)
		defer controller.Finish()
		serviceMock := mock.NewMockServiceClienter(controller)
		jwtVerifierMock := jwtCommonMock.NewMockTokenVerifierer(controller)
		jwtTokenInspectorMock := jwtCommonMock.NewMockTokenInspectorer(controller)
		authenticationMiddleware := &AutheticationMiddleware{
			serviceMock,
			jwtVerifierMock,
			jwtTokenInspectorMock,
		}
		ctx, w := createTestContext("GET", "/test", nil, nil)

		authenticationMiddleware.RequireAuthentication(ctx)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("RequireAuthentication_No_Authorization_Header_Error", func(t *testing.T) {
		controller := gomock.NewController(t)
		defer controller.Finish()
		serviceMock := mock.NewMockServiceClienter(controller)
		jwtVerifierMock := jwtCommonMock.NewMockTokenVerifierer(controller)
		jwtTokenInspectorMock := jwtCommonMock.NewMockTokenInspectorer(controller)
		authenticationMiddleware := &AutheticationMiddleware{
			serviceMock,
			jwtVerifierMock,
			jwtTokenInspectorMock,
		}
		loggerMock := loggerCommonMock.NewMockLoggerer(controller)

		ctx, w := createTestContextWithLogger(loggerMock, nil)

		loggerMock.EXPECT().Error(nil, "No authorization header was present in the request")

		authenticationMiddleware.RequireAuthentication(ctx)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("RequireAuthentication_Wrong_Authorization_Header_Error", func(t *testing.T) {
		controller := gomock.NewController(t)
		defer controller.Finish()
		serviceMock := mock.NewMockServiceClienter(controller)
		jwtVerifierMock := jwtCommonMock.NewMockTokenVerifierer(controller)
		jwtTokenInspectorMock := jwtCommonMock.NewMockTokenInspectorer(controller)
		authenticationMiddleware := &AutheticationMiddleware{
			serviceMock,
			jwtVerifierMock,
			jwtTokenInspectorMock,
		}
		loggerMock := loggerCommonMock.NewMockLoggerer(controller)

		authHeader := "test-header"
		ctx, w := createTestContextWithLogger(loggerMock, &authHeader)

		loggerMock.EXPECT().Error(nil, "No bearer token was present in the authorization header")

		authenticationMiddleware.RequireAuthentication(ctx)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("RequireAuthentication_Empty_Authorization_Header_Error", func(t *testing.T) {
		controller := gomock.NewController(t)
		defer controller.Finish()
		serviceMock := mock.NewMockServiceClienter(controller)
		jwtVerifierMock := jwtCommonMock.NewMockTokenVerifierer(controller)
		jwtTokenInspectorMock := jwtCommonMock.NewMockTokenInspectorer(controller)
		authenticationMiddleware := &AutheticationMiddleware{
			serviceMock,
			jwtVerifierMock,
			jwtTokenInspectorMock,
		}
		loggerMock := loggerCommonMock.NewMockLoggerer(controller)

		authHeader := "Bearer"
		ctx, w := createTestContextWithLogger(loggerMock, &authHeader)

		loggerMock.EXPECT().Error(nil, "No bearer token was present in the authorization header")

		authenticationMiddleware.RequireAuthentication(ctx)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("RequireAuthentication_Invalid_Authorization_Header_Error", func(t *testing.T) {
		controller := gomock.NewController(t)
		defer controller.Finish()
		serviceMock := mock.NewMockServiceClienter(controller)
		jwtVerifierMock := jwtCommonMock.NewMockTokenVerifierer(controller)
		jwtTokenInspectorMock := jwtCommonMock.NewMockTokenInspectorer(controller)
		authenticationMiddleware := &AutheticationMiddleware{
			serviceMock,
			jwtVerifierMock,
			jwtTokenInspectorMock,
		}
		loggerMock := loggerCommonMock.NewMockLoggerer(controller)

		exampleError := errors.New("example error")
		authHeader := "Bearer invalid-header"
		ctx, w := createTestContextWithLogger(loggerMock, &authHeader)

		loggerMock.EXPECT().Error(exampleError, "The bearer token was invalid")
		jwtVerifierMock.EXPECT().Verify("invalid-header").Return(nil, exampleError)

		authenticationMiddleware.RequireAuthentication(ctx)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("RequireAuthentication_Type_Claim_Authorization_Header_Error", func(t *testing.T) {
		controller := gomock.NewController(t)
		defer controller.Finish()
		serviceMock := mock.NewMockServiceClienter(controller)
		jwtVerifierMock := jwtCommonMock.NewMockTokenVerifierer(controller)
		jwtTokenInspectorMock := jwtCommonMock.NewMockTokenInspectorer(controller)
		authenticationMiddleware := &AutheticationMiddleware{
			serviceMock,
			jwtVerifierMock,
			jwtTokenInspectorMock,
		}
		loggerMock := loggerCommonMock.NewMockLoggerer(controller)

		exampleError := errors.New("example error")
		authHeader := "Bearer test-header"
		testToken := jwt.Token{}

		ctx, w := createTestContextWithLogger(loggerMock, &authHeader)

		loggerMock.EXPECT().Error(exampleError, "Could not obtain type from bearer token")
		jwtVerifierMock.EXPECT().Verify("test-header").Return(&testToken, nil)
		jwtTokenInspectorMock.EXPECT().GetTypeFromToken(&testToken).Return(nil, exampleError)

		authenticationMiddleware.RequireAuthentication(ctx)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("RequireAuthentication_Wrong_Type_Claim_Authorization_Header_Error", func(t *testing.T) {
		controller := gomock.NewController(t)
		defer controller.Finish()
		serviceMock := mock.NewMockServiceClienter(controller)
		jwtVerifierMock := jwtCommonMock.NewMockTokenVerifierer(controller)
		jwtTokenInspectorMock := jwtCommonMock.NewMockTokenInspectorer(controller)
		authenticationMiddleware := &AutheticationMiddleware{
			serviceMock,
			jwtVerifierMock,
			jwtTokenInspectorMock,
		}
		loggerMock := loggerCommonMock.NewMockLoggerer(controller)

		authHeader := "Bearer test-header"
		testToken := jwt.Token{}
		tokenTypeValue := "invalid-type"

		ctx, w := createTestContextWithLogger(loggerMock, &authHeader)

		loggerMock.EXPECT().Error(nil, "The bearer token was not an AccessTokenType")
		jwtVerifierMock.EXPECT().Verify("test-header").Return(&testToken, nil)
		jwtTokenInspectorMock.EXPECT().GetTypeFromToken(&testToken).Return(&tokenTypeValue, nil)

		authenticationMiddleware.RequireAuthentication(ctx)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("RequireAuthentication_Email_Claim_Authorization_Header_Error", func(t *testing.T) {
		controller := gomock.NewController(t)
		defer controller.Finish()
		serviceMock := mock.NewMockServiceClienter(controller)
		jwtVerifierMock := jwtCommonMock.NewMockTokenVerifierer(controller)
		jwtTokenInspectorMock := jwtCommonMock.NewMockTokenInspectorer(controller)
		authenticationMiddleware := &AutheticationMiddleware{
			serviceMock,
			jwtVerifierMock,
			jwtTokenInspectorMock,
		}
		loggerMock := loggerCommonMock.NewMockLoggerer(controller)

		exampleError := errors.New("example error")
		authHeader := "Bearer test-header"
		testToken := jwt.Token{}
		tokenTypeValue := "AccessTokenType"

		ctx, w := createTestContextWithLogger(loggerMock, &authHeader)

		loggerMock.EXPECT().Error(exampleError, "Could not obtain email from bearer token")
		jwtVerifierMock.EXPECT().Verify("test-header").Return(&testToken, nil)
		jwtTokenInspectorMock.EXPECT().GetTypeFromToken(&testToken).Return(&tokenTypeValue, nil)
		jwtTokenInspectorMock.EXPECT().GetEmailFromToken(&testToken).Return(nil, exampleError)

		authenticationMiddleware.RequireAuthentication(ctx)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("RequireAuthentication_Expiry_Claim_Authorization_Header_Error", func(t *testing.T) {
		controller := gomock.NewController(t)
		defer controller.Finish()
		serviceMock := mock.NewMockServiceClienter(controller)
		jwtVerifierMock := jwtCommonMock.NewMockTokenVerifierer(controller)
		jwtTokenInspectorMock := jwtCommonMock.NewMockTokenInspectorer(controller)
		authenticationMiddleware := &AutheticationMiddleware{
			serviceMock,
			jwtVerifierMock,
			jwtTokenInspectorMock,
		}
		loggerMock := loggerCommonMock.NewMockLoggerer(controller)

		exampleError := errors.New("example error")
		authHeader := "Bearer test-header"
		testToken := jwt.Token{}
		tokenTypeValue := "AccessTokenType"
		testEmail := "test@email.com"

		ctx, w := createTestContextWithLogger(loggerMock, &authHeader)

		loggerMock.EXPECT().Error(exampleError, "Could not obtain expiry from bearer token")
		jwtVerifierMock.EXPECT().Verify("test-header").Return(&testToken, nil)
		jwtTokenInspectorMock.EXPECT().GetTypeFromToken(&testToken).Return(&tokenTypeValue, nil)
		jwtTokenInspectorMock.EXPECT().GetEmailFromToken(&testToken).Return(&testEmail, nil)
		jwtTokenInspectorMock.EXPECT().GetExpiryFromToken(&testToken).Return(nil, exampleError)

		authenticationMiddleware.RequireAuthentication(ctx)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("RequireAuthentication_Wrong_Expiry_Claim_Authorization_Header_Error", func(t *testing.T) {
		controller := gomock.NewController(t)
		defer controller.Finish()
		serviceMock := mock.NewMockServiceClienter(controller)
		jwtVerifierMock := jwtCommonMock.NewMockTokenVerifierer(controller)
		jwtTokenInspectorMock := jwtCommonMock.NewMockTokenInspectorer(controller)
		authenticationMiddleware := &AutheticationMiddleware{
			serviceMock,
			jwtVerifierMock,
			jwtTokenInspectorMock,
		}
		loggerMock := loggerCommonMock.NewMockLoggerer(controller)

		authHeader := "Bearer test-header"
		testToken := jwt.Token{}
		tokenTypeValue := "AccessTokenType"
		testEmail := "test@email.com"
		testExpiry := time.Now().Add(-1 * time.Second)

		ctx, w := createTestContextWithLogger(loggerMock, &authHeader)

		loggerMock.EXPECT().Error(nil, "The bearer token has expired")
		jwtVerifierMock.EXPECT().Verify("test-header").Return(&testToken, nil)
		jwtTokenInspectorMock.EXPECT().GetTypeFromToken(&testToken).Return(&tokenTypeValue, nil)
		jwtTokenInspectorMock.EXPECT().GetEmailFromToken(&testToken).Return(&testEmail, nil)
		jwtTokenInspectorMock.EXPECT().GetExpiryFromToken(&testToken).Return(&testExpiry, nil)

		authenticationMiddleware.RequireAuthentication(ctx)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("RequireAuthentication_Wrong_Expiry_Claim_Authorization_Header_Error", func(t *testing.T) {
		controller := gomock.NewController(t)
		defer controller.Finish()
		serviceMock := mock.NewMockServiceClienter(controller)
		jwtVerifierMock := jwtCommonMock.NewMockTokenVerifierer(controller)
		jwtTokenInspectorMock := jwtCommonMock.NewMockTokenInspectorer(controller)
		authenticationMiddleware := &AutheticationMiddleware{
			serviceMock,
			jwtVerifierMock,
			jwtTokenInspectorMock,
		}
		loggerMock := loggerCommonMock.NewMockLoggerer(controller)

		authHeader := "Bearer test-header"
		testToken := jwt.Token{}
		tokenTypeValue := "AccessTokenType"
		testEmail := "test@email.com"
		testExpiry := time.Now().Add(1 * time.Second)

		ctx, w := createTestContextWithLogger(loggerMock, &authHeader)

		jwtVerifierMock.EXPECT().Verify("test-header").Return(&testToken, nil)
		jwtTokenInspectorMock.EXPECT().GetTypeFromToken(&testToken).Return(&tokenTypeValue, nil)
		jwtTokenInspectorMock.EXPECT().GetEmailFromToken(&testToken).Return(&testEmail, nil)
		jwtTokenInspectorMock.EXPECT().GetExpiryFromToken(&testToken).Return(&testExpiry, nil)
		loggerMock.EXPECT().Info("Successfully authenticated user")

		authenticationMiddleware.RequireAuthentication(ctx)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	// Refresh Authentication
	t.Run("RefreshAuthentication_Wrong_Type_Claim_Authorization_Header_Error", func(t *testing.T) {
		controller := gomock.NewController(t)
		defer controller.Finish()
		serviceMock := mock.NewMockServiceClienter(controller)
		jwtVerifierMock := jwtCommonMock.NewMockTokenVerifierer(controller)
		jwtTokenInspectorMock := jwtCommonMock.NewMockTokenInspectorer(controller)
		authenticationMiddleware := &AutheticationMiddleware{
			serviceMock,
			jwtVerifierMock,
			jwtTokenInspectorMock,
		}
		loggerMock := loggerCommonMock.NewMockLoggerer(controller)

		authHeader := "Bearer test-header"
		testToken := jwt.Token{}
		tokenTypeValue := "invalid-type"

		ctx, w := createTestContextWithLogger(loggerMock, &authHeader)

		loggerMock.EXPECT().Error(nil, "The bearer token was not an RefreshTokenType")
		jwtVerifierMock.EXPECT().Verify("test-header").Return(&testToken, nil)
		jwtTokenInspectorMock.EXPECT().GetTypeFromToken(&testToken).Return(&tokenTypeValue, nil)

		authenticationMiddleware.RefreshAuthentication(ctx)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}
