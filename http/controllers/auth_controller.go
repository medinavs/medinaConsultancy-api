package controllers

import (
	"medina-consultancy-api/database"
	"medina-consultancy-api/models"
	jwtPkg "medina-consultancy-api/pkg/jwt"
	"medina-consultancy-api/pkg/response"
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type AuthResponse struct {
	Token string       `json:"token"`
	User  UserResponse `json:"user"`
}

type UserResponse struct {
	ID    uint   `json:"id"`
	Email string `json:"email"`
}

func Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.SendGinResponse(c, http.StatusBadRequest, nil, nil, err.Error())
		return
	}

	var existingUser models.User
	if err := database.DB.Where("email = ?", req.Email).First(&existingUser).Error; err == nil {
		response.SendGinResponse(c, http.StatusConflict, nil, nil, "User already exists")
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		response.SendGinResponse(c, http.StatusInternalServerError, nil, nil, "Failed to hash password")
		return
	}

	user := models.User{
		Email:    req.Email,
		Password: string(hashedPassword),
	}

	if err := database.DB.Create(&user).Error; err != nil {
		response.SendGinResponse(c, http.StatusInternalServerError, nil, nil, "Failed to create user")
		return
	}

	token, err := jwtPkg.GenerateToken(user.ID, user.Email)
	if err != nil {
		response.SendGinResponse(c, http.StatusInternalServerError, nil, nil, "Failed to generate token")
		return
	}

	authResponse := AuthResponse{
		Token: token,
		User: UserResponse{
			ID:    user.ID,
			Email: user.Email,
		},
	}

	response.SendGinResponse(c, http.StatusCreated, authResponse, nil, "")
}

func Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.SendGinResponse(c, http.StatusBadRequest, nil, nil, err.Error())
		return
	}

	var user models.User
	if err := database.DB.Where("email = ?", req.Email).First(&user).Error; err != nil {
		response.SendGinResponse(c, http.StatusUnauthorized, nil, nil, "Invalid email or password")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		response.SendGinResponse(c, http.StatusUnauthorized, nil, nil, "Invalid email or password")
		return
	}

	token, err := jwtPkg.GenerateToken(user.ID, user.Email)
	if err != nil {
		response.SendGinResponse(c, http.StatusInternalServerError, nil, nil, "Failed to generate token")
		return
	}

	authResponse := AuthResponse{
		Token: token,
		User: UserResponse{
			ID:    user.ID,
			Email: user.Email,
		},
	}

	response.SendGinResponse(c, http.StatusOK, authResponse, nil, "")
}

func GetProfile(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		response.SendGinResponse(c, http.StatusUnauthorized, nil, nil, "User not authenticated")
		return
	}

	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		response.SendGinResponse(c, http.StatusNotFound, nil, nil, "User not found")
		return
	}

	userResponse := UserResponse{
		ID:    user.ID,
		Email: user.Email,
	}

	response.SendGinResponse(c, http.StatusOK, userResponse, nil, "")
}
