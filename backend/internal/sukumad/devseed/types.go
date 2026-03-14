package devseed

const (
	DemoSeedTag      = "sukumad-demo-v1"
	DemoServerPrefix = "demo-dhis2-"
)

type Summary struct {
	SeedTag        string
	ServersSeeded  int
	RequestsSeeded int
}
