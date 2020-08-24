package library

import (
	"fmt"
	"strings"

	"github.com/Masterminds/semver/v3"
	"golang.org/x/xerrors"

	"github.com/aquasecurity/trivy-db/pkg/db"
	dbTypes "github.com/aquasecurity/trivy-db/pkg/types"
	"github.com/aquasecurity/trivy/pkg/scanner/utils"
	"github.com/aquasecurity/trivy/pkg/types"
)

type Advisory struct {
	lang     string
	comparer comparer
}

func NewAdvisory(lang string) *Advisory {
	return &Advisory{
		lang:     lang,
		comparer: newComparer(lang),
	}
}

func (s *Advisory) DetectVulnerabilities(pkgName string, pkgVer *semver.Version) ([]types.DetectedVulnerability, error) {
	prefix := fmt.Sprintf("%s::", s.lang)
	advisories, err := db.Config{}.GetAdvisories(prefix, pkgName)
	if err != nil {
		return nil, xerrors.Errorf("failed to get %s advisories: %w", s.lang, err)
	}

	var vulns []types.DetectedVulnerability
	for _, advisory := range advisories {
		if !s.comparer.isVulnerable(pkgVer, advisory) {
			continue
		}

		vuln := types.DetectedVulnerability{
			VulnerabilityID:  advisory.VulnerabilityID,
			PkgName:          pkgName,
			InstalledVersion: pkgVer.String(),
			FixedVersion:     s.createFixedVersions(advisory),
		}
		vulns = append(vulns, vuln)
	}

	return vulns, nil
}

func (s *Advisory) createFixedVersions(advisory dbTypes.Advisory) string {
	if len(advisory.PatchedVersions) != 0 {
		return strings.Join(advisory.PatchedVersions, ", ")
	}

	var fixedVersions []string
	for _, version := range advisory.VulnerableVersions {
		for _, s := range strings.Split(version, ",") {
			s = strings.TrimSpace(s)
			if !strings.HasPrefix(s, "<=") && strings.HasPrefix(s, "<") {
				s = strings.TrimPrefix(s, "<")
				fixedVersions = append(fixedVersions, strings.TrimSpace(s))
			}
		}
	}
	return strings.Join(fixedVersions, ", ")
}

type comparer interface {
	isVulnerable(pkgVer *semver.Version, advisory dbTypes.Advisory) bool
}

func newComparer(lang string) comparer {
	switch lang {
	case "java":
		// TODO
	}
	return generalComparer{}
}

type generalComparer struct{}

func (c generalComparer) isVulnerable(pkgVer *semver.Version, advisory dbTypes.Advisory) bool {
	if len(advisory.VulnerableVersions) != 0 {
		return utils.MatchVersions(pkgVer, advisory.VulnerableVersions)
	}

	if utils.MatchVersions(pkgVer, advisory.PatchedVersions) ||
		utils.MatchVersions(pkgVer, advisory.UnaffectedVersions) {
		return false
	}

	return true
}