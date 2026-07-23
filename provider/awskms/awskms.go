// Package awskms unwraps JWE content keys with AWS KMS for the "aws-kms" scheme.
package awskms

import (
	"context"
	"fmt"
	"os"

	"decryptor/provider"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kms"
)

const scheme = "aws-kms"

func init() {
	provider.Register(scheme, awsKMS{})
}

// awsKMS unwraps CEKs with AWS KMS. keyID is the key ARN, which carries the
// region; RSA-OAEP-256 matches the JWE "alg".
type awsKMS struct{}

func (awsKMS) Unwrapper(ctx context.Context, keyID string) (provider.UnwrapFunc, error) {
	parsedArn, err := arn.Parse(keyID)
	if err != nil {
		return nil, fmt.Errorf("parse AWS KMS key ARN %q: %w", keyID, err)
	}

	sess, err := session.NewSession(&aws.Config{Region: aws.String(parsedArn.Region)})
	if err != nil {
		return nil, err
	}
	if endpoint := os.Getenv("DEV_AWS_ENDPOINT"); endpoint != "" {
		sess.Config.Endpoint = aws.String(endpoint)
	}
	svc := kms.New(sess)

	return func(encryptedKey []byte) ([]byte, error) {
		out, err := svc.DecryptWithContext(ctx, &kms.DecryptInput{
			KeyId:               aws.String(keyID),
			CiphertextBlob:      encryptedKey,
			EncryptionAlgorithm: aws.String(kms.EncryptionAlgorithmSpecRsaesOaepSha256),
		})
		if err != nil {
			return nil, fmt.Errorf("unwrap content key via AWS KMS: %w", err)
		}
		return out.Plaintext, nil
	}, nil
}
