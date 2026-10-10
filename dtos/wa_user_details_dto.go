package dtos

type WAUserDetailsResponse struct {
	Name                        string  `json:"name"`
	Email                       string  `json:"email"`
	Gender                      *int    `json:"gender"`
	DOB                         *string `json:"dob"`
	ProfilePicture              *string `json:"profilePicture"`
	IsActive                    bool    `json:"isActive"`
	MainRole                    string  `json:"mainRole"`
	AvatarID                    int     `json:"avatarId"`
	AvatarURL                   *string `json:"avatarURL"`
	PhoneNumber                 string  `json:"phoneNumber"`
	AlternateMobile             *string `json:"alternateMobile"`
	CurrentBalance              *int    `json:"currentBalance"`
	DefaultAddressCity          *string `json:"defaultAddressCity"`
	ProfileCompletionPercentage int     `json:"profileCompletionPercentage"`
	LastLanguageSelection       int     `json:"lastLanguageSelection"`
	Rank                        int     `json:"rank"`
	Coins                       int     `json:"coins"`
	IsCreator                   bool    `json:"isCreator"`
	Ques1                       bool    `json:"ques1"`
	Ques2                       bool    `json:"ques2"`
	Ques3                       bool    `json:"ques3"`
	Ques4                       bool    `json:"ques4"`
	Ques5                       bool    `json:"ques5"`
	Ques6                       bool    `json:"ques6"`
}
