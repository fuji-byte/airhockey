package services

import (
	"main.go/models"
	"main.go/repositories"
)

type IUserService interface {
	// EnterRoom(WriteInput *dto.WriteInput) (*models.InputNum, error)
	// Find(num *dto.WriteInput) (*models.OutNum, error)
	Find(model models.Message) (*models.Client, error)
	Create(model *models.Client) error
	Delete(ID string) error
	Update(ID string, model *models.Client) error
	// CreateRoomNum() (*models.OutNum, error)
	// RoomClients(ID string, model *models.OutNum) error
}

type IUserMemoryService interface {
	FindMessageMemory(model models.Message) (*models.Client, error)
	FindMemories(clientID string) (*[]models.Client, error)
	FindMemory(clientID string) (*models.Client, error)
	// EnterRoomMemory(content models.InputNum) (*models.InputNum, error)
	CreateMemory(model *models.Client) error
	DeleteMemory(ID string) error
	UpdateMemory(ID string, model *models.Client) error
}

type UserService struct {
	repository repositories.IUserRepository
}

type UserMemoryService struct {
	memoryrepository repositories.IUserMemoryRepository
}

func NewUserService(repository repositories.IUserRepository) IUserService {
	return &UserService{repository: repository}
}

func NewUserMemoryService(memoryrepository repositories.IUserMemoryRepository) IUserMemoryService {
	return &UserMemoryService{memoryrepository: memoryrepository}
}

func (s *UserMemoryService) CreateMemory(model *models.Client) error {
	return s.memoryrepository.CreateMemory(*model)
}

func (s *UserService) Create(model *models.Client) error {
	return s.repository.Create(*model)
}

func (s *UserMemoryService) FindMessageMemory(model models.Message) (*models.Client, error) {
	content, err := s.memoryrepository.FindMessageMemory(model)
	if err != nil {
		return nil, err
	}
	return content, nil
}

func (s *UserService) Find(model models.Message) (*models.Client, error) {
	content, err := s.repository.Find(model)
	if err != nil {
		return nil, err
	}
	return content, nil
}
func (s *UserMemoryService) FindMemories(clientID string) (*[]models.Client, error) {
	content, err := s.memoryrepository.FindMemories(clientID)
	if err != nil {
		return nil, err
	}
	return content, nil
}
func (s *UserMemoryService) FindMemory(clientID string) (*models.Client, error) {
	content, err := s.memoryrepository.FindMemory(clientID)
	if err != nil {
		return nil, err
	}
	return content, nil
}

func (s *UserService) Delete(ID string) error {
	return s.repository.Delete(ID)
}
func (s *UserMemoryService) DeleteMemory(ID string) error {
	return s.memoryrepository.DeleteMemory(ID)
}

func (s *UserService) Update(ID string, model *models.Client) error {
	return s.repository.Update(ID, model)
}

func (s *UserMemoryService) UpdateMemory(ID string, model *models.Client) error {
	return s.memoryrepository.UpdateMemory(ID, model)
}
