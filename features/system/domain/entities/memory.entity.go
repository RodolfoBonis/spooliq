package entities

// Memory representa a entidade de memória RAM do sistema.
// @Description Memory data
// @Example {"Total": "16GB", "Available": "8GB", "Used": "8GB", "Percentage": "50%"}
type Memory struct {
	Total      string
	Available  string
	Used       string
	Percentage string
}
