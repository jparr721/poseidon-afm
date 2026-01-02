module github.com/jparr721/poseidon-afm

go 1.25.1

//replace github.com/MythicMeta/MythicContainer => ../../../../MythicMeta/MythicContainer

require (
	github.com/MythicMeta/MythicContainer v1.6.3
	github.com/google/uuid v1.6.0
	github.com/jparr721/poseidon-afm/poseidon/agent_code v0.0.0
	github.com/mitchellh/mapstructure v1.5.0
	github.com/pelletier/go-toml v1.9.5
	golang.org/x/exp v0.0.0-20251209150349-8475f28825e9
)

replace github.com/jparr721/poseidon-afm/poseidon/agent_code => ./poseidon/agent_code

require (
	github.com/creack/pty v1.1.24 // indirect
	github.com/djherbis/atime v1.1.0 // indirect
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/gen2brain/shm v0.1.1 // indirect
	github.com/go-viper/mapstructure/v2 v2.4.0 // indirect
	github.com/godbus/dbus/v5 v5.1.0 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/jezek/xgb v1.1.1 // indirect
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51 // indirect
	github.com/kbinani/screenshot v0.0.0-20250624051815-089614a94018 // indirect
	github.com/lxn/win v0.0.0-20210218163916-a377121e959e // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/miekg/dns v1.1.69 // indirect
	github.com/pelletier/go-toml/v2 v2.2.4 // indirect
	github.com/rabbitmq/amqp091-go v1.10.0 // indirect
	github.com/rs/zerolog v1.34.0 // indirect
	github.com/sagikazarmark/locafero v0.12.0 // indirect
	github.com/spf13/afero v1.15.0 // indirect
	github.com/spf13/cast v1.10.0 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
	github.com/spf13/viper v1.21.0 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	github.com/tmc/scp v0.0.0-20170824174625-f7b48647feef // indirect
	github.com/xorrior/keyctl v1.0.1-0.20210425144957-8746c535bf58 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/crypto v0.46.0 // indirect
	golang.org/x/mod v0.31.0 // indirect
	golang.org/x/net v0.48.0 // indirect
	golang.org/x/sync v0.19.0 // indirect
	golang.org/x/sys v0.39.0 // indirect
	golang.org/x/text v0.32.0 // indirect
	golang.org/x/tools v0.40.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251213004720-97cd9d5aeac2 // indirect
	google.golang.org/grpc v1.77.0 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.2.1 // indirect
	howett.net/plist v1.0.1 // indirect
)
