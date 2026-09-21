package check

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/veighnsche/can-lang/compiler/internal/ir"
	"github.com/veighnsche/can-lang/compiler/internal/project"
	"github.com/veighnsche/can-lang/compiler/internal/syntax"
)

const assetURL = "can.std.asset@1::url"

// resolveAssetName looks only at the calling project's manifest. A name that
// exists solely in another project is unowned; a name absent from the graph is
// missing. Both are html::invalid_url reasons, not filesystem lookups.
func resolveAssetName(graph *project.Graph, owner *project.Project, name string) ir.AssetResolution {
	if owner != nil {
		for _, asset := range owner.CheckedAssets {
			if asset.Name == name {
				return ir.AssetResolution{URL: asset.URL}
			}
		}
	}
	if graph != nil {
		for _, candidate := range graph.Projects {
			if owner != nil && candidate == owner {
				continue
			}
			for _, asset := range candidate.CheckedAssets {
				if asset.Name == name {
					return ir.AssetResolution{Reason: "unowned"}
				}
			}
		}
	}
	return ir.AssetResolution{Reason: "missing"}
}

func (c *regionChecker) resolveAsset(args []syntax.Argument) (ir.AssetResolution, error) {
	if c.context.Asset == nil {
		return ir.AssetResolution{}, fmt.Errorf("asset lookup requires its project manifest")
	}
	if len(args) != 1 || args[0].Spread || args[0].Group != nil {
		return ir.AssetResolution{}, fmt.Errorf("asset name must be a static manifest key")
	}
	literal, ok := fetchUngroup(args[0].Value).(*syntax.LiteralExpr)
	if !ok || literal.Token.Kind != syntax.String {
		return ir.AssetResolution{}, fmt.Errorf("asset name must be a static manifest key")
	}
	name := literal.Token.Value
	if name == "" || !utf8.ValidString(name) || strings.ContainsRune(name, 0) || strings.ContainsAny(name, "/\\.") || strings.Contains(name, "..") {
		return ir.AssetResolution{}, fmt.Errorf("asset name must be a static manifest key")
	}
	return c.context.Asset(name), nil
}
