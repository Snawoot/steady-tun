package main

import (
    "net"
    "strings"
    "github.com/caarlos0/env/v6"
)

type PluginArgs struct {
    RemoteHost string `env:"SS_REMOTE_HOST,required"`
    RemotePort string `env:"SS_REMOTE_PORT,required"`
    LocalHost  string `env:"SS_LOCAL_HOST,required"`
    LocalPort  string `env:"SS_LOCAL_PORT,required"`
    Options    string `env:"SS_PLUGIN_OPTIONS"`
}

func NewPluginArgs() (*PluginArgs, error) {
    args := PluginArgs{}
    if err := env.Parse(&args); err != nil {
        return nil, err
    }
    return &args, nil
}

func (args *PluginArgs) GetRemoteAddr() string {
    return net.JoinHostPort(args.RemoteHost, args.RemotePort)
}

func (args *PluginArgs) GetLocalAddr() string {
    return net.JoinHostPort(args.LocalHost, args.LocalPort)
}

func (args *PluginArgs) ExportOptions() []string {
    CLIopts := []string{}
    opts := strings.Split(args.Options, ";")
    for _, opt := range opts {
        if opt == "" {
            continue
        }
        pair := strings.SplitN(opt, "=", 2)
        switch len(pair) {
        case 1:
            CLIopts = append(CLIopts, "-" + pair[0])
        case 2:
            CLIopts = append(CLIopts, "-" + pair[0], pair[1])
        }
    }
    return CLIopts
}
