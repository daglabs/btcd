// Copyright (c) 2013-2017 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package txscript

import (
	"fmt"
	"math/big"

	"github.com/daglabs/btcd/btcec"
	"github.com/daglabs/btcd/wire"
)

// ScriptFlags is a bitmask defining additional operations or tests that will be
// done when executing a script pair.
type ScriptFlags uint32

const (

	// ScriptNoFlags is used when you want to use ScriptFlags without raising any flags
	ScriptNoFlags ScriptFlags = 0

	// ScriptDiscourageUpgradableNops defines whether to verify that
	// NOP1 through NOP10 are reserved for future soft-fork upgrades.  This
	// flag must not be used for consensus critical code nor applied to
	// blocks as this flag is only for stricter standard transaction
	// checks.  This flag is only applied when the above opcodes are
	// executed.
	ScriptDiscourageUpgradableNops ScriptFlags = 1 << iota
)

const (
	// MaxStackSize is the maximum combined height of stack and alt stack
	// during execution.
	MaxStackSize = 244

	// MaxScriptSize is the maximum allowed length of a raw script.
	MaxScriptSize = 10000
)

// halforder is used to tame ECDSA malleability (see BIP0062).
var halfOrder = new(big.Int).Rsh(btcec.S256().N, 1)

// Engine is the virtual machine that executes scripts.
type Engine struct {
	scripts         [][]parsedOpcode
	scriptIdx       int
	scriptOff       int
	dstack          stack // data stack
	astack          stack // alt stack
	tx              wire.MsgTx
	txIdx           int
	condStack       []int
	numOps          int
	flags           ScriptFlags
	sigCache        *SigCache
	isP2SH          bool     // treat execution as pay-to-script-hash
	savedFirstStack [][]byte // stack from first script for ps2h scripts
}

// hasFlag returns whether the script engine instance has the passed flag set.
func (vm *Engine) hasFlag(flag ScriptFlags) bool {
	return vm.flags&flag == flag
}

// isBranchExecuting returns whether or not the current conditional branch is
// actively executing.  For example, when the data stack has an OP_FALSE on it
// and an OP_IF is encountered, the branch is inactive until an OP_ELSE or
// OP_ENDIF is encountered.  It properly handles nested conditionals.
func (vm *Engine) isBranchExecuting() bool {
	if len(vm.condStack) == 0 {
		return true
	}
	return vm.condStack[len(vm.condStack)-1] == OpCondTrue
}

// executeOpcode peforms execution on the passed opcode.  It takes into account
// whether or not it is hidden by conditionals, but some rules still must be
// tested in this case.
func (vm *Engine) executeOpcode(pop *parsedOpcode) error {
	// Disabled opcodes are fail on program counter.
	if pop.isDisabled() {
		str := fmt.Sprintf("attempt to execute disabled opcode %s",
			pop.opcode.name)
		return scriptError(ErrDisabledOpcode, str)
	}

	// Always-illegal opcodes are fail on program counter.
	if pop.alwaysIllegal() {
		str := fmt.Sprintf("attempt to execute reserved opcode %s",
			pop.opcode.name)
		return scriptError(ErrReservedOpcode, str)
	}

	// Note that this includes OP_RESERVED which counts as a push operation.
	if pop.opcode.value > Op16 {
		vm.numOps++
		if vm.numOps > MaxOpsPerScript {
			str := fmt.Sprintf("exceeded max operation limit of %d",
				MaxOpsPerScript)
			return scriptError(ErrTooManyOperations, str)
		}

	} else if len(pop.data) > MaxScriptElementSize {
		str := fmt.Sprintf("element size %d exceeds max allowed size %d",
			len(pop.data), MaxScriptElementSize)
		return scriptError(ErrElementTooBig, str)
	}

	// Nothing left to do when this is not a conditional opcode and it is
	// not in an executing branch.
	if !vm.isBranchExecuting() && !pop.isConditional() {
		return nil
	}

	// Ensure all executed data push opcodes use the minimal encoding when
	// the minimal data verification flag is set.
	if vm.isBranchExecuting() &&
		pop.opcode.value >= 0 && pop.opcode.value <= OpPushData4 {

		if err := pop.checkMinimalDataPush(); err != nil {
			return err
		}
	}

	return pop.opcode.opfunc(pop, vm)
}

// disasm is a helper function to produce the output for DisasmPC and
// DisasmScript.  It produces the opcode prefixed by the program counter at the
// provided position in the script.  It does no error checking and leaves that
// to the caller to provide a valid offset.
func (vm *Engine) disasm(scriptIdx int, scriptOff int) string {
	return fmt.Sprintf("%02x:%04x: %s", scriptIdx, scriptOff,
		vm.scripts[scriptIdx][scriptOff].print(false))
}

