package repositories

import (
	"errors"

	"gorm.io/gorm"
	"main.go/models"
)

type IUserRepository interface {
	EnterRoom(content models.InputNum) (*models.InputNum, error)
	Find(model models.Message) (*models.Client, error)
	Create(model models.Client) error
	Delete(ID string) error
	DeleteAll() error
	Update(ID string, model *models.Client) error
	// Find(num models.InputNum) (*models.OutNum, error)
	// CreateRoomNum(content models.OutNum) (*models.OutNum, error)
	// FindClients(ID string) (*models.Client, error)
	// UpdateClients(ID string, newContent models.Client) (*models.Client, error)
}

type IUserMemoryRepository interface {
	FindMessageMemory(model models.Message) (*models.Client, error)
	FindMemories(ID string) (*[]models.Client, error)
	FindMemory(ID string) (*models.Client, error)
	CreateMemory(model models.Client) error
	DeleteMemory(ID string) error
	UpdateMemory(ID string, model *models.Client) error
}

type UserMemoryRepository struct {
	users []*models.Client
}

func NewUserMemoryRepository(users []*models.Client) IUserMemoryRepository {
	return &UserMemoryRepository{users: users}
}

func (r *UserMemoryRepository) FindMessageMemory(model models.Message) (*models.Client, error) {
	for _, v := range r.users {
		if v.RoomID == model.RoomID {
			if v.Conn == nil {
				return nil, errors.New("websocket connection is nil")
			}
			return v, nil
		}
	}
	return nil, errors.New("room not found")
}

func (r *UserMemoryRepository) FindMemories(ID string) (*[]models.Client, error) {
	clients := []models.Client{}
	for _, v := range r.users {
		if v.ID == ID {
			clients = append(clients, *v)
		}
	}
	if len(clients) == 0 {
		return nil, errors.New("user not found")
	} else {
		return &clients, nil
	}
}

func (r *UserMemoryRepository) FindMemory(ID string) (*models.Client, error) {
	for _, v := range r.users {
		if v.ID == ID {
			return v, nil
		}
	}
	return nil, errors.New("user not found")
}

func (r *UserMemoryRepository) CreateMemory(model models.Client) error {
	if model.Conn == nil {
		return errors.New("websocket connection is nil")
	}
	r.users = append(r.users, &model)
	return nil
}

func (r *UserMemoryRepository) DeleteMemory(ID string) error {
	var client *models.Client
	for _, v := range r.users {
		if v.ID == ID {
			client = v
		}
	}
	if client == nil {
		return errors.New("user not found")
	}

	newUsers := []*models.Client{}
	for _, v := range r.users {
		if v.ID != ID {
			newUsers = append(newUsers, v) // IDが一致しないユーザーのみ保持
		}
	}
	r.users = newUsers
	return nil
}

func (r *UserMemoryRepository) UpdateMemory(ID string, model *models.Client) error {
	var client *models.Client
	for _, v := range r.users {
		if v.ID == ID {
			client = v
		}
	}
	if client == nil {
		return errors.New("user not found")
	}

	for i, v := range r.users {
		if v.ID == ID {
			r.users[i] = model
			return nil
		}
	}
	return errors.New("update failed")
}

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) IUserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) EnterRoom(content models.InputNum) (*models.InputNum, error) {
	result := r.db.Create(&content)
	if result.Error != nil {
		if result.Error.Error() == "record not found" {
			return nil, errors.New("record not found")
		}
		return nil, result.Error
	}
	return &content, nil
}

func (r *UserRepository) Find(model models.Message) (*models.Client, error) {
	var client models.Client
	RoomID := model.RoomID
	result := r.db.First(&client, "room_id = ?", RoomID)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, errors.New("room not found")
	}
	if result.Error != nil {
		return nil, result.Error
	}
	return &client, nil
}

func (r *UserRepository) Create(model models.Client) error {
	result := r.db.Create(&model)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (r *UserRepository) Delete(ID string) error {
	var content models.Client
	result := r.db.Delete(&content, "id = ?", ID)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			// if result.Error.Error() == "record not found" {
			return errors.New("record not found")
		}
		return result.Error
	}
	return nil
}

func (r *UserRepository) DeleteAll() error {
	result := r.db.Exec("DELETE FROM clients") // テーブル名を指定
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (r *UserRepository) Update(ID string, model *models.Client) error {
	result := r.db.Model(&models.Client{}).Where("id = ?", ID).Updates(model)
	if result.Error != nil {
		if result.Error.Error() == "record not found" {
			return errors.New("record not found")
		}
		return result.Error
	}
	return nil
}
