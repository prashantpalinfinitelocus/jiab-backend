package services

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Infinite-Locus-Product/coca_cola_backend/constants"
	"github.com/Infinite-Locus-Product/coca_cola_backend/dtos"
	"github.com/Infinite-Locus-Product/coca_cola_backend/entities"
	errs "github.com/Infinite-Locus-Product/coca_cola_backend/errors"
	"github.com/Infinite-Locus-Product/coca_cola_backend/repository"
	"github.com/Infinite-Locus-Product/coca_cola_backend/utils"
)

type UserService interface {
	GetUser(ctx context.Context, userID string) (*entities.User, error)
	GetUserAddresses(ctx context.Context, userID string) ([]dtos.AddressResponseDTO, error)
	AddUserAddress(ctx context.Context, userID string, req dtos.AddressRequestDTO) (*dtos.AddressResponseDTO, error)
	EditUserAddress(ctx context.Context, userID string, addressID string, req dtos.AddressRequestDTO) (*dtos.AddressResponseDTO, error)
	DeleteUserAddress(ctx context.Context, userID string, addressID string) error
	AnswerQuestions(ctx context.Context, userID string, answers []dtos.AnswerQuestionsRequestDTO) error
	EditProfile(ctx context.Context, userID string, req dtos.EditProfileRequestDTO) (*entities.User, error)
	GetUserRank(ctx context.Context, userID string) (*entities.UserRank, error)
	AnswerQuestion(ctx context.Context, userID string, questionID int, answer *dtos.AnswerQuestionRequestDTO) error
	GetQuestions(ctx context.Context, userID string, languageID int) ([]dtos.QuestionResponseDTO, error)
	GetQuestionByID(ctx context.Context, questionID int, languageID int) (*dtos.QuestionResponseDTO, error)
	GetQuestionIDByText(ctx context.Context, questionText string, languageID int) (int, error)
	WAUserDetails(ctx context.Context, userID string) (*dtos.WAUserDetailsResponse, error)
	EditLastLanguage(ctx context.Context, userID string, languageID int) error
	UpdateProfileComicCoinPopUp(ctx context.Context, userID string) error
	SubmitProfilingQuestionAnswer(ctx context.Context, userID string, answers []dtos.AnswerQuestionsRequestDTO) error
	UpdateProfilingQuestionLastViewed(ctx context.Context, userID string, questionID int, questionNumber int) error
	CheckProfilingQuestionModal(ctx context.Context, userID string, questionID int, questionNumber int) (*dtos.ProfilingQuestionModalResponseDTO, error)
}

type userService struct {
	txnManager                  *utils.TransactionManager
	userRepo                    repository.UserRepository
	addressRepo                 repository.GenericRepository[entities.Address]
	questionAnswerRepo          repository.UserQuestionAnswerRepository
	userRankRepo                repository.UserRankRepository
	stateRepo                   repository.StateRepository
	cityRepo                    repository.CityRepository
	pinCodeRepo                 repository.PinCodeRepository
	questionMasterRepo          repository.QuestionMasterRepository
	questionMasterLanguageRepo  repository.QuestionMasterLanguageRepository
	optionMasterRepo            repository.OptionMasterRepository
	optionMasterLanguageRepo    repository.OptionMasterLanguageRepository
	languageRepo                repository.LanguageRepository
	avatarService               AvatarService
	gcsService                  utils.GCSService
	comicCoinService            ComicCoinService
	notificationService         NotificationService
	userWalletRepo              repository.UserWalletRepository
	profilingQuestionAnswerRepo repository.ProfilingQuestionAnswerRepository
}

func NewUserService(
	txnManager *utils.TransactionManager,
	userRepo repository.UserRepository,
	addressRepo repository.GenericRepository[entities.Address],
	questionAnswerRepo repository.UserQuestionAnswerRepository,
	userRankRepo repository.UserRankRepository,
	stateRepo repository.StateRepository,
	cityRepo repository.CityRepository,
	pinCodeRepo repository.PinCodeRepository,
	questionMasterRepo repository.QuestionMasterRepository,
	questionMasterLanguageRepo repository.QuestionMasterLanguageRepository,
	optionMasterRepo repository.OptionMasterRepository,
	optionMasterLanguageRepo repository.OptionMasterLanguageRepository,
	languageRepo repository.LanguageRepository,
	avatarService AvatarService,
	gcsService utils.GCSService,
	comicCoinService ComicCoinService,
	notificationService NotificationService,
	userWalletRepo repository.UserWalletRepository,
	profilingQuestionAnswerRepo repository.ProfilingQuestionAnswerRepository,
) UserService {
	return &userService{
		txnManager:                  txnManager,
		userRepo:                    userRepo,
		addressRepo:                 addressRepo,
		questionAnswerRepo:          questionAnswerRepo,
		userRankRepo:                userRankRepo,
		stateRepo:                   stateRepo,
		cityRepo:                    cityRepo,
		pinCodeRepo:                 pinCodeRepo,
		questionMasterRepo:          questionMasterRepo,
		questionMasterLanguageRepo:  questionMasterLanguageRepo,
		optionMasterRepo:            optionMasterRepo,
		optionMasterLanguageRepo:    optionMasterLanguageRepo,
		languageRepo:                languageRepo,
		avatarService:               avatarService,
		gcsService:                  gcsService,
		comicCoinService:            comicCoinService,
		notificationService:         notificationService,
		userWalletRepo:              userWalletRepo,
		profilingQuestionAnswerRepo: profilingQuestionAnswerRepo,
	}
}

func (s *userService) GetUser(ctx context.Context, userID string) (*entities.User, error) {
	tx, err := s.txnManager.StartTxn()
	if err != nil {
		return nil, err
	}
	defer s.txnManager.RollbackOnPanic(tx)

	user, err := s.userRepo.FindById(ctx, tx, uuid.MustParse(userID))
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, err
	}

	s.txnManager.CommitTxn(tx)

	return user, nil
}

func (s *userService) GetUserAddresses(ctx context.Context, userID string) ([]dtos.AddressResponseDTO, error) {
	tx, err := s.txnManager.StartTxn()
	if err != nil {
		return nil, err
	}
	defer s.txnManager.RollbackOnPanic(tx)

	// First verify if user exists
	user, err := s.userRepo.FindById(ctx, tx, uuid.MustParse(userID))
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, err
	}

	if user == nil {
		s.txnManager.AbortTxn(tx)
		return nil, nil
	}

	addresses, err := s.addressRepo.FindWithConditions(ctx, tx, map[string]interface{}{
		"user_id": userID,
	})
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, err
	}

	// Collect all IDs for bulk fetching
	stateIDs := make([]int, 0, len(addresses))
	cityIDs := make([]int, 0, len(addresses))
	pinCodeIDs := make([]int, 0, len(addresses))

	for _, addr := range addresses {
		stateIDs = append(stateIDs, addr.StateID)
		cityIDs = append(cityIDs, addr.CityID)
		pinCodeIDs = append(pinCodeIDs, addr.PinCodeID)
	}

	// Fetch all states, cities and pincodes in bulk
	states, err := s.stateRepo.FindByIDs(ctx, tx, stateIDs)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, fmt.Errorf("failed to get state details: %v", err)
	}

	cities, err := s.cityRepo.FindByIDs(ctx, tx, cityIDs)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, fmt.Errorf("failed to get city details: %v", err)
	}

	pincodes, err := s.pinCodeRepo.FindByIDs(ctx, tx, pinCodeIDs)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, fmt.Errorf("failed to get pincode details: %v", err)
	}

	// Create lookup maps for quick access
	stateMap := make(map[int]*entities.State)
	for _, state := range states {
		stateCopy := state // Create a copy to avoid using loop variable
		stateMap[state.ID] = &stateCopy
	}

	cityMap := make(map[int]*entities.City)
	for _, city := range cities {
		cityCopy := city // Create a copy to avoid using loop variable
		cityMap[city.ID] = &cityCopy
	}

	pincodeMap := make(map[int]*entities.PinCode)
	for _, pincode := range pincodes {
		pincodeCopy := pincode // Create a copy to avoid using loop variable
		pincodeMap[pincode.ID] = &pincodeCopy
	}

	// Convert addresses to response DTOs with location details
	addressDTOs := make([]dtos.AddressResponseDTO, 0, len(addresses))
	for _, addr := range addresses {
		state := stateMap[addr.StateID]
		city := cityMap[addr.CityID]
		pincode := pincodeMap[addr.PinCodeID]

		if state == nil || city == nil || pincode == nil {
			s.txnManager.AbortTxn(tx)
			return nil, fmt.Errorf("missing location details for address %d", addr.ID)
		}

		addressDTO := dtos.AddressResponseDTO{
			ID:              addr.ID,
			Address1:        addr.Address1,
			Address2:        addr.Address2,
			Pincode:         pincode.Pincode,
			State:           state.Name,
			City:            city.Name,
			NearestLandmark: addr.NearestLandmark,
			ShippingMobile:  addr.ShippingMobile,
			IsDefault:       addr.IsDefault,
			IsActive:        addr.IsActive,
			CreatedOn:       addr.CreatedOn,
			LastModifiedOn:  addr.LastModifiedOn,
		}
		addressDTOs = append(addressDTOs, addressDTO)
	}

	s.txnManager.CommitTxn(tx)

	return addressDTOs, nil
}

