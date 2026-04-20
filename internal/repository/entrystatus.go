package repository

// Status constants used for the Entry
const (
	EntryStatusReady      uint8 = 0x00
	EntryStatusProcessing uint8 = 0x01
	EntryStatusError      uint8 = 0x02
	EntryStatusDeleting   uint8 = 0x03
)

// GetAllEntryStatuses provides a centralized list of all valid statuses.
func GetAllEntryStatuses() []uint8 {
	return []uint8{
		EntryStatusReady,
		EntryStatusProcessing,
		EntryStatusError,
		EntryStatusDeleting,
	}
}

func GetEntryStatusString(status uint8) string {
	switch status {
	case EntryStatusReady:
		return "ready"
	case EntryStatusProcessing:
		return "processing"
	case EntryStatusError:
		return "error"
	case EntryStatusDeleting:
		return "deleting"
	default:
		return "unknown"
	}
}
