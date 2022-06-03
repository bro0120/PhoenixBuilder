package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"phoenixbuilder/fastbuilder/args"
	"phoenixbuilder/fastbuilder/utils"
	"phoenixbuilder/minecraft/protocol"
	"phoenixbuilder/fastbuilder/command"
	"phoenixbuilder/fastbuilder/configuration"
	"phoenixbuilder/fastbuilder/function"
	I18n "phoenixbuilder/fastbuilder/i18n"
	"phoenixbuilder/fastbuilder/menu"
	"phoenixbuilder/fastbuilder/move"
	script_bridge "phoenixbuilder/fastbuilder/script_engine/bridge"
	"phoenixbuilder/fastbuilder/script_engine/bridge/script_holder"
	"phoenixbuilder/fastbuilder/signalhandler"
	fbtask "phoenixbuilder/fastbuilder/task"
	"phoenixbuilder/fastbuilder/types"
	"phoenixbuilder/fastbuilder/uqHolder"
	"phoenixbuilder/minecraft"
	"phoenixbuilder/minecraft/protocol/packet"
	"phoenixbuilder/omega/cli/embed"
	"runtime"
	"runtime/debug"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/pterm/pterm"
	"golang.org/x/term"

	"phoenixbuilder/fastbuilder/environment"
	"phoenixbuilder/fastbuilder/external"
	"phoenixbuilder/fastbuilder/readline"
)

func main() {
	args.ParseArgs()
	if(!args.DebugMode()&&len(args.ServerAddress())==0) {
		fmt.Printf("Please specify a server address.\n")
		fmt.Printf("--help, -h for help.\n")
		os.Exit(1)
	}
	if len(args.PackScripts()) != 0 {
		os.Exit(script_bridge.MakePackage(args.PackScripts(), args.PackScriptsOut()))
	}
	pterm.Error.Prefix = pterm.Prefix{
		Text:  "ERROR",
		Style: pterm.NewStyle(pterm.BgBlack, pterm.FgRed),
	}

	I18n.Init()

	pterm.DefaultBox.Println(pterm.LightCyan("https://github.com/LNSSPsd/PhoenixBuilder"))
	pterm.Println(pterm.Yellow("Contributors: Ruphane, CAIMEO, CMA2401PT"))
	pterm.Println(pterm.Yellow("Copyright (c) FastBuilder DevGroup, Bouldev 2022"))
	pterm.Println(pterm.Yellow("PhoenixBuilder " + args.GetFBVersion()))

	if !args.NoReadline() {
		readline.InitReadline()
	}

	if I18n.ShouldDisplaySpecial() {
		fmt.Printf("%s", I18n.T(I18n.Special_Startup))
	}

	defer func() {
		if err := recover(); err != nil {
			if !args.NoReadline() {
				readline.HardInterrupt()
			}
			debug.PrintStack()
			pterm.Error.Println(I18n.T(I18n.Crashed_Tip))
			pterm.Error.Println(I18n.T(I18n.Crashed_StackDump_And_Error))
			pterm.Error.Println(err)
			if runtime.GOOS == "windows" {
				pterm.Error.Println(I18n.T(I18n.Crashed_OS_Windows))
				_, _ = bufio.NewReader(os.Stdin).ReadString('\n')
			}
			os.Exit(1)
		}
		os.Exit(0)
	}()
	if args.DebugMode() {
		init_and_run_debug_client()
		return
	}
	/*if !args.ShouldDisableHashCheck() {
		fmt.Printf("Checking update, please wait...")
		hasUpdate, latestVersion := utils.CheckUpdate(args.GetFBVersion())
		fmt.Printf("OK\n")
		if hasUpdate {
			fmt.Printf("A newer version (%s) of PhoenixBuilder is available.\n", latestVersion)
			fmt.Printf("Please update.\n")
			// To ensure user won't ignore it directly, can be suppressed by command line argument.
			os.Exit(0)
		}
	}*/

	runInteractiveClient()
}

func runInteractiveClient() {
	address := args.ServerAddress()
	
	init_and_run_client(address)
}

