/*
Package stores acts as a layer between the internal representation of encrypted files and the encrypted files
themselves.

Subpackages implement serialization and deserialization to multiple formats.

This package defines the structure SOPS files should have and conversions to and from the internal representation. Part
of the purpose of this package is to make it easy to change the SOPS file format while remaining backwards-compatible.
*/
package stores

import (
	"fmt"
	"time"

	"github.com/getsops/sops/v3"
	"github.com/getsops/sops/v3/age"
	"github.com/getsops/sops/v3/azkv"
	"github.com/getsops/sops/v3/gcpkms"
	"github.com/getsops/sops/v3/hcvault"
	"github.com/getsops/sops/v3/kms"
	"github.com/getsops/sops/v3/ocikms"
	"github.com/getsops/sops/v3/pgp"
)

const (
	// SopsMetadataKey is the key used to store SOPS metadata at in SOPS encrypted files.
	SopsMetadataKey = "sops"
)

// SopsFile is a struct used by the stores as a helper to unmarshal the SOPS metadata
type SopsFile struct {
	// Metadata is a pointer so we can easily tell when the field is not present
	// in the SOPS file by checking for nil. This way we can show the user a
	// helpful error message indicating that the metadata wasn't found, instead
	// of showing a cryptic parsing error
	Metadata *Metadata `yaml:"sops" json:"sops" ini:"sops"`
}

// Metadata is stored in SOPS encrypted files, and it contains the information necessary to decrypt the file.
// This struct is just used for serialization, and SOPS uses another struct internally, sops.Metadata. It exists
// in order to allow the binary format to stay backwards compatible over time, but at the same time allow the internal
// representation SOPS uses to change over time.
type Metadata struct {
	ShamirThreshold           int         `yaml:"shamir_threshold,omitempty" json:"shamir_threshold,omitempty"`
	KeyGroups                 []keygroup  `yaml:"key_groups,omitempty" json:"key_groups,omitempty"`
	KMSKeys                   []kmskey    `yaml:"kms,omitempty" json:"kms,omitempty"`
	GCPKMSKeys                []gcpkmskey `yaml:"gcp_kms,omitempty" json:"gcp_kms,omitempty"`
	AzureKeyVaultKeys         []azkvkey   `yaml:"azure_kv,omitempty" json:"azure_kv,omitempty"`
	OCIKMSKeys                []ocikmskey `yaml:"oci_kms,omitempty" json:"oci_kms,omitempty"`
	VaultKeys                 []vaultkey  `yaml:"hc_vault,omitempty" json:"hc_vault,omitempty"`
	AgeKeys                   []agekey    `yaml:"age,omitempty" json:"age,omitempty"`
	LastModified              string      `yaml:"lastmodified" json:"lastmodified"`
	MessageAuthenticationCode string      `yaml:"mac" json:"mac"`
	PGPKeys                   []pgpkey    `yaml:"pgp,omitempty" json:"pgp,omitempty"`
	UnencryptedSuffix         string      `yaml:"unencrypted_suffix,omitempty" json:"unencrypted_suffix,omitempty"`
	EncryptedSuffix           string      `yaml:"encrypted_suffix,omitempty" json:"encrypted_suffix,omitempty"`
	UnencryptedRegex          string      `yaml:"unencrypted_regex,omitempty" json:"unencrypted_regex,omitempty"`
	EncryptedRegex            string      `yaml:"encrypted_regex,omitempty" json:"encrypted_regex,omitempty"`
	UnencryptedCommentRegex   string      `yaml:"unencrypted_comment_regex,omitempty" json:"unencrypted_comment_regex,omitempty"`
	EncryptedCommentRegex     string      `yaml:"encrypted_comment_regex,omitempty" json:"encrypted_comment_regex,omitempty"`
	MACOnlyEncrypted          bool        `yaml:"mac_only_encrypted,omitempty" json:"mac_only_encrypted,omitempty"`
	Version                   string      `yaml:"version" json:"version"`
}

