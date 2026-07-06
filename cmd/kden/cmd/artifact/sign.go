package artifact

import (
	"crypto"
	"fmt"

	"github.com/konfidence-project/konfidence/internal/kden/ocm"
	"github.com/spf13/cobra"
)

var (
	signerSpecFlagName             = "signer-spec"
	signatureNameFlagName          = "signature-name"
	dryRunFlagName                 = "dry-run"
	normalizationAlgorithmFlagName = "normalization-algorithm"
	hashAlgorithmFlagName          = "hash-algorithm"
	overWriteSignaturesFlagName    = "overwrite-signatures"
)

var (
	Sign           = ocm.Sign
	PrintSignature = ocm.PrintSignature
)

func NewSignCmd() *cobra.Command {
	var signCmd = &cobra.Command{
		Use:   "sign",
		Short: "Sign artifact",
		Long:  `Sign an artifact with a signer specification.`,
		Args:  cobra.ExactArgs(1),
		RunE:  sign,
	}
	signCmd.Flags().String(signerSpecFlagName, "", "Path to signer specification file")
	signCmd.Flags().String(signatureNameFlagName, "default", "Name of the signature to use")
	signCmd.Flags().Bool(dryRunFlagName, false, "If enabled, the signature will not be persisted")
	signCmd.Flags().String(normalizationAlgorithmFlagName, "jsonNormalisation/v4alpha1", "Normalization algorithm to use")
	signCmd.Flags().String(hashAlgorithmFlagName, crypto.SHA256.String(), "Hash algorithm to use. Supported values are: "+getSupportedHashingAlgorithms())
	signCmd.Flags().Bool(overWriteSignaturesFlagName, false, "Overwrite if a signature with the same name exists")

	return signCmd
}

func sign(cmd *cobra.Command, args []string) error {
	ocmConfig, err := GetOcmConfiguration(cmd)
	if err != nil {
		return err
	}

	dryRun, err := cmd.Flags().GetBool(dryRunFlagName)
	if err != nil {
		return fmt.Errorf("failed to read dry run flag: %w", err)
	}

	overWriteSignatures, err := cmd.Flags().GetBool(overWriteSignaturesFlagName)
	if err != nil {
		return fmt.Errorf("failed to read overwrite signatures flag: %w", err)
	}

	signingProperties := ocm.SigningProperties{
		ComponentVersion:       args[0],
		SignerSpecPath:         cmd.Flag(signerSpecFlagName).Value.String(),
		SignatureName:          cmd.Flag(signatureNameFlagName).Value.String(),
		DryRun:                 dryRun,
		NormalizationAlgorithm: cmd.Flag(normalizationAlgorithmFlagName).Value.String(),
		HashAlgorithm:          cmd.Flag(hashAlgorithmFlagName).Value.String(),
		OverwriteSignatures:    overWriteSignatures,
	}
	signature, err := Sign(cmd.Context(), signingProperties, ocmConfig)
	if err != nil {
		return fmt.Errorf("failed to sign artifact: %w", err)
	}

	err = PrintSignature(cmd.OutOrStdout(), *signature)
	if err != nil {
		return fmt.Errorf("failed to print signature: %w", err)
	}
	return nil
}

func getSupportedHashingAlgorithms() string {
	return fmt.Sprintf("%s, %s", crypto.SHA512, crypto.SHA256)
}
