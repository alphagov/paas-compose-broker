package broker

type ProvisionParameters struct {
	RestoreFromLatestSnapshotOf *string `json:"restore_from_latest_snapshot_of"`
}

func (pp *ProvisionParameters) Validate() error {
	return nil
}