type keygroup struct {
	PGPKeys           []pgpkey    `yaml:"pgp,omitempty" json:"pgp,omitempty"`
	KMSKeys           []kmskey    `yaml:"kms,omitempty" json:"kms,omitempty"`
	GCPKMSKeys        []gcpkmskey `yaml:"gcp_kms,omitempty" json:"gcp_kms,omitempty"`
	AzureKeyVaultKeys []azkvkey   `yaml:"azure_kv,omitempty" json:"azure_kv,omitempty"`
	OCIKMSKeys        []ocikmskey `yaml:"oci_kms,omitempty" json:"oci_kms,omitempty"`
	VaultKeys         []vaultkey  `yaml:"hc_vault" json:"hc_vault"`
	AgeKeys           []agekey    `yaml:"age" json:"age"`
}

type pgpkey struct {
	CreatedAt        string `yaml:"created_at" json:"created_at"`
	EncryptedDataKey string `yaml:"enc" json:"enc"`
	Fingerprint      string `yaml:"fp" json:"fp"`
}

type kmskey struct {
	Arn              string             `yaml:"arn" json:"arn"`
	Role             string             `yaml:"role,omitempty" json:"role,omitempty"`
	Context          map[string]*string `yaml:"context,omitempty" json:"context,omitempty"`
	CreatedAt        string             `yaml:"created_at" json:"created_at"`
	EncryptedDataKey string             `yaml:"enc" json:"enc"`
	AwsProfile       string             `yaml:"aws_profile" json:"aws_profile"`
}

type gcpkmskey struct {
	ResourceID       string `yaml:"resource_id" json:"resource_id"`
	CreatedAt        string `yaml:"created_at" json:"created_at"`
	EncryptedDataKey string `yaml:"enc" json:"enc"`
}

type vaultkey struct {
	VaultAddress     string `yaml:"vault_address" json:"vault_address"`
	EnginePath       string `yaml:"engine_path" json:"engine_path"`
	KeyName          string `yaml:"key_name" json:"key_name"`
	CreatedAt        string `yaml:"created_at" json:"created_at"`
	EncryptedDataKey string `yaml:"enc" json:"enc"`
}

type azkvkey struct {
	VaultURL         string `yaml:"vault_url" json:"vault_url"`
	Name             string `yaml:"name" json:"name"`
	Version          string `yaml:"version" json:"version"`
	CreatedAt        string `yaml:"created_at" json:"created_at"`
	EncryptedDataKey string `yaml:"enc" json:"enc"`
}

type ocikmskey struct {
	Id               string `yaml:"id" json:"id"`
	CrpytoEndpoint   string `yaml:"crypto_endpoint" json:"crypto_endpoint"`
	KeyVersion       string `yaml:"key_version" json:"key_version"`
	CreatedAt        string `yaml:"created_at" json:"created_at"`
	EncryptedDataKey string `yaml:"enc" json:"enc"`
}

type agekey struct {
	Recipient        string `yaml:"recipient" json:"recipient"`
	EncryptedDataKey string `yaml:"enc" json:"enc"`
}

