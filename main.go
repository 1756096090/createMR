package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	// Endpoint para crear un nuevo registro del paciente
	r.POST("/create", createPatientRecord)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	log.Printf("Servidor corriendo en :%s", port)
	r.Run(":" + port)
}
func createPatientRecord(c *gin.Context) {
	var createRequest map[string]interface{}

	// Bind JSON data from the request body to the createRequest map
	if err := c.ShouldBindJSON(&createRequest); err != nil {
		log.Printf("Error al analizar el JSON: %v", err) // Imprimir error en log
		c.JSON(http.StatusBadRequest, gin.H{"error": "Formato JSON inválido"})
		return
	}

	// Log the incoming JSON data for debugging purposes
	createRequestJSON, _ := json.Marshal(createRequest)
	log.Printf("Datos recibidos: %s", string(createRequestJSON))

	// Verifica que los campos requeridos estén presentes
	requiredFields := []string{"description", "id_patient", "id_user"}
	for _, field := range requiredFields {
		if _, exists := createRequest[field]; !exists {
			log.Printf("Falta el campo requerido: %s", field) // Imprimir error en log
			c.JSON(http.StatusBadRequest, gin.H{"error": "Falta el campo: " + field})
			return
		}
	}

	// Define la consulta SQL
	query := `INSERT INTO medical_records (description, id_patient, id_user) VALUES ($1, $2, $3)`

	args := []interface{}{
		createRequest["description"],
		createRequest["id_patient"],
		createRequest["id_user"],
	}

	queryRequest := map[string]interface{}{
		"sql":  query,
		"args": args,
	}

	// Marshal the query request into JSON format
	queryBody, err := json.Marshal(queryRequest)
	if err != nil {
		log.Printf("Error al preparar la consulta: %v", err) // Imprimir error en log
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al preparar la consulta"})
		return
	}

	// Send the query request to the query service
	queryServiceURL := "http://localhost:8001/query"
	resp, err := http.Post(queryServiceURL, "application/json", bytes.NewBuffer(queryBody))
	if err != nil {
		log.Printf("Error al conectar con el servicio de consulta: %v", err) // Imprimir error en log
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al conectar con el servicio de consulta"})
		return
	}
	defer resp.Body.Close()

	// Handle the response from the query service
	if resp.StatusCode != http.StatusOK {
		log.Printf("Error al crear el registro del paciente, respuesta del servidor: %s", resp.Status) // Imprimir error en log
		c.JSON(resp.StatusCode, gin.H{"error": "Error al crear el registro del paciente"})
		return
	}

	var queryResponse map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&queryResponse); err != nil {
		log.Printf("Error al procesar la respuesta del servicio de consulta: %v", err) // Imprimir error en log
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al procesar la respuesta del servicio de consulta"})
		return
	}

	// Respond with a success message
	c.JSON(http.StatusOK, gin.H{"message": "Registro creado exitosamente"})
}
