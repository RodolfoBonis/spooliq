package entities

// GPU representa a entidade de placa gráfica (Graphics Processing Unit).
// @Description GPU data
// @Example {"Model": "AMD Radeon Pro", "Memory": "4GB", "Available": true, "Cores": 8}
type GPU struct {
	Model     string
	Memory    string
	Available bool
	Cores     int // Número de núcleos da GPU (quando disponível)
}
