package args

import (
	"os"
	"unsafe"
)

/*
extern void free(void *);

extern char args_isDebugMode;
extern char *startup_script;

extern void parse_args(int argc, char **argv);

extern char use_startup_script;
extern char *get_fb_version(void);
extern char *get_fb_plain_version(void);
extern char *commit_hash(void);

extern char *server_address;
extern char *externalListenAddr;
extern char *capture_output_file;
extern char args_no_readline;
extern char *pack_scripts;
extern char *pack_scripts_out;

extern char enable_omega_system;

extern char *gamename;
extern char ingame_response;
*/
import "C"

func charify(val bool) C.char {
	if val {
		return C.char(1)
	} else {
		return C.char(0)
	}
}

func Set_args_isDebugMode(val bool) {
	C.args_isDebugMode = charify(val)
}

func GetFBVersion() string {
	return C.GoString(C.get_fb_version())
}

func GetFBPlainVersion() string {
	return C.GoString(C.get_fb_plain_version())
}

func GetFBCommitHash() string {
	return C.GoString(C.commit_hash())
}

func ParseArgs() {
	argv := make([]*C.char, len(os.Args))
	for i, v := range os.Args {
		cstr := C.CString(v)
		defer C.free(unsafe.Pointer(cstr))
		argv[i] = cstr
	}
	C.parse_args(C.int(len(os.Args)), &argv[0])
}

func boolify(v C.char) bool {
	if int(v) == 0 {
		return false
	}
	return true
}

func DebugMode() bool {
	if int(C.args_isDebugMode) == 0 {
		return false
	}
	return true
}

func ShouldEnableOmegaSystem() bool {
	return boolify(C.enable_omega_system)
}

func StartupScript() string {
	if int(C.use_startup_script) == 0 {
		return ""
	}
	return C.GoString(C.startup_script)
}

func SpecifiedServer() bool {
	return true
}

func ServerAddress() string {
	return C.GoString(C.server_address)
}

var CustomSEConsts map[string]string = map[string]string{}
var CustomSEUndefineConsts []string = []string{}

//export custom_script_engine_const
func custom_script_engine_const(key, val *C.char) {
	CustomSEConsts[C.GoString(key)] = C.GoString(val)
}

//export do_suppress_se_const
func do_suppress_se_const(key *C.char) {
	CustomSEUndefineConsts = append(CustomSEUndefineConsts, C.GoString(key))
}

func ExternalListenAddress() string {
	return C.GoString(C.externalListenAddr)
}

func CaptureOutputFile() string {
	return C.GoString(C.capture_output_file)
}

func NoReadline() bool {
	return boolify(C.args_no_readline)
}

func PackScripts() string {
	return C.GoString(C.pack_scripts)
}

func PackScriptsOut() string {
	return C.GoString(C.pack_scripts_out)
}

func GameName() string {
	return C.GoString(C.gamename)
}

func IngameResponse() bool {
	return boolify(C.ingame_response)
}