func (s *userService) AddUserAddress(ctx context.Context, userID string, req dtos.AddressRequestDTO) (*dtos.AddressResponseDTO, error) {
	tx, err := s.txnManager.StartTxn()
	if err != nil {
		return nil, err
	}
	defer s.txnManager.RollbackOnPanic(tx)

	// First verify if user exists
	user, err := s.userRepo.FindById(ctx, tx, uuid.MustParse(userID))
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, err
	}

	if user == nil {
		s.txnManager.AbortTxn(tx)
		return nil, nil
	}

	// Look up state ID
	state, err := s.stateRepo.FindByName(ctx, tx, req.State)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, fmt.Errorf("failed to find state: %v", err)
	}
	if state == nil {
		s.txnManager.AbortTxn(tx)
		return nil, fmt.Errorf("state '%s' not found", req.State)
	}

	// Look up city ID
	city, err := s.cityRepo.FindByNameAndStateID(ctx, tx, req.City, state.ID)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, fmt.Errorf("failed to find city: %v", err)
	}
	if city == nil {
		s.txnManager.AbortTxn(tx)
		return nil, fmt.Errorf("city '%s' not found in state '%s'", req.City, req.State)
	}

	// Look up pincode and validate deliverability
	pincode, err := s.pinCodeRepo.FindByPincodeAndCityID(ctx, tx, req.Pincode, city.ID)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, fmt.Errorf("failed to find pincode: %v", err)
	}
	if pincode == nil {
		s.txnManager.AbortTxn(tx)
		return nil, fmt.Errorf("pincode '%d' not found in city '%s'", req.Pincode, req.City)
	}

	// Check if pincode is deliverable
	isDeliverable, err := s.pinCodeRepo.IsDeliverable(ctx, tx, req.Pincode, city.ID)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, fmt.Errorf("failed to check pincode deliverability: %v", err)
	}
	if !isDeliverable {
		s.txnManager.AbortTxn(tx)
		return nil, fmt.Errorf("pincode '%d' is not deliverable in city '%s'", req.Pincode, req.City)
	}

	// Check if this is the user's first address
	existingAddresses, err := s.addressRepo.FindWithConditions(ctx, tx, map[string]interface{}{"user_id": userID})
	if err != nil {
		// If no addresses are found, it's not a fatal error for this check
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			s.txnManager.AbortTxn(tx)
			return nil, fmt.Errorf("failed to check for existing addresses: %w", err)
		}
	}
	isFirstAddress := len(existingAddresses) == 0

	// If this is set as default, unset any existing default address
	if req.IsDefault {
		existingAddresses, err := s.addressRepo.FindWithConditions(ctx, tx, map[string]interface{}{
			"user_id":    userID,
			"is_default": true,
		})
		if err != nil {
			s.txnManager.AbortTxn(tx)
			return nil, err
		}

		for _, addr := range existingAddresses {
			if err := s.addressRepo.UpdateFields(ctx, tx, addr.ID, map[string]interface{}{
				"is_default": false,
			}); err != nil {
				s.txnManager.AbortTxn(tx)
				return nil, err
			}
		}
	}

	// Create new address
	now := time.Now()
	address := &entities.Address{
		UserID:          userID,
		Address1:        req.Address1,
		Address2:        req.Address2,
		Pincode:         req.Pincode,
		PinCodeID:       pincode.ID,
		CityID:          city.ID,
		StateID:         state.ID,
		NearestLandmark: req.NearestLandmark,
		ShippingMobile:  req.ShippingMobile,
		IsDefault:       req.IsDefault,
		IsActive:        true,
		IsDeleted:       false,
		CreatedBy:       userID,
		CreatedOn:       now,
	}

	if err := s.addressRepo.Create(ctx, tx, address); err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, err
	}

	// If this was the first address, add 10 points to profile completion
	if isFirstAddress {
		if err := s.addPointsForFirstAddress(ctx, tx, userID); err != nil {
			s.txnManager.AbortTxn(tx)
			return nil, fmt.Errorf("failed to update profile completion for first address: %w", err)
		}
		if err := s.checkAndAwardProfileCompletionBonus(ctx, tx, userID); err != nil {
			s.txnManager.AbortTxn(tx)
			return nil, fmt.Errorf("failed to award profile completion bonus: %w", err)
		}
	}

	// Convert to response DTO
	addressDTO := &dtos.AddressResponseDTO{
		ID:              address.ID,
		Address1:        address.Address1,
		Address2:        address.Address2,
		Pincode:         pincode.Pincode,
		State:           state.Name,
		City:            city.Name,
		NearestLandmark: address.NearestLandmark,
		ShippingMobile:  address.ShippingMobile,
		IsDefault:       address.IsDefault,
		IsActive:        address.IsActive,
		CreatedOn:       address.CreatedOn,
		LastModifiedOn:  address.LastModifiedOn,
	}

	s.txnManager.CommitTxn(tx)

	return addressDTO, nil
}

