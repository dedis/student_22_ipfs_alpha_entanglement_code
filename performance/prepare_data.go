package performance

var alpha, s, p = 3, 5, 5

type FileInfo struct {
	FileCID    string
	MetaCID    string
	TotalBlock int
}

type Metadata struct {
	Alpha int
	S     int
	P     int

	RootCID string

	DataCIDIndexMap map[string]int
	ParityCIDs      [][]string
}

type PerfResult struct {
	PartialSuccessCnt int
	FullSuccessCnt    float32
	RecoverRate       float32
	DownloadParity    float32
	Err               error
}

var InfoMap = map[string]FileInfo{
	"5MB": {
		FileCID:    "QmPhZDvWNwiLjYdMc5Kpijdgiza9ZC1qWFyUFcu6hZVx4w",
		MetaCID:    "QmWePBkj7UbisSXn3KzB24Uyh5EnpPm7SP2Y2o3suq8fMA",
		TotalBlock: 21,
	},
	"20MB": {
		FileCID:    "QmY4ShSx49sYCCZxpqQWMcbjv2hv4yWWp2yprrk53DPqvT",
		MetaCID:    "QmeZmAZ7TiFRT7kqoV4oynn1STKwo8sbiwRnLsM21eZMCc",
		TotalBlock: 81,
	},
	"25MB": {
		FileCID:    "QmNkkcM5tFMqWxdrekyZoJnF5QxWKZnqYdJFBUj1jssRhb",
		MetaCID:    "QmcnV4N1umtzBRk5fC6e8TYkTKhgFkwqxN6LjPZvTworwZ",
		TotalBlock: 101,
	},
	"125MB": {
		FileCID:    "QmRLtPBz7u4V4Pz7qU5ofArnGrEugSsHAMSER51EYRvzoe",
		MetaCID:    "QmQMW8UQyvATTRYdosw9gWyyq1xqEi2Kqv9YV4wycFrF3p",
		TotalBlock: 504,
	},
}
