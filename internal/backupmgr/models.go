package backupmgr

type Backup struct {
	ID        int    `json:"id"`
	Tables    string `json:"tables"`
	SizeBytes int    `json:"size_bytes"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
}
