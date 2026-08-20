package payments

import "go.uber.org/zap"

// RazorpayProcessor struct
type RazorpayProcessor struct {
    KeyID     string
    KeySecret string
    logger    *zap.Logger
}

func NewRazorpayProcessor(keyID, keySecret string, logger *zap.Logger) *RazorpayProcessor {
    return &RazorpayProcessor{
        KeyID:     keyID,
        KeySecret: keySecret,
        logger:    logger,
    }
}

func (r *RazorpayProcessor) Ping() error {
    return nil
}
