package entities

// Environment contém as configurações do ambiente atual.
type environmentsEntity struct {
	Development string
	Staging     string
	Production  string
}

// Environment é uma variável global que contém as configurações do ambiente atual.
var Environment = environmentsEntity{
	Development: "development",
	Staging:     "staging",
	Production:  "production",
}
