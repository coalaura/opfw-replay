package main

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/grafov/m3u8"
	"github.com/yapingcat/gomedia/go-mp4"
	"github.com/yapingcat/gomedia/go-mpeg2"
)

type Stream struct {
	sync.RWMutex
	client   *http.Client
	stop     chan struct{}
	buffer   []*Segment
	sequence uint64
	size     int

	Key       string
	UrlString string
	Url       *url.URL
}

type Segment struct {
	data      []byte
	timestamp time.Time
}

const (
	ReplayDirectory = "replays"
	BufferDuration  = 20 * time.Second
	Interval        = 2 * time.Second
)

func NewStream(cluster, key, uri string) (*Stream, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}

	return &Stream{
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
		stop: make(chan struct{}),

		Key:       key,
		UrlString: uri,
		Url:       u,
	}, nil
}

func (s *Stream) StartReplay() error {
	playlist, err := s.playlist()
	if err != nil {
		return err
	}

	go func() {
		ticker := time.NewTicker(time.Duration(playlist.TargetDuration) * time.Second)

		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				s.clean()

				err := s.collect()
				if err != nil {
					log.Notef("Failed to collect segment for stream %s: %v\n", s.Key, err)

					continue
				}
			case <-s.stop:
				log.Debugf("Stopped replay for stream %s\n", s.Key)

				return
			}
		}
	}()

	return nil
}

func (s *Stream) Capture(wr http.ResponseWriter) error {
	if len(s.buffer) == 0 {
		return errors.New("no segments available")
	}

	writer := NewWriteSeeker(s.size)

	muxer, err := mp4.CreateMp4Muxer(writer)

	var (
		videoTrack uint32
		audioTrack uint32
	)

	s.RLock()
	for _, segment := range s.buffer {
		demuxer := mpeg2.NewTSDemuxer()

		demuxer.OnFrame = func(cid mpeg2.TS_STREAM_TYPE, frame []byte, pts uint64, dts uint64) {
			switch cid {
			case mpeg2.TS_STREAM_H264:
				if videoTrack == 0 {
					videoTrack = muxer.AddVideoTrack(mp4.MP4_CODEC_H264)
				}

				muxer.Write(videoTrack, frame, pts, dts)
			case mpeg2.TS_STREAM_AAC:
				if audioTrack == 0 {
					audioTrack = muxer.AddAudioTrack(mp4.MP4_CODEC_AAC)
				}

				muxer.Write(audioTrack, frame, pts, dts)
			}
		}

		err = demuxer.Input(bytes.NewReader(segment.data))
		if err != nil {
			break
		}
	}
	s.RUnlock()

	if err != nil {
		return err
	}

	err = muxer.WriteTrailer()
	if err != nil {
		return err
	}

	name := "replay-" + time.Now().Format("2006_01_02-15_04_05") + ".mp4"

	wr.Header().Set("Content-Type", "video/mp4")
	wr.Header().Set("Content-Disposition", "attachment; filename="+name)
	wr.WriteHeader(http.StatusOK)

	wr.Write(writer.Bytes())

	return nil
}

func (s *Stream) Stop() {
	close(s.stop)
	s.buffer = nil
}

func (s *Stream) clean() {
	s.Lock()
	defer s.Unlock()

	now := time.Now()

	var index int

	for i, segment := range s.buffer {
		if now.Sub(segment.timestamp) <= BufferDuration {
			break
		}

		s.size -= len(segment.data)
		index = i
	}

	if index == 0 {
		return
	}

	s.buffer = s.buffer[index:]
}

func (s *Stream) playlist() (*m3u8.MediaPlaylist, error) {
	resp, err := s.client.Get(s.UrlString)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	playlist, listType, err := m3u8.DecodeFrom(resp.Body, true)
	if err != nil {
		return nil, err
	}

	if listType != m3u8.MEDIA {
		return nil, errors.New("not a media playlist")
	}

	return playlist.(*m3u8.MediaPlaylist), nil
}

func (s *Stream) collect() error {
	playlist, err := s.playlist()
	if err != nil {
		return err
	}

	if playlist.SeqNo == s.sequence {
		return nil
	}

	if s.sequence == 0 {
		s.sequence = getLastSeqId(playlist.Segments)
	}

	for _, segment := range playlist.Segments {
		if segment == nil || segment.URI == "" || segment.SeqId <= s.sequence {
			continue
		}

		u, err := url.Parse(segment.URI)
		if err != nil {
			return err
		}

		uri := s.Url.ResolveReference(u).String()

		resp, err := s.client.Get(uri)
		if err != nil {
			return err
		}

		data, err := io.ReadAll(resp.Body)
		resp.Body.Close()

		if err != nil {
			return err
		}

		s.Lock()
		s.size += len(data)
		s.buffer = append(s.buffer, &Segment{
			data:      data,
			timestamp: time.Now(),
		})
		s.Unlock()

		s.sequence = segment.SeqId
	}

	return nil
}

func getLastSeqId(segments []*m3u8.MediaSegment) uint64 {
	var segment *m3u8.MediaSegment

	for i := len(segments) - 1; i >= 0; i-- {
		segment = segments[i]

		if segment == nil {
			continue
		}

		return segment.SeqId - 1
	}

	return 0
}
