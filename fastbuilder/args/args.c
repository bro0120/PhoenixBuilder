#include <stdio.h>
#include <stdlib.h>
#include <getopt.h>
#include <string.h>

char args_isDebugMode=0;
char use_startup_script=0;
char *startup_script;
char *server_address="";
char *externalListenAddr="";
char *capture_output_file="";
char args_no_readline=0;
char *pack_scripts="";
char *pack_scripts_out="";
char enable_omega_system=0;
char *gamename="";
char ingame_response=1;

extern void custom_script_engine_const(const char *key, const char *val);
extern void do_suppress_se_const(const char *key);

void print_help(const char *self_name) {
	printf("%s [options]\n",self_name);
	printf("\t--debug: Run in debug mode.\n");
#ifdef WITH_V8
	printf("\t-S, --script=<*.js>: run a .js script at start\n");
	printf("\t--script-engine-const key=value: Define a const value for script engine's \"consts\" const. Can be used to replace the default value. Specify multiple items by using this argument for multiple times.\n");
	printf("\t--script-engine-suppress-const <key>: Undefine a const value for script engine's \"consts\" const. Specify multiple items by using this argument for multiple times.\n");
#endif
	printf("\t-s, --server=<server address>: Specify a server address. Note that only offline mode is supported currently.\n");
	printf("\t-E, --listen-external: Listen on the specified address and wait for external controlling connection.\n\t\tExample: -E 0.0.0.0:5768 - listen on port 5768 and accept connections from anywhere,\n\t\t\t-E 127.0.0.1:5769 - listen on port 5769 and accept connections from localhost only.\n");
	printf("\t--capture=<*.bin>: Capture minecraft packet and dump to target file\n");
	printf("\t--no-readline: Suppress user input.\n");
	printf("\t--pack-scripts <manifest path>: Create a script package.\n");
	printf("\t--pack-scripts-to <path>: Specify the path for the output script package.\n");
	printf("\t-N, --operator_name <name>: Specify the operator's name to give response. Use @ for all possible users.\n");
	printf("\t-X, --no-ingame-response: Disable ingame response.\n");
	printf("\n");
	printf("\t-O, --omega_system: Enable Omega System.\n");
	printf("\n");
	printf("\t-h, --help: Show this help context.\n");
	printf("\t-v, --version: Show the version information of this program.\n");
	printf("\t\t--version-plain: Show the version of this program.\n");
}

char *get_fb_version() {
#ifdef FBGUI_VERSION
	return FB_VERSION "@" FBGUI_VERSION " (" FB_COMMIT ")";
#else
	return FB_VERSION " (" FB_COMMIT ")";
#endif
}

char *get_fb_plain_version() {
#ifdef FBGUI_VERSION
	return FBGUI_VERSION;
#else
	return FB_VERSION;
#endif
}

char *commit_hash() {
	return FB_COMMIT_LONG;
}

void print_version(int detailed) {
	if(!detailed) {
		printf(FB_VERSION "\n");
		return;
	}
	printf("PhoenixBuilder ~ global-bedrock/v1.19.0 " FB_VERSION "\n");
#ifdef FBGUI_VERSION
	printf("With GUI " FBGUI_VERSION "\n");
#endif
#ifdef WITH_V8
	printf("With V8 linked.\n");
#endif
	printf("COMMIT " FB_COMMIT_LONG "\n");
	printf("Bouldev\n");
	printf("Licensed under GNU AGPL v3.\n");
	printf("\n");
}

void read_token(char *token_path) {
	fprintf(stderr, "read_token called with argument token_path: %s\n", token_path);
}

void quickcopy(char **target_ptr) {
	size_t length=strlen(optarg)+1;
	*target_ptr=malloc(length);
	memcpy(*target_ptr, optarg, length);
}

