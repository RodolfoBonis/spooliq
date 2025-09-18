package entities

// CPU representa a entidade de processador (Central Processing Unit).
// @Description CPU data
// @Example {"Model": "Intel(R) Core(TM) i7", "Cores": 8, "Threads": 16, "Usage": "15%"}
type CPU struct {
	Model   string
	Cores   int32
	Threads int32
	Usage   string
}