// validPC returns an error if the current script position is valid for
// execution, nil otherwise.
func (vm *Engine) validPC() error {
	if vm.scriptIdx >= len(vm.scripts) {
		str := fmt.Sprintf("past input scripts %v:%v %v:xxxx",
			vm.scriptIdx, vm.scriptOff, len(vm.scripts))
		return scriptError(ErrInvalidProgramCounter, str)
	}
	if vm.scriptOff >= len(vm.scripts[vm.scriptIdx]) {
		str := fmt.Sprintf("past input scripts %v:%v %v:%04d",
			vm.scriptIdx, vm.scriptOff, vm.scriptIdx,
			len(vm.scripts[vm.scriptIdx]))
		return scriptError(ErrInvalidProgramCounter, str)
	}
	return nil
}

// curPC returns either the current script and offset, or an error if the
// position isn't valid.
func (vm *Engine) curPC() (script int, off int, err error) {
	err = vm.validPC()
	if err != nil {
		return 0, 0, err
	}
	return vm.scriptIdx, vm.scriptOff, nil
}

// DisasmPC returns the string for the disassembly of the opcode that will be
// next to execute when Step() is called.
func (vm *Engine) DisasmPC() (string, error) {
	scriptIdx, scriptOff, err := vm.curPC()
	if err != nil {
		return "", err
	}
	return vm.disasm(scriptIdx, scriptOff), nil
}

// DisasmScript returns the disassembly string for the script at the requested
// offset index.  Index 0 is the signature script and 1 is the public key
// script.
func (vm *Engine) DisasmScript(idx int) (string, error) {
	if idx < 0 {
		str := fmt.Sprintf("script index %d >= total scripts %d", idx,
			len(vm.scripts))
		return "", scriptError(ErrInvalidIndex, str)
	}
	if idx >= len(vm.scripts) {
		str := fmt.Sprintf("script index %d >= total scripts %d", idx,
			len(vm.scripts))
		return "", scriptError(ErrInvalidIndex, str)
	}

	var disstr string
	for i := range vm.scripts[idx] {
		disstr = disstr + vm.disasm(idx, i) + "\n"
	}
	return disstr, nil
}

// CheckErrorCondition returns nil if the running script has ended and was
// successful, leaving a a true boolean on the stack.  An error otherwise,
// including if the script has not finished.
func (vm *Engine) CheckErrorCondition(finalScript bool) error {
	// Check execution is actually done.  When pc is past the end of script
	// array there are no more scripts to run.
	if vm.scriptIdx < len(vm.scripts) {
		return scriptError(ErrScriptUnfinished,
			"error check when script unfinished")
	}

	if finalScript {
		if vm.dstack.Depth() > 1 {
			str := fmt.Sprintf("stack contains %d unexpected items",
				vm.dstack.Depth()-1)
			return scriptError(ErrCleanStack, str)
		} else if vm.dstack.Depth() < 1 {
			return scriptError(ErrEmptyStack,
				"stack empty at end of script execution")
		}
	}

	v, err := vm.dstack.PopBool()
	if err != nil {
		return err
	}
	if !v {
		// Log interesting data.
		log.Tracef("%v", newLogClosure(func() string {
			dis0, _ := vm.DisasmScript(0)
			dis1, _ := vm.DisasmScript(1)
			return fmt.Sprintf("scripts failed: script0: %s\n"+
				"script1: %s", dis0, dis1)
		}))
		return scriptError(ErrEvalFalse,
			"false stack entry at end of script execution")
	}
	return nil
}

