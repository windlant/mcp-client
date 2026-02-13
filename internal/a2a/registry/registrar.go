package registry

import (
	"sync"

	"github.com/windlant/protocol/types/agent_types"
)

// Registrar 管理本地的 AgentCard 缓存
type Registrar struct {
	mu     sync.RWMutex
	agents map[string]agent_types.AgentCard
}

// NewRegistrar 创建新的 Registrar
func NewRegistrar() *Registrar {
	return &Registrar{
		agents: make(map[string]agent_types.AgentCard),
	}
}

// Update 更新或添加一个 AgentCard
func (r *Registrar) Update(card agent_types.AgentCard) {
	if card.AgentID == "" {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.agents[card.AgentID] = card
}

// Remove 从缓存中移除指定 Agent
func (r *Registrar) Remove(agentID string) {
	if agentID == "" {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.agents, agentID)
}

// Get 获取指定 AgentCard，如果不存在则返回 false
func (r *Registrar) Get(agentID string) (agent_types.AgentCard, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	card, ok := r.agents[agentID]
	return card, ok
}

// List 返回当前缓存中所有 AgentCard 的切片
func (r *Registrar) List() []agent_types.AgentCard {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]agent_types.AgentCard, 0, len(r.agents))
	for _, c := range r.agents {
		out = append(out, c)
	}
	return out
}
