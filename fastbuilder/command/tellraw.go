package command

import (
	"fmt"
	"phoenixbuilder/bridge/bridge_fmt"
	"phoenixbuilder/fastbuilder/types"
	"phoenixbuilder/fastbuilder/args"
	"time"
	//"github.com/google/uuid"
	"encoding/json"
	"strings"
)

type TellrawItem struct {
	Text string `json:"text"`
}

type TellrawStruct struct {
	RawText []TellrawItem `json:"rawtext"`
}

func TellRawRequest(target types.Target, lines ...string) string {
	now := time.Now().Format("§6{15:04:05}§b")
	var items []TellrawItem
	for _, text := range lines {
		items=append(items,TellrawItem{Text:msg})
	}
	final := &TellrawStruct {
		RawText: items,
	}
	content, _ := json.Marshal(final)
	cmd := fmt.Sprintf("tellraw %v %s", target, content)
	return cmd
}

func (sender *CommandSender) Tellraw(content string) error {
	bridge_fmt.Printf("%s\n", content)
	if(!args.IngameResponse()) {
		return nil
	}
	return sender.SendSizukanaCommand(RawTellRawRequest(types.AllPlayers, content))
}

func RawTellRawRequest(target types.Target, line string) string {
	items:=[]TellrawItem{TellrawItem{Text:line}}
	final := &TellrawStruct {
		RawText: items,
	}
	content, _ := json.Marshal(final)
	cmd := fmt.Sprintf("tellraw %v %s", target, content)
	return cmd
}

func (cmd_sender *CommandSender) WorldChatTellraw(sender string, content string) error {
	bridge_fmt.Printf("W <%s> %s\n", sender, content)
	str:=fmt.Sprintf("§eW §r<%s> %s",sender,content)
	return cmd_sender.SendSizukanaCommand(RawTellRawRequest(types.AllPlayers, str))
}