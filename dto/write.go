package dto

type WriteInput struct {
	RoomNum int `json:"RoomNum" binding:"required"`
}
