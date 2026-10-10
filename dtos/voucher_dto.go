package dtos

type VoucherInfoDTO struct {
	ID                  int    `json:"id"`
	VoucherCode         string `json:"voucher_code"`
	SecureCode          *int   `json:"secure_code"`
	AssignedOn          string `json:"assigned_on"`
	SecureCodeAvailable bool   `json:"secure_code_available"`
	LastDateToRedeem    string `json:"last_date_to_redeem"`
	VoucherName         string `json:"voucher_name"`
	VoucherImage        string `json:"voucher_image"`
	VoucherDescription  string `json:"voucher_description"`
	VoucherURL          string `json:"voucher_url"`
	ButtonValue         string `json:"button_value"`
}
