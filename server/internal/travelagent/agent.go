package travelagent

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

type AgentConfig struct {
	APIKey  string
	Model   string
	BaseURL string
}

type TravelAdvisorAgent struct {
	chatModel model.BaseChatModel
	builder   *ContextBuilder
}

const (
	defaultSiliconFlowBaseURL = "https://api.siliconflow.cn/v1"
	defaultSiliconFlowModel   = "Qwen/Qwen2.5-72B-Instruct"
)

func NewTravelAdvisorAgent(ctx context.Context, config AgentConfig) (*TravelAdvisorAgent, error) {
	agent := &TravelAdvisorAgent{
		builder: NewContextBuilder(),
	}

	if strings.TrimSpace(config.APIKey) == "" {
		return nil, fmt.Errorf("missing SILICONFLOW_API_KEY")
	}

	modelName := strings.TrimSpace(config.Model)
	if modelName == "" {
		modelName = defaultSiliconFlowModel
	}

	baseURL := strings.TrimSpace(config.BaseURL)
	if baseURL == "" {
		baseURL = defaultSiliconFlowBaseURL
	}

	temperature := float32(0.4)
	chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		APIKey:      config.APIKey,
		Model:       modelName,
		BaseURL:     baseURL,
		Temperature: &temperature,
	})
	if err != nil {
		return nil, fmt.Errorf("create eino siliconflow chat model: %w", err)
	}

	agent.chatModel = chatModel
	return agent, nil
}

func (a *TravelAdvisorAgent) StreamAdvice(ctx context.Context, req AdviceRequest, emit func(StreamEvent) error) error {
	travelContext := a.builder.Build(ctx, req)
	if err := emit(StreamEvent{Type: "context", Data: travelContext}); err != nil {
		return err
	}

	messages := buildPromptMessages(req, travelContext)
	reader, err := a.chatModel.Stream(ctx, messages)
	if err != nil {
		return fmt.Errorf("stream from eino chat model: %w", err)
	}
	defer reader.Close()

	for {
		chunk, err := reader.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("receive eino stream chunk: %w", err)
		}
		if chunk.Content == "" {
			continue
		}
		if err := emit(StreamEvent{Type: "delta", Data: map[string]string{"content": chunk.Content}}); err != nil {
			return err
		}
	}

	return emit(StreamEvent{Type: "done"})
}

func buildPromptMessages(req AdviceRequest, travelContext TravelContext) []*schema.Message {
	contextJSON, _ := json.MarshalIndent(travelContext, "", "  ")
	messages := []*schema.Message{
		schema.SystemMessage(`你是“节气出行顾问”，任务是基于已注入的天气上下文、黄历上下文和风险规则，为用户的出行计划给出中文建议。

规则：
1. 天气、交通安全和用户身体状态优先级高于黄历。
2. 黄历只能作为传统文化参考，不能替代安全、医疗、交通和官方预警。
3. 如果目的地或出行日期缺失，先指出缺失项，再用一句话追问；不要虚构天气。
4. 建议要具体，包含总体判断、天气影响、黄历提示、装备/时间建议、备选方案。
5. “今天、明天、后天”等相对日期必须以出行决策上下文中的 currentTime、currentDate 和 timezone 为准。
6. 不要输出 JSON，不要暴露内部字段名。`),
		schema.SystemMessage("出行决策上下文：\n" + string(contextJSON)),
	}

	for _, history := range trimHistory(req.History, 8) {
		content := strings.TrimSpace(history.Content)
		if content == "" {
			continue
		}

		switch history.Role {
		case "assistant":
			messages = append(messages, schema.AssistantMessage(content, nil))
		default:
			messages = append(messages, schema.UserMessage(content))
		}
	}

	messages = append(messages, schema.UserMessage(strings.TrimSpace(req.Message)))
	return messages
}

func trimHistory(history []ChatMessage, max int) []ChatMessage {
	if len(history) <= max {
		return history
	}
	return history[len(history)-max:]
}
