// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
package messageprocessor

import (
	"errors"
	"fmt"

	lp2 "github.com/ClusterCockpit/cc-lib/v2/ccMessage"
	units "github.com/ClusterCockpit/cc-lib/v2/ccUnits"
	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
)

type MessageLocation int

const (
	MESSAGE_LOCATION_TAGS MessageLocation = iota
	MESSAGE_LOCATION_META
	MESSAGE_LOCATION_FIELDS
)

// Abstract function to move entries from one location to another
func moveInMessage(message lp2.CCMessage, params *map[string]any, checks *map[*vm.Program]messageProcessorTagConfig, from, to MessageLocation) (bool, error) {
	for d, data := range *checks {
		value, err := expr.Run(d, *params)
		if err != nil {
			return false, fmt.Errorf("failed to evaluate: %v", err.Error())
		}
		// cclog.ComponentDebug("MessageProcessor", "Move from", from, "to", to)
		if value.(bool) {
			var v string
			ok := false
			switch from {
			case MESSAGE_LOCATION_TAGS:
				// cclog.ComponentDebug("MessageProcessor", "Getting tag key", data.Key)
				v, ok = message.GetTag(data.Key)
			case MESSAGE_LOCATION_META:
				// cclog.ComponentDebug("MessageProcessor", "Getting meta key", data.Key)
				// cclog.ComponentDebug("MessageProcessor", message.Meta())
				v, ok = message.GetMeta(data.Key)
			case MESSAGE_LOCATION_FIELDS:
				var x any
				// cclog.ComponentDebug("MessageProcessor", "Getting field key", data.Key)
				x, ok = message.GetField(data.Key)
				v = fmt.Sprintf("%v", x)
			}
			if ok {
				switch from {
				case MESSAGE_LOCATION_TAGS:
					// cclog.ComponentDebug("MessageProcessor", "Removing tag key", data.Key)
					message.RemoveTag(data.Key)
					delete((*params)["tag"].(map[string]any), data.Key)
					delete((*params)["tags"].(map[string]any), data.Key)
				case MESSAGE_LOCATION_META:
					// cclog.ComponentDebug("MessageProcessor", "Removing meta key", data.Key)
					message.RemoveMeta(data.Key)
					delete((*params)["meta"].(map[string]any), data.Key)
				case MESSAGE_LOCATION_FIELDS:
					// cclog.ComponentDebug("MessageProcessor", "Removing field key", data.Key)
					message.RemoveField(data.Key)
					delete((*params)["field"].(map[string]any), data.Key)
					delete((*params)["fields"].(map[string]any), data.Key)
				}
				switch to {
				case MESSAGE_LOCATION_TAGS:
					// cclog.ComponentDebug("MessageProcessor", "Adding tag", data.Value, "->", v)
					message.AddTag(data.Value, v)
					(*params)["tag"].(map[string]any)[data.Value] = v
					(*params)["tags"].(map[string]any)[data.Value] = v
				case MESSAGE_LOCATION_META:
					// cclog.ComponentDebug("MessageProcessor", "Adding meta", data.Value, "->", v)
					message.AddMeta(data.Value, v)
					(*params)["meta"].(map[string]any)[data.Value] = v
				case MESSAGE_LOCATION_FIELDS:
					// cclog.ComponentDebug("MessageProcessor", "Adding field", data.Value, "->", v)
					message.AddField(data.Value, v)
					(*params)["field"].(map[string]any)[data.Value] = v
					(*params)["fields"].(map[string]any)[data.Value] = v
				}
			}
		}
	}
	return false, nil
}

func deleteIf(message lp2.CCMessage, params *map[string]any, checks *map[*vm.Program]messageProcessorTagConfig, location MessageLocation) (bool, error) {
	for d, data := range *checks {
		value, err := expr.Run(d, *params)
		if err != nil {
			return true, fmt.Errorf("failed to evaluate: %v", err.Error())
		}
		if value.(bool) {
			switch location {
			case MESSAGE_LOCATION_FIELDS:
				switch data.Key {
				case "value", "event", "log", "control":
					return false, errors.New("cannot delete protected fields")
				default:
					// cclog.ComponentDebug("MessageProcessor", "Removing field for", data.Key)
					message.RemoveField(data.Key)
					delete((*params)["field"].(map[string]any), data.Key)
					delete((*params)["fields"].(map[string]any), data.Key)
				}
			case MESSAGE_LOCATION_TAGS:
				// cclog.ComponentDebug("MessageProcessor", "Removing tag for", data.Key)
				message.RemoveTag(data.Key)
				delete((*params)["tag"].(map[string]any), data.Key)
				delete((*params)["tags"].(map[string]any), data.Key)
			case MESSAGE_LOCATION_META:
				// cclog.ComponentDebug("MessageProcessor", "Removing meta for", data.Key)
				message.RemoveMeta(data.Key)
				delete((*params)["meta"].(map[string]any), data.Key)
			}
		}
	}
	return false, nil
}

