package httpx

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOK(t *testing.T) {
	t.Run("string data", func(t *testing.T) {
		result := OK("Success", "test data")

		assert.True(t, result.Success)
		assert.Equal(t, http.StatusOK, result.StatusCode)
		assert.Equal(t, "Success", result.Message)
		assert.NotNil(t, result.Data)
		assert.Equal(t, "test data", *result.Data)
	})

	t.Run("struct data", func(t *testing.T) {
		data := struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		}{ID: 1, Name: "John"}

		result := OK("User created", data)

		assert.True(t, result.Success)
		assert.Equal(t, http.StatusOK, result.StatusCode)
		assert.Equal(t, "User created", result.Message)
		assert.NotNil(t, result.Data)
		assert.Equal(t, data, *result.Data)
	})
}

func TestFail(t *testing.T) {
	t.Run("bad request", func(t *testing.T) {
		result := Fail("Invalid input", http.StatusBadRequest)

		assert.False(t, result.Success)
		assert.Equal(t, http.StatusBadRequest, result.StatusCode)
		assert.Equal(t, "Invalid input", result.Message)
		assert.Nil(t, result.Data)
	})

	t.Run("unauthorized", func(t *testing.T) {
		result := Fail("Access denied", http.StatusUnauthorized)

		assert.False(t, result.Success)
		assert.Equal(t, http.StatusUnauthorized, result.StatusCode)
		assert.Equal(t, "Access denied", result.Message)
		assert.Nil(t, result.Data)
	})
}
