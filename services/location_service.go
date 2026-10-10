package services

import (
	"context"
	"fmt"

	"github.com/Infinite-Locus-Product/coca_cola_backend/dtos"
	"github.com/Infinite-Locus-Product/coca_cola_backend/errors"
	"github.com/Infinite-Locus-Product/coca_cola_backend/repository"
	"github.com/Infinite-Locus-Product/coca_cola_backend/utils"
)

type LocationService interface {
	GetLocationByPincode(ctx context.Context, pincode int) (*dtos.LocationResponseDTO, error)
}

type locationService struct {
	txnManager  *utils.TransactionManager
	pinCodeRepo repository.PinCodeRepository
	cityRepo    repository.CityRepository
	stateRepo   repository.StateRepository
}

func NewLocationService(txnManager *utils.TransactionManager, pinCodeRepo repository.PinCodeRepository, cityRepo repository.CityRepository, stateRepo repository.StateRepository) LocationService {
	return &locationService{
		txnManager:  txnManager,
		pinCodeRepo: pinCodeRepo,
		cityRepo:    cityRepo,
		stateRepo:   stateRepo,
	}
}

func (s *locationService) GetLocationByPincode(ctx context.Context, pincode int) (*dtos.LocationResponseDTO, error) {
	tx, err := s.txnManager.StartTxn()
	if err != nil {
		return nil, err
	}
	defer s.txnManager.RollbackOnPanic(tx)

	pincodeDetails, err := s.pinCodeRepo.FindByPincode(ctx, tx, pincode)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, fmt.Errorf("failed to find pincode details: %w", err)
	}
	if pincodeDetails == nil {
		s.txnManager.AbortTxn(tx)
		return nil, errors.NewNotFoundError("pincode not found", nil)
	}

	city, err := s.cityRepo.FindByID(ctx, tx, pincodeDetails.CityID)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, fmt.Errorf("failed to find city details: %w", err)
	}
	if city == nil {
		return nil, errors.NewNotFoundError("city not found for the given pincode", nil)
	}

	state, err := s.stateRepo.FindByID(ctx, tx, city.StateID)
	if err != nil {
		s.txnManager.AbortTxn(tx)
		return nil, fmt.Errorf("failed to find state details: %w", err)
	}
	if state == nil {
		s.txnManager.AbortTxn(tx)
		return nil, errors.NewNotFoundError("state not found for the given pincode", nil)
	}

	s.txnManager.CommitTxn(tx)

	return &dtos.LocationResponseDTO{
		Pincode: pincodeDetails.Pincode,
		City:    city.Name,
		State:   state.Name,
	}, nil
}