// MetadataFromInternal converts an internal SOPS metadata representation to a representation appropriate for storage
func MetadataFromInternal(sopsMetadata sops.Metadata) Metadata {
	var m Metadata
	m.LastModified = sopsMetadata.LastModified.Format(time.RFC3339)
	m.UnencryptedSuffix = sopsMetadata.UnencryptedSuffix
	m.EncryptedSuffix = sopsMetadata.EncryptedSuffix
	m.UnencryptedRegex = sopsMetadata.UnencryptedRegex
	m.EncryptedRegex = sopsMetadata.EncryptedRegex
	m.UnencryptedCommentRegex = sopsMetadata.UnencryptedCommentRegex
	m.EncryptedCommentRegex = sopsMetadata.EncryptedCommentRegex
	m.MessageAuthenticationCode = sopsMetadata.MessageAuthenticationCode
	m.MACOnlyEncrypted = sopsMetadata.MACOnlyEncrypted
	m.Version = sopsMetadata.Version
	m.ShamirThreshold = sopsMetadata.ShamirThreshold
	if len(sopsMetadata.KeyGroups) == 1 {
		group := sopsMetadata.KeyGroups[0]
		m.PGPKeys = pgpKeysFromGroup(group)
		m.KMSKeys = kmsKeysFromGroup(group)
		m.GCPKMSKeys = gcpkmsKeysFromGroup(group)
		m.OCIKMSKeys = ocikmsKeysFromGroup(group)
		m.VaultKeys = vaultKeysFromGroup(group)
		m.AzureKeyVaultKeys = azkvKeysFromGroup(group)
		m.AgeKeys = ageKeysFromGroup(group)
	} else {
		for _, group := range sopsMetadata.KeyGroups {
			m.KeyGroups = append(m.KeyGroups, keygroup{
				KMSKeys:           kmsKeysFromGroup(group),
				PGPKeys:           pgpKeysFromGroup(group),
				GCPKMSKeys:        gcpkmsKeysFromGroup(group),
				OCIKMSKeys:        ocikmsKeysFromGroup(group),
				VaultKeys:         vaultKeysFromGroup(group),
				AzureKeyVaultKeys: azkvKeysFromGroup(group),
				AgeKeys:           ageKeysFromGroup(group),
			})
		}
	}
	return m
}

func pgpKeysFromGroup(group sops.KeyGroup) (keys []pgpkey) {
	for _, key := range group {
		switch key := key.(type) {
		case *pgp.MasterKey:
			keys = append(keys, pgpkey{
				Fingerprint:      key.Fingerprint,
				EncryptedDataKey: key.EncryptedKey,
				CreatedAt:        key.CreationDate.Format(time.RFC3339),
			})
		}
	}
	return
}

func kmsKeysFromGroup(group sops.KeyGroup) (keys []kmskey) {
	for _, key := range group {
		switch key := key.(type) {
		case *kms.MasterKey:
			keys = append(keys, kmskey{
				Arn:              key.Arn,
				CreatedAt:        key.CreationDate.Format(time.RFC3339),
				EncryptedDataKey: key.EncryptedKey,
				Context:          key.EncryptionContext,
				Role:             key.Role,
				AwsProfile:       key.AwsProfile,
			})
		}
	}
	return
}

func gcpkmsKeysFromGroup(group sops.KeyGroup) (keys []gcpkmskey) {
	for _, key := range group {
		switch key := key.(type) {
		case *gcpkms.MasterKey:
			keys = append(keys, gcpkmskey{
				ResourceID:       key.ResourceID,
				CreatedAt:        key.CreationDate.Format(time.RFC3339),
				EncryptedDataKey: key.EncryptedKey,
			})
		}
	}
	return
}

func ocikmsKeysFromGroup(group sops.KeyGroup) (keys []ocikmskey) {
	for _, key := range group {
		switch key := key.(type) {
		case *ocikms.MasterKey:
			keys = append(keys, ocikmskey{
				Id:               key.Id,
				CreatedAt:        key.CreationDate.Format(time.RFC3339),
				EncryptedDataKey: key.EncryptedKey,
				CrpytoEndpoint:   key.CryptoEndpoint,
				KeyVersion:       key.KeyVersionId,
			})
		}
	}
	return
}

func vaultKeysFromGroup(group sops.KeyGroup) (keys []vaultkey) {
	for _, key := range group {
		switch key := key.(type) {
		case *hcvault.MasterKey:
			keys = append(keys, vaultkey{
				VaultAddress:     key.VaultAddress,
				EnginePath:       key.EnginePath,
				KeyName:          key.KeyName,
				CreatedAt:        key.CreationDate.Format(time.RFC3339),
				EncryptedDataKey: key.EncryptedKey,
			})
		}
	}
	return
}