// Step will execute the next instruction and move the program counter to the
// next opcode in the script, or the next script if the current has ended.  Step
// will return true in the case that the last opcode was successfully executed.
//
// The result of calling Step or any other method is undefined if an error is
// returned.
func (vm *Engine) Step() (done bool, err error) {
	// Verify that it is pointing to a valid script address.
	err = vm.validPC()
	if err != nil {
		return true, err
	}
	opcode := &vm.scripts[vm.scriptIdx][vm.scriptOff]
	vm.scriptOff++

	// Execute the opcode while taking into account several things such as
	// disabled opcodes, illegal opcodes, maximum allowed operations per
	// script, maximum script element sizes, and conditionals.
	err = vm.executeOpcode(opcode)
	if err != nil {
		return true, err
	}

	// The number of elements in the combination of the data and alt stacks
	// must not exceed the maximum number of stack elements allowed.
	combinedStackSize := vm.dstack.Depth() + vm.astack.Depth()
	if combinedStackSize > MaxStackSize {
		str := fmt.Sprintf("combined stack size %d > max allowed %d",
			combinedStackSize, MaxStackSize)
		return false, scriptError(ErrStackOverflow, str)
	}

	// Prepare for next instruction.
	if vm.scriptOff >= len(vm.scripts[vm.scriptIdx]) {
		// Illegal to have an `if' that straddles two scripts.
		if err == nil && len(vm.condStack) != 0 {
			return false, scriptError(ErrUnbalancedConditional,
				"end of script reached in conditional execution")
		}

		// Alt stack doesn't persist.
		_ = vm.astack.DropN(vm.astack.Depth())

		vm.numOps = 0 // number of ops is per script.
		vm.scriptOff = 0
		if vm.scriptIdx == 0 && vm.isP2SH {
			vm.scriptIdx++
			vm.savedFirstStack = vm.GetStack()
		} else if vm.scriptIdx == 1 && vm.isP2SH {
			// Put us past the end for CheckErrorCondition()
			vm.scriptIdx++
			// Check script ran successfully and pull the script
			// out of the first stack and execute that.
			err := vm.CheckErrorCondition(false)
			if err != nil {
				return false, err
			}

			script := vm.savedFirstStack[len(vm.savedFirstStack)-1]
			pops, err := parseScript(script)
			if err != nil {
				return false, err
			}
			vm.scripts = append(vm.scripts, pops)

			// Set stack to be the stack from first script minus the
			// script itself
			vm.SetStack(vm.savedFirstStack[:len(vm.savedFirstStack)-1])
		} else {
			vm.scriptIdx++
		}
		// there are zero length scripts in the wild
		if vm.scriptIdx < len(vm.scripts) && vm.scriptOff >= len(vm.scripts[vm.scriptIdx]) {
			vm.scriptIdx++
		}
		if vm.scriptIdx >= len(vm.scripts) {
			return true, nil
		}
	}
	return false, nil
}

// Execute will execute all scripts in the script engine and return either nil
// for successful validation or an error if one occurred.
func (vm *Engine) Execute() (err error) {
	done := false
	for !done {
		log.Tracef("%v", newLogClosure(func() string {
			dis, err := vm.DisasmPC()
			if err != nil {
				return fmt.Sprintf("stepping (%v)", err)
			}
			return fmt.Sprintf("stepping %v", dis)
		}))

		done, err = vm.Step()
		if err != nil {
			return err
		}
		log.Tracef("%v", newLogClosure(func() string {
			var dstr, astr string

			// if we're tracing, dump the stacks.
			if vm.dstack.Depth() != 0 {
				dstr = "Stack:\n" + vm.dstack.String()
			}
			if vm.astack.Depth() != 0 {
				astr = "AltStack:\n" + vm.astack.String()
			}

			return dstr + astr
		}))
	}

	return vm.CheckErrorCondition(true)
}

// currentScript returns the script currently being processed.
func (vm *Engine) currentScript() []parsedOpcode {
	return vm.scripts[vm.scriptIdx]
}

// checkHashTypeEncoding returns whether or not the passed hashtype adheres to
// the strict encoding requirements if enabled.
func (vm *Engine) checkHashTypeEncoding(hashType SigHashType) error {
	sigHashType := hashType & ^SigHashAnyOneCanPay
	if sigHashType < SigHashAll || sigHashType > SigHashSingle {
		str := fmt.Sprintf("invalid hash type 0x%x", hashType)
		return scriptError(ErrInvalidSigHashType, str)
	}
	return nil
}

// checkPubKeyEncoding returns whether or not the passed public key adheres to
// the strict encoding requirements if enabled.
func (vm *Engine) checkPubKeyEncoding(pubKey []byte) error {
	if len(pubKey) == 33 && (pubKey[0] == 0x02 || pubKey[0] == 0x03) {
		// Compressed
		return nil
	}
	if len(pubKey) == 65 && pubKey[0] == 0x04 {
		// Uncompressed
		return nil
	}

	return scriptError(ErrPubKeyFormat, "unsupported public key type")
}

