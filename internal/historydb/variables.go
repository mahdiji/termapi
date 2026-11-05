package historydb

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
)

func ExtractVariables(body, domainScope string) []CapturedVariable {
	decoder := json.NewDecoder(strings.NewReader(body))
	decoder.UseNumber()
	var document any
	if err := decoder.Decode(&document); err != nil {
		return nil
	}
	// Reject a second JSON value or trailing non-whitespace data.
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return nil
	}
	variables := make([]CapturedVariable, 0)
	var visit func(any, string, int)
	visit = func(value any, path string, depth int) {
		if len(variables) >= maxCapturedVariables || depth > maxVariableDepth {
			return
		}
		switch typed := value.(type) {
		case map[string]any:
			keys := make([]string, 0, len(typed))
			for key := range typed {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			for _, key := range keys {
				child := typed[key]
				childPath := key
				if path != "" {
					childPath = path + "." + key
				}
				visit(child, childPath, depth+1)
				if len(variables) >= maxCapturedVariables {
					return
				}
			}
		case []any:
			for index, child := range typed {
				visit(child, fmt.Sprintf("%s[%d]", path, index), depth+1)
				if len(variables) >= maxCapturedVariables {
					return
				}
			}
		case nil:
			return
		case string:
			variables = append(variables, CapturedVariable{DomainScope: domainScope, Name: variableName(path), Value: typed})
		case json.Number:
			variables = append(variables, CapturedVariable{DomainScope: domainScope, Name: variableName(path), Value: typed.String()})
		case bool:
			variables = append(variables, CapturedVariable{DomainScope: domainScope, Name: variableName(path), Value: fmt.Sprint(typed)})
		}
	}
	visit(document, "", 0)
	return variables
}

func variableName(path string) string {
	if path == "" {
		return "$"
	}
	return path
}

