package emit

import (
	"encoding/json"
	"fmt"

	"github.com/veighnsche/can-lang/compiler/internal/types"
)

// cryptoOperationBindings maps the concrete password and crypto
// operations to their state-module targets.
func cryptoOperationBindings() bindingContribution {
	functions := map[string]string{
		"can.std.password@1::hash":                   "$canPasswords.hash",
		"can.std.password@1::verify":                 "$canPasswords.verify",
		"can.std.crypto@1::hmac_sha256":              "$canCrypto.hmacSha256",
		"can.std.crypto@1::generate_aes_key":         "$canCryptoKeys.generateAESKey",
		"can.std.crypto@1::generate_ed25519_keypair": "$canCryptoKeys.generateEd25519Keypair",
		"can.std.crypto@1::import_ed25519_public":    "$canCryptoKeys.importEd25519Public",
		"can.std.crypto@1::export_ed25519_public":    "$canCryptoKeys.exportEd25519Public",
		"can.std.crypto@1::encrypt_aes_gcm":          "$canCrypto.encryptAesGcm",
		"can.std.crypto@1::encrypt_aes_gcm_sealed":   "$canCrypto.encryptAesGcmSealed",
		"can.std.crypto@1::decrypt_aes_gcm":          "$canCrypto.decryptAesGcm",
		"can.std.crypto@1::sign_ed25519":             "$canCrypto.signEd25519",
		"can.std.crypto@1::verify_ed25519":           "$canCrypto.verifyEd25519",
	}
	return bindingContribution{domain: "crypto", functions: functions}
}

// cryptoStateImports lists the password, key, and primitive factory
// modules the shared state module needs.
func (assembly *programAssembly) cryptoStateImports(runtime string) []ModuleImport {
	return []ModuleImport{
		{Target: runtime + "/platform/crypto/password.ts", Names: []ImportName{{"createPasswords", "$canCreatePasswords"}}},
		{Target: runtime + "/platform/crypto/keys.ts", Names: []ImportName{{"createCryptoKeys", "$canCreateCryptoKeys"}, {"isCryptoKeyValue", "$canIsCryptoKey"}}},
		{Target: runtime + "/platform/crypto/primitives.ts", Names: []ImportName{{"createCryptoPrimitives", "$canCreateCryptoPrimitives"}}},
	}
}

// cryptoStateValueImportNames lists the crypto factory values authored
// and assertion modules import from the state module.
func cryptoStateValueImportNames() []ImportName {
	return []ImportName{{"$canPasswords", "$canPasswords"}, {"$canCryptoKeys", "$canCryptoKeys"}, {"$canCrypto", "$canCrypto"}}
}

// declareCryptoState emits the password, key, and primitive factory
// bindings.
func (builder *stateBuilder) declareCryptoState() {
	builder.out.WriteString("export let $canPasswords:ReturnType<typeof $canCreatePasswords>;\nexport let $canCryptoKeys:ReturnType<typeof $canCreateCryptoKeys>;\nexport let $canCrypto:ReturnType<typeof $canCreateCryptoPrimitives>;\n")
}

// initializeCryptoState creates the password, key, and primitive
// factories inside the shared initializer, after the domain runtime
// exists.
func (builder *stateBuilder) initializeCryptoState() {
	fmt.Fprintf(&builder.out, "$canPasswords=$canCreatePasswords($canDomain,{costRejected:%s,invalidHash:%s});\n", quote(builder.numberIDs["can.std.password@1::cost_rejected"]), quote(builder.numberIDs["can.std.password@1::invalid_hash"]))
	fmt.Fprintf(&builder.out, "$canCryptoKeys=$canCreateCryptoKeys($canDomain,{keyMisuse:%s,invalidKey:%s,keypair:%s});\n", quote(builder.numberIDs["can.std.crypto@1::key_misuse"]), quote(builder.numberIDs["can.std.crypto@1::invalid_key"]), quote(builder.numberIDs["can.std.crypto@1::keypair"]))
	fmt.Fprintf(&builder.out, "$canCrypto=$canCreateCryptoPrimitives($canDomain,{keyMisuse:%s,invalidKey:%s,invalidNonce:%s,decryptFailed:%s,sealed:%s});\n", quote(builder.numberIDs["can.std.crypto@1::key_misuse"]), quote(builder.numberIDs["can.std.crypto@1::invalid_key"]), quote(builder.numberIDs["can.std.crypto@1::invalid_nonce"]), quote(builder.numberIDs["can.std.crypto@1::decrypt_failed"]), quote(builder.numberIDs["can.std.crypto@1::sealed"]))
}

// emitCryptoKinds emits the opaque-handle kind table the domain predicate
// uses to recognize key values at boundaries.
func (builder *stateBuilder) emitCryptoKinds() error {
	cryptoKinds := map[string]string{}
	for _, typ := range builder.assembly.program.Model.Types() {
		if typ.Kind() != types.Opaque {
			continue
		}
		if typ.Declaration() == "can.std.crypto@1::key" {
			cryptoKinds[typ.Identity()] = "key"
		}
	}
	cryptoKindsJSON, err := json.Marshal(cryptoKinds)
	if err != nil {
		return err
	}
	fmt.Fprintf(&builder.out, "const $canCryptoKinds:Readonly<Record<string,string>>=%s;\n", cryptoKindsJSON)
	return nil
}
