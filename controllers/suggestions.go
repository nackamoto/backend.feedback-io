package controllers

import (
	"errors"
	"fmt"
	"strconv"

	sql "feedback-io.backend/config"
	"feedback-io.backend/models"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func GetSuggestions(c *fiber.Ctx) error {
	var suggestions []models.Suggestion

	offset, err_offset := strconv.Atoi(c.Query("offset", "0"))
	limit, err_limit := strconv.Atoi(c.Query("limit", "10"))
	var count int64

	if err_offset != nil || err_limit != nil {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid offset parameter or limit parameter",
		})
	}

	// Get all suggestions
	// sql.DB.Model(&models.Suggestion{}).Limit(limit).Offset(offset).Find(&suggestions).Count(&count)
	// First get the total count
	if err := sql.DB.Model(&suggestions).Count(&count).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to fetch suggestions count",
		})
	}

	// Then get the paginated results
	if err := sql.DB.Limit(limit).Offset(offset).Find(&suggestions).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to fetch suggestions",
		})
	}
	return c.Status(200).JSON(fiber.Map{
		"success": true,
		"count":   count,
		"data":    suggestions,
	})
}

func GetSuggestion(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid suggestion ID",
		})
	}

	var suggestion models.Suggestion
	// sql.DB.Preload("comments").First(&suggestion, id)
	sql.DB.Where("id = ?", id).Preload("comments").First(&suggestion)

	if suggestion.Id == 0 {
		return c.Status(404).JSON(fiber.Map{
			"success": false,
			"error":   "Suggestion not found",
		})
	}

	return c.Status(200).JSON(fiber.Map{
		"success": true,
		"data":    &suggestion,
	})
}

func VoteSuggestion(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid suggestion ID",
		})
	}

	vote := c.Query("vote", "up")
	if vote != "up" && vote != "down" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid vote parameter: must be 'up' or 'down'",
		})
	}

	// Start transaction
	tx := sql.DB.Begin()
	committed := false

	defer func() {
		if r := recover(); r != nil {
			if !committed {
				tx.Rollback()
			}
			panic(r)
		} else if tx.Error != nil && !committed {
			tx.Rollback()
		}
	}()

	if tx.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to start transaction",
		})
	}

	// Use locking to prevent concurrent votes
	var suggestion models.Suggestion
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&suggestion, id).Error; err != nil {
		fmt.Printf("Error: %v\n", err)
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"success": false,
				"error":   "Suggestion not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to fetch suggestion",
		})
	}

	voteChange := 1
	if vote == "down" {
		voteChange = -1
	}

	if err := tx.Model(&suggestion).
		Where("id = ?", id).
		Update("votes", gorm.Expr("votes + ?", voteChange)).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to update votes",
		})
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to commit transaction",
		})
	}

	// Fetch updated suggestion
	if err := sql.DB.First(&suggestion, id).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to fetch updated suggestion",
		})
	}
	committed = true
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"data":    suggestion,
	})
}
