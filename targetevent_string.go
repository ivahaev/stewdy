// Code generated by "stringer -type=TargetEvent"; DO NOT EDIT.

package stewdy

import "strconv"

const _TargetEvent_name = "EventOriginateEventAnswerEventConnectEventFail"

var _TargetEvent_index = [...]uint8{0, 14, 25, 37, 46}

func (i TargetEvent) String() string {
	i -= 1
	if i < 0 || i >= TargetEvent(len(_TargetEvent_index)-1) {
		return "TargetEvent(" + strconv.FormatInt(int64(i+1), 10) + ")"
	}
	return _TargetEvent_name[_TargetEvent_index[i]:_TargetEvent_index[i+1]]
}
