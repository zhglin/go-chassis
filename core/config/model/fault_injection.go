package model

import "time"

// FaultProtocolStruct fault protocol struct
type FaultProtocolStruct struct {
	Fault map[string]Fault `yaml:"protocols"`
}

// Fault fault struct
type Fault struct {
	Abort Abort `yaml:"abort"` // 终止
	Delay Delay `yaml:"delay"` // 延迟
}

// Abort abort struct
type Abort struct {
	Percent    int `yaml:"percent"`    // 百分比
	HTTPStatus int `yaml:"httpStatus"` // 返回的code
}

// Delay delay struct
type Delay struct {
	Percent    int           `yaml:"percent"`    // 百分比
	FixedDelay time.Duration `yaml:"fixedDelay"` // 延迟时间
}
