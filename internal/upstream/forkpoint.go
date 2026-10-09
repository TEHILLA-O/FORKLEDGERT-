package upstream

import (
	"context"
	"fmt"
	"strings"

	"github.com/TEHILLA-O/forkledger/internal/evidence"
	"github.com/TEHILLA-O/forkledger/internal/git"
)

type ForkPoint struct {
	Commit     string              `json:"commit,omitempty"`
	MergeBases []string            `json:"merge_bases,omitempty"`
	Baseline   string              `json:"baseline,omitempty"`
	Confidence evidence.Assessment `json:"confidence"`
}

func EstimateForkPoint(ctx context.Context, repo *git.Repository, head, upstreamRef string) (ForkPoint, error) {
	fp := ForkPoint{}
	var items []evidence.Item

	bases, err := repo.MergeBases(ctx, head, upstreamRef)
	if err != nil {
		fp.Confidence = evidence.FromWeights("No merge-base. The histories may be unrelated, which is how detached copies start.", nil)
		return fp, fmt.Errorf("merge-base: %w", err)
	}
	if len(bases) == 0 {
		fp.Confidence = evidence.FromWeights("Git found no shared ancestor.", nil)
		return fp, fmt.Errorf("no shared ancestor between HEAD and %s", upstreamRef)
	}
	fp.MergeBases = bases
	fp.Commit = bases[0]
	items = append(items, evidence.New(evidence.KindGitAncestry, "git merge-base HEAD "+upstreamRef).
		WithSource(head).WithTarget(fp.Commit).WithWeight(55))
	if len(bases) > 1 {
		items = append(items, evidence.New(evidence.KindGitAncestry,
			fmt.Sprintf("Git found %d merge-bases. Using %s and keeping the others as evidence.", len(bases), short(fp.Commit))).
			WithWeight(5))
	}

	if tag, err := repo.DescribeNearestTag(ctx, fp.Commit); err == nil && tag != "" {
		fp.Baseline = strings.TrimPrefix(tag, "v")
		items = append(items, evidence.New(evidence.KindReleaseTag, "nearest tag at the fork point is "+tag).
			WithTarget(tag).WithWeight(20))
	}

	fp.Confidence = evidence.FromWeights("Fork point comes from git merge-base, which is ancestry, not similarity.", items)
	return fp, nil
}

func short(hash string) string {
	if len(hash) > 7 {
		return hash[:7]
	}
	return hash
}
