package entities

// SystemStatus representa o status geral do sistema.
// @Description System status data
//
//	@Example {
//	  "OS": "Darwin",
//	  "CPU": {"Model": "Intel(R) Core(TM) i7", "Cores": 8, "Threads": 16, "Usage": "15%"},
//	  "Memory": {"Total": "16GB", "Available": "8GB", "Used": "8GB", "Percentage": "50%"},
//	  "GPU": {"Model": "AMD Radeon Pro", "Memory": "4GB", "Available": true},
//	  "Storage": {"Used": "200GB", "Total": "500GB", "Percentage": "40%"},
//	  "Server": {"Version": "1.0.0", "Active": true}
//	}
type SystemStatus struct {
	OS      string
	CPU     CPU
	Memory  Memory
	GPU     GPU
	Storage Storage
	Server  Server
}
