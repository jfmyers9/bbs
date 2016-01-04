package models

type CellSet map[string]*CellPresence

func NewCellSet() CellSet {
	return make(CellSet)
}

func NewCellSetFromList(cells []*CellPresence) CellSet {
	cellSet := NewCellSet()
	for _, v := range cells {
		cellSet.Add(v)
	}
	return cellSet
}

func (set CellSet) Add(cell *CellPresence) {
	set[cell.CellId] = cell
}

func (set CellSet) Each(predicate func(cell *CellPresence)) {
	for _, cell := range set {
		predicate(cell)
	}
}

func (set CellSet) HasCellID(cellID string) bool {
	_, ok := set[cellID]
	return ok
}

func NewCellCapacity(memoryMB, diskMB, containers int32) CellCapacity {
	return CellCapacity{
		MemoryMb:   memoryMB,
		DiskMb:     diskMB,
		Containers: containers,
	}
}

func (cap CellCapacity) Validate() error {
	var validationError ValidationError

	if cap.MemoryMb <= 0 {
		validationError = validationError.Append(ErrInvalidField{"memory_mb"})
	}

	if cap.DiskMb < 0 {
		validationError = validationError.Append(ErrInvalidField{"disk_mb"})
	}

	if cap.Containers <= 0 {
		validationError = validationError.Append(ErrInvalidField{"containers"})
	}

	if !validationError.Empty() {
		return validationError
	}

	return nil
}

func NewCellPresence(cellID, repAddress, zone string, capacity CellCapacity, rootFSProviders, preloadedRootFSes []string) CellPresence {
	rootFSProviderMap := make(RootFSProviders)

	for _, provider := range rootFSProviders {
		rootFSProviderMap[provider] = &Providers{
			Parameters: []string{},
		}
	}

	rootFSProviderMap["preloaded"] = &Providers{
		Parameters: preloadedRootFSes,
	}

	return CellPresence{
		CellId:          cellID,
		RepAddress:      repAddress,
		Zone:            zone,
		Capacity:        &capacity,
		RootfsProviders: rootFSProviderMap,
	}
}

func (c CellPresence) Validate() error {
	var validationError ValidationError

	if c.CellId == "" {
		validationError = validationError.Append(ErrInvalidField{"cell_id"})
	}

	if c.RepAddress == "" {
		validationError = validationError.Append(ErrInvalidField{"rep_address"})
	}

	if err := c.Capacity.Validate(); err != nil {
		validationError = validationError.Append(err)
	}

	if !validationError.Empty() {
		return validationError
	}

	return nil
}

const (
	EventTypeCellDisappeared = "cell_disappeared"
)

type CellEvent interface {
	EventType() string
	CellIDs() []string
}

type CellDisappearedEvent struct {
	IDs []string
}

func NewCellDisappearedEvent(ids []string) CellDisappearedEvent {
	return CellDisappearedEvent{ids}
}

func (CellDisappearedEvent) EventType() string {
	return EventTypeCellDisappeared
}

func (e CellDisappearedEvent) CellIDs() []string {
	return e.IDs
}