func (s *userService) EditUserAddress(ctx context.Context, userID string, addressID string, req dtos.AddressRequestDTO) (*dtos.AddressResponseDTO, error) {
	tx, err := s.txnManager.StartTxn()
	if err != nil {
		return nil, err
	}
	defer s.txnManager.RollbackOnPanic(tx)

	// First verify if user exists
	user, err := s.userRepo.FindById(ctx, tx, uuid.MustParse(userID))
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, err
	}

	if user == nil {
		s.txnManager.AbortTxn(tx)
		return nil, nil
	}

	// Convert addressID string to int
	addrID, err := strconv.ParseInt(addressID, 10, 64)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, fmt.Errorf("invalid address ID format")
	}

	// Get the address to edit
	address, err := s.addressRepo.FindById(ctx, tx, addrID)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, err
	}

	if address == nil {
		s.txnManager.AbortTxn(tx)
		return nil, fmt.Errorf("address not found")
	}

	// Look up state ID
	state, err := s.stateRepo.FindByName(ctx, tx, req.State)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, fmt.Errorf("failed to find state: %v", err)
	}
	if state == nil {
		s.txnManager.AbortTxn(tx)
		return nil, fmt.Errorf("state '%s' not found", req.State)
	}

	// Look up city ID
	city, err := s.cityRepo.FindByNameAndStateID(ctx, tx, req.City, state.ID)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, fmt.Errorf("failed to find city: %v", err)
	}
	if city == nil {
		s.txnManager.AbortTxn(tx)
		return nil, fmt.Errorf("city '%s' not found in state '%s'", req.City, req.State)
	}

	// Look up pincode and validate deliverability
	pincode, err := s.pinCodeRepo.FindByPincodeAndCityID(ctx, tx, req.Pincode, city.ID)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, fmt.Errorf("failed to find pincode: %v", err)
	}
	if pincode == nil {
		s.txnManager.AbortTxn(tx)
		return nil, fmt.Errorf("pincode '%d' not found in city '%s'", req.Pincode, req.City)
	}

	// Check if pincode is deliverable
	isDeliverable, err := s.pinCodeRepo.IsDeliverable(ctx, tx, req.Pincode, city.ID)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, fmt.Errorf("failed to check pincode deliverability: %v", err)
	}
	if !isDeliverable {
		s.txnManager.AbortTxn(tx)
		return nil, fmt.Errorf("pincode '%d' is not deliverable in city '%s'", req.Pincode, req.City)
	}

	// If this is set as default, unset any existing default address
	if req.IsDefault && !address.IsDefault {
		existingAddresses, err := s.addressRepo.FindWithConditions(ctx, tx, map[string]interface{}{
			"user_id":    userID,
			"is_default": true,
		})
		if err != nil {
			s.txnManager.AbortTxn(tx)
			return nil, err
		}

		for _, addr := range existingAddresses {
			if err := s.addressRepo.UpdateFields(ctx, tx, addr.ID, map[string]interface{}{
				"is_default": false,
			}); err != nil {
				s.txnManager.AbortTxn(tx)
				return nil, err
			}
		}
	}

	// Update address fields
	now := time.Now()
	updateFields := map[string]interface{}{
		"address1":         req.Address1,
		"pincode":          req.Pincode,
		"pin_code_id":      pincode.ID,
		"city_id":          city.ID,
		"state_id":         state.ID,
		"is_default":       req.IsDefault,
		"last_modified_by": userID,
		"last_modified_on": now,
	}

	// Only update optional fields if they are provided
	if req.Address2 != nil {
		updateFields["address2"] = *req.Address2
	}
	if req.NearestLandmark != nil {
		updateFields["nearest_landmark"] = *req.NearestLandmark
	}
	if req.ShippingMobile != nil {
		updateFields["shipping_mobile"] = *req.ShippingMobile
	}

	if err := s.addressRepo.UpdateFields(ctx, tx, addrID, updateFields); err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, err
	}

	// Get updated address
	updatedAddress, err := s.addressRepo.FindById(ctx, tx, addrID)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, err
	}

	// Convert to response DTO
	addressDTO := &dtos.AddressResponseDTO{
		ID:              updatedAddress.ID,
		Address1:        updatedAddress.Address1,
		Address2:        updatedAddress.Address2,
		Pincode:         pincode.Pincode,
		State:           state.Name,
		City:            city.Name,
		NearestLandmark: updatedAddress.NearestLandmark,
		ShippingMobile:  updatedAddress.ShippingMobile,
		IsDefault:       updatedAddress.IsDefault,
		IsActive:        updatedAddress.IsActive,
		CreatedOn:       updatedAddress.CreatedOn,
		LastModifiedOn:  updatedAddress.LastModifiedOn,
	}

	s.txnManager.CommitTxn(tx)

	return addressDTO, nil
}

func (s *userService) DeleteUserAddress(ctx context.Context, userID string, addressID string) error {
	tx, err := s.txnManager.StartTxn()
	if err != nil {
		return err
	}
	defer s.txnManager.RollbackOnPanic(tx)

	// First verify if user exists
	user, err := s.userRepo.FindById(ctx, tx, uuid.MustParse(userID))
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return err
	}

	if user == nil {
		s.txnManager.AbortTxn(tx)
		return fmt.Errorf("user not found")
	}

	// Convert addressID string to int
	addrID, err := strconv.ParseInt(addressID, 10, 64)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return fmt.Errorf("invalid address ID format")
	}

	// Get the address to delete
	address, err := s.addressRepo.FindById(ctx, tx, addrID)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return err
	}

	if address == nil {
		s.txnManager.AbortTxn(tx)
		return fmt.Errorf("address not found")
	}

	// Check if this is the default address
	wasDefault := address.IsDefault

	// Get count of remaining addresses before deletion
	remainingAddresses, err := s.addressRepo.FindWithConditions(ctx, tx, map[string]interface{}{
		"user_id": userID,
	})
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return err
	}

	// Delete the address
	if err := s.addressRepo.Delete(ctx, tx, addrID); err != nil {
		s.txnManager.AbortTxn(tx)
		return err
	}

	// If this was the last address (count was 1 before deletion), decrease profile_percentage by 20
	if len(remainingAddresses) == 1 {
		if err := s.userRepo.UpdateFields(ctx, tx, userID, map[string]interface{}{
			"profile_percentage": gorm.Expr("profile_percentage - ?", 20),
		}); err != nil {
			s.txnManager.AbortTxn(tx)
			return fmt.Errorf("failed to update profile percentage: %w", err)
		}
	}

	// If the deleted address was default, set the most recently created address as default
	if wasDefault && len(remainingAddresses) > 1 {
		// Find the most recently created address
		var mostRecentAddress *entities.Address
		for _, addr := range remainingAddresses {
			if int64(addr.ID) != addrID { // Skip the address being deleted
				if mostRecentAddress == nil || addr.CreatedOn.After(mostRecentAddress.CreatedOn) {
					mostRecentAddress = &addr
				}
			}
		}

		// Set the most recent address as default
		if mostRecentAddress != nil {
			if err := s.addressRepo.UpdateFields(ctx, tx, mostRecentAddress.ID, map[string]interface{}{
				"is_default": true,
			}); err != nil {
				s.txnManager.AbortTxn(tx)
				return err
			}
		}
	}

	s.txnManager.CommitTxn(tx)
	return nil
}

func (s *userService) AnswerQuestions(ctx context.Context, userID string, answers []dtos.AnswerQuestionsRequestDTO) error {
	tx, err := s.txnManager.StartTxn()
	if err != nil {
		return err
	}
	defer s.txnManager.RollbackOnPanic(tx)

	// First verify if user exists
	user, err := s.userRepo.FindById(ctx, tx, uuid.MustParse(userID))
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return err
	}

	if user == nil {
		s.txnManager.AbortTxn(tx)
		return fmt.Errorf("user not found")
	}

	// Check existing answers before creating new ones
	existingAnswers, err := s.questionAnswerRepo.FindByUserID(ctx, tx, userID)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return fmt.Errorf("failed to get existing answers: %v", err)
	}

	// Create a map of existing answers for quick lookup
	existingAnswerMap := make(map[int]entities.UserQuestionAnswer)
	for _, ans := range existingAnswers {
		existingAnswerMap[ans.QuestionMasterID] = ans
	}

	// Process each answer
	now := time.Now()
	newAnswers := make([]entities.UserQuestionAnswer, 0)
	updatedAnswers := make([]entities.UserQuestionAnswer, 0)

	for _, answer := range answers {
		if existingAnswer, exists := existingAnswerMap[answer.QuestionID]; exists {
			// Question already answered, just update the option ID
			existingAnswer.OptionID = answer.AnswerID
			existingAnswer.LastModifiedBy = &userID
			existingAnswer.LastModifiedOn = &now
			updatedAnswers = append(updatedAnswers, existingAnswer)
		} else {
			// New answer, create it
			newAnswer := entities.UserQuestionAnswer{
				UserID:           userID,
				QuestionMasterID: answer.QuestionID,
				OptionID:         answer.AnswerID,
				SelectedAnswer:   true,
				IsActive:         true,
				IsDeleted:        false,
				CreatedBy:        userID,
				CreatedOn:        now,
			}
			newAnswers = append(newAnswers, newAnswer)
		}
	}

	// Update existing answers
	for _, answer := range updatedAnswers {
		if err := s.questionAnswerRepo.Update(ctx, tx, &answer); err != nil {
			s.txnManager.AbortTxn(tx)
			return fmt.Errorf("failed to update existing answer: %v", err)
		}
	}

	// Save new answers
	if len(newAnswers) > 0 {
		if err := s.questionAnswerRepo.CreateMany(ctx, tx, newAnswers); err != nil {
			s.txnManager.AbortTxn(tx)
			return fmt.Errorf("failed to save new answers: %v", err)
		}

		// Only process new answers for points and notifications
		for range newAnswers {
			if err := s.addPointsForQuestionAnswer(ctx, tx, userID); err != nil {
				s.txnManager.AbortTxn(tx)
				return fmt.Errorf("failed to update profile completion for answer: %w", err)
			}

			// if err := s.notificationService.SendPushNotification(ctx, tx, constants.NOTIFICATION_WIN_COIN, userID, &constants.COIN_COUNT_INCREMENT_VALUE); err != nil {
			// 	s.txnManager.AbortTxn(tx)
			// 	return fmt.Errorf("failed to send notification: %w", err)
			// }
			// if _, err := s.comicCoinService.UpdateBalance(ctx, tx, userID, 1); err != nil {
			// 	s.txnManager.AbortTxn(tx)
			// 	return fmt.Errorf("failed to award coin for answering question: %w", err)
			// }

			if (len(existingAnswers) + len(existingAnswers)) >= 3 {

				if err := s.checkAndAwardProfileCompletionBonus(ctx, tx, userID); err != nil {
					s.txnManager.AbortTxn(tx)
					return fmt.Errorf("failed to award profile completion bonus: %w", err)
				}
			}
		}
	}

	s.txnManager.CommitTxn(tx)
	return nil
}