func create_environment() *environment.PBEnvironment {
	env := &environment.PBEnvironment{}
	env.UQHolder = nil
	env.ActivateTaskStatus = make(chan bool)
	env.TaskHolder = fbtask.NewTaskHolder()
	functionHolder := function.NewFunctionHolder(env)
	env.FunctionHolder = functionHolder
	hostBridgeGamma := &script_bridge.HostBridgeGamma{}
	hostBridgeGamma.Init()
	hostBridgeGamma.HostQueryExpose = map[string]func() string{
		"server_code": func() string {
			return env.LoginInfo.ServerCode
		},
		"fb_version": func() string {
			return args.GetFBVersion()
		},
		"uc_username": func() string {
			return env.FBUCUsername
		},
	}
	for _, key := range args.CustomSEUndefineConsts {
		_, found := hostBridgeGamma.HostQueryExpose[key]
		if found {
			delete(hostBridgeGamma.HostQueryExpose, key)
		}
	}
	for key, val := range args.CustomSEConsts {
		hostBridgeGamma.HostQueryExpose[key] = func() string { return val }
	}
	env.ScriptBridge = hostBridgeGamma
	scriptHolder := script_holder.InitScriptHolder(env)
	env.ScriptHolder = scriptHolder
	/*if args.StartupScript() == "" {
		hostBridgeGamma.HostRemoveBlock()
	} else {
		if scriptHolder.LoadScript(args.StartupScript(), env) {
			hostBridgeGamma.HostWaitScriptBlock()
		} else {
			hostBridgeGamma.HostRemoveBlock()
		}
	}*/
	if args.StartupScript() != "" {
		scriptHolder.LoadScript(args.StartupScript(), env)
	}
	env.RespondUser = args.GameName()
	env.TargetServer=args.ServerAddress()
	hostBridgeGamma.HostRemoveBlock()
	return env
}

func init_and_run_debug_client() {
	env := create_environment()
	env.IsDebug = true

	scriptHolder := env.ScriptHolder.(*script_holder.ScriptHolder)
	defer scriptHolder.Destroy()

	runClient(env)
}

func init_and_run_client(address string) {
	env := create_environment()
	env.LoginInfo = environment.LoginInfo{
		Token:          "N/A",
		ServerCode:     "ADDR|"+address,
		ServerPasscode: "N/A",
	}

	scriptHolder := env.ScriptHolder.(*script_holder.ScriptHolder)
	defer scriptHolder.Destroy()

	runClient(env)
}

