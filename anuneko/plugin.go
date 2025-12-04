package anuneko

import (
	"github.com/kohmebot/pkg/chain"
	"github.com/kohmebot/plugin/v2"
	zero "github.com/wdvxdr1123/ZeroBot"
	"github.com/wdvxdr1123/ZeroBot/message"
)

type PluginAnuneko struct {
	env    plugin.Env
	conf   Config
	client *AnuNekoClient
}

func NewPlugin() plugin.Plugin {
	return new(PluginAnuneko)
}

func (p *PluginAnuneko) OnInit(engine plugin.Engine, env plugin.Env) error {
	p.env = env

	if err := env.GetConf(&p.conf); err != nil {
		return err
	}

	p.client = NewClient(p.conf.Token, p.conf.Cookie, p.conf.DeviceId)

	p.SetOnCreateSession(engine)
	p.SetOnSwitchModel(engine)
	p.SetOnAt(engine)

	return nil
}

func (p *PluginAnuneko) OnBoot() {

}

func (p *PluginAnuneko) OnHelp(ctx *zero.Ctx) {
	var msg chain.MessageChain

	msg.Split(
		message.Text("nekonew 开始新会话"),
		message.Text("nekoswitch <模型名称>：切换模型"),
		message.Text("@我 开始聊天"),
	)

	ctx.Send(msg)
}

func (p *PluginAnuneko) Name() string {
	return "anuneko"
}

func (p *PluginAnuneko) Version() string {
	return "v0.0.3"
}