// checkSignatureEncoding returns whether or not the passed signature adheres to
// the strict encoding requirements if enabled.
func (vm *Engine) checkSignatureEncoding(sig []byte) error {

	// The format of a DER encoded signature is as follows:
	//
	// 0x30 <total length> 0x02 <length of R> <R> 0x02 <length of S> <S>
	//   - 0x30 is the ASN.1 identifier for a sequence
	//   - Total length is 1 byte and specifies length of all remaining data
	//   - 0x02 is the ASN.1 identifier that specifies an integer follows
	//   - Length of R is 1 byte and specifies how many bytes R occupies
	//   - R is the arbitrary length big-endian encoded number which
	//     represents the R value of the signature.  DER encoding dictates
	//     that the value must be encoded using the minimum possible number
	//     of bytes.  This implies the first byte can only be null if the
	//     highest bit of the next byte is set in order to prevent it from
	//     being interpreted as a negative number.
	//   - 0x02 is once again the ASN.1 integer identifier
	//   - Length of S is 1 byte and specifies how many bytes S occupies
	//   - S is the arbitrary length big-endian encoded number which
	//     represents the S value of the signature.  The encoding rules are
	//     identical as those for R.

	// Minimum length is when both numbers are 1 byte each.
	// 0x30 + <1-byte> + 0x02 + 0x01 + <byte> + 0x2 + 0x01 + <byte>
	if len(sig) < 8 {
		// Too short
		str := fmt.Sprintf("malformed signature: too short: %d < 8",
			len(sig))
		return scriptError(ErrSigDER, str)
	}

	// Maximum length is when both numbers are 33 bytes each.  It is 33
	// bytes because a 256-bit integer requires 32 bytes and an additional
	// leading null byte might required if the high bit is set in the value.
	// 0x30 + <1-byte> + 0x02 + 0x21 + <33 bytes> + 0x2 + 0x21 + <33 bytes>
	if len(sig) > 72 {
		// Too long
		str := fmt.Sprintf("malformed signature: too long: %d > 72",
			len(sig))
		return scriptError(ErrSigDER, str)
	}
	if sig[0] != 0x30 {
		// Wrong type
		str := fmt.Sprintf("malformed signature: format has wrong "+
			"type: 0x%x", sig[0])
		return scriptError(ErrSigDER, str)
	}
	if int(sig[1]) != len(sig)-2 {
		// Invalid length
		str := fmt.Sprintf("malformed signature: bad length: %d != %d",
			sig[1], len(sig)-2)
		return scriptError(ErrSigDER, str)
	}

	rLen := int(sig[3])

	// Make sure S is inside the signature.
	if rLen+5 > len(sig) {
		return scriptError(ErrSigDER,
			"malformed signature: S out of bounds")
	}

	sLen := int(sig[rLen+5])

	// The length of the elements does not match the length of the
	// signature.
	if rLen+sLen+6 != len(sig) {
		return scriptError(ErrSigDER,
			"malformed signature: invalid R length")
	}

	// R elements must be integers.
	if sig[2] != 0x02 {
		return scriptError(ErrSigDER,
			"malformed signature: missing first integer marker")
	}

	// Zero-length integers are not allowed for R.
	if rLen == 0 {
		return scriptError(ErrSigDER,
			"malformed signature: R length is zero")
	}

	// R must not be negative.
	if sig[4]&0x80 != 0 {
		return scriptError(ErrSigDER,
			"malformed signature: R value is negative")
	}

	// Null bytes at the start of R are not allowed, unless R would
	// otherwise be interpreted as a negative number.
	if rLen > 1 && sig[4] == 0x00 && sig[5]&0x80 == 0 {
		return scriptError(ErrSigDER,
			"malformed signature: invalid R value")
	}

	// S elements must be integers.
	if sig[rLen+4] != 0x02 {
		return scriptError(ErrSigDER,
			"malformed signature: missing second integer marker")
	}

	// Zero-length integers are not allowed for S.
	if sLen == 0 {
		return scriptError(ErrSigDER,
			"malformed signature: S length is zero")
	}

	// S must not be negative.
	if sig[rLen+6]&0x80 != 0 {
		return scriptError(ErrSigDER,
			"malformed signature: S value is negative")
	}

	// Null bytes at the start of S are not allowed, unless S would
	// otherwise be interpreted as a negative number.
	if sLen > 1 && sig[rLen+6] == 0x00 && sig[rLen+7]&0x80 == 0 {
		return scriptError(ErrSigDER,
			"malformed signature: invalid S value")
	}

	// Verify the S value is <= half the order of the curve.  This check is
	// done because when it is higher, the complement modulo the order can
	// be used instead which is a shorter encoding by 1 byte.  Further,
	// without enforcing this, it is possible to replace a signature in a
	// valid transaction with the complement while still being a valid
	// signature that verifies.  This would result in changing the
	// transaction hash and thus is source of malleability.
	sValue := new(big.Int).SetBytes(sig[rLen+6 : rLen+6+sLen])
	if sValue.Cmp(halfOrder) > 0 {
		return scriptError(ErrSigHighS,
			"signature is not canonical due to "+
				"unnecessarily high S value")
	}

	return nil
}

