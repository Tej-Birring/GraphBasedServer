package main

import (
	"encoding/json"
	"github.com/julienschmidt/httprouter"
	"github.com/lestrrat-go/jwx/jwa"
	"github.com/lestrrat-go/jwx/jwt"
	"log"
	"net/http"
)

/*
	AuthProtectedHandleError
	Error bool — Indicates that this is an error response
	TokenValid — If error, indicates that the token is invalid
	TokenExpired — If error, indicates that the token has expired (reason for it being invalid)
	NewToken — This is a new token sent upon every request where authentication is successful, basically replaces existing JWT with one that has an up-to-date/extended expiration time
*/
type AuthProtectedHandleResponse struct {
	IsError             bool
	TokenValid          bool
	TokenExpired        bool
	NewToken            interface{}
	HttpStatusCode      int
	HttpStatusMessage   string
	Reason              string
	UserFriendlyMessage string
}

// log the real & complete reason; return the user-friendly one
func (e *AuthProtectedHandleResponse) Error() string {
	log.Println(e.Reason)
	return e.UserFriendlyMessage
}

func NewAuthProtectedHandleSuccessResponse(data interface{}, tkn *jwt.Token, userMessage string) AuthProtectedHandleResponse {
	// generate new token for the user — TODO handle error
	newTkn, err := GetNewToken(tkn)
	if err != nil {
		return NewAuthProtectedHandleErrorResponse(true, true, http.StatusInternalServerError, err.Error(), "Failed to maintain your session. Please contact us directly to resolve this issue.")
	}
	// send response
	return AuthProtectedHandleResponse{
		false,
		true,
		false,
		string(newTkn),
		http.StatusOK,
		http.StatusText(http.StatusOK),
		"",
		userMessage,
	}
}

func NewAuthProtectedHandleErrorResponse(tokenValid bool, tokenExpired bool, httpStatusCode int, reason string, userMessage string) AuthProtectedHandleResponse {
	return AuthProtectedHandleResponse{
		true,
		tokenValid,
		tokenExpired,
		nil,
		httpStatusCode,
		http.StatusText(httpStatusCode),
		reason,
		userMessage,
	}
}

type AuthProtectedHandleWork func(tkn *jwt.Token, credentials UserQueryCredentials, r *http.Request, p httprouter.Params) AuthProtectedHandleResponse

func NewAuthProtectedHandle(work AuthProtectedHandleWork) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		w.Header().Set("Content-Type", "application/json")

		pubKey, _ := (*_pubSet).Get(0)
		tkn, err := jwt.ParseRequest(r, jwt.WithVerify(jwa.RS512, pubKey)) //jwt.WithValidate(true)
		if err != nil {
			log.Printf("Token verification failed: %s\n", err)
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(NewAuthProtectedHandleErrorResponse(false, false, http.StatusUnauthorized, "Token failed verification with public key.", "Authorisation failed. Your session could not be authenticated."))
			return
		}

		err = jwt.Validate(tkn)
		if err != nil {
			log.Printf("Token validation failed: %s\n", err)
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(NewAuthProtectedHandleErrorResponse(false, err == jwt.ErrTokenExpired(), http.StatusUnauthorized, "Token passed verification with public key BUT failed validation! "+err.Error(), "Authorisation failed. Your session could not be authenticated."))
			return
		}

		err = r.ParseForm()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(NewAuthProtectedHandleErrorResponse(true, true, http.StatusInternalServerError, "Failed to parse form parameters: "+err.Error(), "Something went wrong while we were trying to fulfil your request. Please contact us directly to resolve this issue."))
			return
		}

		workResponse := work(&tkn, GetUserQueryCredentials(&tkn), r, p)
		if workResponse.IsError {
			w.WriteHeader(workResponse.HttpStatusCode)
			json.NewEncoder(w).Encode(workResponse)
		} else {
			json.NewEncoder(w).Encode(workResponse)
		}
	}
}

type UserQueryCredentials struct {
	queryByKey   string
	queryByValue string
}

// GetUserQueryCredentials
// verified phone > verified email > phone > email
func GetUserQueryCredentials(tkn *jwt.Token) UserQueryCredentials {
	phone, _ := (*tkn).Get("phone")
	phoneExists := phone != nil && phone != ""
	email, _ := (*tkn).Get("email")
	emailExists := email != nil && email != ""

	if phoneExists {
		return UserQueryCredentials{"phone", phone.(string)}
	} else if emailExists {
		return UserQueryCredentials{"email", email.(string)}
	} else {
		panic("Shouldn't be here! Neither phone nor email is available for querying the database in behalf of this user!")
	}
}
