package utils

var (
	Configuration *config
)

func FillConfig(endpoint string, relayEnabled bool, allowInsecure bool) {
	Configuration = &config{
		Endpoint:      endpoint,
		RelayEnabled:  relayEnabled,
		AllowInsecure: allowInsecure,
	}
}

type config struct {
	Endpoint               string
	RelayEnabled           bool
	AllowInsecure          bool
	ShouldOverWriteAddress bool
	OverWriteAddress       string
}
