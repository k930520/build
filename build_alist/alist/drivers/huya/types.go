package huya

type File struct {
	Name      string
	RoomId    string
	Type      string
	Thumbnail string
}

type LiveTranscodingTaskList struct {
	TemplateID string `json:"template_id"`
	URL        string `json:"url"`
}

type VideoPreviewPlayInfo struct {
	LiveTranscodingTaskList []LiveTranscodingTaskList `json:"live_transcoding_task_list"`
}

type VideoPreview struct {
	VideoPreviewPlayInfo VideoPreviewPlayInfo `json:"video_preview_play_info"`
}
