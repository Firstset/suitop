package checkpoint

// IsValidatorSigned checks if the validator at a given index signed the checkpoint
// by checking if the index is present in the bitmap (list of signing validator indices).
func IsValidatorSigned(bitmap []uint32, validatorIndex int) bool {
	if validatorIndex < 0 {
		// Validator indices are typically non-negative.
		return false
	}
	targetIndex := uint32(validatorIndex)
	for _, indexInBitmap := range bitmap {
		if indexInBitmap == targetIndex {
			return true
		}
	}
	return false
}

// Add other bitmap helper functions here if needed in the future.
