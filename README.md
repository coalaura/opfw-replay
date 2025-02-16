# OPFW-Replay

OPFW-Replay is a Go-based service that captures and replays HLS (HTTP Live Streaming) video streams, converting them into downloadable MP4 files. This tool is particularly useful for creating video recordings from live HLS streams while maintaining both video and audio quality.

## Features

- HLS stream capture and buffering
- Real-time MP4 conversion
- Configurable buffer duration
- HTTP API for stream access
- Automatic stream cleanup
- Support for H.264 video and AAC audio codecs

## Installation

1. Download the pre-built binary from the [`bin`](bin/replay) directory
2. Create a working directory (e.g., `/var/replay`)
3. Copy the binary to the working directory
4. Create a configuration file (see below)
5. Set up the systemd [service](replay.service) (optional)

## Configuration

Create a configuration file (`config.json`) in the same directory as the binary:
```json
{
    "panel": "/path/to/panel",
    "duration": 30
}
```

|Key|Description|Default|
|---|---|---|
|`panel`|Path to the panel directory|-|
|`duration`|Buffer duration (in seconds)|`30`|
