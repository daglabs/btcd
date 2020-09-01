// Copyright (c) 2013-2016 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package blockdag

import (
	"fmt"

	"github.com/kaspanet/kaspad/util"
	"github.com/kaspanet/kaspad/util/daghash"
	"github.com/kaspanet/kaspad/util/mstime"
)

// NotificationType represents the type of a notification message.
type NotificationType int

// NotificationCallback is used for a caller to provide a callback for
// notifications about various blockDAG events.
type NotificationCallback func(*Notification)

// Constants for the type of a notification message.
const (
	// NTBlockAdded indicates the associated block was added into
	// the blockDAG.
	NTBlockAdded NotificationType = iota

	// NTChainChanged indicates that selected parent
	// chain had changed.
	NTChainChanged

	// NTFinalityConflict indicates that a finality conflict has just occurred
	NTFinalityConflict

	// NTFinalityConflict indicates that a finality conflict has been resolved
	NTFinalityConflictResolved
)

// notificationTypeStrings is a map of notification types back to their constant
// names for pretty printing.
var notificationTypeStrings = map[NotificationType]string{
	NTBlockAdded:               "NTBlockAdded",
	NTChainChanged:             "NTChainChanged",
	NTFinalityConflict:         "NTFinalityConflict",
	NTFinalityConflictResolved: "NTFinalityConflictResolved",
}

// String returns the NotificationType in human-readable form.
func (n NotificationType) String() string {
	if s, ok := notificationTypeStrings[n]; ok {
		return s
	}
	return fmt.Sprintf("Unknown Notification Type (%d)", int(n))
}

// Notification defines notification that is sent to the caller via the callback
// function provided during the call to New and consists of a notification type
// as well as associated data that depends on the type as follows:
// 	- Added:     *util.Block
type Notification struct {
	Type NotificationType
	Data interface{}
}

// Subscribe to block DAG notifications. Registers a callback to be executed
// when various events take place. See the documentation on Notification and
// NotificationType for details on the types and contents of notifications.
func (dag *BlockDAG) Subscribe(callback NotificationCallback) {
	dag.notificationsLock.Lock()
	defer dag.notificationsLock.Unlock()
	dag.notifications = append(dag.notifications, callback)
}

// sendNotification sends a notification with the passed type and data if the
// caller requested notifications by providing a callback function in the call
// to New.
func (dag *BlockDAG) sendNotification(typ NotificationType, data interface{}) {
	// Generate and send the notification.
	n := Notification{Type: typ, Data: data}
	dag.notificationsLock.RLock()
	defer dag.notificationsLock.RUnlock()
	for _, callback := range dag.notifications {
		callback(&n)
	}
}

// BlockAddedNotificationData defines data to be sent along with a BlockAdded
// notification
type BlockAddedNotificationData struct {
	Block         *util.Block
	WasUnorphaned bool
}

// ChainChangedNotificationData defines data to be sent along with a ChainChanged
// notification
type ChainChangedNotificationData struct {
	RemovedChainBlockHashes []*daghash.Hash
	AddedChainBlockHashes   []*daghash.Hash
}

// FinalityConflictNotificationData defines data to be sent along with a
// FinalityConflict notification
type FinalityConflictNotificationData struct {
	ViolatingBlockHash *daghash.Hash
	ConflictTime       mstime.Time
}

// FinalityConflictResolvedNotificationData defines data to be sent along with a
// FinalityConflictResolved notification
type FinalityConflictResolvedNotificationData struct {
	FinalityBlockHash *daghash.Hash
	ResolutionTime    mstime.Time
}
