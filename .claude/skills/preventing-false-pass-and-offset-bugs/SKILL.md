---
name: preventing-false-pass-and-offset-bugs
description: Mandatory verification rules for offset calculations (BOM, frontmatter, section ranges) and preventing false PASS assertions from empty mocks or zero-iteration test loops. Use whenever writing or reviewing tests, offset logic, parity checks, or external data comparisons.
---

# Preventing False Pass & Offset Bugs

This skill defines mandatory rules for test design and offset verification in Gobsidian.

## 1. Disk-Level Verification for Byte Offsets

When writing or reviewing code that manipulates byte offsets (`StripBOM`, `bodyOffset`, `Heading.Start`/`End`, `Block.Start`/`End`):

- **Memory-only AST tests are insufficient**: Verifying `note.BOM == true` or that a Heading struct exists in memory DOES NOT prove that `vault.ReadRange` on disk will fetch the correct bytes.
- **Mandatory disk slice check**: Tests for section or range reading **MUST** invoke the physical reader (`vault.ReadRange` or `os.ReadFile` slicing) on a real file on disk and compare the exact byte slice against expected content.
- **BOM Offset Alignment Rule**: When a file on disk begins with a UTF-8 BOM (`\xEF\xBB\xBF`), the AST parser processes the stripped buffer (`len(body) = len(raw) - 3`). Any offset returned by the parser (`Heading.Start`, etc.) is relative to the stripped body. When fetching range reads from the raw file on disk, **3 bytes MUST be added back** to align with the raw file on disk.

```go
// REQUIRED TEST PATTERN FOR OFFSET/BOM VERIFICATION:
func TestReadSectionWithBOMOnDisk(t *testing.T) {
	root := t.TempDir()
	rawContent := []byte("\xEF\xBB\xBF# Title\n\n## Section\n\nTarget content\n")
	filePath := filepath.Join(root, "bom_test.md")
	os.WriteFile(filePath, rawContent, 0644)

	v, _ := vault.New(root)
	idx := index.New()
	idx.Build(context.Background(), v)

	// Fetch range read using computed offsets
	content, err := v.ReadRange(context.Background(), "bom_test.md", startOffset, endOffset)
	if err != nil {
		t.Fatalf("ReadRange: %v", err)
	}

	// Must compare exact string sliced from raw disk file
	if string(content) != "## Section\n\nTarget content\n" {
		t.Errorf("Sliced content mismatch!\nGot:  %q\nWant: %q", string(content), "## Section\n\nTarget content\n")
	}
}
```

---

## 2. Preventing False "PASS" in Parity and Integration Tests

When writing or reviewing integration tests, golden file harnesses, or parity checkers:

- **No Empty Mock Files**: NEVER create empty JSON objects (`{}`) or empty directories just to make a test suite run without errors.
- **Explicit Artifact Requirement (`t.Skip`)**: If human-generated reference artifacts (e.g. `metadata.json` from the Obsidian parity dumper) are absent, the test MUST call `t.Skip("reference missing...")`.
- **Zero-Iteration Loop Prevention**: If a test iterates over a map or slice of expected test cases (`for path, want := range reference`), the test **MUST assert `len(reference) > 0`** before the loop!

```go
// REQUIRED PATTERN FOR REFERENCE / PARITY TESTS:
func TestParityWithObsidian(t *testing.T) {
	refPath := filepath.Join("testdata", "parity", "metadata.json")
	refData, err := os.ReadFile(refPath)
	if err != nil || len(refData) == 0 || string(bytes.TrimSpace(refData)) == "{}" {
		t.Skip("Corpus de paridade ausente ou vazio; veja tools/parity-dumper/README.md")
	}

	var ref map[string]ObsidianMetadata
	json.Unmarshal(refData, &ref)

	if len(ref) == 0 {
		t.Skip("Corpus de paridade sem entradas válidas; ignorando teste.")
	}

	// Now 100% guaranteed to run at least one real assertion
	for path, want := range ref {
		// ... assertions with detailed failure messages ...
	}
}
```
