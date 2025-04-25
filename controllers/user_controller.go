package controllers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	_struct "music-conveyor/models/struct"
	"music-conveyor/platform/database"
)

type UserCreateRequest struct {
	ID        uint      `json:"id"`
	Username  string    `json:"username" binding:"required"`
	Email     string    `json:"email" binding:"required,email"`
	Role      string    `json:"role" binding:"required"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	IsDeleted bool      `json:"isDeleted"`
}

type UserController struct {
	db *gorm.DB
}

func NewUserController() *UserController {
	return &UserController{
		db: database.DB,
	}
}

func (c *UserController) CreateOrUpdateUser(ctx *gin.Context) {
	var request UserCreateRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format: " + err.Error(),
		})
		return
	}

	user := _struct.User{
		ID:        request.ID,
		Username:  request.Username,
		Email:     request.Email,
		Role:      request.Role,
		CreatedAt: request.CreatedAt,
		UpdatedAt: request.UpdatedAt,
	}

	if user.CreatedAt.IsZero() {
		user.CreatedAt = time.Now()
	}

	if user.UpdatedAt.IsZero() {
		user.UpdatedAt = time.Now()
	}

	if request.IsDeleted {
		if err := c.db.Delete(&user).Error; err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to delete user: " + err.Error(),
			})
			return
		}
		ctx.JSON(http.StatusOK, gin.H{
			"message": "User successfully marked as deleted",
			"user":    user,
		})
		return
	}

	var existingUser _struct.User
	if err := c.db.Where("email = ? AND id != ?", user.Email, user.ID).First(&existingUser).Error; err == nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "User with this email already exists",
		})
		return
	}

	err := c.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&user).Error; err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to save user: " + err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "User successfully saved",
		"user":    user,
	})
}

func (c *UserController) GetUser(ctx *gin.Context) {
	var user _struct.User
	id := ctx.Param("id")

	if err := c.db.First(&user, id).Error; err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{
			"error": "User not found",
		})
		return
	}

	ctx.JSON(http.StatusOK, user)
}

func (c *UserController) DeleteUser(ctx *gin.Context) {
	var user _struct.User
	id := ctx.Param("id")

	if err := c.db.First(&user, id).Error; err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{
			"error": "User not found",
		})
		return
	}

	if err := c.db.Delete(&user).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to delete user: " + err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "User successfully deleted",
	})
}