// validateProfileUpdateFlags checks if the avatar and profile image flags are valid
func (s *userService) validateProfileUpdateFlags(req *dtos.EditProfileRequestDTO) error {
	if req.IsAvatar != nil && req.IsProfile != nil && *req.IsAvatar && *req.IsProfile {
		return fmt.Errorf("cannot set both is_avatar and is_profile to true")
	}
	return nil
}

// handleAvatarUpdate processes avatar selection and updates
func (s *userService) handleAvatarUpdate(ctx context.Context, tx *gorm.DB, user *entities.User, req *dtos.EditProfileRequestDTO) (map[string]interface{}, error) {
	if req.AvatarID == nil {
		return nil, fmt.Errorf("avatar_id is required when is_avatar is true")
	}

	// Verify avatar exists and is active
	avatar, err := s.avatarService.GetAvatar(ctx, *req.AvatarID)
	if err != nil {
		if err.Error() == "avatar not found or inactive" {
			return nil, fmt.Errorf("avatar with ID %d not found or inactive", *req.AvatarID)
		}
		return nil, fmt.Errorf("failed to verify avatar: %v", err)
	}

	if avatar == nil {
		return nil, fmt.Errorf("avatar with ID %d not found", *req.AvatarID)
	}

	updateFields := make(map[string]interface{})

	// Clear profile picture if exists
	if user.ProfilePicture != nil {
		signedURL, err := s.gcsService.GetFileSignedURL(ctx, *user.ProfilePicture, 60*time.Minute)
		if err != nil {
			return nil, fmt.Errorf("failed to generate signed URL for profile picture: %v", err)
		}
		if err := s.gcsService.DeleteFile(ctx, signedURL); err != nil {
			return nil, fmt.Errorf("failed to delete existing profile picture: %v", err)
		}
		updateFields["profile_picture"] = nil
	}

	updateFields["avatar_id"] = *req.AvatarID
	return updateFields, nil
}

// handleProfileImageUpdate processes profile image upload and updates
func (s *userService) handleProfileImageUpdate(ctx context.Context, tx *gorm.DB, user *entities.User, req *dtos.EditProfileRequestDTO) (map[string]interface{}, error) {
	if req.IsProfile == nil || !*req.IsProfile {
		return nil, fmt.Errorf("is_profile is required when is_profile is true")
	}

	if req.ProfileImage == nil {
		return nil, fmt.Errorf("profile_image is required when is_profile is true")
	}

	updateFields := make(map[string]interface{})

	// Upload new profile image
	_, profileImagePath, err := s.gcsService.UploadFile(ctx, req.ProfileImage, "profile_images")
	if err != nil {
		return nil, fmt.Errorf("failed to upload profile image: %v", err)
	}

	// Delete old profile picture if exists
	if user.ProfilePicture != nil {
		signedURL, err := s.gcsService.GetFileSignedURL(ctx, *user.ProfilePicture, 60*time.Minute)
		if err != nil {
			return nil, fmt.Errorf("failed to generate signed URL for profile picture: %v", err)
		}
		if err := s.gcsService.DeleteFile(ctx, signedURL); err != nil {
			return nil, fmt.Errorf("failed to delete existing profile picture: %v", err)
		}
	}

	// Clear avatar if exists
	if user.AvatarID != nil {
		updateFields["avatar_id"] = nil
	}

	updateFields["profile_picture"] = profileImagePath
	return updateFields, nil
}

// handleBasicProfileUpdate processes basic profile fields (name, email, gender, DOB)
func (s *userService) handleBasicProfileUpdate(ctx context.Context, tx *gorm.DB, user *entities.User, req *dtos.EditProfileRequestDTO) (map[string]interface{}, error) {
	updateFields := make(map[string]interface{})

	if req.Name != nil {
		updateFields["name"] = *req.Name
	}

	if req.Email != nil {
		// Check if email is already in use by another user
		if *req.Email != "" {
			existingUser, err := s.userRepo.FindByEmail(ctx, tx, *req.Email)
			if err != nil {
				return nil, fmt.Errorf("failed to check email uniqueness: %v", err)
			}
			// If email exists and belongs to a different user
			if existingUser != nil && existingUser.ID != user.ID {
				return nil, fmt.Errorf("email already in use")
			}
		}
		updateFields["email"] = *req.Email
	}

	if req.Gender != nil {
		genderMap := map[string]int{
			"male":              1,
			"female":            2,
			"other":             3,
			"perfer_not_to_say": 4,
		}
		if genderID, exists := genderMap[*req.Gender]; exists {
			updateFields["gender"] = genderID
		}
	}

	if req.DOB != nil {
		dob, _ := time.Parse("01/02/2006", *req.DOB)
		updateFields["dob"] = dob
	}

	return updateFields, nil
}

func (s *userService) EditProfile(ctx context.Context, userID string, req dtos.EditProfileRequestDTO) (*entities.User, error) {
	tx, err := s.txnManager.StartTxn()
	if err != nil {
		return nil, err
	}
	defer s.txnManager.RollbackOnPanic(tx)

	// First verify if user exists
	user, err := s.userRepo.FindById(ctx, tx, uuid.MustParse(userID))
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, err
	}

	if user == nil {
		s.txnManager.AbortTxn(tx)
		return nil, fmt.Errorf("user not found")
	}

	// Validate update flags
	if err := s.validateProfileUpdateFlags(&req); err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, err
	}

	// Validate avatar requirements first
	if req.IsAvatar != nil && *req.IsAvatar && req.AvatarID == nil {
		s.txnManager.AbortTxn(tx)
		return nil, fmt.Errorf("avatar_id is required when is_avatar is true")
	}

	// Handle avatar update if is_avatar is true
	var avatarUpdates map[string]interface{}
	if req.IsAvatar != nil && *req.IsAvatar {
		// Now we know IsAvatar is true and AvatarID is not nil
		avatarUpdates, err = s.handleAvatarUpdate(ctx, tx, user, &req)
		if err != nil {
			s.txnManager.AbortTxn(tx)
			return nil, fmt.Errorf("avatar update failed: %v", err)
		}
	}

	// Handle profile image update if is_profile is true
	var profileImageUpdates map[string]interface{}
	if req.IsProfile != nil && *req.IsProfile {
		profileImageUpdates, err = s.handleProfileImageUpdate(ctx, tx, user, &req)
		if err != nil {
			s.txnManager.AbortTxn(tx)
			return nil, fmt.Errorf("profile image update failed: %v", err)
		}
	}

	// Handle basic profile fields update
	basicUpdates, err := s.handleBasicProfileUpdate(ctx, tx, user, &req)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, fmt.Errorf("basic profile update failed: %v", err)
	}

	// Merge all updates
	updateFields := make(map[string]interface{})
	for k, v := range avatarUpdates {
		updateFields[k] = v
	}
	for k, v := range profileImageUpdates {
		updateFields[k] = v
	}
	for k, v := range basicUpdates {
		updateFields[k] = v
	}

	// If no fields to update, return the user as is
	if len(updateFields) > 0 {
		// Update the user
		if err := s.userRepo.UpdateFields(ctx, tx, userID, updateFields); err != nil {
			s.txnManager.AbortTxn(tx)
			return nil, fmt.Errorf("failed to update profile: %v", err)
		}
	}

	// Recalculate profile completion after the main update
	if err := s.updateProfileCompletion(ctx, tx, userID); err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, fmt.Errorf("failed to update profile completion: %w", err)
	}

	if user.ProfilePercentage < 100 {
		if err := s.checkAndAwardProfileCompletionBonus(ctx, tx, userID); err != nil {
			s.txnManager.AbortTxn(tx)
			return nil, fmt.Errorf("failed to award profile completion bonus: %w", err)
		}
	}

	// Get the updated user
	updatedUser, err := s.userRepo.FindById(ctx, tx, uuid.MustParse(userID))
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, err
	}

	s.txnManager.CommitTxn(tx)
	return updatedUser, nil
}