// getStack returns the contents of stack as a byte array bottom up
func getStack(stack *stack) [][]byte {
	array := make([][]byte, stack.Depth())
	for i := range array {
		// PeekByteArry can't fail due to overflow, already checked
		array[len(array)-i-1], _ = stack.PeekByteArray(int32(i))
	}
	return array
}

// setStack sets the stack to the contents of the array where the last item in
// the array is the top item in the stack.
func setStack(stack *stack, data [][]byte) {
	// This can not error. Only errors are for invalid arguments.
	_ = stack.DropN(stack.Depth())

	for i := range data {
		stack.PushByteArray(data[i])
	}
}

// GetStack returns the contents of the primary stack as an array. where the
// last item in the array is the top of the stack.
func (vm *Engine) GetStack() [][]byte {
	return getStack(&vm.dstack)
}

// SetStack sets the contents of the primary stack to the contents of the
// provided array where the last item in the array will be the top of the stack.
func (vm *Engine) SetStack(data [][]byte) {
	setStack(&vm.dstack, data)
}

// GetAltStack returns the contents of the alternate stack as an array where the
// last item in the array is the top of the stack.
func (vm *Engine) GetAltStack() [][]byte {
	return getStack(&vm.astack)
}

// SetAltStack sets the contents of the alternate stack to the contents of the
// provided array where the last item in the array will be the top of the stack.
func (vm *Engine) SetAltStack(data [][]byte) {
	setStack(&vm.astack, data)
}

// NewEngine returns a new script engine for the provided public key script,
// transaction, and input index.  The flags modify the behavior of the script
// engine according to the description provided by each flag.
func NewEngine(scriptPubKey []byte, tx *wire.MsgTx, txIdx int, flags ScriptFlags,
	sigCache *SigCache) (*Engine, error) {

	// The provided transaction input index must refer to a valid input.
	if txIdx < 0 || txIdx >= len(tx.TxIn) {
		str := fmt.Sprintf("transaction input index %d is negative or "+
			">= %d", txIdx, len(tx.TxIn))
		return nil, scriptError(ErrInvalidIndex, str)
	}
	scriptSig := tx.TxIn[txIdx].SignatureScript

	// When both the signature script and public key script are empty the
	// result is necessarily an error since the stack would end up being
	// empty which is equivalent to a false top element.  Thus, just return
	// the relevant error now as an optimization.
	if len(scriptSig) == 0 && len(scriptPubKey) == 0 {
		return nil, scriptError(ErrEvalFalse,
			"false stack entry at end of script execution")
	}

	vm := Engine{flags: flags, sigCache: sigCache}

	parsedScriptSig, err := parseScriptAndVerifySize(scriptSig)
	if err != nil {
		return nil, err
	}
	// The signature script must only contain data pushes
	if !isPushOnly(parsedScriptSig) {
		return nil, scriptError(ErrNotPushOnly,
			"signature script is not push only")
	}

	parsedScriptPubKey, err := parseScriptAndVerifySize(scriptPubKey)
	if err != nil {
		return nil, err
	}

	// The engine stores the scripts in parsed form using a slice.  This
	// allows multiple scripts to be executed in sequence.  For example,
	// with a pay-to-script-hash transaction, there will be ultimately be
	// a third script to execute.
	vm.scripts = [][]parsedOpcode{parsedScriptSig, parsedScriptPubKey}

	// Advance the program counter to the public key script if the signature
	// script is empty since there is nothing to execute for it in that
	// case.
	if len(scriptSig) == 0 {
		vm.scriptIdx++
	}

	if isScriptHash(vm.scripts[1]) {
		// Only accept input scripts that push data for P2SH.
		if !isPushOnly(vm.scripts[0]) {
			return nil, scriptError(ErrNotPushOnly,
				"pay to script hash is not push only")
		}
		vm.isP2SH = true
	}

	vm.tx = *tx
	vm.txIdx = txIdx

	return &vm, nil
}

func parseScriptAndVerifySize(script []byte) ([]parsedOpcode, error) {
	if len(script) > MaxScriptSize {
		str := fmt.Sprintf("script size %d is larger than max "+
			"allowed size %d", len(script), MaxScriptSize)
		return nil, scriptError(ErrScriptTooBig, str)
	}
	return parseScript(script)
}
