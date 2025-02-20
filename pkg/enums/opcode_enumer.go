// Code generated by "enumer -trimprefix=Opcode -type=Opcode -json -text"; DO NOT EDIT.

//
package enums

import (
	"encoding/json"
	"fmt"
)

const _OpcodeName = "NoneStepSleepWaitForEvent"

var _OpcodeIndex = [...]uint8{0, 4, 8, 13, 25}

func (i Opcode) String() string {
	if i < 0 || i >= Opcode(len(_OpcodeIndex)-1) {
		return fmt.Sprintf("Opcode(%d)", i)
	}
	return _OpcodeName[_OpcodeIndex[i]:_OpcodeIndex[i+1]]
}

var _OpcodeValues = []Opcode{0, 1, 2, 3}

var _OpcodeNameToValueMap = map[string]Opcode{
	_OpcodeName[0:4]:   0,
	_OpcodeName[4:8]:   1,
	_OpcodeName[8:13]:  2,
	_OpcodeName[13:25]: 3,
}

// OpcodeFromString retrieves an enum value from the enum constants string name.
// Throws an error if the param is not part of the enum.
func OpcodeFromString(s string) (Opcode, error) {
	if val, ok := _OpcodeNameToValueMap[s]; ok {
		return val, nil
	}
	return 0, fmt.Errorf("%s does not belong to Opcode values", s)
}

// OpcodeValues returns all values of the enum
func OpcodeValues() []Opcode {
	return _OpcodeValues
}

// IsAOpcode returns "true" if the value is listed in the enum definition. "false" otherwise
func (i Opcode) IsAOpcode() bool {
	for _, v := range _OpcodeValues {
		if i == v {
			return true
		}
	}
	return false
}

// MarshalJSON implements the json.Marshaler interface for Opcode
func (i Opcode) MarshalJSON() ([]byte, error) {
	return json.Marshal(i.String())
}

// UnmarshalJSON implements the json.Unmarshaler interface for Opcode
func (i *Opcode) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("Opcode should be a string, got %s", data)
	}

	var err error
	*i, err = OpcodeFromString(s)
	return err
}

// MarshalText implements the encoding.TextMarshaler interface for Opcode
func (i Opcode) MarshalText() ([]byte, error) {
	return []byte(i.String()), nil
}

// UnmarshalText implements the encoding.TextUnmarshaler interface for Opcode
func (i *Opcode) UnmarshalText(text []byte) error {
	var err error
	*i, err = OpcodeFromString(string(text))
	return err
}
