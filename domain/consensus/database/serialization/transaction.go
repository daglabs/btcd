package serialization

import (
	"github.com/kaspanet/kaspad/domain/consensus/model/externalapi"
)

func DomainTransactionToDbTransaction(domainTransaction *externalapi.DomainTransaction) *DbTransaction {
	dbInputs := make([]*DbTransactionInput, len(domainTransaction.Inputs))
	for i, domainTransactionInput := range domainTransaction.Inputs {
		dbInputs[i] = &DbTransactionInput{
			PreviousOutpoint: DomainOutpointToDbOutpoint(&domainTransactionInput.PreviousOutpoint),
			SignatureScript:  domainTransactionInput.SignatureScript,
			Sequence:         domainTransactionInput.Sequence,
		}
	}

	dbOutputs := make([]*DbTransactionOutput, len(domainTransaction.Outputs))
	for i, domainTransactionOutput := range domainTransaction.Outputs {
		dbOutputs[i] = &DbTransactionOutput{
			Value:           domainTransactionOutput.Value,
			ScriptPublicKey: domainTransactionOutput.ScriptPublicKey,
		}
	}

	return &DbTransaction{
		Version:      domainTransaction.Version,
		Inputs:       dbInputs,
		Outputs:      dbOutputs,
		LockTime:     domainTransaction.LockTime,
		SubnetworkID: DomainSubnetworkIDToDbSubnetworkId(&domainTransaction.SubnetworkID),
		Gas:          domainTransaction.Gas,
		PayloadHash:  DomainHashToDbHash(&domainTransaction.PayloadHash),
		Payload:      domainTransaction.Payload,
	}
}

func DbTransactionToDomainTransaction(dbTransaction *DbTransaction) (*externalapi.DomainTransaction, error) {
	domainSubnetworkID, err := DbSubnetworkIdToDomainSubnetworkID(dbTransaction.SubnetworkID)
	if err != nil {
		return nil, err
	}
	domainPayloadHash, err := DbHashToDomainHash(dbTransaction.PayloadHash)
	if err != nil {
		return nil, err
	}

	domainInputs := make([]*externalapi.DomainTransactionInput, len(dbTransaction.Inputs))
	for i, dbTransactionInput := range dbTransaction.Inputs {
		domainPreviousOutpoint, err := DbOutpointToDomainOutpoint(dbTransactionInput.PreviousOutpoint)
		if err != nil {
			return nil, err
		}
		domainInputs[i] = &externalapi.DomainTransactionInput{
			PreviousOutpoint: *domainPreviousOutpoint,
			SignatureScript:  dbTransactionInput.SignatureScript,
			Sequence:         dbTransactionInput.Sequence,
		}
	}

	domainOutputs := make([]*externalapi.DomainTransactionOutput, len(dbTransaction.Outputs))
	for i, dbTransactionOutput := range dbTransaction.Outputs {
		domainOutputs[i] = &externalapi.DomainTransactionOutput{
			Value:           dbTransactionOutput.Value,
			ScriptPublicKey: dbTransactionOutput.ScriptPublicKey,
		}
	}

	return &externalapi.DomainTransaction{
		Version:      dbTransaction.Version,
		Inputs:       domainInputs,
		Outputs:      domainOutputs,
		LockTime:     dbTransaction.LockTime,
		SubnetworkID: *domainSubnetworkID,
		Gas:          dbTransaction.Gas,
		PayloadHash:  *domainPayloadHash,
		Payload:      dbTransaction.Payload,
	}, nil
}
