package RequestValidator

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

type ResponseBody struct {
	StatusCode int
	Message    string
	Body       struct{}
}

func BadRequest(c *gin.Context, Message string) {
	response := ResponseBody{
		StatusCode: http.StatusBadRequest,
		Message:    Message,
	}
	c.AbortWithStatusJSON(http.StatusBadRequest, response)
}

func UnprocessableEntity(c *gin.Context, Message string) {
	response := ResponseBody{
		StatusCode: http.StatusUnprocessableEntity,
		Message:    Message,
	}
	c.AbortWithStatusJSON(http.StatusUnprocessableEntity, response)
}

func SuccessResponse(c *gin.Context, Message string) {
	response := ResponseBody{
		StatusCode: http.StatusOK,
		Message:    Message,
	}
	c.JSON(http.StatusOK, response)
}

func requestBodyLogger(c *gin.Context) string {
	requestBody, _ := io.ReadAll(c.Request.Body)
	rdr1 := ioutil.NopCloser(bytes.NewBuffer(requestBody))
	rdr2 := ioutil.NopCloser(bytes.NewBuffer(requestBody))
	c.Request.Body = rdr2
	return readBody(rdr1)
}

func readBody(reader io.Reader) string {
	buf := new(bytes.Buffer)
	buf.ReadFrom(reader)
	s := buf.String()
	return s
}

func ValidateRequest() gin.HandlerFunc {
	return func(c *gin.Context) {
		var jsonData map[string]interface{}
		reqBody := requestBodyLogger(c)
		json.NewDecoder(c.Request.Body).Decode(&jsonData)
		//fmt.Println(reqBody)
		//fmt.Println("jsonData: ", jsonData)
		//Bind the incoming JSON to a map
		// if err := c.ShouldBindJSON(&jsonData); err != nil {
		// 	BadRequest(c, "Failed to bind JSON")
		// 	return
		// }
		// fmt.Printf("jsonData: %#v\n", jsonData)

		var validationErrors []string

		// Validate recursively
		if err := validateNested(jsonData, &validationErrors); err != nil {
			UnprocessableEntity(c, "Validation error")
			return
		}

		// If there are validation errors, return them
		if len(validationErrors) > 0 {
			//c.JSON(http.StatusUnprocessableEntity, gin.H{"errors": validationErrors})
			log.Error("@Validation error:", validationErrors)
			UnprocessableEntity(c, "invalid request")
			return
		}
		// If validation succeeds, set the validated data in context
		c.Set("reqBody", reqBody)
		c.Set("jsonData", jsonData)
		//SuccessResponse(c, "Validation successful")
		c.Next()
	}
}

func validateNested(input interface{}, validationErrors *[]string) error {
	switch v := input.(type) {
	case map[string]interface{}:
		return validateNestedMap(v, validationErrors)
	case []interface{}:
		return validateNestedArray(v, validationErrors)
	default:
		if !isValidGeneralFormat(input) {
			*validationErrors = append(*validationErrors, fmt.Sprintf("Invalid format for value '%v'", input))
		}
		return nil
	}
}

func validateNestedMap(input map[string]interface{}, validationErrors *[]string) error {
	for key, value := range input {
		if value == nil {
			continue // Skip validation for null values
		}
		if err := validateNested(value, validationErrors); err != nil {
			return err
		}
		if err := validateField(key, getStringValue(value), validationErrors); err != nil {
			return err
		}
	}
	return nil
}

func validateNestedArray(input []interface{}, validationErrors *[]string) error {
	for _, item := range input {
		if err := validateNested(item, validationErrors); err != nil {
			return err
		}
	}
	return nil
}

func isValidGeneralFormat(value interface{}) bool {
	switch v := value.(type) {
	case string:
		// Check if the string contains only alphanumeric, ., -, _
		matched, _ := regexp.MatchString(`^[ @/=a-zA-Z0-9\.\-_]*$`, v)
		return matched
	case int, int32, int64, float32, float64:
		// Numeric types, allow any numeric format
		return true
	default:
		// For other types (arrays, etc.), currently assume valid
		return true
	}
}

// getStringValue attempts to convert the input value to string
func getStringValue(value interface{}) string {
	if str, ok := value.(string); ok {
		return str
	}
	return fmt.Sprintf("%v", value) // Fallback to formatting as string
}

// validateField validates a field and appends errors to the provided slice
func validateField(key, value string, validationErrors *[]string) error {
	switch key {
	case "otp":
		if err := validateOTP(value); err != nil {
			*validationErrors = append(*validationErrors, err.Error())
		}
	case "mobile", "contact", "phone":
		if err := validateMobileFormat(value); err != nil {
			*validationErrors = append(*validationErrors, err.Error())
		}
	case "pan":
		if err := validatePanFormat(value); err != nil {
			*validationErrors = append(*validationErrors, err.Error())
		}
	case "email":
		if err := validateEmailFormat(value); err != nil {
			*validationErrors = append(*validationErrors, err.Error())
		}
	default:
		if strings.Contains(key, "id") {
			if err := validateIDFormat(value); err != nil {
				*validationErrors = append(*validationErrors, err.Error())
			}
		}
	}
	return nil
}

// validateMobileFormat validates mobile number format
func validateMobileFormat(mobile string) error {
	if matched, _ := regexp.MatchString(`^[0-9]{10}$`, mobile); !matched {
		return errors.New("invalid mobile number format")
	}
	return nil
}

// validatePanFormat validates PAN card number format
func validatePanFormat(pan string) error {
	if matched, _ := regexp.MatchString(`^[A-Z]{5}[0-9]{4}[A-Z]{1}$`, pan); !matched {
		return errors.New("invalid PAN format")
	}
	return nil
}

// validateEmailFormat validates email format
func validateEmailFormat(email string) error {
	if matched, _ := regexp.MatchString(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`, email); !matched {
		return errors.New("invalid email format")
	}
	return nil
}

// validateIDFormat validates ID format (alphanumeric)
func validateIDFormat(value string) error {
	if matched, _ := regexp.MatchString(`^[A-Za-z=0-9]*$`, value); !matched {
		return errors.New("invalid ID format, should be alphanumeric")
	}
	return nil
}

// validateOTP validates OTP format
func validateOTP(otp string) error {
	if matched, _ := regexp.MatchString(`^\d{6}$`, otp); !matched {
		return errors.New("invalid OTP format")
	}
	return nil
}
