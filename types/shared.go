package types

type PostResponse struct {
	Message string                 `json:"msg"`
	Code    string                 `json:"code"`
	Params  map[string]interface{} `json:"params"`
}

func (p *PostResponse) isSuccess() bool {
	return p.Code == "Success"
}

type FailedUpdates struct {
	ConflictEmails     []string `json:"conflictEmails"`
	ConflictUserIds    []string `json:"conflictUserIds"`
	ForgottenEmails    []string `json:"forgottenEmails"`
	ForgottenUserIds   []string `json:"forgottenUserIds"`
	InvalidDataEmails  []string `json:"invalidDataEmails"`
	InvalidDataUserIds []string `json:"invalidDataUserIds"`
	InvalidEmails      []string `json:"invalidEmails"`
	InvalidUserIds     []string `json:"invalidUserIds"`
	NotFoundEmails     []string `json:"notFoundEmails"`
	NotFoundUserIds    []string `json:"notFoundUserIds"`
}
