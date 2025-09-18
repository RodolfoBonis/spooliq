package entities

// Storage representa a entidade de armazenamento do sistema.
// @Description Storage data
// @Example {"Used": "200GB", "Total": "500GB", "Percentage": "40%"}
type Storage struct {
	Used       string
	Total      string
	Percentage string
}