func addIf(message lp2.CCMessage, params *map[string]any, checks *map[*vm.Program]messageProcessorTagConfig, location MessageLocation) (bool, error) {
	for d, data := range *checks {
		value, err := expr.Run(d, *params)
		if err != nil {
			return true, fmt.Errorf("failed to evaluate: %v", err.Error())
		}
		if value.(bool) {
			switch location {
			case MESSAGE_LOCATION_FIELDS:
				// cclog.ComponentDebug("MessageProcessor", "Adding field", data.Value, "->", data.Value)
				message.AddField(data.Key, data.Value)
				(*params)["field"].(map[string]any)[data.Key] = data.Value
				(*params)["fields"].(map[string]any)[data.Key] = data.Value
			case MESSAGE_LOCATION_TAGS:
				// cclog.ComponentDebug("MessageProcessor", "Adding tag", data.Value, "->", data.Value)
				message.AddTag(data.Key, data.Value)
				(*params)["tag"].(map[string]any)[data.Key] = data.Value
				(*params)["tags"].(map[string]any)[data.Key] = data.Value
			case MESSAGE_LOCATION_META:
				// cclog.ComponentDebug("MessageProcessor", "Adding meta", data.Value, "->", data.Value)
				message.AddMeta(data.Key, data.Value)
				(*params)["meta"].(map[string]any)[data.Key] = data.Value
			}
		}
	}
	return false, nil
}

func deleteTagIf(message lp2.CCMessage, params *map[string]any, checks *map[*vm.Program]messageProcessorTagConfig) (bool, error) {
	return deleteIf(message, params, checks, MESSAGE_LOCATION_TAGS)
}

func addTagIf(message lp2.CCMessage, params *map[string]any, checks *map[*vm.Program]messageProcessorTagConfig) (bool, error) {
	return addIf(message, params, checks, MESSAGE_LOCATION_TAGS)
}

func moveTagToMeta(message lp2.CCMessage, params *map[string]any, checks *map[*vm.Program]messageProcessorTagConfig) (bool, error) {
	return moveInMessage(message, params, checks, MESSAGE_LOCATION_TAGS, MESSAGE_LOCATION_META)
}

func moveTagToField(message lp2.CCMessage, params *map[string]any, checks *map[*vm.Program]messageProcessorTagConfig) (bool, error) {
	return moveInMessage(message, params, checks, MESSAGE_LOCATION_TAGS, MESSAGE_LOCATION_FIELDS)
}

func deleteMetaIf(message lp2.CCMessage, params *map[string]any, checks *map[*vm.Program]messageProcessorTagConfig) (bool, error) {
	return deleteIf(message, params, checks, MESSAGE_LOCATION_META)
}

func addMetaIf(message lp2.CCMessage, params *map[string]any, checks *map[*vm.Program]messageProcessorTagConfig) (bool, error) {
	return addIf(message, params, checks, MESSAGE_LOCATION_META)
}

func moveMetaToTag(message lp2.CCMessage, params *map[string]any, checks *map[*vm.Program]messageProcessorTagConfig) (bool, error) {
	return moveInMessage(message, params, checks, MESSAGE_LOCATION_META, MESSAGE_LOCATION_TAGS)
}

func moveMetaToField(message lp2.CCMessage, params *map[string]any, checks *map[*vm.Program]messageProcessorTagConfig) (bool, error) {
	return moveInMessage(message, params, checks, MESSAGE_LOCATION_META, MESSAGE_LOCATION_FIELDS)
}

func deleteFieldIf(message lp2.CCMessage, params *map[string]any, checks *map[*vm.Program]messageProcessorTagConfig) (bool, error) {
	return deleteIf(message, params, checks, MESSAGE_LOCATION_FIELDS)
}

func addFieldIf(message lp2.CCMessage, params *map[string]any, checks *map[*vm.Program]messageProcessorTagConfig) (bool, error) {
	return addIf(message, params, checks, MESSAGE_LOCATION_FIELDS)
}

func moveFieldToTag(message lp2.CCMessage, params *map[string]any, checks *map[*vm.Program]messageProcessorTagConfig) (bool, error) {
	return moveInMessage(message, params, checks, MESSAGE_LOCATION_FIELDS, MESSAGE_LOCATION_TAGS)
}

