package claude

import (
	"bytes"
	"fmt"

	"gopkg.in/yaml.v3"
)

// marshalYAML 用 2 空格缩进序列化（对齐 npm/k8s/docker-compose 等主流 YAML 风格，
// yaml.Marshal 默认是 4 空格）。
func marshalYAML(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(v); err != nil {
		return nil, fmt.Errorf("claude: marshal yaml: %w", err)
	}
	_ = enc.Close()
	return buf.Bytes(), nil
}
