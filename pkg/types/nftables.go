package types

type NftValidateError struct {
	Section string `json:"section,omitempty" validate:"optional"`
	Line    int    `json:"line"`
	Message string `json:"message"`
}

type NftValidateResult struct {
	OK     bool               `json:"ok"`
	Errors []NftValidateError `json:"errors,omitempty" validate:"optional"`
}