func moveFieldToMeta(message lp2.CCMessage, params *map[string]any, checks *map[*vm.Program]messageProcessorTagConfig) (bool, error) {
	return moveInMessage(message, params, checks, MESSAGE_LOCATION_FIELDS, MESSAGE_LOCATION_META)
}

func dropMessagesIf(params *map[string]any, checks *map[*vm.Program]struct{}) (bool, error) {
	for d := range *checks {
		value, err := expr.Run(d, *params)
		if err != nil {
			return false, fmt.Errorf("failed to evaluate: %v", err.Error())
		}
		if value.(bool) {
			return true, nil
		}
	}
	return false, nil
}

func normalizeUnits(message lp2.CCMessage, params *map[string]any) (bool, error) {
	if in_unit, ok := message.GetMeta("unit"); ok {
		u := units.NewUnit(in_unit)
		if u.Valid() {
			// cclog.ComponentDebug("MessageProcessor", "Update unit with", u.Short())
			message.AddMeta("unit", u.Short())
			(*params)["meta"].(map[string]any)["unit"] = u.Short()
		}
	} else if in_unit, ok := message.GetTag("unit"); ok {
		u := units.NewUnit(in_unit)
		if u.Valid() {
			// cclog.ComponentDebug("MessageProcessor", "Update unit with", u.Short())
			message.AddTag("unit", u.Short())
			(*params)["meta"].(map[string]any)["unit"] = u.Short()
		}
	}
	return false, nil
}

func changeUnitPrefix(message lp2.CCMessage, params *map[string]any, checks *map[*vm.Program]string) (bool, error) {
	for r, n := range *checks {
		value, err := expr.Run(r, *params)
		if err != nil {
			return false, fmt.Errorf("failed to evaluate: %v", err.Error())
		}
		if value.(bool) {
			newPrefix := units.NewPrefix(n)
			// cclog.ComponentDebug("MessageProcessor", "Condition matches, change to prefix", newPrefix.String())
			if in_unit, ok := message.GetMeta("unit"); ok && newPrefix != units.InvalidPrefix {
				u := units.NewUnit(in_unit)
				if u.Valid() {
					// cclog.ComponentDebug("MessageProcessor", "Input unit", u.Short())
					conv, out_unit := units.GetUnitPrefixFactor(u, newPrefix)
					if conv != nil && out_unit.Valid() {
						if val, ok := message.GetField("value"); ok {
							// cclog.ComponentDebug("MessageProcessor", "Update unit with", out_unit.Short())
							newv := conv(val)
							message.AddField("value", newv)
							(*params)["field"].(map[string]any)["value"] = newv
							(*params)["fields"].(map[string]any)["value"] = newv
							message.AddMeta("unit", out_unit.Short())
							(*params)["meta"].(map[string]any)["unit"] = u.Short()
						}
					}
				}

			} else if in_unit, ok := message.GetTag("unit"); ok && newPrefix != units.InvalidPrefix {
				u := units.NewUnit(in_unit)
				if u.Valid() {
					// cclog.ComponentDebug("MessageProcessor", "Input unit", u.Short())
					conv, out_unit := units.GetUnitPrefixFactor(u, newPrefix)
					if conv != nil && out_unit.Valid() {
						if val, ok := message.GetField("value"); ok {
							// cclog.ComponentDebug("MessageProcessor", "Update unit with", out_unit.Short())
							newv := conv(val)
							message.AddField("value", newv)
							(*params)["field"].(map[string]any)["value"] = newv
							(*params)["fields"].(map[string]any)["value"] = newv

							message.AddTag("unit", out_unit.Short())
							(*params)["meta"].(map[string]any)["unit"] = u.Short()
						}
					}
				}

			}
		}
	}
	return false, nil
}

func renameMessagesIf(message lp2.CCMessage, params *map[string]any, checks *map[*vm.Program]string) (bool, error) {
	for d, n := range *checks {
		value, err := expr.Run(d, *params)
		if err != nil {
			return true, fmt.Errorf("failed to evaluate: %v", err.Error())
		}
		if value.(bool) {
			old := message.Name()
			// cclog.ComponentDebug("MessageProcessor", "Rename to", n)
			message.SetName(n)
			(*params)["name"] = n
			// cclog.ComponentDebug("MessageProcessor", "Add old name as 'oldname' to meta", old)
			message.AddMeta("oldname", old)
			(*params)["meta"].(map[string]any)["oldname"] = old
		}
	}
	return false, nil
}
