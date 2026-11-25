package transaction

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/models"
	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/providers"
)

// OfferDescriptor holds backend-authoritative execution parameters & expiration metadata
// so no raw CLI instructions are ever trusted or executed from frontend callers.
type OfferDescriptor struct {
	OfferID         string
	AppID           string
	PackageName     string
	Provider        string
	Repository      string
	Architecture    string
	Branch          string
	Channel         string
	Version         string
	Scope           string
	SignatureStatus string
	Dependencies    []string
	IsEligible      bool
	IsInstalled     bool
	InstalledVer    string
	RegisteredAt    time.Time
	ExpiresAt       time.Time
	MetadataEpoch   int64
}

// ValidateOffer checks all invariants required before dispatching a system package transaction.
func ValidateOffer(rec OfferDescriptor, exists bool, now time.Time, op TransactionOp, caps map[string]providers.ProviderCapability, expectedEpoch int64) (string, error) {
	if !exists {
		return "unknown_offer", fmt.Errorf("unknown or unregistered offerId %q; transaction rejected", rec.OfferID)
	}
	if rec.PackageName == "" || rec.Provider == "" {
		return "invalid_offer", fmt.Errorf("registered offer %q lacks verified package identity", rec.OfferID)
	}
	if !rec.ExpiresAt.IsZero() && now.After(rec.ExpiresAt) {
		return "offer_expired", fmt.Errorf("offer %q expired at %s; refresh application metadata before executing %s", rec.OfferID, rec.ExpiresAt.Format(time.RFC3339), op)
	}
	if expectedEpoch > 0 && rec.MetadataEpoch > 0 && rec.MetadataEpoch < expectedEpoch {
		return "offer_stale", fmt.Errorf("offer %q belongs to superseded metadata epoch (%d < %d); refresh metadata", rec.OfferID, rec.MetadataEpoch, expectedEpoch)
	}
	if !rec.IsEligible {
		return "offer_ineligible", fmt.Errorf("offer %q is marked ineligible on this host", rec.OfferID)
	}
	if capInfo, ok := caps[strings.ToLower(rec.Provider)]; ok {
		if okOp, reason := capInfo.CanPerform(string(op)); !okOp {
			return "provider_unavailable", fmt.Errorf("%s", reason)
		}
	}
	return "", nil
}

// executeProviderOp dispatches structured arguments to the backend provider adapter without shell interpolation.
func executeProviderOp(ctx context.Context, adapter providers.Provider, op TransactionOp, rec OfferDescriptor) (models.Result, int, error) {
	pkg := models.Package{
		Name:     rec.PackageName,
		Version:  rec.Version,
		Provider: rec.Provider,
		Source:   rec.Repository,
		Repo:     rec.Repository,
		Arch:     rec.Architecture,
		Branch:   rec.Branch,
		Channel:  rec.Channel,
		Scope:    rec.Scope,
	}

	var (
		res models.Result
		err error
	)
	switch op {
	case OpInstall:
		res, err = adapter.Install(ctx, pkg)
	case OpRemove:
		res, err = adapter.Remove(ctx, pkg)
	case OpUpdate:
		if scoped, ok := adapter.(providers.ScopedUpdater); ok && pkg.Name != "" && pkg.Name != "system-update" && pkg.Name != "aur-update" {
			res, err = scoped.UpdatePackage(ctx, pkg)
		} else {
			res, err = adapter.Update(ctx)
		}
	default:
		return models.Result{}, 1, fmt.Errorf("unsupported operation %q", op)
	}

	exitCode := 0
	if err != nil || !res.Success {
		exitCode = 1
	}
	return res, exitCode, err
}

func classifyErrorCode(msg string) string {
	low := strings.ToLower(msg)
	switch {
	case strings.Contains(low, "lock") || strings.Contains(low, "db.lck"):
		return "lock_held"
	case strings.Contains(low, "conflict") || strings.Contains(low, "dependency") || strings.Contains(low, "dependencies"):
		return "dependency_conflict"
	case strings.Contains(low, "permission") || strings.Contains(low, "sudo") || strings.Contains(low, "polkit") || strings.Contains(low, "auth"):
		return "auth_failed"
	case strings.Contains(low, "timeout") || strings.Contains(low, "connection") || strings.Contains(low, "resolve host"):
		return "network_error"
	default:
		return "transaction_failed"
	}
}

