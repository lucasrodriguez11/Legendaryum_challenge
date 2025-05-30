package models

type ErrorResponse struct {
	Status  string `json:"status" example:"error"`               // Estado de la respuesta (generalmente "error")
	Message string `json:"message" example:"Detalles del error"` // Mensaje descriptivo del error
}

type SuccessResponse struct {
	Status  string `json:"status" example:"success"`            // Estado de la respuesta (generalmente "success")
	Message string `json:"message" example:"Operación exitosa"` // Mensaje descriptivo de la operación
}
