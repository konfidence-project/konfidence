package ocm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"slices"

	"github.com/konfidence-project/konfidence/internal/kden/config"
	"github.com/konfidence-project/konfidence/internal/kden/log"

	ocmgenericspecv1 "ocm.software/open-component-model/bindings/go/configuration/generic/v1/spec"
	ocmcredentials "ocm.software/open-component-model/bindings/go/credentials"
	ocmdescriptorruntime "ocm.software/open-component-model/bindings/go/descriptor/runtime"
	ocmcompref "ocm.software/open-component-model/bindings/go/oci/compref"
	ocmrepositoryctfv1 "ocm.software/open-component-model/bindings/go/oci/spec/repository/v1/ctf"
	ocmsigningv1alpha1 "ocm.software/open-component-model/bindings/go/rsa/signing/v1alpha1"
	ocmruntime "ocm.software/open-component-model/bindings/go/runtime"
	"sigs.k8s.io/yaml"
)

type SigningProperties struct {
	ComponentVersion       string
	SignerSpecPath         string
	SignatureName          string
	DryRun                 bool
	NormalizationAlgorithm string
	HashAlgorithm          string
	OverwriteSignatures    bool
}

func Sign(ctx context.Context, signingProperties SigningProperties, ocmConfiguration *ocmgenericspecv1.Config) (*ocmdescriptorruntime.Signature, error) {
	pluginManager, err := ocmGetPluginManager(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get plugin manager: %w", err)
	}

	credentials, err := ocmGetCredentialGraph(ctx, pluginManager, ocmConfiguration)
	if err != nil {
		return nil, fmt.Errorf("failed to get credential graph: %w", err)
	}

	componentReference, err := ocmParseComponentReference(signingProperties.ComponentVersion, ocmcompref.WithCTFAccessMode(ocmrepositoryctfv1.AccessModeReadWrite))
	if err != nil {
		return nil, fmt.Errorf("failed to parse component version: %w", err)
	}

	repoResolver, err := ocmNewComponentRepositoryResolver(ctx, pluginManager.ComponentVersionRepositoryRegistry,
		credentials, WithConfig(ocmConfiguration), WithComponentRef(componentReference))
	if err != nil {
		return nil, fmt.Errorf("failed to initialize ocm repository: %w", err)
	}

	repo, err := repoResolver.GetComponentVersionRepositoryForComponent(ctx, componentReference.Component, componentReference.Version)
	if err != nil {
		return nil, fmt.Errorf("failed to access ocm repository: %w", err)
	}

	artifactCv, err := repo.GetComponentVersion(ctx, componentReference.Component, componentReference.Version)
	if err != nil {
		return nil, fmt.Errorf("failed to get component version: %w", err)
	}

	signerSpec, err := loadSignerSpec(signingProperties.SignerSpecPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load signer specification: %w", err)
	}

	signingHandler, err := ocmGetSigningHandler(ctx, pluginManager.SigningRegistry, signerSpec)
	if err != nil {
		return nil, fmt.Errorf("failed to get signature handler: %w", err)
	}

	sigExists := func(sig ocmdescriptorruntime.Signature) bool { return sig.Name == signingProperties.SignatureName }

	if slices.ContainsFunc(artifactCv.Signatures, sigExists) {
		if !signingProperties.OverwriteSignatures {
			return nil, fmt.Errorf("signature %q already exists", signingProperties.SignatureName)
		}
		log.Info("overwriting existing signature", "name", signingProperties.SignatureName)
	}

	unsignedDigest, err := ocmGenerateDigestForSigning(
		ctx, artifactCv, slog.Default(),
		signingProperties.NormalizationAlgorithm,
		signingProperties.HashAlgorithm,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate digest: %w", err)
	}

	var foundCreds ocmruntime.Typed
	if consumerID, err := signingHandler.GetSigningCredentialConsumerIdentity(ctx, signingProperties.SignatureName, *unsignedDigest, signerSpec); err == nil {
		if creds, err := credentials.Resolve(ctx, consumerID); err == nil {
			foundCreds = creds
			log.Debug("using discovered credentials", "type", foundCreds.GetType())
		} else {
			if errors.Is(err, ocmcredentials.ErrNotFound) {
				log.Debug("failed to resolve credentials", "error", err.Error())
			} else {
				return nil, fmt.Errorf("failed to resolve signing credentials: %w", err)
			}
		}
	}

	sigBytes, err := signingHandler.Sign(ctx, *unsignedDigest, signerSpec, foundCreds)
	if err != nil {
		return nil, fmt.Errorf("failed to sign component descriptor: %w", err)
	}

	signature := ocmdescriptorruntime.Signature{
		Name:      signingProperties.SignatureName,
		Digest:    *unsignedDigest,
		Signature: sigBytes,
	}

	if signingProperties.DryRun {
		log.Info("dry run enabled: signature not persisted")
		return &signature, nil
	}

	if idx := slices.IndexFunc(artifactCv.Signatures, sigExists); idx >= 0 {
		artifactCv.Signatures[idx] = signature
	} else {
		artifactCv.Signatures = append(artifactCv.Signatures, signature)
	}

	if err := repo.AddComponentVersion(ctx, artifactCv); err != nil {
		return nil, fmt.Errorf("failed to update component version: %w", err)
	}

	log.Info("signed successfully",
		"name", signingProperties.SignatureName,
		"digest", unsignedDigest.Value,
		"hashAlgorithm", unsignedDigest.HashAlgorithm,
		"normalisationAlgorithm", unsignedDigest.NormalisationAlgorithm,
	)
	return &signature, nil

}

func loadSignerSpec(path string) (_ ocmruntime.Typed, err error) {
	if path == "" {
		spec := &ocmsigningv1alpha1.Config{
			SignatureAlgorithm:      ocmsigningv1alpha1.AlgorithmRSASSAPSS,
			SignatureEncodingPolicy: ocmsigningv1alpha1.SignatureEncodingPolicyPlain,
		}
		log.Info("no signer spec file provided, using default", "algorithm", spec.SignatureAlgorithm, "encodingPolicy", spec.SignatureEncodingPolicy)
		_, _ = ocmsigningv1alpha1.Scheme.DefaultType(spec)
		return spec, nil
	}

	data, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read signer spec %q: %w", path, err)
	}
	defer func() {
		err = errors.Join(err, data.Close())
	}()

	scheme := ocmruntime.NewScheme(ocmruntime.WithAllowUnknown())
	raw := &ocmruntime.Raw{}
	if err := scheme.Decode(data, raw); err != nil {
		return nil, fmt.Errorf("failed to decode signer spec %q: %w", path, err)
	}
	return raw, nil
}

func PrintSignature(output io.Writer, sig ocmdescriptorruntime.Signature) error {
	signature := ocmdescriptorruntime.ConvertToV2Signature(&sig)
	var bytes []byte
	var err error
	switch config.Config.Output {
	case "json":
		if bytes, err = json.MarshalIndent(signature, "", "  "); err != nil {
			return fmt.Errorf("failed to marshall signature to json: %w", err)
		}
		_, err = fmt.Fprintln(output, string(bytes))
	case "yaml", "pretty":
		if bytes, err = yaml.Marshal(signature); err != nil {
			return fmt.Errorf("failed to marshall signature to yaml: %w", err)
		}
		_, err = fmt.Fprintln(output, string(bytes))
	}

	return err
}