func azkvKeysFromGroup(group sops.KeyGroup) (keys []azkvkey) {
	for _, key := range group {
		switch key := key.(type) {
		case *azkv.MasterKey:
			keys = append(keys, azkvkey{
				VaultURL:         key.VaultURL,
				Name:             key.Name,
				Version:          key.Version,
				CreatedAt:        key.CreationDate.Format(time.RFC3339),
				EncryptedDataKey: key.EncryptedKey,
			})
		}
	}
	return
}

func ageKeysFromGroup(group sops.KeyGroup) (keys []agekey) {
	for _, key := range group {
		switch key := key.(type) {
		case *age.MasterKey:
			keys = append(keys, agekey{
				Recipient:        key.Recipient,
				EncryptedDataKey: key.EncryptedKey,
			})
		}
	}
	return
}

// ToInternal converts a storage-appropriate Metadata struct to a SOPS internal representation
func (m *Metadata) ToInternal() (sops.Metadata, error) {
	lastModified, err := time.Parse(time.RFC3339, m.LastModified)
	if err != nil {
		return sops.Metadata{}, err
	}
	groups, err := m.internalKeygroups()
	if err != nil {
		return sops.Metadata{}, err
	}

	cryptRuleCount := 0
	if m.UnencryptedSuffix != "" {
		cryptRuleCount++
	}
	if m.EncryptedSuffix != "" {
		cryptRuleCount++
	}
	if m.UnencryptedRegex != "" {
		cryptRuleCount++
	}
	if m.EncryptedRegex != "" {
		cryptRuleCount++
	}
	if m.UnencryptedCommentRegex != "" {
		cryptRuleCount++
	}
	if m.EncryptedCommentRegex != "" {
		cryptRuleCount++
	}

	if cryptRuleCount > 1 {
		return sops.Metadata{}, fmt.Errorf("Cannot use more than one of encrypted_suffix, unencrypted_suffix, encrypted_regex, unencrypted_regex, encrypted_comment_regex, or unencrypted_comment_regex in the same file")
	}

	if cryptRuleCount == 0 {
		m.UnencryptedSuffix = sops.DefaultUnencryptedSuffix
	}
	return sops.Metadata{
		KeyGroups:                 groups,
		ShamirThreshold:           m.ShamirThreshold,
		Version:                   m.Version,
		MessageAuthenticationCode: m.MessageAuthenticationCode,
		UnencryptedSuffix:         m.UnencryptedSuffix,
		EncryptedSuffix:           m.EncryptedSuffix,
		UnencryptedRegex:          m.UnencryptedRegex,
		EncryptedRegex:            m.EncryptedRegex,
		UnencryptedCommentRegex:   m.UnencryptedCommentRegex,
		EncryptedCommentRegex:     m.EncryptedCommentRegex,
		MACOnlyEncrypted:          m.MACOnlyEncrypted,
		LastModified:              lastModified,
	}, nil
}