func (s *userService) updateProfileCompletion(ctx context.Context, tx *gorm.DB, userID string) error {
	user, err := s.userRepo.FindById(ctx, tx, uuid.MustParse(userID))
	if err != nil {
		return fmt.Errorf("failed to refetch user for completion calculation: %w", err)
	}
	if user == nil {
		return errors.New("user not found for completion calculation")
	}

	percentage := 0
	if user.Name != nil && *user.Name != "" {
		percentage += 20
	}
	if user.DOB != nil {
		percentage += 15
	}
	if user.PhoneNumber != "" {
		percentage += 25
	}
	if user.Gender != nil {
		percentage += 10
	}

	// Check for address
	addresses, err := s.addressRepo.FindWithConditions(ctx, tx, map[string]interface{}{"user_id": user.ID})
	if err != nil {
		// If no addresses are found, it's not a fatal error, just means 0 points.
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("failed to get user addresses: %w", err)
		}
	}
	if len(addresses) > 0 {
		percentage += 20
	}

	// Check for answered questions
	answeredQuestions, err := s.questionAnswerRepo.FindAllByUserID(ctx, tx, user.ID)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("failed to get answered questions: %w", err)
		}
	}
	if len(answeredQuestions) >= 3 {
		percentage += 10
	}

	updates := map[string]interface{}{"profile_percentage": percentage}

	return s.userRepo.UpdateFields(ctx, tx, userID, updates)
}

func (s *userService) GetUserRank(ctx context.Context, userID string) (*entities.UserRank, error) {
	tx, err := s.txnManager.StartTxn()
	if err != nil {
		return nil, err
	}
	defer s.txnManager.RollbackOnPanic(tx)

	// First verify if user exists
	user, err := s.userRepo.FindById(ctx, tx, uuid.MustParse(userID))
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, err
	}

	if user == nil {
		s.txnManager.AbortTxn(tx)
		return nil, fmt.Errorf("user not found")
	}

	// Get user rank
	userRank, err := s.userRankRepo.FindByUserID(ctx, tx, userID)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, fmt.Errorf("failed to fetch user rank: %v", err)
	}

	s.txnManager.CommitTxn(tx)
	return userRank, nil
}

func (s *userService) AnswerQuestion(ctx context.Context, userID string, questionID int, answer *dtos.AnswerQuestionRequestDTO) error {
	tx, err := s.txnManager.StartTxn()
	if err != nil {
		return err
	}
	defer s.txnManager.RollbackOnPanic(tx)

	// Check if question exists
	question, err := s.questionMasterRepo.FindByID(ctx, tx, questionID)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return err
	}
	if question == nil {
		s.txnManager.AbortTxn(tx)
		return errors.New("question not found")
	}

	// Check if option exists and belongs to the question
	option, err := s.optionMasterRepo.FindByID(ctx, tx, answer.OptionID)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return err
	}
	if option == nil || option.QuestionMasterID != questionID {
		s.txnManager.AbortTxn(tx)
		return errors.New("invalid option for this question")
	}

	// Check if user already answered this question
	existingAnswer, err := s.questionAnswerRepo.FindByUserAndQuestionID(ctx, tx, userID, questionID)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return err
	}

	now := time.Now()
	if existingAnswer != nil {
		// Update existing answer
		existingAnswer.OptionID = answer.OptionID
		existingAnswer.SelectedAnswer = answer.SelectedAnswer
		existingAnswer.LastModifiedBy = &userID
		existingAnswer.LastModifiedOn = &now
		if err := s.questionAnswerRepo.Update(ctx, tx, existingAnswer); err != nil {
			s.txnManager.AbortTxn(tx)
			return err
		}
	} else {
		// Create new answer
		newAnswer := &entities.UserQuestionAnswer{
			UserID:           userID,
			QuestionMasterID: questionID,
			OptionID:         answer.OptionID,
			SelectedAnswer:   answer.SelectedAnswer,
			IsActive:         true,
			IsDeleted:        false,
			CreatedBy:        userID,
			CreatedOn:        now,
		}
		if err := s.questionAnswerRepo.Create(ctx, tx, newAnswer); err != nil {
			s.txnManager.AbortTxn(tx)
			return err
		}
	}

	// Update profile completion for the answer
	if err := s.addPointsForQuestionAnswer(ctx, tx, userID); err != nil {
		s.txnManager.AbortTxn(tx)
		return fmt.Errorf("failed to update profile completion for answer: %w", err)
	}

	if _, err := s.comicCoinService.UpdateBalance(ctx, tx, userID, 1); err != nil {
		s.txnManager.AbortTxn(tx)
		return fmt.Errorf("failed to award coin for answering question: %w", err)
	}

	coinCountIncrementValue := constants.COIN_COUNT_INCREMENT_VALUE
	if err := s.notificationService.SendPushNotification(ctx, tx, constants.NOTIFICATION_WIN_COIN, userID, &coinCountIncrementValue); err != nil {
		s.txnManager.AbortTxn(tx)
		return fmt.Errorf("failed to send notification: %w", err)
	}

	if err := s.checkAndAwardProfileCompletionBonus(ctx, tx, userID); err != nil {
		s.txnManager.AbortTxn(tx)
		return fmt.Errorf("failed to award profile completion bonus: %w", err)
	}

	s.txnManager.CommitTxn(tx)
	return nil
}

