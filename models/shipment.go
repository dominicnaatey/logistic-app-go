package models

type Shipment struct {
    ID          string `json:"id"`
    Origin      string `json:"origin" binding:"required"`
    Destination string `json:"destination" binding:"required"`
    Status      string `json:"status"`
}
