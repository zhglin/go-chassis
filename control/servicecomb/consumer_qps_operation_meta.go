package servicecomb

import (
	"strings"

	"github.com/go-chassis/go-chassis/v2/core/common"
)

// ConsumerKeys contain consumer keys
// 限流支持的不同级别的key
type ConsumerKeys struct {
	MicroServiceName       string
	SchemaQualifiedName    string
	OperationQualifiedName string
}

// ProviderKeys contain provider keys
type ProviderKeys struct {
	Global          string
	ServiceOriented string
}

//Prefix is const
const Prefix = "cse.flowcontrol"

/*
cse:
  flowcontrol:
    Consumer:
      qps:
        enabled: true
        limit:
          Server: 100
          Server.schemaId.operationId: 100

    Provider:
      qps:
        enabled: true  # enable rate limiting or not
        global:
          limit: 2   # default limit of provider
		limit:
		  sourceServiceName:1

// cse.flowcontrol."+serviceType+".qps.enabled
*/

// GetConsumerKey get specific key for consumer
// 生成flowControl的前缀key
func GetConsumerKey(sourceName, serviceName, schemaID, OperationID string) *ConsumerKeys {
	keys := new(ConsumerKeys)
	//for mesher to govern
	//来源的service
	if sourceName != "" {
		keys.MicroServiceName = strings.Join([]string{Prefix, sourceName, common.Consumer, "qps.limit", serviceName}, ".")
	} else {
		if serviceName != "" {
			keys.MicroServiceName = strings.Join([]string{Prefix, common.Consumer, "qps.limit", serviceName}, ".")
		}
	}
	if schemaID != "" {
		keys.SchemaQualifiedName = strings.Join([]string{keys.MicroServiceName, schemaID}, ".")
	}
	if OperationID != "" {
		keys.OperationQualifiedName = strings.Join([]string{keys.SchemaQualifiedName, OperationID}, ".")
	}
	return keys
}

// GetProviderKey get specific key for provider
// provider的限流 key是来源的sourceService
func GetProviderKey(sourceServiceName string) *ProviderKeys {
	keys := &ProviderKeys{}
	if sourceServiceName != "" {
		keys.ServiceOriented = strings.Join([]string{Prefix, common.Provider, "qps.limit", sourceServiceName}, ".")
	}

	keys.Global = strings.Join([]string{Prefix, common.Provider, "qps.global.limit"}, ".")
	return keys
}

// GetSchemaQualifiedName get schema qualified name
func (op *ConsumerKeys) GetSchemaQualifiedName() string {
	return op.SchemaQualifiedName
}

// GetMicroServiceSchemaOpQualifiedName get micro-service schema operation qualified name
func (op *ConsumerKeys) GetMicroServiceSchemaOpQualifiedName() string {
	return op.OperationQualifiedName
}

// GetMicroServiceName get micro-service name
func (op *ConsumerKeys) GetMicroServiceName() string {
	return op.MicroServiceName
}
