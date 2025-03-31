package _struct

type AudioFormat struct {
	ID           uint   `json:"id" gorm:"primaryKey"`
	Name         string `json:"name" gorm:"not null"`
	Extension    string `json:"extension" gorm:"not null"`
	MimeType     string `json:"mime_type" gorm:"not null"`
	BitRate      int    `json:"bit_rate"`
	IsLossless   bool   `json:"is_lossless" gorm:"default:false"`
	IsSupported  bool   `json:"is_supported" gorm:"default:true"`
	ConvertFrom  bool   `json:"convert_from" gorm:"default:false"`
	ConvertTo    bool   `json:"convert_to" gorm:"default:false"`
	DisplayOrder int    `json:"display_order" gorm:"default:0"`
}