func (s *userService) GetQuestions(ctx context.Context, userID string, languageID int) ([]dtos.QuestionResponseDTO, error) {
	tx, err := s.txnManager.StartTxn()
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, err
	}
	defer s.txnManager.RollbackOnPanic(tx)
	logrus.Infof("userID: %s, languageID: %d", userID, languageID)

	// Step 1: Fetch all active questions from question_master
	questions, err := s.questionMasterRepo.FindActive(ctx, tx)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, fmt.Errorf("failed to get questions: %v", err)
	}
	logrus.Infof("Found %d active questions", len(questions))

	// Step 2: Get question details from question_master_language table based on question ID and language ID
	questionLanguageMap := make(map[int]string)
	for _, question := range questions {
		questionLanguage, err := s.questionMasterLanguageRepo.FindByQuestionMasterIDAndLanguageID(ctx, tx, question.ID, languageID)
		if err != nil {
			s.txnManager.AbortTxn(tx)
			return nil, fmt.Errorf("failed to get question language text for question %d: %v", question.ID, err)
		}
		if questionLanguage != nil {
			questionLanguageMap[question.ID] = questionLanguage.QuestionText
		} else {
			questionLanguageMap[question.ID] = question.QuestionText // fallback to original text
		}
	}

	// Step 3: Fetch options from option_master table using question IDs
	questionIDs := make([]int, 0)
	for _, q := range questions {
		questionIDs = append(questionIDs, q.ID)
	}
	options, err := s.optionMasterRepo.FindByQuestionIDs(ctx, tx, questionIDs)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, fmt.Errorf("failed to get options: %v", err)
	}
	logrus.Infof("Found %d options", len(options))

	// Step 4: Fetch option details from option_master_language table based on option ID and language ID
	optionLanguageMap := make(map[int]string)
	optionIDs := make([]int, 0)
	for _, opt := range options {
		optionIDs = append(optionIDs, opt.ID)
	}
	optionLanguages, err := s.optionMasterLanguageRepo.FindByOptionMasterIDsAndLanguageID(ctx, tx, optionIDs, languageID)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, fmt.Errorf("failed to get option language texts: %v", err)
	}
	for _, optLang := range optionLanguages {
		optionLanguageMap[optLang.OptionMasterID] = optLang.OptionText
	}

	// Step 5: Fetch all user questions from user_question_answer table
	userAnswers, err := s.questionAnswerRepo.FindByUserID(ctx, tx, userID)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, fmt.Errorf("failed to get user answers: %v", err)
	}
	logrus.Infof("Found %d user answers", len(userAnswers))

	// Create a map of user answers for quick lookup
	userAnswerMap := make(map[int]entities.UserQuestionAnswer)
	for _, ans := range userAnswers {
		userAnswerMap[ans.QuestionMasterID] = ans
	}

	// Step 6: Fill options accordingly and construct response
	response := make([]dtos.QuestionResponseDTO, 0)
	for _, question := range questions {
		// Get question text (language-specific or fallback)
		questionText := questionLanguageMap[question.ID]

		// Get options for this question
		var questionOptions []dtos.OptionDTO
		for _, opt := range options {
			if opt.QuestionMasterID == question.ID {
				// Get option text (language-specific or fallback)
				optionText := opt.OptionText
				if langText, exists := optionLanguageMap[opt.ID]; exists {
					optionText = langText
				}

				questionOptions = append(questionOptions, dtos.OptionDTO{
					ID:           opt.ID,
					OptionText:   optionText,
					DisplayOrder: opt.DisplayOrder,
				})
			}
		}

		// Check if user has answered this question
		var selectedOption *int
		if userAnswer, exists := userAnswerMap[question.ID]; exists {
			selectedOption = &userAnswer.OptionID
		}

		response = append(response, dtos.QuestionResponseDTO{
			ID:             question.ID,
			QuestionText:   questionText,
			LanguageID:     languageID,
			QuesPoint:      question.QuesPoint,
			Options:        questionOptions,
			SelectedOption: selectedOption,
		})
	}

	s.txnManager.CommitTxn(tx)
	return response, nil
}

func (s *userService) GetQuestionIDByText(ctx context.Context, questionText string, languageID int) (int, error) {
	tx, err := s.txnManager.StartTxn()
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return 0, err
	}
	defer s.txnManager.RollbackOnPanic(tx)

	questionLanguage, err := s.questionMasterLanguageRepo.FindByQuestionTextAndLanguageID(ctx, tx, questionText, languageID)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return 0, fmt.Errorf("failed to search question in language table: %v", err)
	}
	if questionLanguage != nil {
		s.txnManager.CommitTxn(tx)
		return questionLanguage.QuestionMasterID, nil
	}

	question, err := s.questionMasterRepo.FindByQuestionTextAndLanguageID(ctx, tx, questionText, languageID)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return 0, fmt.Errorf("failed to search question in master table: %v", err)
	}
	if question != nil {
		s.txnManager.CommitTxn(tx)
		return question.ID, nil
	}

	s.txnManager.AbortTxn(tx)
	return 0, fmt.Errorf("question not found")
}

func (s *userService) GetQuestionByID(ctx context.Context, questionID int, languageID int) (*dtos.QuestionResponseDTO, error) {
	tx, err := s.txnManager.StartTxn()
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, err
	}
	defer s.txnManager.RollbackOnPanic(tx)

	question, err := s.questionMasterRepo.FindByID(ctx, tx, questionID)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, fmt.Errorf("failed to get question: %v", err)
	}
	if question == nil {
		s.txnManager.AbortTxn(tx)
		return nil, fmt.Errorf("question not found")
	}

	questionLanguage, err := s.questionMasterLanguageRepo.FindByQuestionMasterIDAndLanguageID(ctx, tx, question.ID, languageID)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, fmt.Errorf("failed to get question language text: %v", err)
	}

	questionText := question.QuestionText
	if questionLanguage != nil {
		questionText = questionLanguage.QuestionText
	}

	options, err := s.optionMasterRepo.FindByQuestionID(ctx, tx, questionID)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, fmt.Errorf("failed to get options: %v", err)
	}

	optionIDs := make([]int, 0)
	for _, opt := range options {
		optionIDs = append(optionIDs, opt.ID)
	}

	optionLanguages, err := s.optionMasterLanguageRepo.FindByOptionMasterIDsAndLanguageID(ctx, tx, optionIDs, languageID)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, fmt.Errorf("failed to get option language texts: %v", err)
	}

	optionLanguageMap := make(map[int]string)
	for _, optLang := range optionLanguages {
		optionLanguageMap[optLang.OptionMasterID] = optLang.OptionText
	}

	var questionOptions []dtos.OptionDTO
	for _, opt := range options {
		optionText := opt.OptionText
		if langText, exists := optionLanguageMap[opt.ID]; exists {
			optionText = langText
		}

		questionOptions = append(questionOptions, dtos.OptionDTO{
			ID:           opt.ID,
			OptionText:   optionText,
			DisplayOrder: opt.DisplayOrder,
		})
	}

	response := &dtos.QuestionResponseDTO{
		ID:           question.ID,
		QuestionText: questionText,
		LanguageID:   languageID,
		QuesPoint:    question.QuesPoint,
		Options:      questionOptions,
	}

	s.txnManager.CommitTxn(tx)
	return response, nil
}

func (s *userService) SubmitProfilingQuestionAnswer(ctx context.Context, userID string, answers []dtos.AnswerQuestionsRequestDTO) error {
	tx, err := s.txnManager.StartTxn()
	if err != nil {
		return err
	}
	defer s.txnManager.RollbackOnPanic(tx)

	user, err := s.userRepo.FindById(ctx, tx, uuid.MustParse(userID))
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return err
	}

	if user == nil {
		s.txnManager.AbortTxn(tx)
		return fmt.Errorf("user not found")
	}

	existingAnswers, err := s.profilingQuestionAnswerRepo.FindWithConditions(ctx, tx, map[string]interface{}{
		"user_id":    userID,
		"is_deleted": false,
	})
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return fmt.Errorf("failed to get existing answers: %v", err)
	}

	existingAnswerMap := make(map[int]entities.ProfilingQuestionAnswer)
	for _, ans := range existingAnswers {
		existingAnswerMap[ans.QuestionMasterID] = ans
	}

	now := time.Now()
	newAnswers := make([]entities.ProfilingQuestionAnswer, 0)
	updatedAnswers := make([]entities.ProfilingQuestionAnswer, 0)

	for _, answer := range answers {
		question, err := s.questionMasterRepo.FindByID(ctx, tx, answer.QuestionID)
		if err != nil {
			s.txnManager.AbortTxn(tx)
			return fmt.Errorf("failed to get question: %v", err)
		}
		if question == nil {
			s.txnManager.AbortTxn(tx)
			return fmt.Errorf("question not found")
		}

		option, err := s.optionMasterRepo.FindByID(ctx, tx, answer.AnswerID)
		if err != nil {
			s.txnManager.AbortTxn(tx)
			return fmt.Errorf("failed to get option: %v", err)
		}
		if option == nil || option.QuestionMasterID != answer.QuestionID {
			s.txnManager.AbortTxn(tx)
			return fmt.Errorf("invalid option for this question")
		}

		if existingAnswer, exists := existingAnswerMap[answer.QuestionID]; exists {
			existingAnswer.OptionID = answer.AnswerID
			existingAnswer.SelectedAnswer = true
			existingAnswer.LastViewedOn = &now
			existingAnswer.QuestionNumber = answer.QuestionNumber
			updatedAnswers = append(updatedAnswers, existingAnswer)
		} else {
			newAnswer := entities.ProfilingQuestionAnswer{
				UserID:           userID,
				QuestionMasterID: answer.QuestionID,
				OptionID:         answer.AnswerID,
				SelectedAnswer:   true,
				IsActive:         true,
				IsDeleted:        false,
				CreatedBy:        userID,
				CreatedOn:        now,
				QuestionNumber:   answer.QuestionNumber,
				LastViewedOn:     &now,
			}
			newAnswers = append(newAnswers, newAnswer)
		}
	}

	for _, answer := range updatedAnswers {
		if err := s.profilingQuestionAnswerRepo.Update(ctx, tx, answer.ID, &answer); err != nil {
			s.txnManager.AbortTxn(tx)
			return fmt.Errorf("failed to update existing answer: %v", err)
		}
	}

	if len(newAnswers) > 0 {
		if err := s.profilingQuestionAnswerRepo.CreateMany(ctx, tx, newAnswers); err != nil {
			s.txnManager.AbortTxn(tx)
			return fmt.Errorf("failed to save new answers: %v", err)
		}
	}

	s.txnManager.CommitTxn(tx)
	return nil
}

