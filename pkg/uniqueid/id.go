package uniqueid

import (
	"gochat/config"
	"gochat/tools"
)

var (
	nodeId = config.Conf.Common.CommonNode.NodeId
)

func GetSeqId() int64 {
	return tools.GetSnowflakeId(nodeId)
}
