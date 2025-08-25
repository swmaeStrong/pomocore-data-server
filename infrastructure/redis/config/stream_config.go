package config

type StreamInfo struct {
	StreamKey string
	Group     string
	Consumer  string
}

var (
	PomodoroPatternMatch = StreamInfo{
		StreamKey: "pattern_match_stream",
		Group:     "pattern_match_group",
		Consumer:  "pattern_match_consumer",
	}

	SessionScoreSave = StreamInfo{
		StreamKey: "session_score_stream",
		Group:     "session_score_group",
		Consumer:  "session_score_consumer",
	}
)