func (s *userService) UpdateProfilingQuestionLastViewed(ctx context.Context, userID string, questionID int, questionNumber int) error {
	tx, err := s.txnManager.StartTxn()
	if err != nil {
		return err
	}
	defer s.txnManager.RollbackOnPanic(tx)

	user, err := s.userRepo.FindById(ctx, tx, uuid.MustParse(userID))
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return err
	}

	if user == nil {
		s.txnManager.AbortTxn(tx)
		return fmt.Errorf("user not found")
	}

	question, err := s.questionMasterRepo.FindByID(ctx, tx, questionID)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return fmt.Errorf("failed to get question: %v", err)
	}
	if question == nil {
		s.txnManager.AbortTxn(tx)
		return fmt.Errorf("question not found")
	}

	existingAnswer, err := s.profilingQuestionAnswerRepo.FindOneWithConditions(ctx, tx, map[string]interface{}{
		"user_id":            userID,
		"question_master_id": questionID,
		"is_deleted":         false,
	})
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return fmt.Errorf("failed to get existing answer: %v", err)
	}

	now := time.Now()
	if existingAnswer != nil {
		if err := s.profilingQuestionAnswerRepo.UpdateFields(ctx, tx, existingAnswer.ID, map[string]interface{}{
			"last_viewed_on": &now,
		}); err != nil {
			s.txnManager.AbortTxn(tx)
			return fmt.Errorf("failed to update last viewed: %v", err)
		}
	} else {
		newAnswer := entities.ProfilingQuestionAnswer{
			UserID:           userID,
			QuestionMasterID: questionID,
			OptionID:         0,
			SelectedAnswer:   false,
			IsActive:         true,
			IsDeleted:        false,
			CreatedBy:        userID,
			CreatedOn:        now,
			LastViewedOn:     &now,
			QuestionNumber:   questionNumber,
		}
		if err := s.profilingQuestionAnswerRepo.Create(ctx, tx, &newAnswer); err != nil {
			s.txnManager.AbortTxn(tx)
			return fmt.Errorf("failed to create answer record: %v", err)
		}
	}

	s.txnManager.CommitTxn(tx)
	return nil
}

func (s *userService) CheckProfilingQuestionModal(ctx context.Context, userID string, questionID int, questionNumber int) (*dtos.ProfilingQuestionModalResponseDTO, error) {

	tx, err := s.txnManager.StartTxn()
	if err != nil {
		return nil, err
	}
	defer s.txnManager.RollbackOnPanic(tx)

	question, err := s.questionMasterRepo.FindByID(ctx, tx, questionID)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, fmt.Errorf("failed to get question: %v", err)
	}
	if question == nil {
		s.txnManager.AbortTxn(tx)
		return nil, fmt.Errorf("question not found")
	}

	existingAnswers, err := s.profilingQuestionAnswerRepo.FindWithConditions(ctx, tx, map[string]interface{}{
		"user_id":    userID,
		"is_deleted": false,
	})
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, fmt.Errorf("failed to get existing answers: %v", err)
	}
	if existingAnswers == nil {
		existingAnswers = []entities.ProfilingQuestionAnswer{}
	}

	answerByNumber := make(map[int]*entities.ProfilingQuestionAnswer)
	for i := range existingAnswers {
		ans := existingAnswers[i]
		if ans.QuestionNumber >= constants.PROFILING_QUESTION_MIN_NUMBER && ans.QuestionNumber <= constants.PROFILING_QUESTION_MAX_NUMBER {
			existing, ok := answerByNumber[ans.QuestionNumber]
			if !ok || existing == nil || ans.CreatedOn.After(existing.CreatedOn) {
				answerByNumber[ans.QuestionNumber] = &existingAnswers[i]
			}
		}
	}

	if currentAns, ok := answerByNumber[questionNumber]; ok && currentAns != nil && currentAns.SelectedAnswer {
		s.txnManager.CommitTxn(tx)
		return &dtos.ProfilingQuestionModalResponseDTO{
			ShowModal:      false,
			QuestionNumber: questionNumber,
		}, nil
	}

	allAnswered := true
	for q := constants.PROFILING_QUESTION_MIN_NUMBER; q <= constants.PROFILING_QUESTION_MAX_NUMBER; q++ {
		ans, ok := answerByNumber[q]
		if !ok || ans == nil || !ans.SelectedAnswer {
			allAnswered = false
			break
		}
	}
	if allAnswered {
		s.txnManager.CommitTxn(tx)
		return &dtos.ProfilingQuestionModalResponseDTO{
			ShowModal:      false,
			QuestionNumber: questionNumber,
		}, nil
	}

	now := time.Now()

	var lastInteractionTime *time.Time
	lastInteractionQNum := 0
	for qNum, ans := range answerByNumber {
		if ans == nil {
			continue
		}
		var t time.Time
		if ans.LastViewedOn != nil {
			t = *ans.LastViewedOn
		} else {
			t = ans.CreatedOn
		}

		if lastInteractionTime == nil || t.After(*lastInteractionTime) {
			lastInteractionTime = &t
			lastInteractionQNum = qNum
		}
	}

	nextQNum := 0

	if lastInteractionTime == nil || lastInteractionQNum < constants.PROFILING_QUESTION_MIN_NUMBER || lastInteractionQNum > constants.PROFILING_QUESTION_MAX_NUMBER {
		for q := constants.PROFILING_QUESTION_MIN_NUMBER; q <= constants.PROFILING_QUESTION_MAX_NUMBER; q++ {
			ans, ok := answerByNumber[q]
			if !ok || ans == nil || !ans.SelectedAnswer {
				nextQNum = q
				break
			}
		}
	} else {
		if now.Sub(*lastInteractionTime) < constants.PROFILING_QUESTION_MODAL_COOLDOWN {
			s.txnManager.CommitTxn(tx)
			return &dtos.ProfilingQuestionModalResponseDTO{
				ShowModal:      false,
				QuestionNumber: questionNumber,
			}, nil
		}

		questionRange := constants.PROFILING_QUESTION_MAX_NUMBER - constants.PROFILING_QUESTION_MIN_NUMBER + 1
		startOffset := 1
		for i := 0; i < questionRange; i++ {
			offset := (lastInteractionQNum - constants.PROFILING_QUESTION_MIN_NUMBER + startOffset + i)
			modResult := offset % questionRange
			if modResult < 0 {
				modResult += questionRange
			}
			candidate := modResult + constants.PROFILING_QUESTION_MIN_NUMBER

			if candidate < constants.PROFILING_QUESTION_MIN_NUMBER || candidate > constants.PROFILING_QUESTION_MAX_NUMBER {
				continue
			}

			ans, ok := answerByNumber[candidate]
			if !ok || ans == nil || !ans.SelectedAnswer {
				nextQNum = candidate
				break
			}
		}
	}

	if nextQNum == 0 {
		s.txnManager.CommitTxn(tx)
		return &dtos.ProfilingQuestionModalResponseDTO{
			ShowModal:      false,
			QuestionNumber: questionNumber,
		}, nil
	}

	showModal := questionNumber == nextQNum

	s.txnManager.CommitTxn(tx)
	return &dtos.ProfilingQuestionModalResponseDTO{
		ShowModal:      showModal,
		QuestionNumber: questionNumber,
	}, nil
}

func (s *userService) addPointsForFirstAddress(ctx context.Context, tx *gorm.DB, userID string) error {
	return s.userRepo.UpdateFields(ctx, tx, userID, map[string]interface{}{
		"profile_percentage": gorm.Expr("profile_percentage + ?", 20),
	})
}