func internalGroupFrom(kmsKeys []kmskey, pgpKeys []pgpkey, gcpKmsKeys []gcpkmskey, azkvKeys []azkvkey, ociKmsKeys []ocikmskey, vaultKeys []vaultkey, ageKeys []agekey) (sops.KeyGroup, error) {
	var internalGroup sops.KeyGroup
	for _, kmsKey := range kmsKeys {
		k, err := kmsKey.toInternal()
		if err != nil {
			return nil, err
		}
		internalGroup = append(internalGroup, k)
	}
	for _, gcpKmsKey := range gcpKmsKeys {
		k, err := gcpKmsKey.toInternal()
		if err != nil {
			return nil, err
		}
		internalGroup = append(internalGroup, k)
	}
	for _, azkvKey := range azkvKeys {
		k, err := azkvKey.toInternal()
		if err != nil {
			return nil, err
		}
		internalGroup = append(internalGroup, k)
	}
	for _, ociKmsKey := range ociKmsKeys {
		k, err := ociKmsKey.toInternal()
		if err != nil {
			return nil, err
		}
		internalGroup = append(internalGroup, k)
	}
	for _, vaultKey := range vaultKeys {
		k, err := vaultKey.toInternal()
		if err != nil {
			return nil, err
		}
		internalGroup = append(internalGroup, k)
	}
	for _, pgpKey := range pgpKeys {
		k, err := pgpKey.toInternal()
		if err != nil {
			return nil, err
		}
		internalGroup = append(internalGroup, k)
	}
	for _, ageKey := range ageKeys {
		k, err := ageKey.toInternal()
		if err != nil {
			return nil, err
		}
		internalGroup = append(internalGroup, k)
	}
	return internalGroup, nil
}

func (m *Metadata) internalKeygroups() ([]sops.KeyGroup, error) {
	var internalGroups []sops.KeyGroup
	if len(m.PGPKeys) > 0 || len(m.KMSKeys) > 0 || len(m.GCPKMSKeys) > 0 || len(m.OCIKMSKeys) > 0 || len(m.AzureKeyVaultKeys) > 0 || len(m.VaultKeys) > 0 || len(m.AgeKeys) > 0 {
		internalGroup, err := internalGroupFrom(m.KMSKeys, m.PGPKeys, m.GCPKMSKeys, m.AzureKeyVaultKeys, m.OCIKMSKeys, m.VaultKeys, m.AgeKeys)
		if err != nil {
			return nil, err
		}
		internalGroups = append(internalGroups, internalGroup)
		return internalGroups, nil
	} else if len(m.KeyGroups) > 0 {
		for _, group := range m.KeyGroups {
			internalGroup, err := internalGroupFrom(group.KMSKeys, group.PGPKeys, group.GCPKMSKeys, group.AzureKeyVaultKeys, group.OCIKMSKeys, group.VaultKeys, group.AgeKeys)
			if err != nil {
				return nil, err
			}
			internalGroups = append(internalGroups, internalGroup)
		}
		return internalGroups, nil
	} else {
		return nil, fmt.Errorf("No keys found in file")
	}
}

