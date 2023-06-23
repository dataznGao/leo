package constant

const Separator = "/"

const LogContent = "\"this is a log\""

const EnhanceInputPath = "enhance"

const (
	CommonPort  string = "9998"
	InjuredPort string = "9999"
)

type BingoFaultType int

const (
	ValueFault BingoFaultType = iota
	NullFault
	ExceptionShortcircuitFault
	ExceptionUncaughtFault
	ExceptionUnhandledFault
	AttributeShadowedFault
	AttributeReversoFault
	SwitchMissDefaultFault
	ConditionBorderFault
	ConditionInversedFault
	SyncFault
)

// CallGraph 根据数字区分调用图
var CallGraph = make(map[int]map[string]map[string]string)