func (s *userService) addPointsForQuestionAnswer(ctx context.Context, tx *gorm.DB, userID string) error {
	// Get existing answers to check if this is the third question
	existingAnswers, err := s.questionAnswerRepo.FindByUserID(ctx, tx, userID)
	if err != nil {
		return fmt.Errorf("failed to get existing answers: %v", err)
	}

	// If this is the third question (2 existing + 1 new), add 10 to profile_percentage
	if len(existingAnswers) == 3 {
		return s.userRepo.UpdateFields(ctx, tx, userID, map[string]interface{}{
			"profile_percentage": gorm.Expr("profile_percentage + ?", 10),
		})
	}

	return nil
}

func (s *userService) checkAndAwardProfileCompletionBonus(ctx context.Context, tx *gorm.DB, userID string) error {
	user, err := s.userRepo.FindById(ctx, tx, uuid.MustParse(userID))
	if err != nil {
		return err
	}

	if user.ProfilePercentage >= 100 && !user.IsProfileComplete {
		if _, err := s.comicCoinService.UpdateBalance(ctx, tx, user.ID, constants.REWARD_PROFILE_COMPLETION); err != nil {
			return err
		}

		rewardProfileCompletion := constants.REWARD_PROFILE_COMPLETION
		if err := s.notificationService.SendPushNotification(ctx, tx, constants.NOTIFICATION_WIN_COIN, userID, &rewardProfileCompletion); err != nil {
			return err
		}

		if err := s.userRepo.UpdateFields(ctx, tx, user.ID, map[string]interface{}{"is_profile_complete": true}); err != nil {
			return err
		}
	}

	return nil
}

func (s *userService) WAUserDetails(ctx context.Context, userID string) (*dtos.WAUserDetailsResponse, error) {
	tx, err := s.txnManager.StartTxn()
	if err != nil {
		return nil, err
	}
	defer s.txnManager.RollbackOnPanic(tx)
	defer s.txnManager.AbortTxn(tx)

	user, err := s.userRepo.FindById(ctx, tx, uuid.MustParse(userID))
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("user not found")
	}

	var dob *string
	if user.DOB != nil {
		d := user.DOB.Format("2006-01-02")
		dob = &d
	}

	var avatarID int
	if user.AvatarID != nil {
		avatarID = *user.AvatarID
	}

	var avatarURL *string
	if user.AvatarID != nil {
		avatar, err := s.avatarService.GetAvatar(ctx, *user.AvatarID)
		if err != nil {
			// Log error but don't fail the whole request
			fmt.Printf("could not fetch avatar: %v", err)
		}
		if avatar != nil {
			avatarURL = &avatar.ImageURL
		}
	}

	// Fetch user rank and balance
	userRank, err := s.userRankRepo.FindOneWithConditions(ctx, tx, map[string]interface{}{"user_id": userID})
	if err != nil {
		// Log error but don't fail the whole request
		fmt.Printf("could not fetch user rank: %v", err)
	}

	// Fetch user wallet for current balance
	userWallet, err := s.userWalletRepo.FindOneWithConditions(ctx, tx, map[string]interface{}{"user_id": userID})
	if err != nil {
		// Log error but don't fail the whole request
		fmt.Printf("could not fetch user wallet: %v", err)
	}
	fmt.Printf("userWallet: %v", userWallet)
	logrus.Infof("userWallet: %v", userWallet)

	var rank int
	var coins int
	var currentBalance *int
	if userRank != nil {
		rank = int(userRank.UserRank)
	}
	if userWallet != nil {
		coins = userWallet.CurrentBalance
		currentBalance = &userWallet.CurrentBalance
	}

	var defaultAddressCity *string
	defaultAddress, err := s.addressRepo.FindOneWithConditions(ctx, tx, map[string]interface{}{"user_id": userID, "is_default": true})
	if err != nil {
		fmt.Printf("could not fetch default address: %v", err)
	}

	if defaultAddress != nil {
		city, err := s.cityRepo.FindByID(ctx, tx, defaultAddress.CityID)
		if err != nil {
			fmt.Printf("could not fetch city: %v", err)
		}
		if city != nil {
			defaultAddressCity = &city.Name
		}
	}

	// Fetch user question answers
	questionAnswers, err := s.questionAnswerRepo.FindSortedByUserID(ctx, tx, userID)
	if err != nil {
		// Log error but don't fail the whole request
		fmt.Printf("could not fetch user question answers: %v", err)
	}

	quesFlags := make([]bool, 6)
	if questionAnswers != nil {
		for i, answer := range questionAnswers {
			if i < 6 {
				quesFlags[i] = answer.SelectedAnswer
			}
		}
	}

	resp := &dtos.WAUserDetailsResponse{
		Name:                        derefString(user.Name),
		Email:                       derefString(user.Email),
		Gender:                      user.Gender,
		DOB:                         dob,
		ProfilePicture:              user.ProfilePicture,
		IsActive:                    user.IsActive,
		MainRole:                    "Basic", // Hardcoded as per example
		AvatarID:                    avatarID,
		AvatarURL:                   avatarURL,
		PhoneNumber:                 user.PhoneNumber,
		AlternateMobile:             user.AlternateMobile,
		CurrentBalance:              currentBalance,
		DefaultAddressCity:          defaultAddressCity,
		ProfileCompletionPercentage: user.ProfilePercentage,
		LastLanguageSelection:       user.LastLanguageSelection,
		Rank:                        rank,
		Coins:                       coins,
		IsCreator:                   false, // Needs to be implemented
		Ques1:                       quesFlags[0],
		Ques2:                       quesFlags[1],
		Ques3:                       quesFlags[2],
		Ques4:                       quesFlags[3],
		Ques5:                       quesFlags[4],
		Ques6:                       quesFlags[5],
	}

	s.txnManager.CommitTxn(tx)

	return resp, nil
}

func (s *userService) EditLastLanguage(ctx context.Context, userID string, languageID int) error {
	tx, err := s.txnManager.StartTxn()
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return errs.NewInternalServerError("Failed to start transaction", err)
	}
	defer s.txnManager.RollbackOnPanic(tx)

	// First verify if user exists
	user, err := s.userRepo.FindById(ctx, tx, uuid.MustParse(userID))
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return errs.NewInternalServerError("Failed to find user", err)
	}

	if user == nil {
		s.txnManager.AbortTxn(tx)
		return errs.NewNotFoundError("User not found", nil)
	}

	// Validate that the language exists and is active
	language, err := s.languageRepo.FindByID(ctx, tx, languageID)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return errs.NewInternalServerError("Failed to validate language", err)
	}

	if language == nil {
		s.txnManager.AbortTxn(tx)
		return errs.NewNotFoundError("Language not found", nil)
	}

	if !language.IsActive {
		s.txnManager.AbortTxn(tx)
		return errs.NewBadRequestError("Language is not active", nil)
	}

	updateFields := map[string]interface{}{
		"last_language_selection": languageID,
	}

	if err := s.userRepo.UpdateFields(ctx, tx, userID, updateFields); err != nil {
		s.txnManager.AbortTxn(tx)
		return errs.NewInternalServerError("Failed to update user language preference", err)
	}

	s.txnManager.CommitTxn(tx)
	return nil
}

func (s *userService) UpdateProfileComicCoinPopUp(ctx context.Context, userID string) error {
	tx, err := s.txnManager.StartTxn()
	if err != nil {
		return errs.NewInternalServerError("Failed to start transaction", err)
	}
	defer s.txnManager.RollbackOnPanic(tx)

	user, err := s.userRepo.FindById(ctx, tx, uuid.MustParse(userID))
	if err != nil {
		return errs.NewInternalServerError("Failed to find user", err)
	}

	if user == nil {
		return errs.NewNotFoundError("User not found", nil)
	}

	userUpdate := &entities.User{
		IsProfilePopUp: true,
	}

	if err := s.userRepo.Update(ctx, tx, userID, userUpdate); err != nil {
		return errs.NewInternalServerError("Failed to update user comic coin pop up", err)
	}

	s.txnManager.CommitTxn(tx)
	return nil
}