func (kmsKey *kmskey) toInternal() (*kms.MasterKey, error) {
	creationDate, err := time.Parse(time.RFC3339, kmsKey.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &kms.MasterKey{
		Role:              kmsKey.Role,
		EncryptionContext: kmsKey.Context,
		EncryptedKey:      kmsKey.EncryptedDataKey,
		CreationDate:      creationDate,
		Arn:               kmsKey.Arn,
		AwsProfile:        kmsKey.AwsProfile,
	}, nil
}

func (gcpKmsKey *gcpkmskey) toInternal() (*gcpkms.MasterKey, error) {
	creationDate, err := time.Parse(time.RFC3339, gcpKmsKey.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &gcpkms.MasterKey{
		ResourceID:   gcpKmsKey.ResourceID,
		EncryptedKey: gcpKmsKey.EncryptedDataKey,
		CreationDate: creationDate,
	}, nil
}

func (azkvKey *azkvkey) toInternal() (*azkv.MasterKey, error) {
	creationDate, err := time.Parse(time.RFC3339, azkvKey.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &azkv.MasterKey{
		VaultURL:     azkvKey.VaultURL,
		Name:         azkvKey.Name,
		Version:      azkvKey.Version,
		EncryptedKey: azkvKey.EncryptedDataKey,
		CreationDate: creationDate,
	}, nil
}

func (ociKmsKey *ocikmskey) toInternal() (*ocikms.MasterKey, error) {
	creationDate, err := time.Parse(time.RFC3339, ociKmsKey.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &ocikms.MasterKey{
		EncryptedKey:   ociKmsKey.EncryptedDataKey,
		CreationDate:   creationDate,
		Id:             ociKmsKey.Id,
		CryptoEndpoint: ociKmsKey.CrpytoEndpoint,
		KeyVersionId:   ociKmsKey.KeyVersion,
	}, nil
}

func (vaultKey *vaultkey) toInternal() (*hcvault.MasterKey, error) {
	creationDate, err := time.Parse(time.RFC3339, vaultKey.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &hcvault.MasterKey{
		VaultAddress: vaultKey.VaultAddress,
		EnginePath:   vaultKey.EnginePath,
		KeyName:      vaultKey.KeyName,
		CreationDate: creationDate,
		EncryptedKey: vaultKey.EncryptedDataKey,
	}, nil
}

func (pgpKey *pgpkey) toInternal() (*pgp.MasterKey, error) {
	creationDate, err := time.Parse(time.RFC3339, pgpKey.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &pgp.MasterKey{
		EncryptedKey: pgpKey.EncryptedDataKey,
		CreationDate: creationDate,
		Fingerprint:  pgpKey.Fingerprint,
	}, nil
}

func (ageKey *agekey) toInternal() (*age.MasterKey, error) {
	return &age.MasterKey{
		EncryptedKey: ageKey.EncryptedDataKey,
		Recipient:    ageKey.Recipient,
	}, nil
}

// ExampleComplexTree is an example sops.Tree object exhibiting complex relationships
var ExampleComplexTree = sops.Tree{
	Branches: sops.TreeBranches{
		sops.TreeBranch{
			sops.TreeItem{
				Key:   "hello",
				Value: `Welcome to SOPS! Edit this file as you please!`,
			},
			sops.TreeItem{
				Key:   "example_key",
				Value: "example_value",
			},
			sops.TreeItem{
				Key:   sops.Comment{Value: " Example comment"},
				Value: nil,
			},
			sops.TreeItem{
				Key: "example_array",
				Value: []interface{}{
					"example_value1",
					"example_value2",
				},
			},
			sops.TreeItem{
				Key:   "example_number",
				Value: 1234.56789,
			},
			sops.TreeItem{
				Key:   "example_booleans",
				Value: []interface{}{true, false},
			},
		},
	},
}

// ExampleSimpleTree is an example sops.Tree object exhibiting only simple relationships
// with only one nested branch and only simple string values
var ExampleSimpleTree = sops.Tree{
	Branches: sops.TreeBranches{
		sops.TreeBranch{
			sops.TreeItem{
				Key: "Welcome!",
				Value: sops.TreeBranch{
					sops.TreeItem{
						Key:   sops.Comment{Value: " This is an example file."},
						Value: nil,
					},
					sops.TreeItem{
						Key:   "hello",
						Value: "Welcome to SOPS! Edit this file as you please!",
					},
					sops.TreeItem{
						Key:   "example_key",
						Value: "example_value",
					},
				},
			},
		},
	},
}

// ExampleFlatTree is an example sops.Tree object exhibiting only simple relationships
// with no nested branches and only simple string values
var ExampleFlatTree = sops.Tree{
	Branches: sops.TreeBranches{
		sops.TreeBranch{
			sops.TreeItem{
				Key:   sops.Comment{Value: " This is an example file."},
				Value: nil,
			},
			sops.TreeItem{
				Key:   "hello",
				Value: "Welcome to SOPS! Edit this file as you please!",
			},
			sops.TreeItem{
				Key:   "example_key",
				Value: "example_value",
			},
			sops.TreeItem{
				Key:   "example_multiline",
				Value: "foo\nbar\nbaz",
			},
		},
	},
}

// HasSopsTopLevelKey returns true if the given branch has a top-level key called "sops".
func HasSopsTopLevelKey(branch sops.TreeBranch) bool {
	for _, b := range branch {
		if b.Key == SopsMetadataKey {
			return true
		}
	}
	return false
}
