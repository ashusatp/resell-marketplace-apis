package handlers

type CreateListingRequest struct {
	Title       string `json:"title" binding:"required"`
	Description string `json:"description" binding:"required"`
	Price       int64  `json:"price" binding:"required"`
	City        string `json:"city" binding:"required"`
}

type CreateListingResponse struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Created string `json:"created_at"`
}