func runClient(env *environment.PBEnvironment) {
	pterm.Println(pterm.Yellow(fmt.Sprintf("%s: %s", I18n.T(I18n.ServerCodeTrans), env.LoginInfo.ServerCode)))
	var conn *minecraft.Conn
	if env.IsDebug {
		conn = &minecraft.Conn{
			DebugMode: true,
		}
	} else {
		connDeadline := time.NewTimer(time.Minute * 20)
		go func() {
			<-connDeadline.C
			if env.Connection == nil {
				panic("connection not established after very long time (20min)")
			}
		}()
		dialer := minecraft.Dialer{
			
		}
		cconn, err := dialer.Dial("raknet", env.TargetServer)

		if err != nil {
			pterm.Error.Println(err)
			if runtime.GOOS == "windows" {
				pterm.Error.Println(I18n.T(I18n.Crashed_OS_Windows))
				_, _ = bufio.NewReader(os.Stdin).ReadString('\n')
			}
			panic(err)
		}
		conn = cconn
		env.WorldChatChannel = make(chan []string)
	}
	defer conn.Close()
	defer func() {
		env.Stop()
		env.WaitStopped()
	}()

	pterm.Println(pterm.Yellow(I18n.T(I18n.ConnectionEstablished)))
	conn.WritePacket(&packet.ClientCacheStatus{
		Enabled: false,
	})
	env.Connection = conn
	env.UQHolder = nil
	
	if args.ShouldEnableOmegaSystem() {
		fmt.Println("Omega System Enabled!")
		embed.EnableOmegaSystem(env)
	}

	commandSender := command.InitCommandSender(env)
	functionHolder := env.FunctionHolder.(*function.FunctionHolder)
	function.InitInternalFunctions(functionHolder)
	fbtask.InitTaskStatusDisplay(env)
	move.ConnectTime = conn.GameData().ConnectTime
	move.Position = conn.GameData().PlayerPosition
	move.Pitch = conn.GameData().Pitch
	move.Yaw = conn.GameData().Yaw
	move.Connection = conn
	move.RuntimeID = conn.GameData().EntityRuntimeID

	signalhandler.Install(conn, env)

	hostBridgeGamma := env.ScriptBridge.(*script_bridge.HostBridgeGamma)
	hostBridgeGamma.HostSetSendCmdFunc(func(mcCmd string, waitResponse bool) *packet.CommandOutput {
		ud, _ := uuid.NewUUID()
		chann := make(chan *packet.CommandOutput)
		if waitResponse {
			commandSender.UUIDMap.Store(ud.String(), chann)
		}
		commandSender.SendCommand(mcCmd, ud)
		if waitResponse {
			resp := <-chann
			return resp
		} else {
			return nil
		}
	})
	hostBridgeGamma.HostConnectEstablished()
	defer hostBridgeGamma.HostConnectTerminate()

	zeroId, _ := uuid.NewUUID()
	oneId, _ := uuid.NewUUID()
	configuration.ZeroId = zeroId
	configuration.OneId = oneId
	taskholder := env.TaskHolder.(*fbtask.TaskHolder)
	types.ForwardedBrokSender = taskholder.BrokSender
	var captureFp *os.File
	if captureOutputFileName := args.CaptureOutputFile(); captureOutputFileName != "" {
		if fp, err := os.OpenFile(captureOutputFileName, os.O_CREATE|os.O_WRONLY, 0755); err != nil {
			panic(err)
		} else {
			captureFp = fp
			fmt.Println("Capture On: FastBuilder > ", captureOutputFileName)
		}
	}
	go func() {
		if args.NoReadline() {
			return
		}
		for {
			cmd := readline.Readline(env)
			if len(cmd) == 0 {
				continue
			}
			if env.OmegaAdaptorHolder != nil && !strings.Contains(cmd, "exit") {
				env.OmegaAdaptorHolder.(*embed.EmbeddedAdaptor).FeedBackendCommand(cmd)
				continue
			}
			if strings.TrimSpace(cmd) == "capture close" {
				if captureFp != nil {
					captureFp.Close()
					captureFp = nil
					fmt.Println("Capture Closed")
				}
			}
			if cmd[0] == '.' {
				ud, _ := uuid.NewUUID()
				chann := make(chan *packet.CommandOutput)
				commandSender.UUIDMap.Store(ud.String(), chann)
				commandSender.SendCommand(cmd[1:], ud)
				resp := <-chann
				fmt.Printf("%+v\n", resp)
			} else if cmd[0] == '!' {
				ud, _ := uuid.NewUUID()
				chann := make(chan *packet.CommandOutput)
				commandSender.UUIDMap.Store(ud.String(), chann)
				commandSender.SendWSCommand(cmd[1:], ud)
				resp := <-chann
				fmt.Printf("%+v\n", resp)
			}
			if cmd == "menu" {
				menu.OpenMenu(env)
				fmt.Printf("OK\n")
				continue
			}
			if cmd == "move" {
				go func() {
					/*var counter int=0
					var direction bool=false
					for{
						if counter%20==0 {
							//move.Jump()
						}
						if counter>280 {
							counter=0
							direction= !direction
						}
						if direction {
							move.Move(-2+2*moveP/100,0,2*moveP/100)
							time.Sleep(time.Second/20)
							counter++
							continue
						}else{
							move.Move(2*moveP/100,0,-2+2*moveP/100)
							time.Sleep(time.Second/20)
							counter++
							continue
						}
					}*/
					for {
						move.Auto()
						time.Sleep(time.Second / 20)
					}
				}()
				continue
			}
			functionHolder.Process(cmd)
		}
	}()

	if args.ExternalListenAddress() != "" {
		external.ListenExt(env, args.ExternalListenAddress())
	}
	//env.UQHolder.(*uqHolder.UQHolder).UpdateFromConn(conn)
	for {
		pk, err := conn.ReadPacket()
		if err != nil {
			panic(err)
		}
		if env.OmegaAdaptorHolder != nil {
			env.OmegaAdaptorHolder.(*embed.EmbeddedAdaptor).FeedPacket(pk)
			continue
		}
		hostBridgeGamma.HostPumpMcPacket(pk)
		hostBridgeGamma.HostQueryExpose["uqHolder"] = func() string {
			marshal, err := json.Marshal(env.UQHolder.(*uqHolder.UQHolder))
			if err != nil {
				marshalErr, _ := json.Marshal(map[string]string{"err": err.Error()})
				return string(marshalErr)
			}
			return string(marshal)
		}

		switch p := pk.(type) {
		case *packet.StructureTemplateDataResponse:
			fbtask.ExportWaiter <- p.StructureTemplate
			break
		case *packet.Text:
			if p.TextType == packet.TextTypeChat {
				if(!args.IngameResponse()) {
					break
				}
				if(args.GameName()!="@"&&p.SourceName!=args.GameName()) {
					break
				}
				functionHolder.Process(p.Message)
				break
			}
		case *packet.CommandOutput:
			if p.CommandOrigin.UUID.String() == configuration.ZeroId.String() {
				pos, _ := utils.SliceAtoi(p.OutputMessages[0].Parameters)
				if !(p.OutputMessages[0].Message == "commands.generic.unknown") {
					configuration.IsOp = true
				}
				if len(pos) == 0 {
					commandSender.Tellraw(I18n.T(I18n.InvalidPosition))
					break
				}
				configuration.GlobalFullConfig(env).Main().Position = types.Position{
					X: pos[0],
					Y: pos[1],
					Z: pos[2],
				}
				commandSender.Tellraw(fmt.Sprintf("%s: %v", I18n.T(I18n.PositionGot), pos))
				break
			} else if p.CommandOrigin.UUID.String() == configuration.OneId.String() {
				pos, _ := utils.SliceAtoi(p.OutputMessages[0].Parameters)
				if len(pos) == 0 {
					commandSender.Tellraw(I18n.T(I18n.InvalidPosition))
					break
				}
				configuration.GlobalFullConfig(env).Main().End = types.Position{
					X: pos[0],
					Y: pos[1],
					Z: pos[2],
				}
				commandSender.Tellraw(fmt.Sprintf("%s: %v", I18n.T(I18n.PositionGot_End), pos))
				break
			}
			pr, ok := commandSender.UUIDMap.LoadAndDelete(p.CommandOrigin.UUID.String())
			if ok {
				pu := pr.(chan *packet.CommandOutput)
				pu <- p
			}
		case *packet.ActorEvent:
			if p.EventType == packet.ActorEventDeath && p.EntityRuntimeID == conn.GameData().EntityRuntimeID {
				conn.WritePacket(&packet.PlayerAction{
					EntityRuntimeID: conn.GameData().EntityRuntimeID,
					ActionType:      protocol.PlayerActionRespawn,
				})
			}
		case *packet.UpdateBlock:
			channel, h := commandSender.BlockUpdateSubscribeMap.LoadAndDelete(p.Position)
			if h {
				ch := channel.(chan bool)
				ch <- true
			}
		case *packet.Respawn:
			if p.EntityRuntimeID == conn.GameData().EntityRuntimeID {
				move.Position = p.Position
			}
		case *packet.MovePlayer:
			if p.EntityRuntimeID == conn.GameData().EntityRuntimeID {
				move.Position = p.Position
			} else if p.EntityRuntimeID == move.TargetRuntimeID {
				move.Target = p.Position
			}
		case *packet.CorrectPlayerMovePrediction:
			move.MoveP += 10
			if move.MoveP > 100 {
				move.MoveP = 0
			}
			move.Position = p.Position
			move.Jump()
		case *packet.AddPlayer:
			if move.TargetRuntimeID == 0 && p.EntityRuntimeID != conn.GameData().EntityRuntimeID {
				move.Target = p.Position
				move.TargetRuntimeID = p.EntityRuntimeID
				//fmt.Printf("Got target: %s\n",p.Username)
			}
		}
	}
}

func getInput() (string, error) {
	reader := bufio.NewReader(os.Stdin)
	inp, err := reader.ReadString('\n')
	inpl := strings.TrimRight(inp, "\r\n")
	return inpl, err
}

func getInputUserName() (string, error) {
	reader := bufio.NewReader(os.Stdin)
	pterm.Printf(I18n.T(I18n.Enter_FBUC_Username))
	fbusername, err := reader.ReadString('\n')
	return fbusername, err
}

func getRentalServerCode() (string, string, error) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf(I18n.T(I18n.Enter_Rental_Server_Code))
	code, err := reader.ReadString('\n')
	if err != nil {
		return "", "", err
	}
	fmt.Printf(I18n.T(I18n.Enter_Rental_Server_Password))
	bytePassword, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Printf("\n")
	return strings.TrimRight(code, "\r\n"), string(bytePassword), err
}

func readToken(path string) (string, error) {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func loadTokenPath() string {
	homedir, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("WARNING - Failed to obtain the user's home directory. made homedir=\".\";\n")
		homedir = "."
	}
	fbconfigdir := filepath.Join(homedir, ".config/fastbuilder")
	os.MkdirAll(fbconfigdir, 0700)
	token := filepath.Join(fbconfigdir, "fbtoken")
	return token
}
