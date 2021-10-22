package cmd

import (
	"s3Gateway/model/yml"
	"time"
)

const (
	SlashSeparator      = "/"
	globalServerRegion  = "us-east-1"
	globalMaxSkewTime   = 15 * time.Minute // 15 minutes skew allowed.
	stsRequestBodyLimit = 10 * (1 << 20)   // 10 MiB
	requestInfo         = "requestInfo"
)

var GlobalConfig *yml.Config
