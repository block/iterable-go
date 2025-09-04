package api

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"iterable-go/errors"
	"iterable-go/logger"
	"iterable-go/types"
)

const (
	pathUsersGetByEmail              = "users/getByEmail?{query}"
	pathUsersDeleteByEmail           = "users/{email}"
	pathUsersUpdate                  = "users/update"
	pathUsersBulkUpdate              = "users/bulkUpdate"
	pathUsersById                    = "users/byUserId/{userId}"
	pathUsersByIdQuery               = "users/byUserId?userId={userId}"
	pathUsersRegisterBrowser         = "users/registerBrowserToken"
	pathUsersUpdateSubscriptions     = "users/updateSubscriptions"
	pathUsersBulkUpdateSubscriptions = "users/bulkUpdateSubscriptions"
	pathUsersUpdateEmail             = "users/updateEmail"
	pathUsersGetFields               = "users/getFields"
	pathUsersGetSentMessages         = "users/getSentMessages?{query}"
	pathUsersForgotten               = "users/forgotten"
	pathUsersForgottenIds            = "users/forgottenUserIds"
	pathUsersForget                  = "users/forget"
	pathUsersUnForget                = "users/unforget"
	pathUsersInvalidate              = "auth/jwts/invalidate"
	pathUsersRegisterDevice          = "users/registerDeviceToken"
	pathUsersDisableDevice           = "users/disableDevice"
)

// Users implements a set of /api/users API methods,
// See: https://api.iterable.com/api/docs#messageTypes_messageTypes
type Users struct {
	api *apiClient
}

func NewUsersApi(apiKey string, httpClient *http.Client, logger logger.Logger) *Users {
	return &Users{
		api: newApiClient(apiKey, httpClient, logger),
	}
}

func (u *Users) UpdateOrCreate(user types.UserRequest) (*types.PostResponse, error) {
	var res types.PostResponse
	return toNilErr(&res, u.api.postJson(
		pathUsersUpdate, user, &res,
	))
}

func (u *Users) BulkUpdate(update types.BulkUpdateRequest) (*types.BulkUpdateResponse, error) {
	var res types.BulkUpdateResponse
	return toNilErr(&res, u.api.postJson(
		pathUsersBulkUpdate, update, &res,
	))
}

func (u *Users) DeleteById(userId string) (*types.PostResponse, error) {
	var res types.PostResponse
	return toNilErr(&res, u.api.deleteJson(
		strings.Replace(pathUsersById, "{userId}", userId, 1),
		nil,
		&res,
	))
}

func (u *Users) GetById(userId string) (*types.User, bool, error) {
	return u.getUser(strings.Replace(pathUsersByIdQuery, "{userId}", userId, 1))
}

func (u *Users) GetByEmail(email string) (*types.User, bool, error) {
	emailValue := url.Values{}
	emailValue.Add("email", email)
	return u.getUser(strings.Replace(pathUsersGetByEmail, "{query}", emailValue.Encode(), 1))
}

func (u *Users) getUser(path string) (*types.User, bool, error) {
	var res types.UserResponse
	err := u.api.getJson(path, &res)
	if err != nil {
		if err.IterableCode == errors.ITERABLE_NoUserWithIdExists {
			return nil, false, nil
		}
		return nil, false, err
	}
	if res.User.UserId == "" && res.User.Email == "" {
		return nil, false, nil
	}
	return &res.User, true, nil
}

func (u *Users) DeleteByEmail(email string) (*types.PostResponse, error) {
	var res types.PostResponse
	return toNilErr(&res, u.api.deleteJson(
		strings.Replace(pathUsersDeleteByEmail, "{email}", email, 1),
		nil,
		&res,
	))
}

func (u *Users) UpdateSubscriptions(update types.UserUpdateSubscriptionsRequest) (*types.PostResponse, error) {
	var res types.PostResponse
	return toNilErr(&res, u.api.postJson(
		pathUsersUpdateSubscriptions, update, &res,
	))
}

func (u *Users) BulkUpdateSubscriptions(update types.BulkUserUpdateSubscriptionsRequest) (
	*types.BulkUserUpdateSubscriptionsResponse, error) {
	var res types.BulkUserUpdateSubscriptionsResponse
	return toNilErr(&res, u.api.postJson(
		pathUsersBulkUpdateSubscriptions, update, &res,
	))
}

func (u *Users) UpdateEmail(
	email string,
	userId string,
	newEmail string,
) (*types.PostResponse, error) {
	req := types.UserUpdateEmailRequest{
		Email:    email,
		UserId:   userId,
		NewEmail: newEmail,
	}
	var res types.PostResponse
	return toNilErr(&res, u.api.postJson(
		pathUsersUpdateEmail, req, &res,
	))
}

func (u *Users) GetAllFields() (map[string]string, error) {
	var res types.UserFieldsResponse
	return toNilErr(res.Fields, u.api.getJson(
		pathUsersGetFields,
		&res,
	))
}

func (u *Users) GetSentMessages(query types.UserSentMessagesRequest) ([]types.UserSentMessage, error) {
	params := url.Values{}
	if query.Email != "" {
		params.Add("email", query.Email)
	}
	if query.UserId != "" {
		params.Add("userId", query.UserId)
	}
	if query.Limit > 0 {
		params.Add("limit", fmt.Sprint(query.Limit))
	}
	for _, id := range query.CampaignIds {
		params.Add("campaignIds", fmt.Sprint(id))
	}
	if query.StartDate != "" {
		params.Add("startDateTime", query.StartDate)
	}
	if query.EndDate != "" {
		params.Add("endDateTime", query.EndDate)
	}
	params.Add("excludeBlastCampaigns", fmt.Sprint(query.ExcludeBlastCampaign))
	if query.MessageMedium != "" {
		params.Add("messageMedium", query.MessageMedium)
	}

	queryStr := params.Encode()

	var res types.UserSentMessagesResponse
	return toNilErr(res.Messages, u.api.getJson(
		strings.Replace(pathUsersGetSentMessages, "{query}", queryStr, 1),
		&res,
	))
}

func (u *Users) GetForgottenEmails() ([]string, error) {
	var res types.UserForgottenEmailsResponse
	return toNilErr(res.HashedEmails, u.api.getJson(
		pathUsersForgotten,
		&res,
	))
}

func (u *Users) ForgetByEmail(email string) (*types.PostResponse, error) {
	return u.forget(email, "")
}

func (u *Users) ForgetByUserId(userId string) (*types.PostResponse, error) {
	return u.forget("", userId)
}

func (u *Users) forget(email string, userId string) (*types.PostResponse, error) {
	req := types.GdprRequest{
		Email:  email,
		UserId: userId,
	}
	var res types.PostResponse
	return toNilErr(&res, u.api.postJson(pathUsersForget, req, &res))
}

func (u *Users) UnForgetByEmail(email string) (*types.PostResponse, error) {
	return u.unForget(email, "")
}

func (u *Users) UnForgetByUserId(userId string) (*types.PostResponse, error) {
	return u.unForget("", userId)
}

func (u *Users) unForget(email string, userId string) (*types.PostResponse, error) {
	req := types.GdprRequest{
		Email:  email,
		UserId: userId,
	}
	var res types.PostResponse
	return toNilErr(&res, u.api.postJson(pathUsersUnForget, req, &res))
}
