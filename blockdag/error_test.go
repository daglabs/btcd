// Copyright (c) 2014-2017 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package blockdag

import (
	"fmt"
	"testing"
)

// TestErrorCodeStringer tests the stringized output for the ErrorCode type.
func TestErrorCodeStringer(t *testing.T) {
	tests := []struct {
		in   ErrorCode
		want string
	}{
		{ErrDuplicateBlock, "ErrDuplicateBlock"},
		{ErrBlockTooBig, "ErrBlockTooBig"},
		{ErrBlockVersionTooOld, "ErrBlockVersionTooOld"},
		{ErrInvalidTime, "ErrInvalidTime"},
		{ErrTimeTooOld, "ErrTimeTooOld"},
		{ErrTimeTooNew, "ErrTimeTooNew"},
		{ErrNoParents, "ErrNoParents"},
		{ErrWrongParentsOrder, "ErrWrongParentsOrder"},
		{ErrDifficultyTooLow, "ErrDifficultyTooLow"},
		{ErrUnexpectedDifficulty, "ErrUnexpectedDifficulty"},
		{ErrHighHash, "ErrHighHash"},
		{ErrBadMerkleRoot, "ErrBadMerkleRoot"},
		{ErrBadCheckpoint, "ErrBadCheckpoint"},
		{ErrCheckpointTimeTooOld, "ErrCheckpointTimeTooOld"},
		{ErrNoTransactions, "ErrNoTransactions"},
		{ErrNoTxInputs, "ErrNoTxInputs"},
		{ErrTxTooBig, "ErrTxTooBig"},
		{ErrBadTxOutValue, "ErrBadTxOutValue"},
		{ErrDuplicateTxInputs, "ErrDuplicateTxInputs"},
		{ErrBadTxInput, "ErrBadTxInput"},
		{ErrBadCheckpoint, "ErrBadCheckpoint"},
		{ErrMissingTxOut, "ErrMissingTxOut"},
		{ErrUnfinalizedTx, "ErrUnfinalizedTx"},
		{ErrDuplicateTx, "ErrDuplicateTx"},
		{ErrOverwriteTx, "ErrOverwriteTx"},
		{ErrImmatureSpend, "ErrImmatureSpend"},
		{ErrSpendTooHigh, "ErrSpendTooHigh"},
		{ErrBadFees, "ErrBadFees"},
		{ErrTooManySigOps, "ErrTooManySigOps"},
		{ErrFirstTxNotCoinbase, "ErrFirstTxNotCoinbase"},
		{ErrMultipleCoinbases, "ErrMultipleCoinbases"},
		{ErrBadCoinbasePayloadLen, "ErrBadCoinbasePayloadLen"},
		{ErrBadCoinbaseTransaction, "ErrBadCoinbaseTransaction"},
		{ErrScriptMalformed, "ErrScriptMalformed"},
		{ErrScriptValidation, "ErrScriptValidation"},
		{ErrParentBlockUnknown, "ErrParentBlockUnknown"},
		{ErrInvalidAncestorBlock, "ErrInvalidAncestorBlock"},
		{ErrParentBlockNotCurrentTips, "ErrParentBlockNotCurrentTips"},
		{ErrWithDiff, "ErrWithDiff"},
		{ErrFinality, "ErrFinality"},
		{ErrTransactionsNotSorted, "ErrTransactionsNotSorted"},
		{ErrInvalidGas, "ErrInvalidGas"},
		{ErrInvalidPayload, "ErrInvalidPayload"},
		{ErrInvalidPayloadHash, "ErrInvalidPayloadHash"},
		{0xffff, "Unknown ErrorCode (65535)"},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
		result := test.in.String()
		if result != test.want {
			t.Errorf("String #%d\n got: %s want: %s", i, result,
				test.want)
			continue
		}
	}
}

// TestRuleError tests the error output for the RuleError type.
func TestRuleError(t *testing.T) {
	tests := []struct {
		in   RuleError
		want string
	}{
		{
			RuleError{Description: "duplicate block"},
			"duplicate block",
		},
		{
			RuleError{Description: "human-readable error"},
			"human-readable error",
		},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
		result := test.in.Error()
		if result != test.want {
			t.Errorf("Error #%d\n got: %s want: %s", i, result,
				test.want)
			continue
		}
	}
}

// TestDeploymentError tests the stringized output for the DeploymentError type.
func TestDeploymentError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in   DeploymentError
		want string
	}{
		{
			DeploymentError(0),
			"deployment ID 0 does not exist",
		},
		{
			DeploymentError(10),
			"deployment ID 10 does not exist",
		},
		{
			DeploymentError(123),
			"deployment ID 123 does not exist",
		},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
		result := test.in.Error()
		if result != test.want {
			t.Errorf("Error #%d\n got: %s want: %s", i, result,
				test.want)
			continue
		}
	}
}

func TestAssertError(t *testing.T) {
	message := "abc 123"
	err := AssertError(message)
	expectedMessage := fmt.Sprintf("assertion failed: %s", message)
	if expectedMessage != err.Error() {
		t.Errorf("Unexpected AssertError message. "+
			"Got: %s, want: %s", err.Error(), expectedMessage)
	}
}
