package anuneko

import (
	"fmt"
	"github.com/kohmebot/pkg/chain"
	"github.com/kohmebot/plugin/v2"
	zero "github.com/wdvxdr1123/ZeroBot"
	"github.com/wdvxdr1123/ZeroBot/extension"
	"github.com/wdvxdr1123/ZeroBot/message"
	"strings"
)

func (p *PluginAnuneko) SetOnAt(engine plugin.Engine) {
	engine.OnMessage(p.env.Groups().Rule(), func(ctx *zero.Ctx) bool {
		// 只处理at消息
		return ctx.Event.IsToMe
	}).SetBlock(true).Handle(func(ctx *zero.Ctx) {

		uid := ctx.Event.UserID

		var err error
		sid, ok := p.client.GetSessionId(uid)
		if !ok {
			sid, err = p.client.CreateNewSession(uid)
		}
		if err != nil {
			p.env.Error(ctx, fmt.Errorf("创建会话失败: %w", err))
			return
		}

		var builder strings.Builder
		for _, segment := range ctx.Event.Message {
			if segment.Type != "text" {
				continue
			}
			val := segment.Data["text"]
			val = strings.TrimSpace(val)
			if len(val) <= 0 {
				continue
			}
			builder.WriteString(val)
			builder.WriteString("\n")
		}
		text := builder.String()
		if len(text) <= 0 {
			return
		}

		reply, err := p.client.StreamReply(sid, text)
		if err != nil {
			p.env.Error(ctx, err)
			return
		}

		var msgChain chain.MessageChain
		msgChain.Join(message.Reply(ctx.Event.MessageID))
		msgChain.Join(message.At(ctx.Event.Sender.ID))
		msgChain.Join(message.Text(" " + reply))
		ctx.Send(msgChain)

	})
}

func (p *PluginAnuneko) SetOnCreateSession(engine plugin.Engine) {
	engine.OnCommand("nekonew", p.env.Groups().Rule()).SetBlock(true).Handle(func(ctx *zero.Ctx) {
		uid := ctx.Event.UserID
		_, err := p.client.CreateNewSession(uid)
		if err != nil {
			p.env.Error(ctx, fmt.Errorf("创建会话失败: %w", err))
			return
		}
		var msgChain chain.MessageChain
		msgChain.Join(message.Reply(ctx.Event.MessageID))
		msgChain.Join(message.At(ctx.Event.Sender.ID))
		msgChain.Join(message.Text(" " + "已创建会话，@我开始聊天"))
		ctx.Send(msgChain)
	})
}

func (p *PluginAnuneko) SetOnSwitchModel(engine plugin.Engine) {
	engine.OnCommand("nekoswitch", p.env.Groups().Rule()).SetBlock(true).Handle(func(ctx *zero.Ctx) {
		uid := ctx.Event.UserID
		var cmdModel extension.CommandModel

		_ = ctx.Parse(&cmdModel)

		modelName := cmdModel.Args

		if len(modelName) <= 0 {
			p.env.Error(ctx, fmt.Errorf("模型名不能为空"))
			return
		}

		sid, ok := p.client.GetSessionId(uid)
		if !ok {
			p.env.Error(ctx, fmt.Errorf("会话不存在"))
			return
		}
		if !p.client.SwitchModel(uid, sid, modelName) {
			p.env.Error(ctx, fmt.Errorf("切换模型失败"))
			return
		}

		var msgChain chain.MessageChain
		msgChain.Join(message.Reply(ctx.Event.MessageID))
		msgChain.Join(message.At(ctx.Event.Sender.ID))
		msgChain.Join(message.Text(" " + "已切换模型为" + modelName))
		ctx.Send(msgChain)
	})
}
