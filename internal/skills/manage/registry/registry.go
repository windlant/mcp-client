package registry

import (
	"github.com/windlant/protocol/types/skill_types"
)

// Registry 管理已注册的技能定义
type Registry struct {
	skills map[string]skill_types.SkillDefinition
}

// NewRegistry 创建一个新的技能注册表
func NewRegistry() *Registry {
	return &Registry{
		skills: make(map[string]skill_types.SkillDefinition),
	}
}

// Register 注册一个技能定义（以技能名称为键）
func (r *Registry) Register(def skill_types.SkillDefinition) {
	r.skills[def.Name] = def
}

// Get 根据名称获取技能定义，若不存在则返回 false
func (r *Registry) Get(name string) (skill_types.SkillDefinition, bool) {
	def, ok := r.skills[name]
	return def, ok
}

// ListAll 返回所有已注册的技能定义列表
func (r *Registry) ListAll() []skill_types.SkillDefinition {
	defs := make([]skill_types.SkillDefinition, 0, len(r.skills))
	for _, def := range r.skills {
		defs = append(defs, def)
	}
	return defs
}
