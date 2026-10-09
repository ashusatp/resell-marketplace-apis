package jobs

import "github.com/google/uuid"

const KindProcessingListingImages = "precess_listing_images"

type ImageSource struct {
	OriginalKey string `json:"original_key"`
	FinalKey    string `json:"final_key"`
}

type ProcessImagepayload struct {
	ListingID    uuid.UUID     `json:"listing_id"`
	ImageSources []ImageSource `json:"image_sources"`
}