int _parse_args(int argc, char **argv) {
	while(1) {
		static struct option opts[]={
			{"debug", no_argument, 0, 0}, // 0
			{"help", no_argument, 0, 'h'}, // 1
			{"auth-server", required_argument, 0, 'A'}, //2
			{"no-update-check", no_argument, 0, 0}, //3
			{"no-world-chat", no_argument, 0, 'M'}, //4
			{"no-pyrpc", no_argument, 0, 0}, //5
			{"no-nbt", no_argument, 0, 0}, //6
			{"script", required_argument, 0, 'S'}, //7
			{"version", no_argument, 0, 'v'}, //8
			{"version-plain", no_argument, 0, 0}, //9
			{"server", required_argument, 0, 's'}, //10
			{"password", required_argument, 0, 'p'}, //11
			{"token", required_argument, 0, 't'}, //12
			{"plain-token", required_argument, 0, 'T'}, //13
			{"script-engine-const", required_argument, 0, 0}, //14
			{"script-engine-suppress-const", required_argument, 0, 0}, //15
			{"listen-external", required_argument, 0, 'E'}, // 16
			{"no-readline", no_argument, 0, 0}, //17
			{"pack-scripts", required_argument, 0, 0}, //18
			{"pack-scripts-to", required_argument, 0, 0}, //19
			{"capture", required_argument, 0, 0}, // 20
			{"omega_system", no_argument, 0, 'O'}, // 21
			{"operator_name", required_argument, 0, 'N'}, // 22
			{"no-ingame-response", no_argument, 0, 'X'}, // 23
			{0, 0, 0, 0}
		};
		int option_index;
		int c=getopt_long(argc,argv,"hA:MvS:s:p:t:T:ON:", opts, &option_index);
		if(c==-1)
			break;
		switch(c) {
		case 0:
			switch(option_index) {
			case 0:
				args_isDebugMode=1;
				break;
			case 3:
				fprintf(stderr, "WARNING: --no-update-check option isn't available.\n");
				break;
			case 5:
				fprintf(stderr, "WARNING: --no-pyrpc option isn't available.\n");
				break;
			case 6:
				fprintf(stderr, "--no-nbt option is no longer available.\n");
				return 10;
				break;
			case 9:
				print_version(0);
				return 0;
			case 14:
#ifndef WITH_V8
				fprintf(stderr,"--script-engine-const argument isn't available: Non-v8-linked version.\n");
				return 10;
#endif
				{
					int break_switch_14=0;
					for(char *ptr=optarg;*ptr!=0;ptr++) {
						if(*ptr=='=') {
							*ptr=0;
							ptr++;
							custom_script_engine_const(optarg, ptr);
							break_switch_14=1;
							break;
						}
					}
					if(break_switch_14)break;
					fprintf(stderr, "--script-engine-const: Format: key=val\n");
					print_help(argv[0]);
					return 1;
				}
			case 15:
#ifndef WITH_V8
				fprintf(stderr,"--script-engine-suppress-const argument isn't available: Non-v8-linked version.\n");
				return 10;
#endif
				do_suppress_se_const(optarg);
				break;
			case 17:
				args_no_readline=1;
				break;
			case 18:
				quickcopy(&pack_scripts);
				break;
			case 19:
				quickcopy(&pack_scripts_out);
				break;
			case 20:
				quickcopy(&capture_output_file);
				break;
			};
			break;
		case 'h':
			print_help(argv[0]);
			return 0;
		case 'A':
			fprintf(stderr, "WARNING: -A option isn't available\n");
			break;
		case 'M':
			fprintf(stderr, "WARNING: -M option isn't available\n");
			break;
		case 'S':
#ifndef WITH_V8
			fprintf(stderr,"-S, --script option isn't available: No V8 linked for this version.\n");
			return 10;
#endif
			use_startup_script=1;
			quickcopy(&startup_script);
			break;
		case 's':
			quickcopy(&server_address);
			break;
		case 'p':
			fprintf(stderr, "WARNING: -p, --password option isn't available.\n");
			break;
		case 'E':
			quickcopy(&externalListenAddr);
			break;
		case 'v':
			print_version(1);
			return 0;
		case 'O':
			enable_omega_system=1;
			break;
		case 'N':
			quickcopy(&gamename);
			break;
		case 'X':
			ingame_response=0;
			break;
		default:
			print_help(argv[0]);
			return 1;
		};
	};
	return -1;
}

void parse_args(int argc, char **argv) {
	int ec;
	if((ec=_parse_args(argc,argv))!=-1) {
		exit(ec);
	}
	return;
}
