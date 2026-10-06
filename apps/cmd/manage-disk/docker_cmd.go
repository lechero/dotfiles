package main

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/lechero/dotfiles/apps/internal/disk/clean"
	"github.com/lechero/dotfiles/apps/internal/disk/docker"
	"github.com/lechero/dotfiles/apps/internal/disk/human"
)

// listDocker prints Docker's disk, every image, container, volume and slice of
// build cache with its verdict and the evidence for it, and the cheapest way to
// target free. It removes nothing; measuring the disk runs one throwaway
// container.
func listDocker(w io.Writer, env *clean.Env, target int64) error {
	rep := clean.AuditDocker(context.Background(), env)
	if rep.Err != "" {
		fmt.Fprintln(w, rep.Err+".")
		return nil
	}
	if d := rep.Disk; d.OK() {
		fmt.Fprintf(w, "Docker's disk: %s free of %s (%.0f%% full)\n",
			human.Bytes(d.Free), human.Bytes(d.Total), 100*float64(d.Used())/float64(d.Total))
	} else {
		fmt.Fprintf(w, "Docker's disk: not measured (%s)\n", d.Err)
	}
	var removable int64
	for _, r := range rep.Resources {
		if r.Verdict.Removable() {
			removable += r.Size
		}
	}
	fmt.Fprintf(w, "%d resources, about %s removable", len(rep.Resources), human.Bytes(removable))
	if rep.SharedCache > 0 {
		fmt.Fprintf(w, "; %s more build cache is shared with images", human.Bytes(rep.SharedCache))
	}
	fmt.Fprintln(w)

	for v := docker.Unused; v >= docker.InUse; v-- {
		var group []docker.Resource
		var size int64
		for _, r := range rep.Resources {
			if r.Verdict == v {
				group = append(group, r)
				size += r.Size
			}
		}
		if len(group) == 0 {
			continue
		}
		fmt.Fprintf(w, "\n%s (%d, %s) — %s\n", strings.ToUpper(v.String()), len(group), human.Bytes(size), dockerMeaning[v])
		for _, r := range group {
			fmt.Fprintf(w, "  %-11s %-52s %10s\n", r.Kind, truncate(r.Name, 52), human.Bytes(r.Size))
			fmt.Fprintf(w, "  %-11s %s\n", "", strings.Join(r.Reasons, "; "))
		}
	}
	printPlan(w, rep, docker.Suggest(rep, target))
	return nil
}

func printPlan(w io.Writer, rep docker.Report, p docker.Plan) {
	target := human.Bytes(p.Target)
	switch {
	case p.Err != "":
		fmt.Fprintf(w, "\nNo plan for %s free: %s\n", target, p.Err)
		return
	case p.Reached():
		fmt.Fprintf(w, "\nDocker's disk has the %s target free.\n", target)
		return
	}
	byKey := map[string]docker.Resource{}
	for _, r := range rep.Resources {
		byKey[r.Key()] = r
	}
	fmt.Fprintf(w, "\nTo reach %s free (%s to go), the cheapest removals:\n", target, human.Bytes(p.Need()))
	for _, pk := range p.Picks {
		r := byKey[pk.Key]
		fmt.Fprintf(w, "  %-11s %-52s %10s  %s\n", r.Kind, truncate(r.Name, 52), human.Bytes(pk.Frees), pk.Cost)
	}
	if p.Short == 0 {
		fmt.Fprintf(w, "That frees about %s → about %s free. In the Docker tab, s picks these.\n",
			human.Bytes(p.Frees), human.Bytes(p.Free+p.Frees))
		return
	}
	fmt.Fprintf(w, "Everything but volumes frees about %s: still %s short.\n", human.Bytes(p.Frees), human.Bytes(p.Short))
	if len(p.Volumes) > 0 {
		fmt.Fprintln(w, "The largest volumes, toward the rest; their data is gone for good, so they are yours to decide:")
		for i, k := range p.Volumes {
			if i == 5 {
				fmt.Fprintf(w, "  … and %d smaller\n", len(p.Volumes)-5)
				break
			}
			r := byKey[k]
			fmt.Fprintf(w, "  %-11s %-52s %10s  %s\n", r.Kind, truncate(r.Name, 52), human.Bytes(r.Size), strings.Join(r.Reasons, "; "))
		}
	}
	fmt.Fprintln(w, "Or raise Docker's disk limit: Docker Desktop → Settings → Resources → Virtual disk limit.")
}

var dockerMeaning = map[docker.Verdict]string{
	docker.Unused: "nothing refers to it: untagged images, stale build cache, empty volumes",
	docker.Orphan: "its compose project's folder is gone",
	docker.Old:    "nothing has used it for 30+ days, and it can come back",
	docker.Review: "removable, but it looks wanted or only a rebuild brings it back",
	docker.Data:   "a volume: its data exists nowhere else, so it is never picked for you",
	docker.InUse:  "a running container, or what one runs from or mounts",
}
