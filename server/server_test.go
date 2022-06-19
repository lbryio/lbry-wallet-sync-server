package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"orblivion/lbry-id/auth"
	"orblivion/lbry-id/wallet"
	"strings"
	"testing"
)

// Implementing interfaces for stubbed out packages

type TestAuth struct {
	TestToken    auth.TokenString
	FailGenToken bool
}

func (a *TestAuth) NewToken(userId auth.UserId, deviceId auth.DeviceId, scope auth.AuthScope) (*auth.AuthToken, error) {
	if a.FailGenToken {
		return nil, fmt.Errorf("Test error: fail to generate token")
	}
	return &auth.AuthToken{Token: a.TestToken, UserId: userId, DeviceId: deviceId, Scope: scope}, nil
}

type TestStoreFunctionsCalled struct {
	SaveToken     bool
	GetToken      bool
	GetUserId     bool
	CreateAccount bool
	SetWallet     bool
	GetWallet     bool
}

type TestStoreFunctionsErrors struct {
	SaveToken     error
	GetToken      error
	GetUserId     error
	CreateAccount error
	SetWallet     error
	GetWallet     error
}

type TestStore struct {
	// Fake store functions will set these to `true` as they are called
	Called TestStoreFunctionsCalled

	// Fake store functions will return the errors (including `nil`) specified in
	// the test setup
	Errors TestStoreFunctionsErrors
}

func (s *TestStore) SaveToken(token *auth.AuthToken) error {
	s.Called.SaveToken = true
	return s.Errors.SaveToken
}

func (s *TestStore) GetToken(auth.TokenString) (*auth.AuthToken, error) {
	s.Called.GetToken = true
	return nil, s.Errors.GetToken
}

func (s *TestStore) GetUserId(auth.Email, auth.Password) (auth.UserId, error) {
	s.Called.GetUserId = true
	return 0, s.Errors.GetUserId
}

func (s *TestStore) CreateAccount(auth.Email, auth.Password) error {
	s.Called.CreateAccount = true
	return s.Errors.CreateAccount
}

func (s *TestStore) SetWallet(
	UserId auth.UserId,
	encryptedWallet wallet.EncryptedWallet,
	sequence wallet.Sequence,
	hmac wallet.WalletHmac,
) (latestEncryptedWallet wallet.EncryptedWallet, latestSequence wallet.Sequence, latestHmac wallet.WalletHmac, sequenceCorrect bool, err error) {
	s.Called.SetWallet = true
	err = s.Errors.SetWallet
	return
}

func (s *TestStore) GetWallet(userId auth.UserId) (encryptedWallet wallet.EncryptedWallet, sequence wallet.Sequence, hmac wallet.WalletHmac, err error) {
	s.Called.GetWallet = true
	err = s.Errors.GetWallet
	return
}

func TestServerHelperCheckAuthSuccess(t *testing.T) {
	t.Fatalf("Test me: checkAuth success")
}

func TestServerHelperCheckAuthErrors(t *testing.T) {
	t.Fatalf("Test me: checkAuth failure")
}

func TestServerHelperGetGetDataSuccess(t *testing.T) {
	t.Fatalf("Test me: getGetData success")
}
func TestServerHelperGetGetDataErrors(t *testing.T) {
	t.Fatalf("Test me: getGetData failure")
}

type TestReqStruct struct{ key string }

func (t *TestReqStruct) validate() bool { return t.key != "" }

func TestServerHelperGetPostDataSuccess(t *testing.T) {
	requestBody := []byte(`{}`)
	req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewBuffer(requestBody))
	w := httptest.NewRecorder()
	success := getPostData(w, req, &TestReqStruct{key: "hi"})
	if !success {
		t.Errorf("getPostData failed unexpectedly")
	}
}

func TestServerHelperGetPostDataErrors(t *testing.T) {
	tt := []struct {
		name                string
		method              string
		requestBody         string
		expectedStatusCode  int
		expectedErrorString string
	}{
		{
			name:                "bad method",
			method:              http.MethodGet,
			requestBody:         "",
			expectedStatusCode:  http.StatusMethodNotAllowed,
			expectedErrorString: http.StatusText(http.StatusMethodNotAllowed),
		},
		{
			name:                "request body too large",
			method:              http.MethodPost,
			requestBody:         fmt.Sprintf(`{"key": "%s"}`, strings.Repeat("a", 10000)),
			expectedStatusCode:  http.StatusRequestEntityTooLarge,
			expectedErrorString: http.StatusText(http.StatusRequestEntityTooLarge),
		},
		{
			name:                "malformed request body JSON",
			method:              http.MethodPost,
			requestBody:         "{",
			expectedStatusCode:  http.StatusBadRequest,
			expectedErrorString: http.StatusText(http.StatusBadRequest) + ": Request body JSON malformed or structure mismatch",
		},
		{
			name:                "body JSON failed validation",
			method:              http.MethodPost,
			requestBody:         "{}",
			expectedStatusCode:  http.StatusBadRequest,
			expectedErrorString: http.StatusText(http.StatusBadRequest) + ": Request failed validation",
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			// Make request
			req := httptest.NewRequest(tc.method, PathAuthToken, bytes.NewBuffer([]byte(tc.requestBody)))
			w := httptest.NewRecorder()

			success := getPostData(w, req, &TestReqStruct{})
			if success {
				t.Errorf("getPostData succeeded unexpectedly")
			}

			if want, got := tc.expectedStatusCode, w.Result().StatusCode; want != got {
				t.Errorf("StatusCode: expected %d, got %d", want, got)
			}

			body, _ := ioutil.ReadAll(w.Body)

			var result ErrorResponse
			if err := json.Unmarshal(body, &result); err != nil {
				t.Fatalf("Error decoding error message %s: `%s`", err, body)
			}

			if want, got := tc.expectedErrorString, result.Error; want != got {
				t.Errorf("Error String: expected %s, got %s", want, got)
			}
		})
	}
}

func TestServerHelperRequestOverheadSuccess(t *testing.T) {
	t.Fatalf("Test me: requestOverhead success")
}
func TestServerHelperRequestOverheadErrors(t *testing.T) {
	t.Fatalf("Test me: requestOverhead failures")
}
