package main

import (
    "bytes"
    "encoding/json"
    "io/ioutil"
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

    // Leer y loggear el body raw
    bodyBytes, err := ioutil.ReadAll(c.Request.Body)
    if err != nil {
        log.Printf("Error al leer el body: %v", err)
        c.JSON(http.StatusBadRequest, gin.H{"error": "Error al leer el request"})
        return
    }
    
    // Importante: Restaurar el body para que pueda ser leído nuevamente
    c.Request.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))
    
    // Loggear el body raw
    log.Printf("Body raw recibido: %s", string(bodyBytes))

    // Bind JSON data from the request body to the createRequest map
    if err := c.ShouldBindJSON(&createRequest); err != nil {
        log.Printf("Error al analizar el JSON: %v", err)
        c.JSON(http.StatusBadRequest, gin.H{"error": "Formato JSON inválido"})
        return
    }

    // Log the incoming JSON data for debugging purposes
    createRequestJSON, _ := json.Marshal(createRequest)
    log.Printf("Datos procesados: %s", string(createRequestJSON))

    // Verifica que los campos requeridos estén presentes
    requiredFields := []string{"description", "id_patient", "id_user"}
    for _, field := range requiredFields {
        if _, exists := createRequest[field]; !exists {
            log.Printf("Falta el campo requerido: %s", field)
            c.JSON(http.StatusBadRequest, gin.H{"error": "Falta el campo: " + field})
            return
        }
    }

    // Define la consulta SQL
    query := `INSERT INTO medical_records (description, id_patient, id_user) VALUES ($1, $2, $3) RETURNING id;`

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
        log.Printf("Error al preparar la consulta: %v", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al preparar la consulta"})
        return
    }

    // Send the query request to the query service
    queryServiceURL := "http://localhost:8001/query"
    resp, err := http.Post(queryServiceURL, "application/json", bytes.NewBuffer(queryBody))
    if err != nil {
        log.Printf("Error al conectar con el servicio de consulta: %v", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al conectar con el servicio de consulta"})
        return
    }
    defer resp.Body.Close()

    // Leer y loggear el body de la respuesta
    respBody, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        log.Printf("Error al leer la respuesta: %v", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al leer la respuesta"})
        return
    }
    
    // Loggear el body de la respuesta
    log.Printf("Respuesta del servicio de consulta: %s", string(respBody))
    
    // Restaurar el body para poder decodificarlo
    resp.Body = ioutil.NopCloser(bytes.NewBuffer(respBody))

    // Handle the response from the query service
    if resp.StatusCode != http.StatusOK {
        log.Printf("Error al crear el registro del paciente, respuesta del servidor: %s", resp.Status)
        c.JSON(resp.StatusCode, gin.H{"error": "Error al crear el registro del paciente"})
        return
    }

    var queryResponse map[string]interface{}
    if err := json.NewDecoder(resp.Body).Decode(&queryResponse); err != nil {
        log.Printf("Error al procesar la respuesta del servicio de consulta: %v", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al procesar la respuesta del servicio de consulta"})
        return
    }

    // Extraer el ID del registro creado usando la estructura correcta
    data, ok := queryResponse["data"].([]interface{})
    if !ok || len(data) == 0 {
        log.Printf("Error: no se recibió el ID del registro creado")
        c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo obtener el ID del registro creado"})
        return
    }

    firstRow, ok := data[0].(map[string]interface{})
    if !ok {
        log.Printf("Error: formato de respuesta inválido")
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Formato de respuesta inválido"})
        return
    }

    id, ok := firstRow["id"]
    if !ok {
        log.Printf("Error: ID no encontrado en la respuesta")
        c.JSON(http.StatusInternalServerError, gin.H{"error": "ID no encontrado en la respuesta"})
        return
    }

    // Respond with the created record ID
    c.JSON(http.StatusOK, gin.H{
        "message": "Registro creado exitosamente",
        "id": id,
    })
}