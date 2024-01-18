package main

import (
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Item struct {
	Description string `json:"shortDescription"`
	Price       string `json:"price"`
}

type Receipt struct {
	Retailer string `json:"retailer"`
	Date     string `json:"purchaseDate"`
	Time     string `json:"purchaseTime"`
	Items    []Item `json:"items"`
	Total    string `json:"total"`
}

// Global map to track all receipt point values
var receiptPoints map[string]int

/*
Reads through a receipt object to determine its point value, saves the
receipt points to the global map, then returns the unique id for that
receipt's points.
*/
func scanReceipt(context *gin.Context) {
	var receipt Receipt

	// read the JSON from the request
	if err := context.BindJSON(&receipt); err != nil {
		context.IndentedJSON(
			http.StatusBadRequest,
			gin.H{"message": "Failed to bind the request's JSON to type: Receipt."},
		)
		return
	}

	// parse the receipt's total
	receiptTotal, receiptTotalError := strconv.ParseFloat(receipt.Total, 64)
	if receiptTotalError != nil {
		context.IndentedJSON(
			http.StatusBadRequest,
			gin.H{"message": "Failed to parse receipt total to float."},
		)
		return
	}

	// parse the receipt's date
	receiptDate, receiptDateError := time.Parse("2006-01-02", receipt.Date)
	if receiptDateError != nil {
		context.IndentedJSON(
			http.StatusBadRequest,
			gin.H{"message": "Failed to parse receipt purchaseDate."},
		)
		return
	}

	// parse the receipt's time
	receiptTime, receiptTimeError := time.Parse("15:04", receipt.Time)
	if receiptTimeError != nil {
		context.IndentedJSON(
			http.StatusBadRequest,
			gin.H{"message": "Failed to parse receipt purchaseTime."},
		)
		return
	}

	// Begin tallying points for the receipt
	var totalPoints int = 0

	// 1 point for every alphanumeric character in the retailer name
	var retailerAlphanumericChars []rune
	for _, char := range receipt.Retailer {
		if unicode.IsLetter(char) || unicode.IsDigit(char) {
			retailerAlphanumericChars = append(retailerAlphanumericChars, char)
		}
	}
	totalPoints += len(retailerAlphanumericChars)

	// 50 points if the total is a round dollar amount with no cents
	if math.Floor(receiptTotal) == receiptTotal {
		totalPoints += 50
	}

	// 25 points if the total is a multiple of 0.25
	if math.Mod(receiptTotal, 0.25) == 0 {
		totalPoints += 25
	}

	// 5 points for every two items on the receipt
	totalPoints += (5 * (len(receipt.Items) / 2))

	/* If the trimmed length of the item description is a multiple of 3,
	multiply the price by 0.2 and round up to the nearest integer.
	The result is the number of points earned */
	for _, item := range receipt.Items {
		trimmedDescLength := len(strings.TrimSpace(item.Description))
		if trimmedDescLength%3 == 0 {
			priceFloat, err := strconv.ParseFloat(item.Price, 64)
			if err != nil {
				context.IndentedJSON(
					http.StatusBadRequest,
					gin.H{"message": "Failed to parse price to float for item: " + item.Description},
				)
				return
			}
			totalPoints += int(math.Ceil(priceFloat * 0.2))
		}
	}

	// 6 points if the day in the purchase date is odd
	if (receiptDate.Day() % 2) != 0 {
		totalPoints += 6
	}

	// 10 points if the time of purchase is after 2:00pm and before 4:00pm
	if (receiptTime.Hour() == 14 && receiptTime.Minute() > 0) ||
		(receiptTime.Hour() > 14) && (receiptTime.Hour() < 16) {
		totalPoints += 10
	}

	uniqueID := uuid.New().String()
	receiptPoints[uniqueID] = totalPoints

	context.IndentedJSON(
		http.StatusCreated,
		gin.H{"id": uniqueID},
	)
}

// Retrieve a receipt's point count using its unique id.
func getPoints(context *gin.Context) {
	inputId := context.Param("id")
	points, exists := receiptPoints[inputId]

	if exists {
		context.IndentedJSON(
			http.StatusOK,
			gin.H{"points": points},
		)
	} else {
		context.IndentedJSON(
			http.StatusNotFound,
			gin.H{"message": "Points not found for that id."},
		)
	}
}

func main() {
	receiptPoints = make(map[string]int)
	router := gin.Default()
	router.POST("receipts/process", scanReceipt)
	router.GET("/receipts/:id/points", getPoints)
	router.Run("localhost:9090")
}
