package command

import (
	"phoenixbuilder/minecraft"
	"phoenixbuilder/minecraft/protocol"
	"phoenixbuilder/minecraft/protocol/packet"
	"github.com/google/uuid"
	"sync"
)

func (sender *CommandSender) GetBlockUpdateSubscribeMap() *sync.Map {
	return &sender.BlockUpdateSubscribeMap
}

func (sender *CommandSender) GetUUIDMap() *sync.Map {
	return &sender.UUIDMap
}

func (sender *CommandSender) ClearUUIDMap() {
	sender.UUIDMap=sync.Map{}
}

func (sender *CommandSender) getConn() *minecraft.Conn {
	conn:=sender.env.Connection.(*minecraft.Conn)
	return conn
}

func (sender *CommandSender) SendCommand(command string, UUID uuid.UUID) error {
	requestId, _ := uuid.Parse("96045347-a6a3-4114-94c0-1bc4cc561694")
	origin := protocol.CommandOrigin{
		Origin:         protocol.CommandOriginPlayer,
		UUID:           UUID,
		RequestID:      requestId.String(),
		PlayerUniqueID: 0,
	}
	commandRequest := &packet.CommandRequest{
		CommandLine:   command,
		CommandOrigin: origin,
		Internal:      false,
	}
	return sender.getConn().WritePacket(commandRequest)
}

func (sender *CommandSender) SendWSCommand(command string, UUID uuid.UUID) error {
	requestId, _ := uuid.Parse("96045347-a6a3-4114-94c0-1bc4cc561694")
	origin := protocol.CommandOrigin{
		Origin:         protocol.CommandOriginAutomationPlayer,
		UUID:           UUID,
		RequestID:      requestId.String(),
		PlayerUniqueID: 0,
	}
	commandRequest := &packet.CommandRequest{
		CommandLine:   command,
		CommandOrigin: origin,
		Internal:      false,
	}
	return sender.getConn().WritePacket(commandRequest)
}

func (sender *CommandSender) SendSizukanaCommand(command string) error {
	return sender.SendWSCommand(command, uuid.New())
	// The method below seems no longer works.
	/*return sender.getConn().WritePacket(&packet.SettingsCommand{
		CommandLine: command,
		SuppressOutput: true,
	})*/
}

func (sender *CommandSender) SendChat(content string) error {
	conn:=sender.getConn()
	idd:=conn.IdentityData()
	return conn.WritePacket(&packet.Text {
		TextType: packet.TextTypeChat,
		NeedsTranslation: false,
		SourceName: idd.DisplayName,
		Message: content,
		XUID: idd.XUID,
	})
}